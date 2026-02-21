package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/xela-io/xelanote/internal/htmlutil"
)

func (s *Server) importRecipeFromURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req importRecipeFromURLRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.URL) == "" {
		respondError(w, http.StatusBadRequest, "url is required")
		return
	}

	recipe, err := s.recipeSuggestionService.ExtractRecipeFromURL(r.Context(), userID, req.URL, req.Locale)
	if err != nil {
		switch {
		case errors.Is(err, htmlutil.ErrInvalidURL), errors.Is(err, htmlutil.ErrDisallowedAddress):
			respondError(w, http.StatusBadRequest, "invalid or disallowed URL")
		case errors.Is(err, htmlutil.ErrFetchFailed):
			respondError(w, http.StatusBadGateway, "failed to fetch recipe URL")
		default:
			s.handleSuggestionError(w, err)
		}
		return
	}

	respondJSON(w, http.StatusOK, recipe)
}

func filterMainImageCandidates(candidates []string) []string {
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		u := strings.ToLower(strings.TrimSpace(candidate))
		if u == "" {
			continue
		}
		if strings.Contains(u, "logo") || strings.Contains(u, "icon") || strings.Contains(u, "avatar") {
			continue
		}
		result = append(result, candidate)
	}
	return result
}
