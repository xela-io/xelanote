package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/service"
)

// --- Recipe handlers ---

func (s *Server) listRecipes(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	fields := r.URL.Query().Get("fields")
	recipes, err := s.recipeService.ListRecipes(userID, fields)
	if err != nil {
		s.respondInternalErr(w, "failed to list recipes", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"recipes": ensureSlice(recipes),
	})
}

func (s *Server) getRecipeDetail(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")

	detail, err := s.recipeService.GetRecipeDetail(userID, noteID)
	if err != nil {
		if errors.Is(err, service.ErrRecipeEncrypted) {
			respondError(w, http.StatusForbidden, "recipe is encrypted")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "recipe not found")
			return
		}
		s.respondInternalErr(w, "failed to get recipe detail", err)
		return
	}

	// Sign image URLs
	s.signRecipeImageURLs(detail)

	respondJSON(w, http.StatusOK, detail)
}

func (s *Server) updateRecipeMetadata(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")

	var req UpdateRecipeMetadataRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	meta := &service.RecipeMetadata{
		Servings:        req.Servings,
		PrepTimeMinutes: req.PrepTimeMinutes,
		CookTimeMinutes: req.CookTimeMinutes,
		SourceURL:       req.SourceURL,
		Difficulty:      req.Difficulty,
	}

	err := s.recipeService.UpdateRecipeMetadata(userID, noteID, meta, req.ExpectedUpdatedAt)
	if err != nil {
		if errors.Is(err, service.ErrVersionMismatch) {
			respondError(w, http.StatusConflict, "version mismatch - recipe was modified")
			return
		}
		if errors.Is(err, service.ErrRecipeEncrypted) {
			respondError(w, http.StatusBadRequest, "cannot update encrypted recipe metadata")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		s.respondInternalErr(w, "failed to update recipe", err)
		return
	}

	// Return updated metadata
	updatedMeta, err := s.recipeService.GetRecipeDetail(userID, noteID)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	respondJSON(w, http.StatusOK, updatedMeta.Metadata)
}

func (s *Server) setRecipeIngredients(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")

	var req SetRecipeIngredientsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := s.recipeService.SetRecipeIngredients(userID, noteID, req.Ingredients, req.ExpectedUpdatedAt)
	if err != nil {
		if errors.Is(err, service.ErrVersionMismatch) {
			respondError(w, http.StatusConflict, "version mismatch - recipe was modified")
			return
		}
		if errors.Is(err, service.ErrRecipeEncrypted) {
			respondError(w, http.StatusBadRequest, "cannot update encrypted recipe ingredients")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		s.respondInternalErr(w, "failed to update ingredients", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) getScaledIngredients(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")

	servingsStr := r.URL.Query().Get("servings")
	if servingsStr == "" {
		respondError(w, http.StatusBadRequest, "servings query parameter required")
		return
	}
	targetServings, err := strconv.Atoi(servingsStr)
	if err != nil || targetServings < 1 || targetServings > 999 {
		respondError(w, http.StatusBadRequest, "servings must be between 1 and 999")
		return
	}

	scaled, err := s.recipeService.GetScaledIngredients(userID, noteID, targetServings)
	if err != nil {
		if errors.Is(err, service.ErrRecipeEncrypted) {
			respondError(w, http.StatusBadRequest, "cannot scale encrypted recipe")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "recipe not found")
			return
		}
		s.respondInternalErr(w, "failed to get scaled ingredients", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ingredients": scaled,
	})
}
