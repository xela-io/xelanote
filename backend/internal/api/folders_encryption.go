package api

import (
	"net/http"
)

// ============================================================================
// Encryption Default Endpoints
// ============================================================================

// updateFolderEncryptionDefault toggles the encryption_default flag for a folder.
// New notes created in this folder will inherit this setting.
func (s *Server) updateFolderEncryptionDefault(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id, ok := parseIntParam(w, r, "id", "invalid folder id")
	if !ok {
		return
	}

	var req UpdateFolderEncryptionDefaultRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.noteService.UpdateFolderEncryptionDefault(userID, id, req.Encrypted); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"encrypted": req.Encrypted,
	})
}

// getFolderEncryptionDefault returns the encryption_default status for a folder.
func (s *Server) getFolderEncryptionDefault(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id, ok := parseIntParam(w, r, "id", "invalid folder id")
	if !ok {
		return
	}

	encrypted, err := s.noteService.GetFolderEncryptionDefault(userID, id)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"encrypted": encrypted,
	})
}
