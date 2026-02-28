package api

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

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
	if req.RecoveryKeyHash == "" && req.RecoveryKey == "" {
		respondError(w, http.StatusBadRequest, "recovery_key_hash or recovery_key is required")
		return
	}
	if req.RecoveryKeyHash != "" && req.RecoveryKey != "" {
		respondError(w, http.StatusBadRequest, "provide either recovery_key_hash or recovery_key, not both")
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

	recoveryKeyHash := req.RecoveryKeyHash
	if recoveryKeyHash == "" {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(req.RecoveryKey), 12)
		if hashErr != nil {
			s.logger().Error("failed to hash recovery key", "error", hashErr)
			respondError(w, http.StatusInternalServerError, "failed to set recovery key")
			return
		}
		recoveryKeyHash = string(hash)
	}

	// Set recovery key
	err = s.userService.SetRecoveryKeyWithRecoveryWrappedDEKs(
		userID,
		recoveryKeyHash,
		salt,
		req.RecoveryWrappedNoteDEKs,
		req.RecoveryWrappedVersionDEKs,
	)
	if err != nil {
		if errors.Is(err, service.ErrRecoveryKeyBlockedEncrypted) ||
			errors.Is(err, service.ErrRecoveryWrappedDEKsRequired) {
			respondError(
				w,
				http.StatusConflict,
				"recovery key setup for encrypted accounts requires full recovery DEK re-wrapping",
			)
			return
		}
		errMsg := err.Error()
		if errMsg == "no encrypted content to re-wrap" ||
			strings.HasPrefix(errMsg, "missing ") ||
			strings.HasPrefix(errMsg, "invalid ") ||
			strings.HasPrefix(errMsg, "unexpected ") {
			respondError(w, http.StatusBadRequest, errMsg)
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
		if errors.Is(err, service.ErrNotFound) || errors.Is(err, service.ErrRecoveryKeyBlockedEncrypted) {
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
