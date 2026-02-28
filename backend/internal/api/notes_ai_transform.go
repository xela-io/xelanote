package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/llm"
	"github.com/xela-io/xelanote/internal/service"
)

// ============================================================================
// LLM Feature: AI Transform
// ============================================================================

// aiTransform performs AI-based text transformation.
// POST /api/notes/{id}/ai-transform
func (s *Server) aiTransform(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID is required")
		return
	}

	var req AITransformRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate action
	action := service.AITransformAction(req.Action)
	validActions := map[service.AITransformAction]bool{
		service.ActionFormat:           true,
		service.ActionSummarize:        true,
		service.ActionExpand:           true,
		service.ActionTranslateDE:      true,
		service.ActionTranslateEN:      true,
		service.ActionFormal:           true,
		service.ActionInformal:         true,
		service.ActionCustom:           true,
		service.ActionDictationCleanup: true,
	}

	if !validActions[action] {
		respondError(w, http.StatusBadRequest, "unknown action")
		return
	}

	// Validate custom prompt for custom action
	if action == service.ActionCustom && strings.TrimSpace(req.CustomPrompt) == "" {
		respondError(w, http.StatusBadRequest, "custom_prompt is required for custom action")
		return
	}

	// Determine content source: PlaintextContent takes precedence
	content := req.PlaintextContent
	if content == "" {
		content = req.Content
	}

	if strings.TrimSpace(content) == "" {
		respondError(w, http.StatusBadRequest, "content is required")
		return
	}

	// Perform transformation
	result, err := s.summarizeService.AITransform(
		r.Context(),
		userID,
		noteID,
		action,
		content,
		req.CustomPrompt,
	)
	if err != nil {
		s.logger().Error("AI transform failed",
			"error", err,
			"note_id", noteID,
			"user_id", userID,
			"action", req.Action,
		)

		// Map errors to appropriate HTTP status codes
		switch {
		case errors.Is(err, llm.ErrNoProviderAvailable):
			respondError(w, http.StatusPreconditionFailed, "AI provider required - add API key in settings")
		case errors.Is(err, llm.ErrNoteNotAIEnabled):
			respondError(w, http.StatusForbidden, "AI features are disabled for this note")
		case errors.Is(err, service.ErrContentTooLarge):
			respondError(w, http.StatusRequestEntityTooLarge, "Content too large (max 50KB)")
		case errors.Is(err, service.ErrContentTooShort):
			respondError(w, http.StatusBadRequest, "Content too short (min 10 characters)")
		case errors.Is(err, service.ErrContentEmpty):
			respondError(w, http.StatusBadRequest, "No content to transform")
		case errors.Is(err, service.ErrResponseTooLarge):
			respondError(w, http.StatusRequestEntityTooLarge, "Response too large")
		case errors.Is(err, service.ErrUnknownAction):
			respondError(w, http.StatusBadRequest, "Unknown action")
		case errors.Is(err, service.ErrCustomPromptRequired):
			respondError(w, http.StatusBadRequest, "custom_prompt is required for custom action")
		case errors.Is(err, r.Context().Err()):
			respondError(w, http.StatusGatewayTimeout, "Request timed out - try shorter content")
		default:
			respondError(w, http.StatusInternalServerError, "Transformation failed")
		}
		return
	}

	respondJSON(w, http.StatusOK, AITransformResponse{
		TransformedContent: result,
	})
}
