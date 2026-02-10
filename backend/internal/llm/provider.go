// Package llm provides LLM provider abstractions and implementations.
package llm

import "context"

// Provider defines the interface for LLM providers (Claude, Gemini).
// All LLM providers must implement this interface.
type Provider interface {
	// Summarize generates a summary for the given text content.
	// Returns the summary or an error if generation fails.
	Summarize(ctx context.Context, content string) (string, error)

	// Generate sends a prompt to the LLM and returns the response.
	// maxTokens controls the maximum number of tokens in the response.
	Generate(ctx context.Context, prompt string, maxTokens int) (string, error)

	// IsAvailable checks if the provider is reachable and configured.
	IsAvailable(ctx context.Context) bool

	// Name returns the provider name (e.g., "claude", "gemini").
	Name() string
}

// ProviderType represents the type of LLM provider.
type ProviderType string

const (
	// ProviderTypeClaude represents the Anthropic Claude API provider.
	ProviderTypeClaude ProviderType = "claude"
	// ProviderTypeGemini represents the Google Gemini API provider.
	ProviderTypeGemini ProviderType = "gemini"
)

// VisionProvider extends Provider with image understanding capabilities.
// Providers that support multimodal input (text + image) implement this interface.
type VisionProvider interface {
	Provider
	// GenerateWithImage sends a prompt with an image to the LLM and returns the response.
	// imageData is the raw image bytes, mimeType is the validated MIME type (e.g., "image/jpeg").
	GenerateWithImage(ctx context.Context, prompt string, imageData []byte, mimeType string, maxTokens int) (string, error)
}

// Interface compliance checks
var (
	_ Provider       = (*ClaudeClient)(nil)
	_ Provider       = (*GeminiClient)(nil)
	_ VisionProvider = (*ClaudeClient)(nil)
	_ VisionProvider = (*GeminiClient)(nil)
)
