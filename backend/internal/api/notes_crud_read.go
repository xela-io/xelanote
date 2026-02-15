package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/service"
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

	// Validate fields parameter: only "" (empty) or "slim" are allowed
	fields := r.URL.Query().Get("fields")
	if fields != "" && fields != "slim" {
		respondError(w, http.StatusBadRequest, "invalid fields parameter: must be 'slim' or omitted")
		return
	}

	// Validate updated_since parameter if present
	updatedSince := r.URL.Query().Get("updated_since")
	if updatedSince != "" {
		if _, _, err := db.ParseSyncToken(updatedSince); err != nil {
			respondError(w, http.StatusBadRequest, "invalid updated_since parameter: expected format timestamp|id")
			return
		}
	}

	opts := db.ListNotesOptions{
		Fields:       fields,
		UpdatedSince: updatedSince,
	}

	var notes []service.Note
	var nextCursor string
	var err error

	if folderPath != "" {
		notes, err = s.noteService.GetNotesByFolder(userID, folderPath, fields)
	} else {
		notes, nextCursor, err = s.noteService.ListNotes(userID, limit, cursor, opts)
	}

	if err != nil {
		s.respondInternalErr(w, "failed to list notes", err)
		return
	}

	noteSlice := ensureSlice(notes)

	// Compute sync_token (high-watermark of result set)
	isDelta := updatedSince != ""
	syncToken := db.HighWatermark(noteSlice, isDelta)

	respondJSON(w, http.StatusOK, NoteListResponse{
		Notes:      noteSlice,
		NextCursor: nextCursor,
		SyncToken:  syncToken,
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
