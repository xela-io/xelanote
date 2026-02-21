package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/service"
)

// --- Collection handlers ---

func (s *Server) listRecipeCollections(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collections, err := s.recipeService.ListCollections(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to list recipe collections", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"collections": collections,
	})
}

func (s *Server) createRecipeCollection(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req CreateCollectionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	coll, err := s.recipeService.CreateCollection(userID, req.Name, req.Description, req.Color)
	if err != nil {
		s.respondInternalErr(w, "failed to create collection", err)
		return
	}

	respondJSON(w, http.StatusCreated, coll)
}

func (s *Server) updateRecipeCollection(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collID, ok := parseIntParam(w, r, "id", "invalid collection id")
	if !ok {
		return
	}

	var req UpdateCollectionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.recipeService.UpdateCollection(userID, collID, req.Name, req.Description, req.Color); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "collection not found")
			return
		}
		s.respondInternalErr(w, "failed to update collection", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) deleteRecipeCollection(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collID, ok := parseIntParam(w, r, "id", "invalid collection id")
	if !ok {
		return
	}

	if err := s.recipeService.DeleteCollection(userID, collID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "collection not found")
			return
		}
		s.respondInternalErr(w, "failed to delete recipe collection", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) addRecipeToCollection(w http.ResponseWriter, r *http.Request) {
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

	if err := s.recipeService.AddToCollection(userID, collID, req.NoteID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "collection not found")
			return
		}
		s.respondInternalErr(w, "failed to add recipe to collection", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) removeRecipeFromCollection(w http.ResponseWriter, r *http.Request) {
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

	if err := s.recipeService.RemoveFromCollection(userID, collID, noteID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "not found")
			return
		}
		s.respondInternalErr(w, "failed to remove recipe from collection", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listCollectionItems(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collID, ok := parseIntParam(w, r, "id", "invalid collection id")
	if !ok {
		return
	}

	recipes, err := s.recipeService.ListCollectionItems(userID, collID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "collection not found")
			return
		}
		s.respondInternalErr(w, "failed to list collection items", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"recipes": recipes,
	})
}
