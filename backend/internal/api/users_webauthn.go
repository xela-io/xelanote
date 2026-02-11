package api

import (
	"net/http"
)

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
