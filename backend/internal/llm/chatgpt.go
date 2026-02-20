package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	// ChatGPTAPIURL is the OpenAI Chat Completions endpoint.
	ChatGPTAPIURL = "https://api.openai.com/v1/chat/completions"
	// ChatGPTDefaultModel is the default OpenAI model to use.
	ChatGPTDefaultModel = "gpt-4o-mini"
	// ChatGPTDefaultTimeout is the default timeout for OpenAI API requests.
	ChatGPTDefaultTimeout = 60 * time.Second
	// ChatGPTDefaultMaxTokens is the default max tokens for OpenAI responses.
	ChatGPTDefaultMaxTokens = 1024
)

// ChatGPTClient is an HTTP client for the OpenAI Chat Completions API.
type ChatGPTClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	maxTokens  int

	// Cached availability result (avoids repeated API calls)
	availableCache       *bool
	availableCacheExpiry time.Time
}

type openAITextPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type openAIImageURL struct {
	URL string `json:"url"`
}

type openAIImagePart struct {
	Type     string         `json:"type"`
	ImageURL openAIImageURL `json:"image_url"`
}

type openAIMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

type openAIChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
}

type openAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewChatGPTClient creates a new ChatGPT client with the given API key.
func NewChatGPTClient(apiKey string) *ChatGPTClient {
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = ChatGPTDefaultModel
	}

	return &ChatGPTClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: ChatGPTDefaultTimeout,
		},
		maxTokens: ChatGPTDefaultMaxTokens,
	}
}

// NewChatGPTClientWithConfig creates a new ChatGPT client with explicit configuration.
func NewChatGPTClientWithConfig(apiKey, model string, timeout time.Duration, maxTokens int) *ChatGPTClient {
	if model == "" {
		model = ChatGPTDefaultModel
	}
	if timeout == 0 {
		timeout = ChatGPTDefaultTimeout
	}
	if maxTokens == 0 {
		maxTokens = ChatGPTDefaultMaxTokens
	}

	return &ChatGPTClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxTokens: maxTokens,
	}
}

// Summarize generates a summary for the given text content using ChatGPT.
func (c *ChatGPTClient) Summarize(ctx context.Context, content string) (string, error) {
	if content == "" {
		return "", nil
	}

	prompt := BuildSummarizePrompt(content)
	return c.Generate(ctx, prompt, c.maxTokens)
}

// Generate sends a prompt to ChatGPT and returns the response.
func (c *ChatGPTClient) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	if prompt == "" {
		return "", nil
	}
	if maxTokens <= 0 {
		maxTokens = c.maxTokens
	}

	req := openAIRequest{
		Model: c.model,
		Messages: []openAIMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   maxTokens,
		Temperature: 0.7,
	}

	var openAIResp openAIResponse
	err := doJSONRequest(ctx, c.httpClient, "POST", ChatGPTAPIURL, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + c.apiKey,
	}, req, &openAIResp, parseOpenAIError, "openai")
	if err != nil {
		return "", err
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	return CleanMarkdownCodeBlock(openAIResp.Choices[0].Message.Content), nil
}

// GenerateWithImage sends a prompt with an image to ChatGPT and returns the response.
func (c *ChatGPTClient) GenerateWithImage(ctx context.Context, prompt string, imageData []byte, mimeType string, maxTokens int) (string, error) {
	if prompt == "" {
		return "", nil
	}
	if len(imageData) == 0 {
		return "", fmt.Errorf("image data is empty")
	}
	if maxTokens <= 0 {
		maxTokens = c.maxTokens
	}

	dataURL := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(imageData)
	req := openAIRequest{
		Model: c.model,
		Messages: []openAIMessage{
			{
				Role: "user",
				Content: []interface{}{
					openAITextPart{
						Type: "text",
						Text: prompt,
					},
					openAIImagePart{
						Type: "image_url",
						ImageURL: openAIImageURL{
							URL: dataURL,
						},
					},
				},
			},
		},
		MaxTokens:   maxTokens,
		Temperature: 0.7,
	}

	var openAIResp openAIResponse
	err := doJSONRequest(ctx, c.httpClient, "POST", ChatGPTAPIURL, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + c.apiKey,
	}, req, &openAIResp, parseOpenAIError, "openai")
	if err != nil {
		return "", err
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	return CleanMarkdownCodeBlock(openAIResp.Choices[0].Message.Content), nil
}

// IsAvailable checks if the OpenAI API is reachable with the configured API key.
// Results are cached for 5 minutes to avoid repeated API calls.
func (c *ChatGPTClient) IsAvailable(ctx context.Context) bool {
	if c.apiKey == "" {
		return false
	}

	if c.availableCache != nil && time.Now().Before(c.availableCacheExpiry) {
		return *c.availableCache
	}

	req := openAIRequest{
		Model: c.model,
		Messages: []openAIMessage{
			{
				Role:    "user",
				Content: "Hi",
			},
		},
		MaxTokens: 1,
	}

	var openAIResp openAIResponse
	err := doJSONRequest(ctx, c.httpClient, "POST", ChatGPTAPIURL, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + c.apiKey,
	}, req, &openAIResp, parseOpenAIError, "openai")
	if err != nil {
		return false
	}

	available := true
	c.availableCache = &available
	c.availableCacheExpiry = time.Now().Add(5 * time.Minute)
	return available
}

// Name returns the provider name for ChatGPTClient.
func (c *ChatGPTClient) Name() string {
	return string(ProviderTypeChatGPT)
}

// Model returns the configured model name.
func (c *ChatGPTClient) Model() string {
	return c.model
}

func parseOpenAIError(body []byte) error {
	var errResp openAIErrorResponse
	if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
		return fmt.Errorf("openai API error (%s): %s", errResp.Error.Type, errResp.Error.Message)
	}
	return nil
}
