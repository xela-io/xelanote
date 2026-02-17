package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// registerCanvasRoutes registers canvas-specific API routes.
func (s *Server) registerCanvasRoutes(r chi.Router) {
	r.Route("/canvas", func(r chi.Router) {
		r.Get("/", s.listCanvasNotes)
		r.Get("/{id}/export", s.exportCanvas)
	})
}

func (s *Server) listCanvasNotes(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	notes, err := s.canvasService.ListCanvasNotes(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to list canvas notes", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"notes": ensureSlice(notes),
	})
}

func (s *Server) exportCanvas(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "missing note id")
		return
	}

	note, err := s.canvasService.ExportCanvas(userID, noteID)
	if err != nil {
		respondError(w, http.StatusNotFound, "canvas not found")
		return
	}

	filename := note.Title + ".canvas"
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(note.Content))
}
