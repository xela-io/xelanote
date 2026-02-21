package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/xela-io/xelanote/internal/llm"
	"github.com/xela-io/xelanote/internal/service"
)

const (
	maxAutoImportRecipeImages   = 3
	maxRecipeImageCandidates    = 8
	maxAutoImportedImageSize    = 5 << 20 // 5MB
	minMainRecipeImageDimension = 300
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

type importRecipeFromURLRequest struct {
	URL    string `json:"url"`
	Locale string `json:"locale"`
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

	// Sanitize ingredients: enforce allowlist, length limits, max count
	sanitized := llm.SanitizeIngredients(req.Ingredients)
	if len(sanitized) == 0 {
		respondError(w, http.StatusBadRequest, "no valid ingredients provided")
		return
	}
	req.Ingredients = sanitized

	result, err := s.recipeSuggestionService.SuggestByIngredients(
		r.Context(), userID, req.Ingredients, req.CollectionID, req.Locale,
	)
	if err != nil {
		s.handleSuggestionError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// handleSuggestionError maps service-layer errors to appropriate HTTP status codes.
func (s *Server) handleSuggestionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, llm.ErrNoProviderAvailable):
		respondError(w, http.StatusFailedDependency, "no AI provider configured - add API key in settings")
	case errors.Is(err, llm.ErrVisionNotAvailable):
		respondError(w, http.StatusFailedDependency, "no vision-capable AI provider available")
	case errors.Is(err, service.ErrNoRecipeFound):
		respondError(w, http.StatusUnprocessableEntity, "no recipe found in input")
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
