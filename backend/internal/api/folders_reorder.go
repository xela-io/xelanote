package api

import "net/http"

// reorderFolders updates the display order of folders.
func (s *Server) reorderFolders(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req ReorderFoldersRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Items) == 0 {
		respondError(w, http.StatusBadRequest, "items cannot be empty")
		return
	}

	err := s.noteService.ReorderFolders(userID, req.ParentID, req.Items)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
