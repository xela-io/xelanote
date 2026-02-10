package api

import (
	"encoding/base64"
	"log/slog"
	"net/http"
	"strings"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/service"
)

// Request/Response types for user preferences

type webAuthnCredentialInfo struct {
	ID           int64   `json:"id"`
	CredentialID string  `json:"credential_id"`
	DeviceName   string  `json:"device_name"`
	CreatedAt    string  `json:"created_at"`
	LastUsedAt   *string `json:"last_used_at,omitempty"`
}

type preferencesResponse struct {
	Theme           string                   `json:"theme"`
	EditorMode      string                   `json:"editor_mode"`
	KeywordsEnabled bool                     `json:"keywords_enabled"`
	EncryptTitles   bool                     `json:"encrypt_titles"`
	SecurityLevel   string                   `json:"security_level"`
	AutoLockTimeout int                      `json:"auto_lock_timeout"`
	Credentials     []webAuthnCredentialInfo `json:"webauthn_credentials"`
	Created         bool                     `json:"created"`
}

type updatePreferencesRequest struct {
	Theme      string `json:"theme"`
	EditorMode string `json:"editor_mode"`
}

type updateEncryptionPreferencesRequest struct {
	KeywordsEnabled bool `json:"keywords_enabled"`
	EncryptTitles   bool `json:"encrypt_titles"`
}

type updateSecurityPreferencesRequest struct {
	SecurityLevel   *string `json:"security_level"`
	AutoLockTimeout *int    `json:"auto_lock_timeout"`
}

type addWebAuthnCredentialRequest struct {
	CredentialID string `json:"credential_id"`
	DeviceName   string `json:"device_name"`
}

// convertWebAuthnCredentials converts db.WebAuthnCredential slice to webAuthnCredentialInfo slice
func convertWebAuthnCredentials(creds []db.WebAuthnCredential) []webAuthnCredentialInfo {
	result := make([]webAuthnCredentialInfo, 0, len(creds))
	for _, c := range creds {
		result = append(result, webAuthnCredentialInfo{
			ID:           c.ID,
			CredentialID: c.CredentialID,
			DeviceName:   c.DeviceName,
			CreatedAt:    c.CreatedAt,
			LastUsedAt:   c.LastUsedAt,
		})
	}
	return result
}

type setRecoveryKeyRequest struct {
	RecoveryKeyHash string `json:"recovery_key_hash"` // bcrypt hash
	Salt            string `json:"salt"`              // Base64-encoded
}

type changeEmailRequest struct {
	NewEmail        string `json:"new_email"`
	CurrentPassword string `json:"current_password"`
}

type changePasswordRequest struct {
	CurrentPassword      string            `json:"current_password"`
	NewPassword          string            `json:"new_password"`
	ReWrappedNoteDEKs    map[string]string `json:"re_wrapped_note_deks,omitempty"`    // noteID -> wrapped_dek (optional)
	ReWrappedVersionDEKs map[string]string `json:"re_wrapped_version_deks,omitempty"` // versionID -> wrapped_dek (optional)
}

// getPreferences returns user preferences, creating defaults if not exist
func (s *Server) getPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	prefs, created, err := s.userService.GetOrCreatePreferences(userID)
	if err != nil {
		s.logger().Error("failed to get preferences", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get preferences")
		return
	}

	// Load WebAuthn credentials
	credentials, err := s.userService.GetWebAuthnCredentials(int64(userID))
	if err != nil {
		s.logger().Error("failed to get webauthn credentials", "error", err)
		// Non-fatal, continue with empty list
		credentials = []db.WebAuthnCredential{}
	}

	respondJSON(w, http.StatusOK, preferencesResponse{
		Theme:           prefs.Theme,
		EditorMode:      prefs.EditorMode,
		KeywordsEnabled: prefs.KeywordsEnabled,
		EncryptTitles:   prefs.EncryptTitles,
		SecurityLevel:   prefs.SecurityLevel,
		AutoLockTimeout: prefs.AutoLockTimeout,
		Credentials:     convertWebAuthnCredentials(credentials),
		Created:         created,
	})
}

// updatePreferences updates user preferences
func (s *Server) updatePreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updatePreferencesRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	prefs, err := s.userService.UpdatePreferences(userID, req.Theme, req.EditorMode)
	if err != nil {
		switch err {
		case service.ErrInvalidTheme:
			respondError(w, http.StatusBadRequest, "invalid theme")
		case service.ErrInvalidEditorMode:
			respondError(w, http.StatusBadRequest, "invalid editor mode")
		default:
			s.logger().Error("failed to update preferences", "error", err)
			respondError(w, http.StatusInternalServerError, "failed to update preferences")
		}
		return
	}

	// Load WebAuthn credentials
	credentials, err := s.userService.GetWebAuthnCredentials(int64(userID))
	if err != nil {
		s.logger().Warn("failed to load webauthn credentials", "user_id", userID, "error", err)
		credentials = []db.WebAuthnCredential{}
	}

	respondJSON(w, http.StatusOK, preferencesResponse{
		Theme:           prefs.Theme,
		EditorMode:      prefs.EditorMode,
		KeywordsEnabled: prefs.KeywordsEnabled,
		EncryptTitles:   prefs.EncryptTitles,
		SecurityLevel:   prefs.SecurityLevel,
		AutoLockTimeout: prefs.AutoLockTimeout,
		Credentials:     convertWebAuthnCredentials(credentials),
		Created:         false,
	})
}

// changeEmail updates user email with password verification
func (s *Server) changeEmail(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req changeEmailRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.NewEmail == "" {
		respondError(w, http.StatusBadRequest, "new email is required")
		return
	}

	if req.CurrentPassword == "" {
		respondError(w, http.StatusBadRequest, "current password is required")
		return
	}

	// Get current refresh token from cookie or request body
	currentRefreshToken := getRefreshTokenFromCookie(r)

	err := s.userService.ChangeEmail(userID, req.NewEmail, req.CurrentPassword, currentRefreshToken)
	if err != nil {
		switch err {
		case service.ErrInvalidPassword:
			respondError(w, http.StatusUnauthorized, "incorrect password")
		case service.ErrEmailInUse:
			respondError(w, http.StatusConflict, "email already in use")
		case service.ErrInvalidEmail:
			respondError(w, http.StatusBadRequest, "invalid email format")
		default:
			s.logger().Error("failed to change email", "error", err)
			respondError(w, http.StatusInternalServerError, "failed to change email")
		}
		return
	}

	// Log email change event (PII protection: email not logged)
	s.logger().Info("email_changed",
		slog.Int("user_id", userID),
		slog.String("event", "email_change"),
		slog.String("remote_ip", getClientIPSafe(r)))

	respondJSON(w, http.StatusOK, map[string]string{"message": "email changed successfully"})
}

// changePassword updates user password with verification
func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req changePasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CurrentPassword == "" {
		respondError(w, http.StatusBadRequest, "current password is required")
		return
	}

	if req.NewPassword == "" {
		respondError(w, http.StatusBadRequest, "new password is required")
		return
	}

	// Get current refresh token from cookie
	currentRefreshToken := getRefreshTokenFromCookie(r)

	err := s.userService.ChangePasswordWithDEKRewrap(
		userID,
		req.CurrentPassword,
		req.NewPassword,
		req.ReWrappedNoteDEKs,
		req.ReWrappedVersionDEKs,
		currentRefreshToken,
	)
	if err != nil {
		switch err {
		case service.ErrInvalidPassword:
			respondError(w, http.StatusUnauthorized, "incorrect password")
		case service.ErrPasswordTooShort:
			respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		default:
			// Check if it's a DEK re-wrapping error
			errMsg := err.Error()
			if errMsg == "DEK re-wrapping required: user has encrypted notes or versions" {
				respondError(w, http.StatusBadRequest, "DEK re-wrapping required")
			} else if strings.HasPrefix(errMsg, "missing") {
				// "missing re-wrapped DEK for note X"
				respondError(w, http.StatusBadRequest, errMsg)
			} else {
				s.logger().Error("failed to change password", "error", err)
				respondError(w, http.StatusInternalServerError, "failed to change password")
			}
		}
		return
	}

	// Log password change event
	s.logger().Info("password_changed",
		slog.Int("user_id", userID),
		slog.String("event", "password_change"),
		slog.String("remote_ip", getClientIPSafe(r)))

	respondJSON(w, http.StatusOK, map[string]string{
		"message":                  "password changed successfully",
		"recovery_key_invalidated": "true",
	})
}

// updateEncryptionPreferences updates encryption-related preferences
func (s *Server) updateEncryptionPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateEncryptionPreferencesRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := s.userService.UpdateEncryptionPreferences(userID, req.KeywordsEnabled, req.EncryptTitles)
	if err != nil {
		s.logger().Error("failed to update encryption preferences", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update encryption preferences")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "encryption preferences updated successfully"})
}

// setRecoveryKey sets a recovery key for the authenticated user
func (s *Server) setRecoveryKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setRecoveryKeyRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate inputs
	if req.RecoveryKeyHash == "" {
		respondError(w, http.StatusBadRequest, "recovery_key_hash is required")
		return
	}
	if req.Salt == "" {
		respondError(w, http.StatusBadRequest, "salt is required")
		return
	}

	// Decode base64 salt
	salt, err := base64.StdEncoding.DecodeString(req.Salt)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid base64 salt")
		return
	}

	// Set recovery key
	err = s.userService.SetRecoveryKey(userID, req.RecoveryKeyHash, salt)
	if err != nil {
		s.logger().Error("failed to set recovery key", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to set recovery key")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "recovery key set successfully"})
}

// getRecoveryKeySalt retrieves the recovery key salt for the authenticated user
func (s *Server) getRecoveryKeySalt(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	salt, err := s.userService.GetRecoveryKeySalt(userID)
	if err != nil {
		if err == db.ErrNotFound {
			respondError(w, http.StatusNotFound, "no recovery key set")
			return
		}
		s.logger().Error("failed to get recovery key salt", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get recovery key salt")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"salt": base64.StdEncoding.EncodeToString(salt),
	})
}

// updateSecurityPreferences updates security-related preferences (security_level and auto_lock_timeout)
func (s *Server) updateSecurityPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateSecurityPreferencesRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate security level
	if req.SecurityLevel != nil {
		validLevels := map[string]bool{"paranoid": true, "balanced": true, "convenient": true}
		if !validLevels[*req.SecurityLevel] {
			respondError(w, http.StatusBadRequest, "invalid security level")
			return
		}
	}

	// Validate auto-lock timeout
	if req.AutoLockTimeout != nil {
		validTimeouts := map[int]bool{0: true, 5: true, 15: true, 30: true, 60: true}
		if !validTimeouts[*req.AutoLockTimeout] {
			respondError(w, http.StatusBadRequest, "invalid auto-lock timeout")
			return
		}
	}

	// Update preferences
	prefs, err := s.userService.UpdateSecurityPreferences(userID, req.SecurityLevel, req.AutoLockTimeout)
	if err != nil {
		s.logger().Error("failed to update security preferences", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update preferences")
		return
	}

	// Load credentials
	credentials, err := s.userService.GetWebAuthnCredentials(int64(userID))
	if err != nil {
		s.logger().Warn("failed to load webauthn credentials", "user_id", userID, "error", err)
		credentials = []db.WebAuthnCredential{}
	}

	respondJSON(w, http.StatusOK, preferencesResponse{
		Theme:           prefs.Theme,
		EditorMode:      prefs.EditorMode,
		KeywordsEnabled: prefs.KeywordsEnabled,
		EncryptTitles:   prefs.EncryptTitles,
		SecurityLevel:   prefs.SecurityLevel,
		AutoLockTimeout: prefs.AutoLockTimeout,
		Credentials:     convertWebAuthnCredentials(credentials),
		Created:         false,
	})
}

// addWebAuthnCredential adds a new WebAuthn credential for the user
func (s *Server) addWebAuthnCredential(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req addWebAuthnCredentialRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CredentialID == "" {
		respondError(w, http.StatusBadRequest, "credential_id required")
		return
	}

	cred, err := s.userService.AddWebAuthnCredential(int64(userID), req.CredentialID, req.DeviceName)
	if err != nil {
		s.logger().Error("failed to add webauthn credential", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to add credential")
		return
	}

	respondJSON(w, http.StatusCreated, webAuthnCredentialInfo{
		ID:           cred.ID,
		CredentialID: cred.CredentialID,
		DeviceName:   cred.DeviceName,
		CreatedAt:    cred.CreatedAt,
		LastUsedAt:   cred.LastUsedAt,
	})
}

// deleteWebAuthnCredential deletes a WebAuthn credential
func (s *Server) deleteWebAuthnCredential(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	credentialID := r.URL.Query().Get("credential_id")
	if credentialID == "" {
		respondError(w, http.StatusBadRequest, "credential_id query parameter required")
		return
	}

	if err := s.userService.DeleteWebAuthnCredential(int64(userID), credentialID); err != nil {
		s.logger().Error("failed to delete webauthn credential", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to delete credential")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// touchWebAuthnCredential updates the last_used_at timestamp for a credential
func (s *Server) touchWebAuthnCredential(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	credentialID := r.URL.Query().Get("credential_id")
	if credentialID == "" {
		respondError(w, http.StatusBadRequest, "credential_id query parameter required")
		return
	}

	if err := s.userService.TouchWebAuthnCredential(int64(userID), credentialID); err != nil {
		s.logger().Error("failed to touch webauthn credential", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update credential")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// --- Claude API Key Management (BYOK) ---

type setClaudeAPIKeyRequest struct {
	APIKey string `json:"api_key"`
}

// setClaudeAPIKey stores a Claude API key for the user (encrypted)
func (s *Server) setClaudeAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setClaudeAPIKeyRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.APIKey == "" {
		respondError(w, http.StatusBadRequest, "api_key is required")
		return
	}

	err := s.userService.SetClaudeAPIKey(userID, req.APIKey)
	if err != nil {
		switch err {
		case service.ErrInvalidClaudeAPIKey:
			respondError(w, http.StatusBadRequest, "invalid Claude API key format (must start with sk-ant-)")
		default:
			s.logger().Error("failed to set Claude API key", "error", err)
			respondError(w, http.StatusInternalServerError, "failed to store API key")
		}
		return
	}

	// Invalidate cached Claude client so the new key is used
	s.summarizeService.InvalidateClaudeClient(userID)

	s.logger().Info("claude_api_key_set",
		slog.Int("user_id", userID),
		slog.String("event", "api_key_set"),
		slog.String("remote_ip", getClientIPSafe(r)))

	respondJSON(w, http.StatusOK, map[string]string{"message": "API key stored successfully"})
}

// deleteClaudeAPIKey removes the user's Claude API key
func (s *Server) deleteClaudeAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	err := s.userService.DeleteClaudeAPIKey(userID)
	if err != nil {
		s.logger().Error("failed to delete Claude API key", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to delete API key")
		return
	}

	// Invalidate cached Claude client
	s.summarizeService.InvalidateClaudeClient(userID)

	s.logger().Info("claude_api_key_deleted",
		slog.Int("user_id", userID),
		slog.String("event", "api_key_deleted"),
		slog.String("remote_ip", getClientIPSafe(r)))

	respondJSON(w, http.StatusOK, map[string]string{"message": "API key deleted successfully"})
}

// getClaudeAPIKeyStatus returns the status of the user's Claude API key (NOT the actual key)
func (s *Server) getClaudeAPIKeyStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	status, err := s.userService.GetClaudeAPIKeyStatus(userID)
	if err != nil {
		s.logger().Error("failed to get Claude API key status", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get API key status")
		return
	}

	respondJSON(w, http.StatusOK, status)
}

// --- Gemini API Key Management (BYOK) ---

type setGeminiAPIKeyRequest struct {
	APIKey string `json:"api_key"`
}

// setGeminiAPIKey stores a Gemini API key for the user (encrypted)
func (s *Server) setGeminiAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setGeminiAPIKeyRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.APIKey == "" {
		respondError(w, http.StatusBadRequest, "api_key is required")
		return
	}

	err := s.userService.SetGeminiAPIKey(userID, req.APIKey)
	if err != nil {
		switch err {
		case service.ErrInvalidGeminiAPIKey:
			respondError(w, http.StatusBadRequest, "invalid Gemini API key format (must start with AIza)")
		default:
			s.logger().Error("failed to set Gemini API key", "error", err)
			respondError(w, http.StatusInternalServerError, "failed to store API key")
		}
		return
	}

	// Invalidate cached Gemini client so the new key is used
	s.summarizeService.InvalidateGeminiClient(userID)

	s.logger().Info("gemini_api_key_set",
		slog.Int("user_id", userID),
		slog.String("event", "api_key_set"),
		slog.String("remote_ip", getClientIPSafe(r)))

	respondJSON(w, http.StatusOK, map[string]string{"message": "API key stored successfully"})
}

// deleteGeminiAPIKey removes the user's Gemini API key
func (s *Server) deleteGeminiAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	err := s.userService.DeleteGeminiAPIKey(userID)
	if err != nil {
		s.logger().Error("failed to delete Gemini API key", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to delete API key")
		return
	}

	// Invalidate cached Gemini client
	s.summarizeService.InvalidateGeminiClient(userID)

	s.logger().Info("gemini_api_key_deleted",
		slog.Int("user_id", userID),
		slog.String("event", "api_key_deleted"),
		slog.String("remote_ip", getClientIPSafe(r)))

	respondJSON(w, http.StatusOK, map[string]string{"message": "API key deleted successfully"})
}

// getGeminiAPIKeyStatus returns the status of the user's Gemini API key (NOT the actual key)
func (s *Server) getGeminiAPIKeyStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	status, err := s.userService.GetGeminiAPIKeyStatus(userID)
	if err != nil {
		s.logger().Error("failed to get Gemini API key status", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get API key status")
		return
	}

	respondJSON(w, http.StatusOK, status)
}
