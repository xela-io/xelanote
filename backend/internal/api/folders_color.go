package api

import (
	"net/http"
)

// updateFolderColor updates the color of a folder.
func (s *Server) updateFolderColor(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id, ok := parseIntParam(w, r, "id", "invalid folder id")
	if !ok {
		return
	}

	var req UpdateColorRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := s.noteService.UpdateFolderColor(userID, id, req.Color)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
