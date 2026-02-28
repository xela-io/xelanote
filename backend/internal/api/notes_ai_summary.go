package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/llm"
)

// --- Summary Endpoints ---

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
			switch {
			case errors.Is(err, llm.ErrNoteNotAIEnabled):
				respondError(w, http.StatusForbidden, "AI features not enabled for this note")
			case strings.Contains(err.Error(), "not found"):
				respondError(w, http.StatusNotFound, "note not found")
			default:
				respondError(w, http.StatusInternalServerError, "failed to store summary")
			}
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

		switch {
		case errors.Is(err, llm.ErrNoteNotAIEnabled):
			respondError(w, http.StatusForbidden, "AI features not enabled for this note")
			return
		case errors.Is(err, llm.ErrNoProviderAvailable):
			respondError(w, http.StatusPreconditionFailed, "AI provider required - add API key in settings")
			return
		case strings.Contains(err.Error(), "not found"):
			respondError(w, http.StatusNotFound, "note not found")
			return
		case strings.Contains(err.Error(), "plaintext content required"):
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

	// Get plaintext_content via one-time stream token.
	// Query parameter fallback was removed to prevent plaintext leakage via URLs.
	var plaintextContent string
	streamToken := strings.TrimSpace(r.URL.Query().Get("token"))
	if streamToken != "" {
		entry, err := s.streamContent.get(streamToken)
		if err != nil || entry.userID != userID || entry.noteID != noteID {
			respondError(w, http.StatusBadRequest, "invalid or expired stream token")
			return
		}
		plaintextContent = entry.plaintextContent
	} else if r.URL.Query().Get("plaintext_content") != "" {
		respondError(w, http.StatusBadRequest, "plaintext_content query parameter is no longer supported; use /summarize/prepare")
		return
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
		// Do not leak raw internal/provider error details to clients.
		clientError := "failed to generate summary"
		switch {
		case errors.Is(err, llm.ErrNoteNotAIEnabled):
			clientError = "AI features not enabled for this note"
		case errors.Is(err, llm.ErrNoProviderAvailable):
			clientError = "AI provider required - add API key in settings"
		case strings.Contains(err.Error(), "not found"):
			clientError = "note not found"
		case strings.Contains(err.Error(), "plaintext content required"):
			clientError = "plaintext_content is required for encrypted notes"
		}
		errPayload, marshalErr := json.Marshal(map[string]string{"error": clientError})
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
