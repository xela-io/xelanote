package api

import (
	"net/http"

	"github.com/xela-io/xelanote/internal/service"
)

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
