package api

import (
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/xela-io/xelanote/internal/service"
)

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
		if errors.Is(err, service.ErrRecoveryKeyBlockedEncrypted) {
			respondError(w, http.StatusConflict, "recovery key setup is unavailable for accounts with encrypted notes")
			return
		}
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
		if errors.Is(err, service.ErrNotFound) {
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
