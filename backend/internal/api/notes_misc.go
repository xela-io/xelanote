package api

import (
	"net/http"

	"github.com/xela-io/xelanote/internal/db"
)

// listNoteTitles returns a lightweight list of note titles for link suggestions.
// Only returns unencrypted titles to avoid sending encrypted data to LLM.
// Limited to MaxNoteTitlesForSuggestions to prevent memory exhaustion.
const MaxNoteTitlesForSuggestions = 1000

func (s *Server) listNoteTitles(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Fetch notes with pagination, limited to prevent memory exhaustion
	var allNotes []db.Note
	cursor := ""
	for {
		notes, nextCursor, err := s.noteService.ListNotes(userID, 500, cursor)
		if err != nil {
			s.respondInternalErr(w, "failed to list note titles", err)
			return
		}
		allNotes = append(allNotes, notes...)
		// Early exit if we have enough notes
		if len(allNotes) >= MaxNoteTitlesForSuggestions || nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	// Build title list - only include unencrypted titles
	titles := make([]NoteTitleInfo, 0, len(allNotes))
	for _, note := range allNotes {
		// Skip notes with encrypted titles (privacy-first)
		if note.TitleEncrypted {
			continue
		}
		titles = append(titles, NoteTitleInfo{
			ID:        note.ID,
			Title:     note.Title,
			Encrypted: note.ContentEncrypted,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"titles": titles,
	})
}

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
