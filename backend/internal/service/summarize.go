// Package service contains the business logic for xelanote.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
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

// ============================================================================
// LLM Feature: Tag Suggestions
// ============================================================================

// TagSuggestion represents a suggested tag from the LLM
type TagSuggestion struct {
	Name  string  `json:"name"`
	IsNew bool    `json:"is_new"`
	Score float64 `json:"score"`
}

// MaxTagSuggestions is the maximum number of tag suggestions to return
const MaxTagSuggestions = 5

// MaxPlaintextContent is the maximum content size for tag/link/summary suggestions (50KB)
const MaxPlaintextContent = 50000

// SuggestTags generates tag suggestions for the given content.
// If noteID is provided and the note has ai_enabled=true, Claude/Gemini will be used.
func (s *SummarizeService) SuggestTags(ctx context.Context, userID int, title, content string) ([]TagSuggestion, error) {
	return s.SuggestTagsForNote(ctx, userID, "", title, content)
}

// SuggestTagsForNote generates tag suggestions for a specific note.
// Uses Claude/Gemini if the note has ai_enabled=true and user has a provider configured.
func (s *SummarizeService) SuggestTagsForNote(ctx context.Context, userID int, noteID, title, content string) ([]TagSuggestion, error) {
	if content == "" {
		return nil, fmt.Errorf("content is required for tag suggestions")
	}

	// Validate content size
	if len(content) > MaxPlaintextContent {
		return nil, fmt.Errorf("content too large (max %d bytes)", MaxPlaintextContent)
	}

	// Get all existing tags for the user
	existingTags, err := s.db.GetAllTags(userID)
	if err != nil {
		s.logger.Error("failed to get existing tags", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to load existing tags: %w", err)
	}

	// Extract tag names
	existingTagNames := make([]string, len(existingTags))
	existingTagSet := make(map[string]bool)
	for i, tag := range existingTags {
		existingTagNames[i] = tag.Name
		existingTagSet[strings.ToLower(tag.Name)] = true
	}

	// Get the appropriate provider
	var provider llm.Provider
	if noteID != "" {
		p, err := s.router.GetProviderForNote(ctx, userID, noteID)
		if err != nil {
			return nil, fmt.Errorf("no AI provider available: %w", err)
		}
		provider = p
	} else {
		// For new notes without noteID, use any available provider
		p, err := s.router.GetAnyProvider(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("no AI provider available: %w", err)
		}
		provider = p
	}

	// Build and send prompt (including title for better context)
	prompt := llm.BuildTagSuggestionPrompt(title, content, existingTagNames)
	response, err := provider.Generate(ctx, prompt, 500)
	if err != nil {
		s.logger.Error("LLM tag suggestion failed", "error", err, "provider", provider.Name())
		return nil, fmt.Errorf("failed to generate suggestions: %w", err)
	}

	// Parse JSON response
	suggestions, err := parseTagSuggestions(response, existingTagSet)
	if err != nil {
		s.logger.Warn("failed to parse LLM response", "error", err, "response", response)
		return nil, fmt.Errorf("failed to parse suggestions: %w", err)
	}

	// Limit to MaxTagSuggestions
	if len(suggestions) > MaxTagSuggestions {
		suggestions = suggestions[:MaxTagSuggestions]
	}

	return suggestions, nil
}

// parseTagSuggestions parses the LLM JSON response into TagSuggestion structs
func parseTagSuggestions(response string, existingTagSet map[string]bool) ([]TagSuggestion, error) {
	// Clean up response - LLM might include markdown code blocks
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// Try to find JSON array in response
	startIdx := strings.Index(response, "[")
	endIdx := strings.LastIndex(response, "]")
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return nil, fmt.Errorf("no JSON array found in response")
	}
	response = response[startIdx : endIdx+1]

	// Parse JSON
	var rawSuggestions []struct {
		Name  string  `json:"name"`
		Score float64 `json:"score"`
	}

	if err := json.Unmarshal([]byte(response), &rawSuggestions); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	// Convert to TagSuggestion with is_new flag
	suggestions := make([]TagSuggestion, 0, len(rawSuggestions))
	for _, raw := range rawSuggestions {
		name := strings.TrimSpace(strings.ToLower(raw.Name))
		if name == "" {
			continue
		}

		isNew := !existingTagSet[name]

		suggestions = append(suggestions, TagSuggestion{
			Name:  name,
			IsNew: isNew,
			Score: raw.Score,
		})
	}

	return suggestions, nil
}

// ============================================================================
// LLM Feature: Link Suggestions
// ============================================================================

// LinkSuggestion represents a suggested wikilink from the LLM
type LinkSuggestion struct {
	Term        string  `json:"term"`         // The term in the text to link
	TargetTitle string  `json:"target_title"` // The note title to link to
	Confidence  float64 `json:"confidence"`
}

// MaxLinkSuggestions is the maximum number of link suggestions to return
const MaxLinkSuggestions = 10

// MaxNoteTitles is the maximum number of note titles to send to LLM
const MaxNoteTitles = 500

// SuggestLinks generates link suggestions for the given content.
func (s *SummarizeService) SuggestLinks(ctx context.Context, userID int, content string, noteTitles, existingLinks []string) ([]LinkSuggestion, error) {
	return s.SuggestLinksForNote(ctx, userID, "", content, noteTitles, existingLinks)
}

// SuggestLinksForNote generates link suggestions for a specific note.
// Uses Claude/Gemini if the note has ai_enabled=true and user has a provider configured.
func (s *SummarizeService) SuggestLinksForNote(ctx context.Context, userID int, noteID, content string, noteTitles, existingLinks []string) ([]LinkSuggestion, error) {
	if content == "" {
		return nil, fmt.Errorf("content is required for link suggestions")
	}

	// Validate content size
	if len(content) > MaxPlaintextContent {
		return nil, fmt.Errorf("content too large (max %d bytes)", MaxPlaintextContent)
	}

	// Validate title count
	if len(noteTitles) > MaxNoteTitles {
		return nil, fmt.Errorf("too many note titles (max %d)", MaxNoteTitles)
	}

	// Get the appropriate provider
	var provider llm.Provider
	if noteID != "" && userID > 0 {
		p, err := s.router.GetProviderForNote(ctx, userID, noteID)
		if err != nil {
			return nil, fmt.Errorf("no AI provider available: %w", err)
		}
		provider = p
	} else {
		// For new notes without noteID, use any available provider
		p, err := s.router.GetAnyProvider(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("no AI provider available: %w", err)
		}
		provider = p
	}

	// Build and send prompt
	prompt := llm.BuildLinkSuggestionPrompt(content, noteTitles, existingLinks)
	response, err := provider.Generate(ctx, prompt, 1000)
	if err != nil {
		s.logger.Error("LLM link suggestion failed", "error", err, "provider", provider.Name())
		return nil, fmt.Errorf("failed to generate suggestions: %w", err)
	}

	// Parse JSON response
	suggestions, err := parseLinkSuggestions(response, noteTitles, existingLinks)
	if err != nil {
		s.logger.Warn("failed to parse LLM response", "error", err, "response", response)
		return nil, fmt.Errorf("failed to parse suggestions: %w", err)
	}

	// Limit to MaxLinkSuggestions
	if len(suggestions) > MaxLinkSuggestions {
		suggestions = suggestions[:MaxLinkSuggestions]
	}

	return suggestions, nil
}

// parseLinkSuggestions parses the LLM JSON response into LinkSuggestion structs
func parseLinkSuggestions(response string, validTitles, existingLinks []string) ([]LinkSuggestion, error) {
	// Clean up response
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// Try to find JSON array
	startIdx := strings.Index(response, "[")
	endIdx := strings.LastIndex(response, "]")
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return nil, fmt.Errorf("no JSON array found in response")
	}
	response = response[startIdx : endIdx+1]

	// Parse JSON
	var rawSuggestions []struct {
		Term        string  `json:"term"`
		TargetTitle string  `json:"target_title"`
		Confidence  float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(response), &rawSuggestions); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	// Build lookup sets for validation
	validTitleSet := make(map[string]string) // lowercase -> original
	for _, t := range validTitles {
		validTitleSet[strings.ToLower(t)] = t
	}

	existingLinkSet := make(map[string]bool)
	for _, l := range existingLinks {
		existingLinkSet[strings.ToLower(l)] = true
	}

	// Filter and validate suggestions
	suggestions := make([]LinkSuggestion, 0, len(rawSuggestions))
	for _, raw := range rawSuggestions {
		term := strings.TrimSpace(raw.Term)
		targetTitle := strings.TrimSpace(raw.TargetTitle)
		if term == "" || targetTitle == "" {
			continue
		}

		// Validate target is in the valid titles list
		originalTitle, ok := validTitleSet[strings.ToLower(targetTitle)]
		if !ok {
			continue
		}

		// Skip if already linked
		if existingLinkSet[strings.ToLower(targetTitle)] {
			continue
		}

		suggestions = append(suggestions, LinkSuggestion{
			Term:        term,
			TargetTitle: originalTitle, // Use original casing
			Confidence:  raw.Confidence,
		})
	}

	return suggestions, nil
}

// ============================================================================
// LLM Feature: Spell Check
// ============================================================================

// SpellIssue represents a spelling or grammar issue found by the LLM
type SpellIssue struct {
	ByteOffset  int      `json:"byte_offset"` // UTF-8 byte position in original text
	ByteLength  int      `json:"byte_length"` // UTF-8 byte length of the issue
	Original    string   `json:"original"`    // The problematic text for fallback matching
	Message     string   `json:"message"`     // Description of the issue
	Suggestions []string `json:"suggestions"` // Correction suggestions
	Type        string   `json:"type"`        // "spelling" or "grammar"
}

// MaxSpellCheckText is the maximum text size for spell checking (10KB)
const MaxSpellCheckText = 10000

// MaxSpellIssues is the maximum number of issues to return
const MaxSpellIssues = 20

// SpellCheck checks the given text for spelling and grammar issues
func (s *SummarizeService) SpellCheck(ctx context.Context, userID int, text, language string) ([]SpellIssue, error) {
	if text == "" {
		return []SpellIssue{}, nil
	}

	// Validate text size
	if len(text) > MaxSpellCheckText {
		return nil, fmt.Errorf("text too large (max %d bytes)", MaxSpellCheckText)
	}

	// Validate language
	if language != "de" && language != "en" {
		language = "en" // Default to English
	}

	// Get any available provider for the user
	provider, err := s.router.GetAnyProvider(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("no AI provider configured - please add API key in settings: %w", err)
	}

	// Build and send prompt
	prompt := llm.BuildSpellCheckPrompt(text, language)
	response, err := provider.Generate(ctx, prompt, 2000)
	if err != nil {
		s.logger.Error("LLM spell check failed", "error", err, "provider", provider.Name())
		return nil, fmt.Errorf("failed to check spelling: %w", err)
	}

	// Parse JSON response
	issues, err := parseSpellIssues(response, text)
	if err != nil {
		s.logger.Warn("failed to parse LLM response", "error", err, "response", response)
		return nil, fmt.Errorf("failed to parse spell check results: %w", err)
	}

	// Limit to MaxSpellIssues
	if len(issues) > MaxSpellIssues {
		issues = issues[:MaxSpellIssues]
	}

	return issues, nil
}

// parseSpellIssues parses the LLM JSON response and calculates byte offsets
func parseSpellIssues(response, originalText string) ([]SpellIssue, error) {
	// Clean up response
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// Try to find JSON array
	startIdx := strings.Index(response, "[")
	endIdx := strings.LastIndex(response, "]")
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		// Empty array is valid (no issues found)
		if strings.Contains(response, "[]") {
			return []SpellIssue{}, nil
		}
		return nil, fmt.Errorf("no JSON array found in response")
	}
	response = response[startIdx : endIdx+1]

	// Parse JSON - handle both flat array and nested array (LLM sometimes returns [[...]])
	type rawIssue struct {
		Original    string   `json:"original"`
		Message     string   `json:"message"`
		Suggestions []string `json:"suggestions"`
		Type        string   `json:"type"`
	}

	var rawIssues []rawIssue

	// First try to parse as flat array
	if err := json.Unmarshal([]byte(response), &rawIssues); err != nil {
		// If that fails, try to parse as nested array [[...]]
		var nestedIssues [][]rawIssue
		if nestedErr := json.Unmarshal([]byte(response), &nestedIssues); nestedErr != nil {
			return nil, fmt.Errorf("JSON parse error: %w (nested: %v)", err, nestedErr)
		}
		// Flatten the nested array
		for _, inner := range nestedIssues {
			rawIssues = append(rawIssues, inner...)
		}
	}

	// Calculate byte offsets by finding each original text in the content
	issues := make([]SpellIssue, 0, len(rawIssues))
	searchFrom := 0

	for _, raw := range rawIssues {
		original := strings.TrimSpace(raw.Original)
		if original == "" {
			continue
		}

		// Find the original text in the content (case-insensitive)
		// Start searching from the last found position to handle duplicates
		idx := strings.Index(strings.ToLower(originalText[searchFrom:]), strings.ToLower(original))
		if idx == -1 {
			// Try from the beginning if not found
			idx = strings.Index(strings.ToLower(originalText), strings.ToLower(original))
			if idx == -1 {
				// Still not found - skip this issue
				continue
			}
		} else {
			idx += searchFrom
		}

		// Calculate byte offset and length
		byteOffset := len(originalText[:idx])
		byteLength := len(original)

		issues = append(issues, SpellIssue{
			ByteOffset:  byteOffset,
			ByteLength:  byteLength,
			Original:    original,
			Message:     raw.Message,
			Suggestions: raw.Suggestions,
			Type:        raw.Type,
		})

		// Update search position to after this match
		searchFrom = idx + len(original)
	}

	return issues, nil
}

// ============================================================================
// LLM Feature: AI Transform (Format, Summarize, Expand, Translate, Tone)
// ============================================================================

// Validation errors for AI Transform
var (
	ErrContentEmpty         = errors.New("content is empty")
	ErrContentTooShort      = errors.New("content too short")
	ErrContentTooLarge      = errors.New("content too large")
	ErrResponseTooLarge     = errors.New("response too large")
	ErrUnknownAction        = errors.New("unknown action")
	ErrCustomPromptRequired = errors.New("custom_prompt is required for custom action")
)

// AITransformAction represents the type of AI transformation to perform
type AITransformAction string

const (
	ActionFormat      AITransformAction = "format"
	ActionSummarize   AITransformAction = "summarize"
	ActionExpand      AITransformAction = "expand"
	ActionTranslateDE AITransformAction = "translate_de"
	ActionTranslateEN AITransformAction = "translate_en"
	ActionFormal      AITransformAction = "formal"
	ActionInformal    AITransformAction = "informal"
	ActionCustom      AITransformAction = "custom"
)

// Content limits
const (
	FormatMinLength       = 10         // Minimum 10 characters
	FormatMaxLength       = 50 * 1024  // Maximum 50KB
	FormatMaxLines        = 1000       // Maximum 1000 lines
	FormatResponseLimit   = 60 * 1024  // Response max 60KB (default)
	ExpandedResponseLimit = 100 * 1024 // Response max 100KB (for expand/custom)
)

// validateFormatContent validates content for formatting.
func validateFormatContent(content string) error {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) == 0 {
		return ErrContentEmpty
	}
	if len(trimmed) < FormatMinLength {
		return ErrContentTooShort
	}
	if len(content) > FormatMaxLength {
		return ErrContentTooLarge
	}
	if strings.Count(content, "\n") > FormatMaxLines {
		return ErrContentTooLarge
	}
	return nil
}

// extractMarkdownFromResponse removes LLM preamble/postamble.
// Called AFTER stripCodeBlocks() which handles ```markdown fences.
func extractMarkdownFromResponse(response string) string {
	// stripCodeBlocks already handles fenced code blocks
	response = stripCodeBlocks(response)

	// Remove leading explanatory text (e.g., "Here is the formatted...")
	lines := strings.Split(response, "\n")
	startIdx := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Start at first markdown-like line or empty line
		if strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "-") ||
			strings.HasPrefix(trimmed, "*") ||
			strings.HasPrefix(trimmed, ">") ||
			strings.HasPrefix(trimmed, "```") ||
			strings.HasPrefix(trimmed, "1.") ||
			trimmed == "" {
			startIdx = i
			break
		}
	}

	return strings.TrimSpace(strings.Join(lines[startIdx:], "\n"))
}

// stripCodeBlocks removes markdown code fences from LLM responses.
func stripCodeBlocks(response string) string {
	response = strings.TrimSpace(response)

	// Remove ```markdown or ``` prefix
	if strings.HasPrefix(response, "```markdown") {
		response = strings.TrimPrefix(response, "```markdown")
		response = strings.TrimPrefix(response, "\n")
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimPrefix(response, "\n")
	}

	// Remove trailing ```
	if strings.HasSuffix(response, "```") {
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSuffix(response, "\n")
	}

	return strings.TrimSpace(response)
}

// getMaxTokensForAction returns the max_tokens limit for an AI action
func getMaxTokensForAction(action AITransformAction) int {
	switch action {
	case ActionExpand, ActionCustom:
		return 8000
	case ActionSummarize:
		return 2000
	default:
		return 4000
	}
}

// getResponseLimitForAction returns the response size limit for an AI action
func getResponseLimitForAction(action AITransformAction) int {
	switch action {
	case ActionExpand, ActionCustom:
		return ExpandedResponseLimit
	default:
		return FormatResponseLimit
	}
}

// isValidAction checks if the action is a known AITransformAction
func isValidAction(action AITransformAction) bool {
	switch action {
	case ActionFormat, ActionSummarize, ActionExpand, ActionTranslateDE, ActionTranslateEN, ActionFormal, ActionInformal, ActionCustom:
		return true
	default:
		return false
	}
}

// buildPromptForAction builds the appropriate prompt for the given action
func buildPromptForAction(action AITransformAction, content, customPrompt string) string {
	switch action {
	case ActionFormat:
		return llm.BuildFormatMarkdownPrompt(content)
	case ActionSummarize:
		return llm.BuildSummarizeSelectionPrompt(content)
	case ActionExpand:
		return llm.BuildExpandPrompt(content)
	case ActionTranslateDE:
		return llm.BuildTranslateToGermanPrompt(content)
	case ActionTranslateEN:
		return llm.BuildTranslateToEnglishPrompt(content)
	case ActionFormal:
		return llm.BuildFormalTonePrompt(content)
	case ActionInformal:
		return llm.BuildInformalTonePrompt(content)
	case ActionCustom:
		return llm.BuildCustomTransformPrompt(content, customPrompt)
	default:
		return llm.BuildFormatMarkdownPrompt(content)
	}
}

// AITransform performs an AI transformation on the given content.
// action: one of format, summarize, expand, translate_de, translate_en, formal, informal, custom
// customPrompt: only required when action is "custom"
func (s *SummarizeService) AITransform(
	ctx context.Context,
	userID int,
	noteID string,
	action AITransformAction,
	content string,
	customPrompt string,
) (string, error) {
	// Validate action
	if !isValidAction(action) {
		return "", ErrUnknownAction
	}

	// Validate custom prompt requirement
	if action == ActionCustom && strings.TrimSpace(customPrompt) == "" {
		return "", ErrCustomPromptRequired
	}

	// Validate content
	if err := validateFormatContent(content); err != nil {
		return "", err
	}

	// Get the appropriate provider (respects ai_enabled flag)
	provider, err := s.router.GetProviderForNote(ctx, userID, noteID)
	if err != nil {
		s.logger.Error("failed to get provider for note",
			slog.String("note_id", noteID),
			slog.Int("user_id", userID),
			slog.String("action", string(action)),
			slog.String("error", err.Error()),
		)
		return "", err
	}

	// Build prompt for the action
	prompt := buildPromptForAction(action, content, customPrompt)
	maxTokens := getMaxTokensForAction(action)
	responseLimit := getResponseLimitForAction(action)

	// Generate response
	response, err := provider.Generate(ctx, prompt, maxTokens)
	if err != nil {
		s.logger.Error("LLM AI transform failed",
			slog.String("note_id", noteID),
			slog.Int("user_id", userID),
			slog.String("action", string(action)),
			slog.String("provider", provider.Name()),
			slog.String("error", err.Error()),
		)
		return "", fmt.Errorf("transformation failed: %w", err)
	}

	// Extract clean text from response (for format action, use extractMarkdownFromResponse)
	var result string
	if action == ActionFormat {
		result = extractMarkdownFromResponse(response)
	} else {
		result = stripCodeBlocks(strings.TrimSpace(response))
	}

	// Validate response size
	if len(result) > responseLimit {
		s.logger.Error("LLM response too large",
			slog.String("note_id", noteID),
			slog.String("action", string(action)),
			slog.Int("response_size", len(result)),
			slog.Int("limit", responseLimit),
		)
		return "", ErrResponseTooLarge
	}

	s.logger.Info("AI transform completed successfully",
		slog.String("note_id", noteID),
		slog.Int("user_id", userID),
		slog.String("action", string(action)),
		slog.Int("input_length", len(content)),
		slog.Int("output_length", len(result)),
		slog.String("provider", provider.Name()),
	)

	return result, nil
}

// FormatMarkdown formats markdown content using an LLM provider.
// For notes with ai_enabled=true, uses GetProviderForNote.
// Returns the formatted markdown content.
// Deprecated: Use AITransform with ActionFormat instead. Kept for backward compatibility.
func (s *SummarizeService) FormatMarkdown(ctx context.Context, userID int, noteID string, content string) (string, error) {
	return s.AITransform(ctx, userID, noteID, ActionFormat, content, "")
}
