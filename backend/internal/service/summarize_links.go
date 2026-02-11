package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xela-io/xelanote/internal/llm"
)

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
