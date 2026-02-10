package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/xela-io/xelanote/internal/llm"
	"github.com/xela-io/xelanote/internal/service"
)

// ============================================================================
// LLM Feature: Spell Check
// ============================================================================

// SpellCheckRequest represents the request body for spell checking.
type SpellCheckRequest struct {
	Text     string `json:"text"`
	Language string `json:"language"` // "de" or "en"
}

// SpellCheckResponse represents the response from spell checking.
type SpellCheckResponse struct {
	Issues []service.SpellIssue `json:"issues"`
}

// spellCheck performs LLM-based spell checking on the provided text.
func (s *Server) spellCheck(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req SpellCheckRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Text == "" {
		respondJSON(w, http.StatusOK, SpellCheckResponse{
			Issues: []service.SpellIssue{},
		})
		return
	}

	// Validate text size
	if len(req.Text) > service.MaxSpellCheckText {
		respondError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("text too large (max %d bytes)", service.MaxSpellCheckText))
		return
	}

	// Default language to English if not specified or invalid
	language := req.Language
	if language != "de" && language != "en" {
		language = "en"
	}

	// Perform spell check (now requires userID for provider lookup)
	issues, err := s.summarizeService.SpellCheck(r.Context(), userID, req.Text, language)
	if err != nil {
		s.logger().Error("spell check failed",
			"error", err,
			"user_id", userID,
		)
		// Distinguish "no provider" from other errors
		if errors.Is(err, llm.ErrNoProviderAvailable) {
			respondError(w, http.StatusPreconditionFailed, "AI provider required - add API key in settings")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to check spelling")
		return
	}

	respondJSON(w, http.StatusOK, SpellCheckResponse{
		Issues: issues,
	})
}
