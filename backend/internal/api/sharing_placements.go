package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/service"
)

// placeSharedNoteHandler handles POST /api/shared/{id}/placement
func (s *Server) placeSharedNoteHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	var req PlacementRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.FolderID <= 0 {
		respondError(w, http.StatusBadRequest, "folder_id required")
		return
	}

	err := s.sharingService.PlaceSharedNote(userID, noteID, req.FolderID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNoShareAccess):
			respondError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, service.ErrCannotPlaceOwnNote):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			s.logger().Error("unexpected placement error", "error", err)
			respondError(w, http.StatusInternalServerError, "an unexpected error occurred")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// removePlacementHandler handles DELETE /api/shared/{id}/placement
func (s *Server) removePlacementHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	err := s.sharingService.RemovePlacement(userID, noteID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "placement not found")
			return
		}
		s.respondInternalErr(w, "failed to remove placement", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
