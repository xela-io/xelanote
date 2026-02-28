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
// LLM Feature: Format Markdown
// ============================================================================

// formatMarkdown formats markdown content using an LLM provider.
// POST /api/notes/{id}/format-markdown
func (s *Server) formatMarkdown(w http.ResponseWriter, r *http.Request) {
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

	var req FormatMarkdownRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get note to determine content source
	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil {
		s.respondInternalErr(w, "failed to get note for formatting", err)
		return
	}
	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	// P0 privacy hardening: never process encrypted-note plaintext server-side.
	if note.ContentEncrypted {
		respondError(w, http.StatusForbidden, errEncryptedNoteAIProcessingDisabled)
		return
	}
	if strings.TrimSpace(req.PlaintextContent) != "" {
		respondError(w, http.StatusBadRequest, errAIPlaintextContentDeprecated)
		return
	}

	// Determine content to format
	var content string
	if req.SelectionOnly != "" {
		// Formatting a selection
		content = req.SelectionOnly
	} else {
		// Plaintext note: use content from database
		content = note.Content
	}

	// Format the content
	formatted, err := s.summarizeService.FormatMarkdown(r.Context(), userID, noteID, content)
	if err != nil {
		s.logger().Error("format markdown failed",
			"error", err,
			"note_id", noteID,
			"user_id", userID,
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
			respondError(w, http.StatusBadRequest, "No content to format")
		case errors.Is(err, r.Context().Err()):
			respondError(w, http.StatusGatewayTimeout, "Request timed out - try shorter selection")
		default:
			respondError(w, http.StatusInternalServerError, "Formatting failed")
		}
		return
	}

	respondJSON(w, http.StatusOK, FormatMarkdownResponse{
		FormattedContent: formatted,
	})
}
