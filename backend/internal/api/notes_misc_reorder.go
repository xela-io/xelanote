package api

import "net/http"

// reorderNotes updates the display order of notes within a folder.
func (s *Server) reorderNotes(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req struct {
		FolderPath string   `json:"folder_path"`
		Items      []string `json:"items"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Items) == 0 {
		respondError(w, http.StatusBadRequest, "items cannot be empty")
		return
	}

	if len(req.FolderPath) > MaxFolderPathLength {
		respondError(w, http.StatusBadRequest, "folder path too long")
		return
	}

	err := s.noteService.ReorderNotes(userID, req.FolderPath, req.Items)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
