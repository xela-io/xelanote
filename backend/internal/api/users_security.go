package api

import (
	"net/http"

	"github.com/xela-io/xelanote/internal/service"
)

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
		credentials = []service.WebAuthnCredential{}
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
