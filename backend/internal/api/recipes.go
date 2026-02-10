package api

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/auth"
	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/service"
)

// --- Request/Response types ---

type UpdateRecipeMetadataRequest struct {
	Servings          int     `json:"servings"`
	PrepTimeMinutes   *int    `json:"prep_time_minutes,omitempty"`
	CookTimeMinutes   *int    `json:"cook_time_minutes,omitempty"`
	SourceURL         *string `json:"source_url,omitempty"`
	Difficulty        *string `json:"difficulty,omitempty"`
	ExpectedUpdatedAt string  `json:"expected_updated_at"`
}

type SetRecipeIngredientsRequest struct {
	Ingredients       []db.RecipeIngredient `json:"ingredients"`
	ExpectedUpdatedAt string                `json:"expected_updated_at"`
}

type CreateCollectionRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
}

type UpdateCollectionRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
}

type AddToCollectionRequest struct {
	NoteID string `json:"note_id"`
}

// --- Handlers ---

func (s *Server) listRecipes(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	recipes, err := s.recipeService.ListRecipes(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to list recipes", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"recipes": recipes,
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
		if errors.Is(err, db.ErrNotFound) {
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

	meta := &db.RecipeMetadata{
		Servings:        req.Servings,
		PrepTimeMinutes: req.PrepTimeMinutes,
		CookTimeMinutes: req.CookTimeMinutes,
		SourceURL:       req.SourceURL,
		Difficulty:      req.Difficulty,
	}

	err := s.recipeService.UpdateRecipeMetadata(userID, noteID, meta, req.ExpectedUpdatedAt)
	if err != nil {
		if errors.Is(err, db.ErrVersionMismatch) {
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
		respondError(w, http.StatusBadRequest, err.Error())
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
		if errors.Is(err, db.ErrVersionMismatch) {
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
		respondError(w, http.StatusBadRequest, err.Error())
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
		if errors.Is(err, db.ErrNotFound) {
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
		respondError(w, http.StatusBadRequest, err.Error())
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

	collID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid collection id")
		return
	}

	var req UpdateCollectionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.recipeService.UpdateCollection(userID, collID, req.Name, req.Description, req.Color); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "collection not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
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

	collID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid collection id")
		return
	}

	if err := s.recipeService.DeleteCollection(userID, collID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
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

	collID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid collection id")
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
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "collection not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
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

	collID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid collection id")
		return
	}

	noteID := chi.URLParam(r, "noteId")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note id is required")
		return
	}

	if err := s.recipeService.RemoveFromCollection(userID, collID, noteID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
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

	collID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid collection id")
		return
	}

	recipes, err := s.recipeService.ListCollectionItems(userID, collID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
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

// --- Collection Sharing handlers ---

// ShareCollectionRequest represents the request to share a collection.
type ShareCollectionRequest struct {
	Identifier string `json:"identifier"` // Username or email
	Role       string `json:"role"`       // "viewer" or "editor"
}

// UpdateCollectionShareRoleRequest represents the request to update a collection share role.
type UpdateCollectionShareRoleRequest struct {
	Role string `json:"role"` // "viewer" or "editor"
}

// shareCollection handles POST /api/recipes/collections/{id}/shares
func (s *Server) shareCollection(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid collection id")
		return
	}

	var req ShareCollectionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Identifier == "" {
		respondError(w, http.StatusBadRequest, "identifier (username or email) required")
		return
	}
	if req.Role != "viewer" && req.Role != "editor" {
		respondError(w, http.StatusBadRequest, "role must be 'viewer' or 'editor'")
		return
	}

	share, err := s.recipeService.ShareCollection(userID, collID, req.Identifier, req.Role)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrNotFound):
			respondError(w, http.StatusNotFound, "collection not found")
		case errors.Is(err, service.ErrNotCollectionOwner):
			respondError(w, http.StatusForbidden, "only the collection owner can share")
		case errors.Is(err, service.ErrCollectionHasEncryptedRecipes):
			respondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrCannotShareWithSelf):
			respondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrCollectionAlreadyShared):
			respondError(w, http.StatusConflict, err.Error())
		case errors.Is(err, service.ErrUserNotFound):
			respondError(w, http.StatusBadRequest, "unable to share with specified user")
		default:
			s.logger().Error("unexpected sharing error", "error", err)
			respondError(w, http.StatusInternalServerError, "an unexpected error occurred")
		}
		return
	}

	respondJSON(w, http.StatusCreated, share)
}

// getCollectionShares handles GET /api/recipes/collections/{id}/shares
func (s *Server) getCollectionShares(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid collection id")
		return
	}

	shares, err := s.recipeService.GetCollectionShares(userID, collID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "collection not found")
			return
		}
		if errors.Is(err, service.ErrNotCollectionOwner) {
			respondError(w, http.StatusForbidden, "only the collection owner can view shares")
			return
		}
		s.respondInternalErr(w, "failed to get collection shares", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"shares": shares,
	})
}

// updateCollectionShareRole handles PUT /api/recipes/collections/{id}/shares/{userId}
func (s *Server) updateCollectionShareRole(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid collection id")
		return
	}

	targetUserID, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req UpdateCollectionShareRoleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Role != "viewer" && req.Role != "editor" {
		respondError(w, http.StatusBadRequest, "role must be 'viewer' or 'editor'")
		return
	}

	if err := s.recipeService.UpdateCollectionShareRole(userID, collID, targetUserID, req.Role); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "share not found")
			return
		}
		if errors.Is(err, service.ErrNotCollectionOwner) {
			respondError(w, http.StatusForbidden, "only the collection owner can manage shares")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// removeCollectionShare handles DELETE /api/recipes/collections/{id}/shares/{userId}
func (s *Server) removeCollectionShare(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid collection id")
		return
	}

	targetUserID, err := strconv.Atoi(chi.URLParam(r, "userId"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := s.recipeService.UnshareCollection(userID, collID, targetUserID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "share not found")
			return
		}
		if errors.Is(err, service.ErrNotCollectionOwner) {
			respondError(w, http.StatusForbidden, "only the collection owner can manage shares")
			return
		}
		s.respondInternalErr(w, "failed to remove collection share", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

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

	if recipes == nil {
		recipes = []db.SharedNote{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"recipes": recipes,
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

	if collections == nil {
		collections = []db.SharedCollection{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"collections": collections,
	})
}

// listSharedCollectionItems handles GET /api/shared/collections/{id}/items
func (s *Server) listSharedCollectionItems(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid collection id")
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

	collID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid collection id")
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

	collID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid collection id")
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
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "not found")
			return
		}
		s.respondInternalErr(w, "failed to remove from shared collection", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Recipe Image handlers ---

// AddRecipeImageRequest represents the request to add an image to a recipe.
type AddRecipeImageRequest struct {
	ImageURL string  `json:"image_url"`
	Caption  *string `json:"caption,omitempty"`
}

// UpdateRecipeImageCaptionRequest represents the request to update an image caption.
type UpdateRecipeImageCaptionRequest struct {
	Caption *string `json:"caption"`
}

// ReorderRecipeImagesRequest represents the request to reorder images.
type ReorderRecipeImagesRequest struct {
	ImageIDs []int `json:"image_ids"`
}

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

// signRecipeImageURLs signs all image URLs in a RecipeDetail with fresh signatures.
func (s *Server) signRecipeImageURLs(detail *db.RecipeDetail) {
	for i, img := range detail.Images {
		signed, err := s.signImageURL(img.ImageURL)
		if err != nil {
			s.logger().Warn("failed to sign recipe image URL", "error", err, "url", img.ImageURL)
			continue
		}
		detail.Images[i].ImageURL = signed
	}
}

// signImageURL takes a base path like /api/uploads/{userID}/{filename}
// and returns a signed URL with signature and expiry query params.
func (s *Server) signImageURL(baseURL string) (string, error) {
	// Parse: /api/uploads/{userID}/{filename}
	trimmed := strings.TrimPrefix(baseURL, "/api/uploads/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid image URL format: %s", baseURL)
	}

	userIDStr := parts[0]
	filename := filepath.Base(parts[1])
	if filename == "." || filename == ".." || filename == "" {
		return "", fmt.Errorf("invalid filename in image URL: %s", baseURL)
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return "", fmt.Errorf("invalid user ID in image URL: %s", baseURL)
	}

	sig, expires, err := auth.GenerateUploadSignature(userID, filename, s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("generate signature: %w", err)
	}

	return fmt.Sprintf("/api/uploads/%d/%s?signature=%s&expires=%d", userID, filename, sig, expires), nil
}
