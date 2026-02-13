package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/llm"
	"github.com/xela-io/xelanote/internal/service"
)

// ============================================================================
// LLM Feature: Tag Suggestions
// ============================================================================

// suggestTags generates tag suggestions for a note using LLM.
func (s *Server) suggestTags(w http.ResponseWriter, r *http.Request) {
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

	var req SuggestTagsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		req = SuggestTagsRequest{}
	}

	// Get note to check if encrypted
	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil {
		s.respondInternalErr(w, "failed to get note for tag suggestions", err)
		return
	}
	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	// Determine content source
	var content string
	if note.ContentEncrypted {
		if req.PlaintextContent == "" {
			respondError(w, http.StatusBadRequest, "plaintext_content is required for encrypted notes")
			return
		}
		content = req.PlaintextContent
	} else {
		content = note.Content
	}

	// Validate content size
	if len(content) > service.MaxPlaintextContent {
		respondError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("content too large (max %d bytes)", service.MaxPlaintextContent))
		return
	}

	// Generate suggestions (include title for better context)
	suggestions, err := s.summarizeService.SuggestTagsForNote(r.Context(), userID, noteID, note.Title, content)
	if err != nil {
		s.logger().Error("tag suggestion failed",
			"error", err,
			"note_id", noteID,
			"user_id", userID,
		)

		switch {
		case errors.Is(err, llm.ErrNoteNotAIEnabled):
			respondError(w, http.StatusForbidden, "AI features not enabled for this note")
		case errors.Is(err, llm.ErrNoProviderAvailable):
			respondError(w, http.StatusPreconditionFailed, "AI provider required - add API key in settings")
		default:
			respondError(w, http.StatusInternalServerError, "failed to generate tag suggestions")
		}
		return
	}

	respondJSON(w, http.StatusOK, SuggestTagsResponse{
		Suggestions: suggestions,
	})
}

// ============================================================================
// LLM Feature: Link Suggestions
// ============================================================================

// suggestLinks generates wikilink suggestions for a note using LLM.
func (s *Server) suggestLinks(w http.ResponseWriter, r *http.Request) {
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

	var req SuggestLinksRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate title count
	if len(req.NoteTitles) > service.MaxNoteTitles {
		respondError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("too many note titles (max %d)", service.MaxNoteTitles))
		return
	}

	// Get note to check if encrypted
	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil {
		s.respondInternalErr(w, "failed to get note for link suggestions", err)
		return
	}
	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	// Determine content source
	var content string
	if note.ContentEncrypted {
		if req.PlaintextContent == "" {
			respondError(w, http.StatusBadRequest, "plaintext_content is required for encrypted notes")
			return
		}
		content = req.PlaintextContent
	} else {
		content = note.Content
	}

	// Validate content size
	if len(content) > service.MaxPlaintextContent {
		respondError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("content too large (max %d bytes)", service.MaxPlaintextContent))
		return
	}

	// Generate suggestions (using SuggestLinksForNote with userID and noteID)
	suggestions, err := s.summarizeService.SuggestLinksForNote(r.Context(), userID, noteID, content, req.NoteTitles, req.ExistingLinks)
	if err != nil {
		s.logger().Error("link suggestion failed",
			"error", err,
			"note_id", noteID,
			"user_id", userID,
		)

		switch {
		case errors.Is(err, llm.ErrNoteNotAIEnabled):
			respondError(w, http.StatusForbidden, "AI features not enabled for this note")
		case errors.Is(err, llm.ErrNoProviderAvailable):
			respondError(w, http.StatusPreconditionFailed, "AI provider required - add API key in settings")
		default:
			respondError(w, http.StatusInternalServerError, "failed to generate link suggestions")
		}
		return
	}

	respondJSON(w, http.StatusOK, SuggestLinksResponse{
		Suggestions: suggestions,
	})
}

// ============================================================================
// AI-Enabled (Claude API Opt-In) Endpoints
// ============================================================================

// updateNoteAIEnabled toggles the ai_enabled flag for a note.
// When ai_enabled=true, Cloud-KI features (Claude API) are allowed for this note.
func (s *Server) updateNoteAIEnabled(w http.ResponseWriter, r *http.Request) {
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

	var req UpdateAIEnabledRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Check if note exists and belongs to user
	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil {
		s.respondInternalErr(w, "failed to get note", err)
		return
	}
	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	// Update ai_enabled flag
	if err := s.noteService.UpdateNoteAIEnabled(userID, noteID, req.AIEnabled); err != nil {
		s.respondInternalErr(w, "failed to update AI enabled flag", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "ok",
		"ai_enabled": req.AIEnabled,
	})
}

// getNoteAIEnabled returns the ai_enabled status for a note.
func (s *Server) getNoteAIEnabled(w http.ResponseWriter, r *http.Request) {
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

	aiEnabled, err := s.noteService.GetNoteAIEnabled(userID, noteID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		s.respondInternalErr(w, "failed to get AI enabled status", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ai_enabled": aiEnabled,
	})
}
