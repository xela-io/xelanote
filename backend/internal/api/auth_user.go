package api

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/xela-io/xelanote/internal/service"
)

// me returns current authenticated user information
// This is a protected endpoint that requires valid JWT
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (set by authMiddleware)
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get user from database
	user, err := s.authService.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to retrieve user")
		return
	}

	// Return user info (without password hash)
	respondJSON(w, http.StatusOK, UserResponse{
		ID:             user.ID,
		Username:       user.Username,
		Email:          user.Email,
		IsAdmin:        user.IsAdmin,
		EncryptionSalt: user.EncryptionSalt, // Include encryption salt for E2E encryption setup
	})
}

// getRecoveryKeySaltByEmail retrieves the recovery key salt for password recovery (public endpoint)
func (s *Server) getRecoveryKeySaltByEmail(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	// Ensure minimum response time to prevent timing attacks
	defer func() {
		elapsed := time.Since(startTime)
		if elapsed < 100*time.Millisecond {
			time.Sleep(100*time.Millisecond - elapsed)
		}
	}()

	var req RecoverySaltRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "email is required")
		return
	}

	salt, err := s.userService.GetRecoveryKeySaltByEmail(req.Email)
	if err != nil {
		// Return generic error to avoid user enumeration
		respondError(w, http.StatusNotFound, "recovery key not available")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"salt": base64.StdEncoding.EncodeToString(salt),
	})
}

// verifyRecoveryKey verifies email + recovery key and returns a one-time recovery reset token.
func (s *Server) verifyRecoveryKey(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		elapsed := time.Since(startTime)
		if elapsed < 200*time.Millisecond {
			time.Sleep(200*time.Millisecond - elapsed)
		}
	}()

	var req RecoveryVerifyRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.RecoveryKey == "" {
		respondError(w, http.StatusBadRequest, "recovery_key is required")
		return
	}

	verifyResult, err := s.userService.BeginRecoveryResetByEmail(req.Email, req.RecoveryKey)
	if err != nil {
		s.logger().Warn("recovery verify failed", "email", req.Email, "error", err.Error())
		respondError(w, http.StatusUnauthorized, "invalid email or recovery key")
		return
	}

	respondJSON(w, http.StatusOK, recoveryVerifyResponse{
		RecoveryResetToken: verifyResult.RecoveryResetToken,
		EncryptionSalt:     verifyResult.EncryptionSalt,
	})
}

// getRecoveryWrappedDEKs returns encrypted note/version recovery wrappers for the provided reset token.
func (s *Server) getRecoveryWrappedDEKs(w http.ResponseWriter, r *http.Request) {
	token := getRecoveryResetToken(r)
	if token == "" {
		respondError(w, http.StatusBadRequest, "recovery reset token is required")
		return
	}

	notes, versions, err := s.userService.GetRecoveryWrappedDEKs(token)
	if err != nil {
		switch err {
		case service.ErrInvalidRecoveryResetToken:
			respondError(w, http.StatusUnauthorized, "invalid or expired recovery reset token")
		case service.ErrRecoveryRewrapNotConfigured:
			respondError(w, http.StatusConflict, "encrypted recovery wrappers are missing")
		default:
			s.logger().Error("failed to load recovery wrapped DEKs", "error", err)
			respondError(w, http.StatusInternalServerError, "failed to load encrypted recovery data")
		}
		return
	}

	noteItems := make([]recoveryWrappedDEKResponseItem, 0, len(notes))
	for _, note := range notes {
		noteItems = append(noteItems, recoveryWrappedDEKResponseItem{
			ID:                 note.ID,
			WrappedDEKRecovery: note.WrappedDEKRecovery,
		})
	}

	versionItems := make([]recoveryWrappedDEKResponseItem, 0, len(versions))
	for _, version := range versions {
		versionItems = append(versionItems, recoveryWrappedDEKResponseItem{
			ID:                 version.ID,
			WrappedDEKRecovery: version.WrappedDEKRecovery,
		})
	}

	respondJSON(w, http.StatusOK, recoveryWrappedDEKsResponse{
		Notes:    noteItems,
		Versions: versionItems,
	})
}

// resetPasswordWithRecoveryKey resets a user's password using recovery key (public endpoint)
func (s *Server) resetPasswordWithRecoveryKey(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	// Ensure minimum response time to prevent timing attacks
	defer func() {
		elapsed := time.Since(startTime)
		if elapsed < 200*time.Millisecond {
			time.Sleep(200*time.Millisecond - elapsed)
		}
	}()

	var req RecoveryResetPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.RecoveryKey == "" {
		respondError(w, http.StatusBadRequest, "recovery_key is required")
		return
	}
	if req.NewPassword == "" {
		respondError(w, http.StatusBadRequest, "new_password is required")
		return
	}

	// Attempt password recovery
	err := s.userService.RecoverPasswordWithRecoveryKeyByEmail(req.Email, req.RecoveryKey, req.NewPassword)
	if err != nil {
		// Return generic error message for security
		s.logger().Warn("password recovery attempt failed", "email", req.Email, "error", err.Error())
		respondError(w, http.StatusUnauthorized, "invalid email, recovery key, or password requirements not met")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "password reset successfully, please login with your new password",
	})
}

// resetPasswordWithRecoveryToken finalizes password recovery with a one-time recovery reset token.
func (s *Server) resetPasswordWithRecoveryToken(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		elapsed := time.Since(startTime)
		if elapsed < 200*time.Millisecond {
			time.Sleep(200*time.Millisecond - elapsed)
		}
	}()

	var req RecoveryResetPasswordWithTokenRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RecoveryResetToken == "" {
		respondError(w, http.StatusBadRequest, "recovery_reset_token is required")
		return
	}
	if req.NewPassword == "" {
		respondError(w, http.StatusBadRequest, "new_password is required")
		return
	}

	err := s.userService.FinalizeRecoveryResetWithToken(
		req.RecoveryResetToken,
		req.NewPassword,
		req.ReWrappedNoteDEKs,
		req.ReWrappedVersionDEKs,
	)
	if err != nil {
		switch err {
		case service.ErrPasswordTooShort:
			respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		case service.ErrInvalidRecoveryResetToken:
			respondError(w, http.StatusUnauthorized, "invalid or expired recovery reset token")
		default:
			errMsg := err.Error()
			if errMsg == "DEK re-wrapping required: user has encrypted notes or versions" ||
				errMsg == "no encrypted content to re-wrap" ||
				strings.HasPrefix(errMsg, "missing re-wrapped DEK") ||
				strings.HasPrefix(errMsg, "invalid re-wrapped DEK") ||
				strings.HasPrefix(errMsg, "unexpected re-wrapped DEK") {
				respondError(w, http.StatusBadRequest, errMsg)
			} else {
				s.logger().Warn("recovery reset finalize failed", "error", err.Error())
				respondError(w, http.StatusInternalServerError, "failed to reset password")
			}
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "password reset successfully, please login with your new password",
	})
}

func getRecoveryResetToken(r *http.Request) string {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	return strings.TrimSpace(r.Header.Get("X-Recovery-Reset-Token"))
}
