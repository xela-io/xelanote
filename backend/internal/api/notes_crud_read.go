package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/db"
)

func (s *Server) listNotes(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (set by authMiddleware)
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = min(parsed, 500) // Cap at 500 to prevent memory exhaustion
		}
	}
	cursor := r.URL.Query().Get("cursor")
	folderPath := r.URL.Query().Get("folder")

	var notes []db.Note
	var nextCursor string
	var err error

	if folderPath != "" {
		notes, err = s.noteService.GetNotesByFolder(userID, folderPath)
	} else {
		notes, nextCursor, err = s.noteService.ListNotes(userID, limit, cursor)
	}

	if err != nil {
		s.respondInternalErr(w, "failed to list notes", err)
		return
	}

	respondJSON(w, http.StatusOK, NoteListResponse{
		Notes:      ensureSlice(notes),
		NextCursor: nextCursor,
	})
}

func (s *Server) getNote(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	note, err := s.noteService.GetNote(userID, id)
	if err != nil {
		s.respondInternalErr(w, "failed to get note", err)
		return
	}

	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	w.Header().Set("ETag", generateETag(note.ID, note.Version))
	respondJSON(w, http.StatusOK, note)
}
