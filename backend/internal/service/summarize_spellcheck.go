package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xela-io/xelanote/internal/llm"
)

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
