package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

const (
	// WhisperAPIURL is the OpenAI Audio Transcriptions endpoint.
	WhisperAPIURL = "https://api.openai.com/v1/audio/transcriptions"
	// WhisperModel is the model used for audio transcription.
	WhisperModel = "gpt-4o-mini-transcribe"
	// WhisperMaxFileSize is the maximum audio file size (25 MB, OpenAI limit).
	WhisperMaxFileSize = 25 * 1024 * 1024
)

type whisperResponse struct {
	Text string `json:"text"`
}

// Transcribe sends audio data to OpenAI's Whisper API and returns the transcribed text.
// The filename should include the appropriate extension (e.g., "audio.webm").
// Language is auto-detected by the API.
func (c *ChatGPTClient) Transcribe(ctx context.Context, audioData []byte, filename string) (string, error) {
	if len(audioData) == 0 {
		return "", fmt.Errorf("audio data is empty")
	}
	if len(audioData) > WhisperMaxFileSize {
		return "", fmt.Errorf("audio file too large (max %d MB)", WhisperMaxFileSize/(1024*1024))
	}

	// Build multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add audio file
	filePart, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := filePart.Write(audioData); err != nil {
		return "", fmt.Errorf("failed to write audio data: %w", err)
	}

	// Add model field
	if err := writer.WriteField("model", WhisperModel); err != nil {
		return "", fmt.Errorf("failed to write model field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", WhisperAPIURL, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("whisper API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse OpenAI error
		if apiErr := parseOpenAIError(body); apiErr != nil {
			return "", apiErr
		}
		return "", fmt.Errorf("whisper API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result whisperResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse whisper response: %w", err)
	}

	return result.Text, nil
}
