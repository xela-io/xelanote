package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/xela-io/xelanote/internal/llm"
)

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
	ActionFormat           AITransformAction = "format"
	ActionSummarize        AITransformAction = "summarize"
	ActionExpand           AITransformAction = "expand"
	ActionTranslateDE      AITransformAction = "translate_de"
	ActionTranslateEN      AITransformAction = "translate_en"
	ActionFormal           AITransformAction = "formal"
	ActionInformal         AITransformAction = "informal"
	ActionCustom           AITransformAction = "custom"
	ActionDictationCleanup AITransformAction = "dictation_cleanup"
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
	case ActionDictationCleanup:
		return 4000
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
	case ActionFormat, ActionSummarize, ActionExpand, ActionTranslateDE, ActionTranslateEN, ActionFormal, ActionInformal, ActionCustom, ActionDictationCleanup:
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
	case ActionDictationCleanup:
		return llm.BuildDictationCleanupPrompt(content)
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

	// Validate content (dictation_cleanup allows shorter input)
	if action == ActionDictationCleanup {
		trimmed := strings.TrimSpace(content)
		if len(trimmed) == 0 {
			return "", ErrContentEmpty
		}
		if len(content) > FormatMaxLength {
			return "", ErrContentTooLarge
		}
	} else if err := validateFormatContent(content); err != nil {
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
