package llm

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/xela-io/xelanote/internal/crypto"
	"github.com/xela-io/xelanote/internal/db"
)

var (
	// ErrNoProviderAvailable is returned when no LLM provider is available.
	ErrNoProviderAvailable = errors.New("no LLM provider available - please add API key in settings")
	// ErrClaudeNotConfigured is returned when Claude is requested but no API key is configured.
	ErrClaudeNotConfigured = errors.New("Claude API key not configured")
	// ErrGeminiNotConfigured is returned when Gemini is requested but no API key is configured.
	ErrGeminiNotConfigured = errors.New("Gemini API key not configured")
	// ErrNoteNotAIEnabled is returned when trying to use AI features for a note that doesn't have AI enabled.
	ErrNoteNotAIEnabled = errors.New("AI features not enabled for this note")
	// ErrVisionNotAvailable is returned when no vision-capable LLM provider is available.
	ErrVisionNotAvailable = errors.New("no vision-capable LLM provider available")
)

// ProviderRouter routes LLM requests to the appropriate provider based on context.
// It manages provider instances and decides which provider to use for each request.
type ProviderRouter struct {
	db            *db.DB
	claudeClients map[int]*ClaudeClient // userID -> ClaudeClient
	geminiClients map[int]*GeminiClient // userID -> GeminiClient
	mu            sync.RWMutex
}

// NewProviderRouter creates a new provider router.
func NewProviderRouter(database *db.DB) *ProviderRouter {
	return &ProviderRouter{
		db:            database,
		claudeClients: make(map[int]*ClaudeClient),
		geminiClients: make(map[int]*GeminiClient),
	}
}

// GetProviderForNote returns the appropriate provider for a specific note.
// Returns a provider if:
//   - The note has ai_enabled=true
//   - The user has a Claude or Gemini API key configured
//
// Returns an error if:
//   - The note has ai_enabled=false (ErrNoteNotAIEnabled)
//   - No provider is configured (ErrNoProviderAvailable)
func (r *ProviderRouter) GetProviderForNote(ctx context.Context, userID int, noteID string) (Provider, error) {
	// Check if the note has AI enabled
	aiEnabled, err := r.db.GetNoteAIEnabled(userID, noteID)
	if err != nil {
		return nil, fmt.Errorf("failed to check note AI status: %w", err)
	}

	if !aiEnabled {
		return nil, ErrNoteNotAIEnabled
	}

	// Claude preferred
	if claude, err := r.getClaudeClient(userID); err == nil {
		return claude, nil
	}

	// Fallback: Gemini
	if gemini, err := r.getGeminiClient(userID); err == nil {
		return gemini, nil
	}

	return nil, ErrNoProviderAvailable
}

// GetAnyProvider returns any configured provider for the user (Claude preferred).
// This is used for features that don't require a specific note (e.g., spell-check).
func (r *ProviderRouter) GetAnyProvider(ctx context.Context, userID int) (Provider, error) {
	// Claude preferred
	if claude, err := r.getClaudeClient(userID); err == nil {
		return claude, nil
	}

	// Fallback: Gemini
	if gemini, err := r.getGeminiClient(userID); err == nil {
		return gemini, nil
	}

	return nil, ErrNoProviderAvailable
}

// GetClaudeProvider returns the Claude provider for a user if configured.
// Returns ErrClaudeNotConfigured if the user hasn't set up their API key.
func (r *ProviderRouter) GetClaudeProvider(ctx context.Context, userID int) (Provider, error) {
	return r.getClaudeClient(userID)
}

// GetGeminiProvider returns the Gemini provider for a user if configured.
// Returns ErrGeminiNotConfigured if the user hasn't set up their API key.
func (r *ProviderRouter) GetGeminiProvider(ctx context.Context, userID int) (Provider, error) {
	return r.getGeminiClient(userID)
}

// getClaudeClient returns or creates a Claude client for the given user.
// Clients are cached per user to avoid repeated decryption.
func (r *ProviderRouter) getClaudeClient(userID int) (*ClaudeClient, error) {
	r.mu.RLock()
	client, exists := r.claudeClients[userID]
	r.mu.RUnlock()

	if exists {
		return client, nil
	}

	// Need to create a new client
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := r.claudeClients[userID]; exists {
		return client, nil
	}

	// Get encrypted API key from database
	encryptedKey, err := r.db.GetClaudeAPIKey(userID)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, ErrClaudeNotConfigured
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	// Decrypt the API key
	apiKey, err := crypto.DecryptAPIKey(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt API key: %w", err)
	}

	// Create and cache the client
	client = NewClaudeClient(apiKey)
	r.claudeClients[userID] = client

	return client, nil
}

// InvalidateClaudeClient removes the cached Claude client for a user.
// Should be called when the user updates or deletes their API key.
func (r *ProviderRouter) InvalidateClaudeClient(userID int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.claudeClients, userID)
}

// getGeminiClient returns or creates a Gemini client for the given user.
// Clients are cached per user to avoid repeated decryption.
func (r *ProviderRouter) getGeminiClient(userID int) (*GeminiClient, error) {
	r.mu.RLock()
	client, exists := r.geminiClients[userID]
	r.mu.RUnlock()

	if exists {
		return client, nil
	}

	// Need to create a new client
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := r.geminiClients[userID]; exists {
		return client, nil
	}

	// Get encrypted API key from database
	encryptedKey, err := r.db.GetGeminiAPIKey(userID)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, ErrGeminiNotConfigured
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	// Decrypt the API key
	apiKey, err := crypto.DecryptAPIKey(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt API key: %w", err)
	}

	// Create and cache the client
	client = NewGeminiClient(apiKey)
	r.geminiClients[userID] = client

	return client, nil
}

// InvalidateGeminiClient removes the cached Gemini client for a user.
// Should be called when the user updates or deletes their API key.
func (r *ProviderRouter) InvalidateGeminiClient(userID int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.geminiClients, userID)
}

// SummarizeNote generates a summary for a note, using the appropriate provider.
func (r *ProviderRouter) SummarizeNote(ctx context.Context, userID int, noteID string, content string) (string, error) {
	provider, err := r.GetProviderForNote(ctx, userID, noteID)
	if err != nil {
		return "", err
	}

	return provider.Summarize(ctx, content)
}

// IsClaudeAvailableForNote checks if Claude can be used for a specific note.
func (r *ProviderRouter) IsClaudeAvailableForNote(ctx context.Context, userID int, noteID string) bool {
	// Check if note has AI enabled
	aiEnabled, err := r.db.GetNoteAIEnabled(userID, noteID)
	if err != nil || !aiEnabled {
		return false
	}

	// Check if user has Claude configured
	_, err = r.getClaudeClient(userID)
	return err == nil
}

// HasClaudeConfigured checks if a user has Claude API configured.
func (r *ProviderRouter) HasClaudeConfigured(userID int) bool {
	_, err := r.getClaudeClient(userID)
	return err == nil
}

// HasGeminiConfigured checks if a user has Gemini API configured.
func (r *ProviderRouter) HasGeminiConfigured(userID int) bool {
	_, err := r.getGeminiClient(userID)
	return err == nil
}

// GetVisionProvider returns a vision-capable provider for the user.
// Both Claude and Gemini implement VisionProvider, so this returns whichever is configured.
func (r *ProviderRouter) GetVisionProvider(ctx context.Context, userID int) (VisionProvider, error) {
	// Claude preferred
	if claude, err := r.getClaudeClient(userID); err == nil {
		return claude, nil
	}

	// Fallback: Gemini
	if gemini, err := r.getGeminiClient(userID); err == nil {
		return gemini, nil
	}

	return nil, ErrVisionNotAvailable
}

// HasAnyProviderConfigured checks if a user has any LLM provider configured.
func (r *ProviderRouter) HasAnyProviderConfigured(userID int) bool {
	return r.HasClaudeConfigured(userID) || r.HasGeminiConfigured(userID)
}

// GetProviderStatus returns status information about available providers for a user.
func (r *ProviderRouter) GetProviderStatus(ctx context.Context, userID int) *ProviderStatusInfo {
	status := &ProviderStatusInfo{
		ClaudeConfigured: false,
		GeminiConfigured: false,
	}

	if client, err := r.getClaudeClient(userID); err == nil {
		status.ClaudeConfigured = true
		status.ClaudeModel = client.Model()
		status.ClaudeAvailable = client.IsAvailable(ctx)
	}

	if client, err := r.getGeminiClient(userID); err == nil {
		status.GeminiConfigured = true
		status.GeminiModel = client.Model()
		status.GeminiAvailable = client.IsAvailable(ctx)
	}

	return status
}

// ProviderStatusInfo contains status information about LLM providers.
type ProviderStatusInfo struct {
	ClaudeConfigured bool   `json:"claude_configured"`
	ClaudeAvailable  bool   `json:"claude_available,omitempty"`
	ClaudeModel      string `json:"claude_model,omitempty"`
	GeminiConfigured bool   `json:"gemini_configured"`
	GeminiAvailable  bool   `json:"gemini_available,omitempty"`
	GeminiModel      string `json:"gemini_model,omitempty"`
}
