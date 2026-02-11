package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid folder id")
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

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid folder id")
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
