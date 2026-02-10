package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// ClaudeAPIURL is the base URL for the Anthropic Claude API.
	ClaudeAPIURL = "https://api.anthropic.com/v1/messages"
	// ClaudeDefaultModel is the default Claude model to use.
	ClaudeDefaultModel = "claude-3-haiku-20240307"
	// ClaudeAPIVersion is the required API version header.
	ClaudeAPIVersion = "2023-06-01"
	// ClaudeDefaultTimeout is the default timeout for Claude API requests.
	ClaudeDefaultTimeout = 60 * time.Second
	// ClaudeDefaultMaxTokens is the default max tokens for Claude responses.
	ClaudeDefaultMaxTokens = 1024
)

// ClaudeClient is an HTTP client for the Anthropic Claude API.
type ClaudeClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	maxTokens  int
}

// ClaudeMessage represents a message in the Claude API format.
// Content can be a string (text-only) or []ClaudeContentItem (multimodal).
type ClaudeMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// ClaudeContentItem represents a content block in a multimodal Claude message.
type ClaudeContentItem struct {
	Type   string             `json:"type"`
	Text   string             `json:"text,omitempty"`
	Source *ClaudeImageSource `json:"source,omitempty"`
}

// ClaudeImageSource holds base64-encoded image data for the Claude Vision API.
type ClaudeImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// ClaudeRequest represents a request to the Claude messages endpoint.
type ClaudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []ClaudeMessage `json:"messages"`
	System    string          `json:"system,omitempty"`
}

// ClaudeContentBlock represents a content block in Claude's response.
type ClaudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ClaudeResponse represents a response from the Claude messages endpoint.
type ClaudeResponse struct {
	ID           string               `json:"id"`
	Type         string               `json:"type"`
	Role         string               `json:"role"`
	Content      []ClaudeContentBlock `json:"content"`
	Model        string               `json:"model"`
	StopReason   string               `json:"stop_reason"`
	StopSequence string               `json:"stop_sequence,omitempty"`
	Usage        ClaudeUsage          `json:"usage"`
}

// ClaudeUsage contains token usage information.
type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ClaudeErrorResponse represents an error response from the Claude API.
type ClaudeErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewClaudeClient creates a new Claude client with the given API key.
func NewClaudeClient(apiKey string) *ClaudeClient {
	return &ClaudeClient{
		apiKey: apiKey,
		model:  ClaudeDefaultModel,
		httpClient: &http.Client{
			Timeout: ClaudeDefaultTimeout,
		},
		maxTokens: ClaudeDefaultMaxTokens,
	}
}

// NewClaudeClientWithConfig creates a new Claude client with explicit configuration.
func NewClaudeClientWithConfig(apiKey, model string, timeout time.Duration, maxTokens int) *ClaudeClient {
	if model == "" {
		model = ClaudeDefaultModel
	}
	if timeout == 0 {
		timeout = ClaudeDefaultTimeout
	}
	if maxTokens == 0 {
		maxTokens = ClaudeDefaultMaxTokens
	}

	return &ClaudeClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxTokens: maxTokens,
	}
}

// Summarize generates a summary for the given text content using Claude.
func (c *ClaudeClient) Summarize(ctx context.Context, content string) (string, error) {
	if content == "" {
		return "", nil
	}

	// Use the same prompt as Ollama for consistency
	prompt := BuildSummarizePrompt(content)

	return c.Generate(ctx, prompt, c.maxTokens)
}

// Generate sends a prompt to Claude and returns the response.
func (c *ClaudeClient) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	if prompt == "" {
		return "", nil
	}

	if maxTokens <= 0 {
		maxTokens = c.maxTokens
	}

	req := ClaudeRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", ClaudeAPIURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", ClaudeAPIVersion)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		// Try to parse as Claude error response
		var errResp ClaudeErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return "", fmt.Errorf("claude API error (%s): %s", errResp.Error.Type, errResp.Error.Message)
		}

		return "", fmt.Errorf("claude returned status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract text from content blocks
	var result string
	for _, block := range claudeResp.Content {
		if block.Type == "text" {
			result += block.Text
		}
	}

	return result, nil
}

// GenerateWithImage sends a prompt with an image to Claude and returns the response.
func (c *ClaudeClient) GenerateWithImage(ctx context.Context, prompt string, imageData []byte, mimeType string, maxTokens int) (string, error) {
	if prompt == "" {
		return "", nil
	}
	if len(imageData) == 0 {
		return "", fmt.Errorf("image data is empty")
	}

	if maxTokens <= 0 {
		maxTokens = c.maxTokens
	}

	req := ClaudeRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		Messages: []ClaudeMessage{
			{
				Role: "user",
				Content: []ClaudeContentItem{
					{
						Type: "image",
						Source: &ClaudeImageSource{
							Type:      "base64",
							MediaType: mimeType,
							Data:      base64.StdEncoding.EncodeToString(imageData),
						},
					},
					{
						Type: "text",
						Text: prompt,
					},
				},
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", ClaudeAPIURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", ClaudeAPIVersion)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		var errResp ClaudeErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return "", fmt.Errorf("claude API error (%s): %s", errResp.Error.Type, errResp.Error.Message)
		}

		return "", fmt.Errorf("claude returned status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	var result string
	for _, block := range claudeResp.Content {
		if block.Type == "text" {
			result += block.Text
		}
	}

	return result, nil
}

// IsAvailable checks if the Claude API is reachable with the configured API key.
// Makes a minimal API call to verify credentials.
func (c *ClaudeClient) IsAvailable(ctx context.Context) bool {
	if c.apiKey == "" {
		return false
	}

	// Send a minimal request to verify the API key works
	req := ClaudeRequest{
		Model:     c.model,
		MaxTokens: 1,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: "Hi",
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return false
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", ClaudeAPIURL, bytes.NewReader(reqBody))
	if err != nil {
		return false
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", ClaudeAPIVersion)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 200 means the API is working
	// 401 means invalid API key
	// Other errors might be temporary
	return resp.StatusCode == http.StatusOK
}

// Name returns the provider name for ClaudeClient.
func (c *ClaudeClient) Name() string {
	return string(ProviderTypeClaude)
}

// Model returns the configured model name.
func (c *ClaudeClient) Model() string {
	return c.model
}

// Ensure ClaudeClient implements Provider interface.
var _ Provider = (*ClaudeClient)(nil)
