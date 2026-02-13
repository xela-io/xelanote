package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/service"
)

// --- Shared recipe/collection endpoints (recipient view) ---

// listSharedRecipes handles GET /api/shared/recipes
func (s *Server) listSharedRecipes(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	recipes, err := s.recipeService.ListSharedRecipes(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to list shared recipes", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"recipes": ensureSlice(recipes),
	})
}

// listSharedCollections handles GET /api/shared/collections
func (s *Server) listSharedCollections(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collections, err := s.recipeService.GetSharedCollectionsForUser(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to list shared collections", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"collections": ensureSlice(collections),
	})
}

// listSharedCollectionItems handles GET /api/shared/collections/{id}/items
func (s *Server) listSharedCollectionItems(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collID, ok := parseIntParam(w, r, "id", "invalid collection id")
	if !ok {
		return
	}

	recipes, err := s.recipeService.ListSharedCollectionItems(userID, collID)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			respondError(w, http.StatusForbidden, "no access to this collection")
			return
		}
		s.respondInternalErr(w, "failed to list shared collection items", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"recipes": recipes,
	})
}

// addToSharedCollection handles POST /api/shared/collections/{id}/items
func (s *Server) addToSharedCollection(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collID, ok := parseIntParam(w, r, "id", "invalid collection id")
	if !ok {
		return
	}

	var req AddToCollectionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.NoteID == "" {
		respondError(w, http.StatusBadRequest, "note_id is required")
		return
	}

	if err := s.recipeService.AddToSharedCollection(userID, collID, req.NoteID); err != nil {
		if errors.Is(err, service.ErrForbidden) {
			respondError(w, http.StatusForbidden, "editor access required")
			return
		}
		if errors.Is(err, service.ErrNotRecipeNote) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, service.ErrRecipeEncrypted) {
			respondError(w, http.StatusBadRequest, "encrypted recipes cannot be added to shared collections")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// removeFromSharedCollection handles DELETE /api/shared/collections/{id}/items/{noteId}
func (s *Server) removeFromSharedCollection(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collID, ok := parseIntParam(w, r, "id", "invalid collection id")
	if !ok {
		return
	}

	noteID := chi.URLParam(r, "noteId")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note id is required")
		return
	}

	if err := s.recipeService.RemoveFromSharedCollection(userID, collID, noteID); err != nil {
		if errors.Is(err, service.ErrForbidden) {
			respondError(w, http.StatusForbidden, "editor access required")
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "not found")
			return
		}
		s.respondInternalErr(w, "failed to remove from shared collection", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
