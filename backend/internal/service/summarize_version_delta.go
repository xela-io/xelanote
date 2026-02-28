package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/xela-io/xelanote/internal/llm"
)

// SummarizeVersionDelta generates a concise change summary between two note versions.
// Access is gated by note ai_enabled via ProviderRouter.GetProviderForNote.
func (s *SummarizeService) SummarizeVersionDelta(
	ctx context.Context,
	userID int,
	noteID string,
	fromVersion int,
	toVersion int,
	fromContent string,
	toContent string,
) (string, error) {
	if strings.TrimSpace(fromContent) == "" || strings.TrimSpace(toContent) == "" {
		return "", ErrContentEmpty
	}

	if len(fromContent) > MaxPlaintextContent || len(toContent) > MaxPlaintextContent {
		return "", ErrContentTooLarge
	}

	provider, err := s.router.GetProviderForNote(ctx, userID, noteID)
	if err != nil {
		return "", err
	}

	prompt := llm.BuildVersionDeltaSummaryPrompt(fromVersion, toVersion, fromContent, toContent)
	response, err := provider.Generate(ctx, prompt, 1200)
	if err != nil {
		return "", fmt.Errorf("delta summary failed: %w", err)
	}

	result := strings.TrimSpace(stripCodeBlocks(response))
	if result == "" {
		return "", fmt.Errorf("delta summary empty")
	}

	return result, nil
}
