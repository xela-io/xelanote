package api

import (
	"bytes"
	"errors"
	"image"
	// Register image decoders for content sniffing
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"

	"github.com/xela-io/xelanote/internal/llm"
	"github.com/xela-io/xelanote/internal/service"
	// WebP decoder (requires golang.org/x/image)
	_ "golang.org/x/image/webp"
)

// --- Request/Response types ---

type similarRecipesRequest struct {
	NoteID       string `json:"note_id"`
	CollectionID *int   `json:"collection_id,omitempty"`
	Locale       string `json:"locale"`
}

type byIngredientsRequest struct {
	Ingredients  []string `json:"ingredients"`
	CollectionID *int     `json:"collection_id,omitempty"`
	Locale       string   `json:"locale"`
}

// --- Handlers ---

func (s *Server) findSimilarRecipes(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req similarRecipesRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.NoteID == "" {
		respondError(w, http.StatusBadRequest, "note_id is required")
		return
	}

	results, err := s.recipeSuggestionService.FindSimilarRecipes(
		r.Context(), userID, req.NoteID, req.CollectionID, req.Locale,
	)
	if err != nil {
		s.handleSuggestionError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
	})
}

func (s *Server) suggestByIngredients(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req byIngredientsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Ingredients) == 0 {
		respondError(w, http.StatusBadRequest, "at least one ingredient is required")
		return
	}

	result, err := s.recipeSuggestionService.SuggestByIngredients(
		r.Context(), userID, req.Ingredients, req.CollectionID, req.Locale,
	)
	if err != nil {
		s.handleSuggestionError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func (s *Server) saveGeneratedRecipe(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req service.SaveGeneratedRecipeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	note, err := s.recipeSuggestionService.SaveGeneratedRecipe(userID, req)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"note_id": note.ID,
		"title":   note.Title,
	})
}

func (s *Server) extractIngredientsFromPhoto(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse multipart form (5MB limit)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		respondError(w, http.StatusRequestEntityTooLarge, "file too large (max 5MB)")
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		respondError(w, http.StatusBadRequest, "image file is required")
		return
	}
	defer file.Close()

	imageData, err := io.ReadAll(io.LimitReader(file, 5<<20+1))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to read image")
		return
	}
	if len(imageData) > 5<<20 {
		respondError(w, http.StatusRequestEntityTooLarge, "image exceeds 5MB limit")
		return
	}

	// Content-sniff: decode the image to verify it's actually an image
	_, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid image file: could not decode")
		return
	}

	// Derive MIME type from decoded format (not from HTTP header)
	decodedMime := "image/" + format

	locale := r.FormValue("locale")

	ingredients, err := s.recipeSuggestionService.ExtractIngredientsFromPhoto(
		r.Context(), userID, imageData, decodedMime, locale,
	)
	if err != nil {
		s.handleSuggestionError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ingredients": ingredients,
	})
}

// handleSuggestionError maps service-layer errors to appropriate HTTP status codes.
func (s *Server) handleSuggestionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, llm.ErrNoProviderAvailable):
		respondError(w, http.StatusFailedDependency, "no AI provider configured - add API key in settings")
	case errors.Is(err, llm.ErrVisionNotAvailable):
		respondError(w, http.StatusFailedDependency, "no vision-capable AI provider available")
	case errors.Is(err, service.ErrRecipeEncrypted):
		respondError(w, http.StatusForbidden, "not available for encrypted recipes")
	default:
		errMsg := err.Error()
		if isUpstreamError(errMsg) {
			respondError(w, http.StatusBadGateway, "AI provider temporarily unavailable")
		} else {
			s.log.Error("recipe suggestion error", "error", errMsg)
			respondError(w, http.StatusInternalServerError, "failed to process request")
		}
	}
}

// isUpstreamError checks if an error message indicates an upstream LLM failure.
func isUpstreamError(msg string) bool {
	for _, pattern := range []string{"API error", "returned status", "failed to send request"} {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}
