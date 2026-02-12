package api

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// PendingLogin stores the state of a login that passed password check but awaits 2FA.
type PendingLogin struct {
	UserID   int
	Username string
}

// FIDO2RegisterRequest is the request body for starting FIDO2 registration.
type FIDO2RegisterRequest struct {
	DeviceName string `json:"device_name"`
}

// FIDO2AuthBeginRequest is the request body for starting FIDO2 authentication.
type FIDO2AuthBeginRequest struct {
	PendingLoginToken string `json:"pending_login_token"`
}

// FIDO2AuthFinishRequest is the request body for finishing FIDO2 authentication.
type FIDO2AuthFinishRequest struct {
	PendingLoginToken string `json:"pending_login_token"`
}

// generatePendingLoginToken creates a cryptographically random token.
func generatePendingLoginToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// --- Protected endpoints (behind authMiddleware) ---

// beginFIDO2Registration handles POST /api/2fa/fido2/register/begin
func (s *Server) beginFIDO2Registration(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := s.authService.GetUserByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get user details")
		return
	}

	creation, err := s.fido2Service.BeginRegistration(userID, user.Username)
	if err != nil {
		s.logger().Error("fido2 begin registration failed", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "failed to start registration")
		return
	}

	respondJSON(w, http.StatusOK, creation)
}

// finishFIDO2Registration handles POST /api/2fa/fido2/register/finish
func (s *Server) finishFIDO2Registration(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Read device_name from query parameter (body is the WebAuthn response)
	deviceName := r.URL.Query().Get("device_name")
	if deviceName == "" {
		deviceName = "Security Key"
	}

	user, err := s.authService.GetUserByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get user details")
		return
	}

	result, err := s.fido2Service.FinishRegistration(userID, user.Username, deviceName, r)
	if err != nil {
		s.logger().Error("fido2 finish registration failed", slog.Any("error", err))
		respondError(w, http.StatusBadRequest, "failed to complete registration")
		return
	}

	response := map[string]any{
		"credential_id": result.CredentialID,
	}
	if len(result.BackupCodes) > 0 {
		response["backup_codes"] = result.BackupCodes
	}

	respondJSON(w, http.StatusOK, response)
}

// listFIDO2Credentials handles GET /api/2fa/fido2/credentials
func (s *Server) listFIDO2Credentials(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	creds, err := s.fido2Service.ListCredentials(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list credentials")
		return
	}

	respondJSON(w, http.StatusOK, creds)
}

// deleteFIDO2Credential handles DELETE /api/2fa/fido2/credentials/{id}
func (s *Server) deleteFIDO2Credential(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	credIDStr := chi.URLParam(r, "id")
	credID, err := strconv.ParseInt(credIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid credential id")
		return
	}

	if err := s.fido2Service.DeleteCredential(userID, credID); err != nil {
		respondError(w, http.StatusNotFound, "credential not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Public endpoints (for login flow) ---

// beginFIDO2Auth handles POST /api/auth/fido2/begin
func (s *Server) beginFIDO2Auth(w http.ResponseWriter, r *http.Request) {
	var req FIDO2AuthBeginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.PendingLoginToken == "" {
		respondError(w, http.StatusBadRequest, "pending_login_token is required")
		return
	}

	// Security note (SEC-005): The pending login token is consumed (Get deletes)
	// then re-stored for the finish step. The WebAuthn challenge (generated per
	// request, cryptographically bound) is the actual security control, not this
	// token. Accepted risk per security audit.
	pending, err := s.getPendingLogin(req.PendingLoginToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid or expired login token")
		return
	}

	// Re-store the token so it can be used again in finish step
	if err := s.fido2Service.StorePendingLogin(req.PendingLoginToken, pending); err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	assertion, err := s.fido2Service.BeginAuthentication(pending.UserID, pending.Username)
	if err != nil {
		s.logger().Error("fido2 begin auth failed", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "failed to start authentication")
		return
	}

	respondJSON(w, http.StatusOK, assertion)
}

// finishFIDO2Auth handles POST /api/auth/fido2/finish
func (s *Server) finishFIDO2Auth(w http.ResponseWriter, r *http.Request) {
	// Read pending_login_token from query param (body is WebAuthn response)
	pendingToken := r.URL.Query().Get("pending_login_token")
	if pendingToken == "" {
		respondError(w, http.StatusBadRequest, "pending_login_token is required")
		return
	}

	// Consume the pending login token (one-time use)
	pending, err := s.getPendingLogin(pendingToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid or expired login token")
		return
	}

	// Verify the WebAuthn assertion
	if err := s.fido2Service.FinishAuthentication(pending.UserID, pending.Username, r); err != nil {
		s.logger().Warn("fido2 auth failed",
			slog.Int("user_id", pending.UserID),
			slog.Any("error", err),
			securityIPAttr(r))
		s.accountLockout.RecordFailure(pending.Username, getClientIPSafe(r))
		respondError(w, http.StatusUnauthorized, "authentication failed")
		return
	}

	// Clear lockout on success
	s.accountLockout.RecordSuccess(pending.Username)

	// Issue tokens
	accessToken, refreshToken, err := s.authService.IssueTokens(r.Context(), pending.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}

	// Get user info
	user, err := s.authService.GetUserByID(pending.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to retrieve user information")
		return
	}

	// Fetch encryption salt
	salt, err := s.getOrGenerateUserSalt(user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch encryption salt")
		return
	}

	// Set cookies + CSRF token
	setAccessTokenCookie(w, accessToken)
	setRefreshTokenCookie(w, refreshToken)

	csrfToken, err := generateCSRFToken()
	if err != nil {
		s.logger().Error("failed to generate CSRF token", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	setCSRFTokenCookie(w, csrfToken)

	s.logger().Info("user_logged_in_fido2",
		slog.Int("user_id", user.ID),
		slog.String("event", "login_success_fido2"),
		securityIPAttr(r))

	respondJSON(w, http.StatusOK, AuthResponse{
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		EncryptionSalt: base64.StdEncoding.EncodeToString(salt),
		User: UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  user.IsAdmin,
		},
	})
}

// getPendingLogin retrieves and removes a pending login from the session store.
func (s *Server) getPendingLogin(token string) (*PendingLogin, error) {
	data, err := s.fido2Service.GetPendingLogin(token)
	if err != nil {
		return nil, err
	}
	pending, ok := data.(*PendingLogin)
	if !ok {
		return nil, fmt.Errorf("invalid pending login data")
	}
	return pending, nil
}

// storePendingLoginToken creates and stores a pending login token.
func (s *Server) storePendingLoginToken(userID int, username string) (string, error) {
	token, err := generatePendingLoginToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate pending login token: %w", err)
	}

	if err := s.fido2Service.StorePendingLogin(token, &PendingLogin{
		UserID:   userID,
		Username: username,
	}); err != nil {
		return "", fmt.Errorf("failed to store pending login: %w", err)
	}

	return token, nil
}
