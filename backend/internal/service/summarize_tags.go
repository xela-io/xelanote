package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xela-io/xelanote/internal/llm"
)

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
