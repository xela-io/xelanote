package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// GeminiAPIURL is the base URL for the Google Gemini API.
	GeminiAPIURL = "https://generativelanguage.googleapis.com/v1beta/models"
	// GeminiDefaultModel is the default Gemini model to use.
	// Using gemini-2.5-flash as it has better free-tier availability.
	GeminiDefaultModel = "gemini-2.5-flash"
	// GeminiDefaultTimeout is the default timeout for Gemini API requests.
	GeminiDefaultTimeout = 60 * time.Second
	// GeminiDefaultMaxTokens is the default max tokens for Gemini responses.
	GeminiDefaultMaxTokens = 1024
)

// GeminiClient is an HTTP client for the Google Gemini API.
type GeminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	maxTokens  int

	// Cached availability result (avoids repeated API calls)
	availableCache       *bool
	availableCacheExpiry time.Time
}

// GeminiPart represents a part of a message content.
type GeminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *GeminiInlineData `json:"inline_data,omitempty"`
}

// GeminiInlineData holds base64-encoded inline data for the Gemini Vision API.
type GeminiInlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

// GeminiContent represents message content in Gemini API format.
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

// GeminiGenerationConfig contains generation configuration.
type GeminiGenerationConfig struct {
	MaxOutputTokens  int     `json:"maxOutputTokens,omitempty"`
	Temperature      float64 `json:"temperature,omitempty"`
	ResponseMimeType string  `json:"responseMimeType,omitempty"`
	// ThinkingConfig controls the thinking/reasoning feature
	ThinkingConfig *GeminiThinkingConfig `json:"thinkingConfig,omitempty"`
}

// GeminiThinkingConfig controls the thinking feature.
type GeminiThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget"` // 0 to disable thinking
}

// GeminiRequest represents a request to the Gemini generateContent endpoint.
type GeminiRequest struct {
	Contents         []GeminiContent         `json:"contents"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

// GeminiCandidate represents a response candidate.
type GeminiCandidate struct {
	Content      GeminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
	Index        int           `json:"index"`
}

// GeminiUsageMetadata contains token usage information.
type GeminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
	ThoughtsTokenCount   int `json:"thoughtsTokenCount,omitempty"`
}

// GeminiResponse represents a response from the Gemini generateContent endpoint.
type GeminiResponse struct {
	Candidates    []GeminiCandidate   `json:"candidates"`
	UsageMetadata GeminiUsageMetadata `json:"usageMetadata"`
	ModelVersion  string              `json:"modelVersion"`
}

// GeminiErrorResponse represents an error response from the Gemini API.
type GeminiErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// NewGeminiClient creates a new Gemini client with the given API key.
func NewGeminiClient(apiKey string) *GeminiClient {
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = GeminiDefaultModel
	}

	return &GeminiClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: GeminiDefaultTimeout,
		},
		maxTokens: GeminiDefaultMaxTokens,
	}
}

// NewGeminiClientWithConfig creates a new Gemini client with explicit configuration.
func NewGeminiClientWithConfig(apiKey, model string, timeout time.Duration, maxTokens int) *GeminiClient {
	if model == "" {
		model = GeminiDefaultModel
	}
	if timeout == 0 {
		timeout = GeminiDefaultTimeout
	}
	if maxTokens == 0 {
		maxTokens = GeminiDefaultMaxTokens
	}

	return &GeminiClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxTokens: maxTokens,
	}
}

// Summarize generates a summary for the given text content using Gemini.
func (c *GeminiClient) Summarize(ctx context.Context, content string) (string, error) {
	if content == "" {
		return "", nil
	}

	prompt := BuildSummarizePrompt(content)
	return c.Generate(ctx, prompt, c.maxTokens)
}

// Generate sends a prompt to Gemini and returns the response.
func (c *GeminiClient) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	if prompt == "" {
		return "", nil
	}

	if maxTokens <= 0 {
		maxTokens = c.maxTokens
	}

	req := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: &GeminiGenerationConfig{
			MaxOutputTokens: maxTokens,
			Temperature:     0.7,
			// Disable thinking to save tokens on simple tasks
			ThinkingConfig: &GeminiThinkingConfig{
				ThinkingBudget: 0,
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent", GeminiAPIURL, c.model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read error response: %w", err)
		}

		// Try to parse as Gemini error response
		var errResp GeminiErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return "", fmt.Errorf("gemini API error (%s): %s", errResp.Error.Status, errResp.Error.Message)
		}

		return "", fmt.Errorf("gemini returned status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract text from candidates
	if len(geminiResp.Candidates) == 0 {
		return "", fmt.Errorf("gemini returned no candidates")
	}

	var result string
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		result += part.Text
	}

	// Clean up markdown code blocks if present (Gemini sometimes wraps JSON in ```json...```)
	result = CleanMarkdownCodeBlock(result)

	return result, nil
}

// CleanMarkdownCodeBlock removes markdown code block formatting if present.
func CleanMarkdownCodeBlock(s string) string {
	s = strings.TrimSpace(s)

	// Check for ```json or ``` at start
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}

	// Check for ``` at end
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}

	return strings.TrimSpace(s)
}

// GenerateWithImage sends a prompt with an image to Gemini and returns the response.
func (c *GeminiClient) GenerateWithImage(ctx context.Context, prompt string, imageData []byte, mimeType string, maxTokens int) (string, error) {
	if prompt == "" {
		return "", nil
	}
	if len(imageData) == 0 {
		return "", fmt.Errorf("image data is empty")
	}

	if maxTokens <= 0 {
		maxTokens = c.maxTokens
	}

	req := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{
						InlineData: &GeminiInlineData{
							MimeType: mimeType,
							Data:     base64.StdEncoding.EncodeToString(imageData),
						},
					},
					{Text: prompt},
				},
			},
		},
		GenerationConfig: &GeminiGenerationConfig{
			MaxOutputTokens: maxTokens,
			Temperature:     0.7,
			ThinkingConfig: &GeminiThinkingConfig{
				ThinkingBudget: 0,
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent", GeminiAPIURL, c.model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read error response: %w", err)
		}

		var errResp GeminiErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return "", fmt.Errorf("gemini API error (%s): %s", errResp.Error.Status, errResp.Error.Message)
		}

		return "", fmt.Errorf("gemini returned status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 {
		return "", fmt.Errorf("gemini returned no candidates")
	}

	var result string
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		if part.Text != "" {
			result += part.Text
		}
	}

	result = CleanMarkdownCodeBlock(result)

	return result, nil
}

// IsAvailable checks if the Gemini API is reachable with the configured API key.
// Results are cached for 5 minutes to avoid repeated API calls.
func (c *GeminiClient) IsAvailable(ctx context.Context) bool {
	if c.apiKey == "" {
		return false
	}

	// Return cached result if still valid
	if c.availableCache != nil && time.Now().Before(c.availableCacheExpiry) {
		return *c.availableCache
	}

	// List models to verify the API key works
	httpReq, err := http.NewRequestWithContext(ctx, "GET", GeminiAPIURL, nil)
	if err != nil {
		return false
	}
	httpReq.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	available := resp.StatusCode == http.StatusOK
	c.availableCache = &available
	c.availableCacheExpiry = time.Now().Add(5 * time.Minute)
	return available
}

// Name returns the provider name for GeminiClient.
func (c *GeminiClient) Name() string {
	return string(ProviderTypeGemini)
}

// Model returns the configured model name.
func (c *GeminiClient) Model() string {
	return c.model
}

// Ensure GeminiClient implements Provider interface.
var _ Provider = (*GeminiClient)(nil)
