package api

import (
	"net/http"

	"github.com/xela-io/xelanote/internal/service"
)

func (s *Server) listNoteTitles(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Fetch notes with pagination, limited to prevent memory exhaustion
	var allNotes []service.Note
	cursor := ""
	for {
		notes, nextCursor, err := s.noteService.ListNotes(userID, 500, cursor, service.ListNotesOptions{})
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
