package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/service"
)

func (s *Server) getBacklinks(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	backlinks, err := s.noteService.GetBacklinks(userID, id)
	if err != nil {
		s.respondInternalErr(w, "failed to get backlinks", err)
		return
	}

	if backlinks == nil {
		backlinks = []service.Backlink{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"backlinks": backlinks,
	})
}
