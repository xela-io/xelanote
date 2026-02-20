// Package service contains the business logic for xelanote.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/llm"
)

// SummarizeService handles LLM-based note summarization.
type SummarizeService struct {
	db     *db.DB
	router *llm.ProviderRouter
	logger *slog.Logger
	// Per-note mutex to prevent concurrent summarization of the same note
	locks sync.Map // map[string]*sync.Mutex
}

// NewSummarizeService creates a new SummarizeService with a ProviderRouter.
func NewSummarizeService(database *db.DB, router *llm.ProviderRouter, logger *slog.Logger) *SummarizeService {
	return &SummarizeService{
		db:     database,
		router: router,
		logger: logger,
	}
}

// acquireLock acquires a per-note mutex to prevent race conditions.
// Returns an unlock function that must be called when done.
func (s *SummarizeService) acquireLock(noteID string) func() {
	lock, _ := s.locks.LoadOrStore(noteID, &sync.Mutex{})
	lock.(*sync.Mutex).Lock()
	return lock.(*sync.Mutex).Unlock
}

// SummarizeNote generates a summary for a note.
// For unencrypted notes, content is read from the database.
// For encrypted notes, plaintext content must be provided.
// Uses idempotency check: skips if content hash is unchanged and summary exists.
func (s *SummarizeService) SummarizeNote(ctx context.Context, userID int, noteID string, plaintextContent string) (string, error) {
	// Acquire per-note lock to prevent race conditions
	unlock := s.acquireLock(noteID)
	defer unlock()

	// Get current note state
	note, err := s.db.GetNote(userID, noteID)
	if err != nil {
		return "", fmt.Errorf("failed to get note: %w", err)
	}
	if note == nil {
		return "", fmt.Errorf("note not found")
	}

	// Determine content source
	var content string
	if note.ContentEncrypted {
		// Encrypted note: use provided plaintext content
		if plaintextContent == "" {
			return "", fmt.Errorf("plaintext content required for encrypted notes")
		}
		content = plaintextContent
	} else {
		// Unencrypted note: use content from database
		content = note.Content
	}

	// Compute content hash
	newHash := db.ComputeContentHash(content)

	// Idempotency check: skip if hash is unchanged and summary exists
	if note.ContentHash != nil && *note.ContentHash == newHash &&
		note.SummaryGeneratedAt != nil {
		// Summary is up-to-date
		if note.ContentEncrypted && note.EncryptedSummary != nil {
			return *note.EncryptedSummary, nil
		}
		if !note.ContentEncrypted && note.Summary != nil {
			return *note.Summary, nil
		}
	}

	// Get the appropriate provider (Claude or Gemini if ai_enabled)
	provider, err := s.router.GetProviderForNote(ctx, userID, noteID)
	if err != nil {
		s.logger.Error("failed to get provider for note",
			slog.String("note_id", noteID),
			slog.Int("user_id", userID),
			slog.String("error", err.Error()),
		)
		return "", err
	}

	// Generate summary via LLM
	summary, err := provider.Summarize(ctx, content)
	if err != nil {
		s.logger.Error("LLM summarization failed",
			slog.String("note_id", noteID),
			slog.Int("user_id", userID),
			slog.String("provider", provider.Name()),
			slog.String("error", err.Error()),
		)
		return "", fmt.Errorf("summarization failed: %w", err)
	}

	s.logger.Debug("summary generated",
		slog.String("note_id", noteID),
		slog.String("provider", provider.Name()),
	)

	// Update content hash
	if err := s.db.UpdateNoteContentHash(userID, noteID, newHash); err != nil {
		s.logger.Error("failed to update content hash",
			slog.String("note_id", noteID),
			slog.String("error", err.Error()),
		)
		// Don't fail the request - summary is generated
	}

	// Store summary
	now := time.Now().UTC()
	if err := s.db.UpdateNoteSummary(userID, noteID, summary, note.ContentEncrypted, now); err != nil {
		s.logger.Error("failed to store summary",
			slog.String("note_id", noteID),
			slog.String("error", err.Error()),
		)
		return summary, fmt.Errorf("failed to store summary: %w", err)
	}

	s.logger.Info("note summarized successfully",
		slog.String("note_id", noteID),
		slog.Int("user_id", userID),
		slog.Int("summary_length", len(summary)),
	)

	return summary, nil
}

// SummarizeNoteStream generates a summary and returns it.
// Cloud providers don't support token streaming, so this calls SummarizeNote
// internally and returns the complete result.
func (s *SummarizeService) SummarizeNoteStream(ctx context.Context, userID int, noteID string, plaintextContent string, onToken func(token string)) (bool, string, error) {
	summary, err := s.SummarizeNote(ctx, userID, noteID, plaintextContent)
	if err != nil {
		return false, "", err
	}
	return true, summary, nil
}

// SummarizeNoteEncrypted stores a pre-encrypted summary for an encrypted note.
// Used when the frontend encrypts the summary before sending it to the server.
func (s *SummarizeService) SummarizeNoteEncrypted(ctx context.Context, userID int, noteID, encryptedSummary, plaintextContentHash string) error {
	unlock := s.acquireLock(noteID)
	defer unlock()

	// Get current note state
	note, err := s.db.GetNote(userID, noteID)
	if err != nil {
		return fmt.Errorf("failed to get note: %w", err)
	}
	if note == nil {
		return fmt.Errorf("note not found")
	}

	if !note.ContentEncrypted {
		return fmt.Errorf("note is not encrypted")
	}

	// Update content hash with the plaintext hash from frontend
	if err := s.db.UpdateNoteContentHash(userID, noteID, plaintextContentHash); err != nil {
		s.logger.Error("failed to update content hash",
			slog.String("note_id", noteID),
			slog.String("error", err.Error()),
		)
	}

	// Store encrypted summary
	now := time.Now().UTC()
	if err := s.db.UpdateNoteSummary(userID, noteID, encryptedSummary, true, now); err != nil {
		return fmt.Errorf("failed to store encrypted summary: %w", err)
	}

	s.logger.Info("encrypted summary stored",
		slog.String("note_id", noteID),
		slog.Int("user_id", userID),
	)

	return nil
}

// SummarizePendingNotes processes notes that need summary generation.
// Only processes unencrypted notes for users who have a provider configured.
// Returns the number of notes processed.
func (s *SummarizeService) SummarizePendingNotes(ctx context.Context, limit int) (int, error) {
	notes, err := s.db.GetNotesNeedingSummary(limit)
	if err != nil {
		return 0, fmt.Errorf("failed to get notes needing summary: %w", err)
	}

	processed := 0
	for _, note := range notes {
		select {
		case <-ctx.Done():
			s.logger.Info("summarization interrupted",
				slog.Int("processed", processed),
				slog.Int("total", len(notes)),
			)
			return processed, ctx.Err()
		default:
		}

		// Check if user has any provider configured
		if !s.router.HasAnyProviderConfigured(note.UserID) {
			continue
		}

		// Compute current hash
		currentHash := db.ComputeContentHash(note.Content)

		// Skip if content hash matches and we have a hash (already summarized with this content)
		if note.ContentHash != nil && *note.ContentHash == currentHash {
			continue
		}

		// Generate summary
		_, err := s.SummarizeNote(ctx, note.UserID, note.ID, "")
		if err != nil {
			s.logger.Warn("failed to summarize note",
				slog.String("note_id", note.ID),
				slog.Int("user_id", note.UserID),
				slog.String("error", err.Error()),
			)
			// Continue with next note
			continue
		}

		processed++
	}

	return processed, nil
}

// GetProviderStatus returns status information about available LLM providers for a user.
func (s *SummarizeService) GetProviderStatus(ctx context.Context, userID int) *llm.ProviderStatusInfo {
	return s.router.GetProviderStatus(ctx, userID)
}

// InvalidateClaudeClient invalidates the cached Claude client for a user.
// Should be called when the user updates or deletes their API key.
func (s *SummarizeService) InvalidateClaudeClient(userID int) {
	s.router.InvalidateClaudeClient(userID)
}

// IsClaudeAvailableForNote checks if Claude can be used for a specific note.
func (s *SummarizeService) IsClaudeAvailableForNote(ctx context.Context, userID int, noteID string) bool {
	return s.router.IsClaudeAvailableForNote(ctx, userID, noteID)
}

// HasClaudeConfigured checks if a user has Claude API configured.
func (s *SummarizeService) HasClaudeConfigured(userID int) bool {
	return s.router.HasClaudeConfigured(userID)
}

// InvalidateGeminiClient invalidates the cached Gemini client for a user.
// Should be called when the user updates or deletes their API key.
func (s *SummarizeService) InvalidateGeminiClient(userID int) {
	s.router.InvalidateGeminiClient(userID)
}

// HasGeminiConfigured checks if a user has Gemini API configured.
func (s *SummarizeService) HasGeminiConfigured(userID int) bool {
	return s.router.HasGeminiConfigured(userID)
}

// InvalidateChatGPTClient invalidates the cached ChatGPT client for a user.
// Should be called when the user updates or deletes their API key.
func (s *SummarizeService) InvalidateChatGPTClient(userID int) {
	s.router.InvalidateChatGPTClient(userID)
}

// HasChatGPTConfigured checks if a user has ChatGPT API configured.
func (s *SummarizeService) HasChatGPTConfigured(userID int) bool {
	return s.router.HasChatGPTConfigured(userID)
}

// InvalidateAllAIClients invalidates all cached AI clients for a user.
func (s *SummarizeService) InvalidateAllAIClients(userID int) {
	s.router.InvalidateAllClients(userID)
}
