package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/llm"
	"github.com/xela-io/xelanote/internal/service"
)

// --- Summary Endpoints ---

// SummarizeNoteRequest represents the request body for summarizing a note.
type SummarizeNoteRequest struct {
	// PlaintextContent is required for encrypted notes (decrypted content from frontend)
	PlaintextContent string `json:"plaintext_content,omitempty"`
	// PlaintextContentHash is the SHA256 hash of plaintext (for E2E notes, computed by frontend)
	PlaintextContentHash string `json:"plaintext_content_hash,omitempty"`
	// EncryptedSummary is the encrypted summary (for E2E notes, encrypted by frontend)
	EncryptedSummary string `json:"encrypted_summary,omitempty"`
}

// SummarizeNoteResponse represents the response from summarizing a note.
type SummarizeNoteResponse struct {
	Summary string `json:"summary"`
}

// summarizeNote generates or retrieves a summary for a note.
// For plaintext notes: generates summary server-side
// For encrypted notes: requires frontend to send decrypted content, generates summary, returns for frontend encryption
func (s *Server) summarizeNote(w http.ResponseWriter, r *http.Request) {
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

	var req SummarizeNoteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		// Allow empty body for plaintext notes
		req = SummarizeNoteRequest{}
	}

	// Check if this is storing a pre-encrypted summary (E2E flow)
	if req.EncryptedSummary != "" && req.PlaintextContentHash != "" {
		// Store pre-encrypted summary
		if err := s.summarizeService.SummarizeNoteEncrypted(
			r.Context(),
			userID,
			noteID,
			req.EncryptedSummary,
			req.PlaintextContentHash,
		); err != nil {
			s.logger().Error("failed to store encrypted summary",
				"error", err,
				"note_id", noteID,
				"user_id", userID,
			)
			respondError(w, http.StatusInternalServerError, "failed to store summary")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
		return
	}

	// Generate summary (for plaintext or intermediate E2E step)
	summary, err := s.summarizeService.SummarizeNote(
		r.Context(),
		userID,
		noteID,
		req.PlaintextContent,
	)
	if err != nil {
		s.logger().Error("failed to summarize note",
			"error", err,
			"note_id", noteID,
			"user_id", userID,
		)

		// Check for specific error types
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		if strings.Contains(err.Error(), "plaintext content required") {
			respondError(w, http.StatusBadRequest, "plaintext_content is required for encrypted notes")
			return
		}

		respondError(w, http.StatusInternalServerError, "failed to generate summary")
		return
	}

	respondJSON(w, http.StatusOK, SummarizeNoteResponse{
		Summary: summary,
	})
}

// prepareSummarizeStream stores plaintext content and returns a one-time token
// for the SSE stream endpoint, avoiding plaintext in URL query parameters.
func (s *Server) prepareSummarizeStream(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		PlaintextContent string `json:"plaintext_content"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.PlaintextContent) > streamStoreMaxContentSize {
		respondError(w, http.StatusRequestEntityTooLarge, "content too large")
		return
	}

	token, err := generateStreamToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate stream token")
		return
	}

	if err := s.streamContent.store(token, userID, noteID, req.PlaintextContent); err != nil {
		respondError(w, http.StatusTooManyRequests, "too many pending stream requests, try again later")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"stream_token": token})
}

// summarizeNoteStream generates a summary with Server-Sent Events for progress.
func (s *Server) summarizeNoteStream(w http.ResponseWriter, r *http.Request) {
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

	// Get plaintext_content via stream token (preferred) or query parameter (deprecated)
	var plaintextContent string
	streamToken := r.URL.Query().Get("token")
	if streamToken != "" {
		entry, err := s.streamContent.get(streamToken)
		if err != nil || entry.userID != userID || entry.noteID != noteID {
			respondError(w, http.StatusBadRequest, "invalid or expired stream token")
			return
		}
		plaintextContent = entry.plaintextContent
	} else if pc := r.URL.Query().Get("plaintext_content"); pc != "" {
		s.logger().Warn("deprecated: plaintext_content in query param, use /prepare endpoint",
			"note_id", noteID, "user_id", userID)
		plaintextContent = pc
	}

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Extend write deadline for SSE streaming (5 minutes max)
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Now().Add(5 * time.Minute)); err != nil {
		s.logger().Warn("failed to extend SSE write deadline", "error", err)
	}

	// Send initial event
	fmt.Fprintf(w, "event: start\ndata: {\"status\":\"generating\"}\n\n")
	flusher.Flush()

	// Generate summary with streaming callback
	cached, summary, err := s.summarizeService.SummarizeNoteStream(
		r.Context(),
		userID,
		noteID,
		plaintextContent,
		func(token string) {
			data, err := json.Marshal(token)
			if err != nil {
				s.logger().Error("failed to marshal SSE token", "error", err)
				fmt.Fprintf(w, "event: error\ndata: {\"error\":\"internal encoding error\"}\n\n")
				flusher.Flush()
				return
			}
			fmt.Fprintf(w, "event: token\ndata: %s\n\n", data)
			flusher.Flush()
		},
	)

	if err != nil {
		s.logger().Error("failed to summarize note (stream)",
			"error", err,
			"note_id", noteID,
			"user_id", userID,
		)
		errPayload, marshalErr := json.Marshal(map[string]string{"error": err.Error()})
		if marshalErr != nil {
			s.logger().Error("failed to marshal SSE error", "error", marshalErr)
			errPayload = []byte(`{"error":"internal error"}`)
		}
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", errPayload)
		flusher.Flush()
		return
	}

	// Send completion event
	if cached {
		// For cached summaries, send the full text at once
		cachedPayload, marshalErr := json.Marshal(map[string]string{"summary": summary})
		if marshalErr != nil {
			s.logger().Error("failed to marshal SSE cached summary", "error", marshalErr)
			fmt.Fprintf(w, "event: error\ndata: {\"error\":\"internal encoding error\"}\n\n")
		} else {
			fmt.Fprintf(w, "event: cached\ndata: %s\n\n", cachedPayload)
		}
	} else {
		fmt.Fprintf(w, "event: done\ndata: {\"status\":\"complete\"}\n\n")
	}
	flusher.Flush()
}

// ============================================================================
// LLM Feature: Tag Suggestions
// ============================================================================

// SuggestTagsRequest represents the request body for tag suggestions.
type SuggestTagsRequest struct {
	PlaintextContent string `json:"plaintext_content,omitempty"`
}

// SuggestTagsResponse represents the response from tag suggestions.
type SuggestTagsResponse struct {
	Suggestions []service.TagSuggestion `json:"suggestions"`
}

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

// SuggestLinksRequest represents the request body for link suggestions.
type SuggestLinksRequest struct {
	PlaintextContent string   `json:"plaintext_content,omitempty"`
	NoteTitles       []string `json:"note_titles"`
	ExistingLinks    []string `json:"existing_links"`
}

// SuggestLinksResponse represents the response from link suggestions.
type SuggestLinksResponse struct {
	Suggestions []service.LinkSuggestion `json:"suggestions"`
}

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

// UpdateAIEnabledRequest represents the request body for toggling ai_enabled.
type UpdateAIEnabledRequest struct {
	AIEnabled bool `json:"ai_enabled"`
}

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
		if errors.Is(err, db.ErrNotFound) {
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

// ============================================================================
// LLM Feature: Format Markdown
// ============================================================================

// FormatMarkdownRequest represents the request body for formatting markdown.
type FormatMarkdownRequest struct {
	PlaintextContent string `json:"plaintext_content,omitempty"` // For E2E-encrypted notes
	SelectionOnly    string `json:"selection_only,omitempty"`    // When only part is formatted
}

// FormatMarkdownResponse represents the response from formatting markdown.
type FormatMarkdownResponse struct {
	FormattedContent string `json:"formatted_content"`
}

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

	// Determine content to format
	var content string
	if req.SelectionOnly != "" {
		// Formatting a selection
		content = req.SelectionOnly
	} else if req.PlaintextContent != "" {
		// Encrypted note: use provided plaintext
		content = req.PlaintextContent
	} else if note.ContentEncrypted {
		respondError(w, http.StatusBadRequest, "plaintext_content or selection_only is required for encrypted notes")
		return
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

// ============================================================================
// LLM Feature: AI Transform
// ============================================================================

// AITransformRequest represents the request body for AI text transformation.
type AITransformRequest struct {
	Action           string `json:"action"`                      // format, summarize, expand, translate_de, translate_en, formal, informal, custom
	Content          string `json:"content,omitempty"`           // Plain text content
	PlaintextContent string `json:"plaintext_content,omitempty"` // E2E decrypted content (takes precedence)
	CustomPrompt     string `json:"custom_prompt,omitempty"`     // Only for action="custom"
}

// AITransformResponse represents the response from AI text transformation.
type AITransformResponse struct {
	TransformedContent string `json:"transformed_content"`
}

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
		service.ActionFormat:      true,
		service.ActionSummarize:   true,
		service.ActionExpand:      true,
		service.ActionTranslateDE: true,
		service.ActionTranslateEN: true,
		service.ActionFormal:      true,
		service.ActionInformal:    true,
		service.ActionCustom:      true,
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

// listNoteTitlesAIEnabled returns titles of notes with ai_enabled=true.
// Used for Claude API link suggestions (only AI-enabled notes are included).
func (s *Server) listNoteTitlesAIEnabled(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	titles, err := s.noteService.GetNoteTitlesAIEnabled(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to list AI-enabled note titles", err)
		return
	}

	if titles == nil {
		titles = []string{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"titles": titles,
	})
}
