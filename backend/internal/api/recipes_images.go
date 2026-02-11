package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/service"
)

// --- Recipe Image handlers ---

// addRecipeImage handles POST /api/recipes/{id}/images
func (s *Server) addRecipeImage(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")

	var req AddRecipeImageRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ImageURL == "" {
		respondError(w, http.StatusBadRequest, "image_url is required")
		return
	}

	img, err := s.recipeService.AddRecipeImage(userID, noteID, req.ImageURL, req.Caption)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecipeEncrypted):
			respondError(w, http.StatusBadRequest, "cannot add images to encrypted recipe")
		case errors.Is(err, service.ErrForbidden):
			respondError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, service.ErrMaxImagesReached):
			respondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrInvalidImageURL):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			s.respondInternalErr(w, "failed to add recipe image", err)
		}
		return
	}

	respondJSON(w, http.StatusCreated, img)
}

// updateRecipeImageCaption handles PUT /api/recipes/{id}/images/{imageId}
func (s *Server) updateRecipeImageCaption(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	imageID, err := strconv.Atoi(chi.URLParam(r, "imageId"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid image id")
		return
	}

	var req UpdateRecipeImageCaptionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.recipeService.UpdateRecipeImageCaption(userID, noteID, imageID, req.Caption); err != nil {
		switch {
		case errors.Is(err, service.ErrRecipeEncrypted):
			respondError(w, http.StatusBadRequest, "cannot update caption on encrypted recipe")
		case errors.Is(err, service.ErrForbidden):
			respondError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, db.ErrNotFound):
			respondError(w, http.StatusNotFound, "image not found")
		default:
			s.respondInternalErr(w, "failed to update recipe image caption", err)
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// deleteRecipeImage handles DELETE /api/recipes/{id}/images/{imageId}
func (s *Server) deleteRecipeImage(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	imageID, err := strconv.Atoi(chi.URLParam(r, "imageId"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid image id")
		return
	}

	if err := s.recipeService.DeleteRecipeImage(userID, noteID, imageID); err != nil {
		switch {
		case errors.Is(err, service.ErrRecipeEncrypted):
			respondError(w, http.StatusBadRequest, "cannot delete image from encrypted recipe")
		case errors.Is(err, service.ErrForbidden):
			respondError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, db.ErrNotFound):
			respondError(w, http.StatusNotFound, "image not found")
		default:
			s.respondInternalErr(w, "failed to delete recipe image", err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// reorderRecipeImages handles PUT /api/recipes/{id}/images/order
func (s *Server) reorderRecipeImages(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")

	var req ReorderRecipeImagesRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.ImageIDs) == 0 {
		respondError(w, http.StatusBadRequest, "image_ids must not be empty")
		return
	}

	if err := s.recipeService.ReorderRecipeImages(userID, noteID, req.ImageIDs); err != nil {
		switch {
		case errors.Is(err, service.ErrRecipeEncrypted):
			respondError(w, http.StatusBadRequest, "cannot reorder images on encrypted recipe")
		case errors.Is(err, service.ErrForbidden):
			respondError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, "image_ids must not be empty")
		default:
			respondError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
