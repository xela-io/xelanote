package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/db"
)

func (s *Server) listTrash(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
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

	notes, nextCursor, err := s.noteService.ListDeletedNotes(userID, limit, cursor)
	if err != nil {
		s.respondInternalErr(w, "failed to list trash", err)
		return
	}

	respondJSON(w, http.StatusOK, NoteListResponse{
		Notes:      ensureSlice(notes),
		NextCursor: nextCursor,
	})
}

func (s *Server) getTrashCount(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	count, err := s.noteService.GetDeletedNotesCount(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to get trash count", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"count": count,
	})
}

func (s *Server) restoreNote(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	note, err := s.noteService.RestoreNote(userID, id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		s.respondInternalErr(w, "failed to restore note", err)
		return
	}

	w.Header().Set("ETag", generateETag(note.ID, note.Version))
	respondJSON(w, http.StatusOK, note)
}

func (s *Server) permanentlyDeleteNote(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	err := s.noteService.PermanentlyDeleteNote(userID, id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found or not deleted")
			return
		}
		s.respondInternalErr(w, "failed to permanently delete note", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) emptyTrash(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	count, err := s.noteService.EmptyTrash(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to empty trash", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"deleted_count": count,
	})
}
