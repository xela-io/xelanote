package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/service"
)

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
		if errors.Is(err, service.ErrNotFound) {
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
		if errors.Is(err, service.ErrNotFound) {
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
