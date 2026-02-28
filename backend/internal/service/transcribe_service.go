package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/xela-io/xelanote/internal/llm"
)

// TranscribeAudio transcribes audio data to text using the user's OpenAI API key (Whisper).
// This requires the ChatGPT/OpenAI provider specifically, as Whisper is an OpenAI-only feature.
func (s *SummarizeService) TranscribeAudio(ctx context.Context, userID int, audioData []byte, filename string) (string, error) {
	if len(audioData) == 0 {
		return "", fmt.Errorf("audio data is empty")
	}
	if len(audioData) > llm.WhisperMaxFileSize {
		return "", ErrContentTooLarge
	}

	client, err := s.router.GetChatGPTClient(userID)
	if err != nil {
		return "", fmt.Errorf("OpenAI provider required for transcription: %w", err)
	}

	text, err := client.Transcribe(ctx, audioData, filename)
	if err != nil {
		return "", fmt.Errorf("transcription failed: %w", err)
	}

	s.logger.Info("audio transcribed successfully",
		slog.Int("user_id", userID),
		slog.Int("audio_size", len(audioData)),
		slog.Int("text_length", len(text)),
	)

	return text, nil
}
