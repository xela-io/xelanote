package service

import "errors"

// --- Claude API Key Management (BYOK) ---

// Validation errors for Claude API Key
var (
	ErrInvalidClaudeAPIKey = errors.New("invalid Claude API key format")
	ErrNoClaudeAPIKey      = errors.New("no Claude API key configured")
)

// SetClaudeAPIKey validates, encrypts, and stores a Claude API key for a user.
// The key is encrypted with AES-256-GCM before storage.
func (s *UserService) SetClaudeAPIKey(userID int, apiKey string) error {
	// Import crypto package inline to avoid circular dependency
	// crypto.ValidateClaudeAPIKey and crypto.EncryptAPIKey will be used
	return setClaudeAPIKeyImpl(s.db, userID, apiKey)
}

// GetClaudeAPIKey retrieves and decrypts the Claude API key for a user.
// Returns ErrNoClaudeAPIKey if no key is stored.
func (s *UserService) GetClaudeAPIKey(userID int) (string, error) {
	return getClaudeAPIKeyImpl(s.db, userID)
}

// DeleteClaudeAPIKey removes the Claude API key for a user.
func (s *UserService) DeleteClaudeAPIKey(userID int) error {
	return s.db.DeleteClaudeAPIKey(userID)
}

// HasClaudeAPIKey checks if a user has a Claude API key stored.
func (s *UserService) HasClaudeAPIKey(userID int) (bool, error) {
	return s.db.HasClaudeAPIKey(userID)
}

// GetClaudeAPIKeyStatus returns status information about the stored API key.
// Does NOT return the actual key, only metadata.
func (s *UserService) GetClaudeAPIKeyStatus(userID int) (*ClaudeAPIKeyStatus, error) {
	hasKey, err := s.db.HasClaudeAPIKey(userID)
	if err != nil {
		return nil, err
	}

	if !hasKey {
		return &ClaudeAPIKeyStatus{
			HasKey:    false,
			UpdatedAt: nil,
			MaskedKey: nil,
		}, nil
	}

	updatedAt, err := s.db.GetClaudeAPIKeyUpdatedAt(userID)
	if err != nil {
		return nil, err
	}

	// Get masked key (decrypt and mask)
	decryptedKey, err := getClaudeAPIKeyImpl(s.db, userID)
	if err != nil {
		return nil, err
	}

	masked := maskClaudeAPIKey(decryptedKey)

	return &ClaudeAPIKeyStatus{
		HasKey:    true,
		UpdatedAt: updatedAt,
		MaskedKey: &masked,
	}, nil
}

// ClaudeAPIKeyStatus represents the status of a user's Claude API key.
type ClaudeAPIKeyStatus struct {
	HasKey    bool    `json:"has_key"`
	UpdatedAt *string `json:"updated_at,omitempty"`
	MaskedKey *string `json:"masked_key,omitempty"` // e.g., "sk-ant-api0...xxxx"
}

// maskClaudeAPIKey returns a masked version of the API key for display.
// Shows first 10 and last 4 characters.
func maskClaudeAPIKey(apiKey string) string {
	if len(apiKey) <= 14 {
		return "****"
	}
	return apiKey[:10] + "..." + apiKey[len(apiKey)-4:]
}

// --- Gemini API Key Management (BYOK) ---

// Validation errors for Gemini API Key
var (
	ErrInvalidGeminiAPIKey = errors.New("invalid Gemini API key format")
	ErrNoGeminiAPIKey      = errors.New("no Gemini API key configured")
)

// SetGeminiAPIKey validates, encrypts, and stores a Gemini API key for a user.
// The key is encrypted with AES-256-GCM before storage.
func (s *UserService) SetGeminiAPIKey(userID int, apiKey string) error {
	return setGeminiAPIKeyImpl(s.db, userID, apiKey)
}

// GetGeminiAPIKey retrieves and decrypts the Gemini API key for a user.
// Returns ErrNoGeminiAPIKey if no key is stored.
func (s *UserService) GetGeminiAPIKey(userID int) (string, error) {
	return getGeminiAPIKeyImpl(s.db, userID)
}

// DeleteGeminiAPIKey removes the Gemini API key for a user.
func (s *UserService) DeleteGeminiAPIKey(userID int) error {
	return s.db.DeleteGeminiAPIKey(userID)
}

// HasGeminiAPIKey checks if a user has a Gemini API key stored.
func (s *UserService) HasGeminiAPIKey(userID int) (bool, error) {
	return s.db.HasGeminiAPIKey(userID)
}

// GetGeminiAPIKeyStatus returns status information about the stored API key.
// Does NOT return the actual key, only metadata.
func (s *UserService) GetGeminiAPIKeyStatus(userID int) (*GeminiAPIKeyStatus, error) {
	hasKey, err := s.db.HasGeminiAPIKey(userID)
	if err != nil {
		return nil, err
	}

	if !hasKey {
		return &GeminiAPIKeyStatus{
			HasKey:    false,
			UpdatedAt: nil,
			MaskedKey: nil,
		}, nil
	}

	updatedAt, err := s.db.GetGeminiAPIKeyUpdatedAt(userID)
	if err != nil {
		return nil, err
	}

	// Get masked key (decrypt and mask)
	decryptedKey, err := getGeminiAPIKeyImpl(s.db, userID)
	if err != nil {
		return nil, err
	}

	masked := maskGeminiAPIKey(decryptedKey)

	return &GeminiAPIKeyStatus{
		HasKey:    true,
		UpdatedAt: updatedAt,
		MaskedKey: &masked,
	}, nil
}

// GeminiAPIKeyStatus represents the status of a user's Gemini API key.
type GeminiAPIKeyStatus struct {
	HasKey    bool    `json:"has_key"`
	UpdatedAt *string `json:"updated_at,omitempty"`
	MaskedKey *string `json:"masked_key,omitempty"` // e.g., "AIzaSy...xxxx"
}

// maskGeminiAPIKey returns a masked version of the API key for display.
// Shows first 7 and last 4 characters.
func maskGeminiAPIKey(apiKey string) string {
	if len(apiKey) <= 11 {
		return "****"
	}
	return apiKey[:7] + "..." + apiKey[len(apiKey)-4:]
}

// --- OpenAI API Key Management (BYOK) ---

// Validation errors for OpenAI API Key
var (
	ErrInvalidOpenAIAPIKey = errors.New("invalid OpenAI API key format")
	ErrNoOpenAIAPIKey      = errors.New("no OpenAI API key configured")
)

// SetOpenAIAPIKey validates, encrypts, and stores an OpenAI API key for a user.
// The key is encrypted with AES-256-GCM before storage.
func (s *UserService) SetOpenAIAPIKey(userID int, apiKey string) error {
	return setOpenAIAPIKeyImpl(s.db, userID, apiKey)
}

// GetOpenAIAPIKey retrieves and decrypts the OpenAI API key for a user.
// Returns ErrNoOpenAIAPIKey if no key is stored.
func (s *UserService) GetOpenAIAPIKey(userID int) (string, error) {
	return getOpenAIAPIKeyImpl(s.db, userID)
}

// DeleteOpenAIAPIKey removes the OpenAI API key for a user.
func (s *UserService) DeleteOpenAIAPIKey(userID int) error {
	return s.db.DeleteOpenAIAPIKey(userID)
}

// HasOpenAIAPIKey checks if a user has an OpenAI API key stored.
func (s *UserService) HasOpenAIAPIKey(userID int) (bool, error) {
	return s.db.HasOpenAIAPIKey(userID)
}

// GetOpenAIAPIKeyStatus returns status information about the stored API key.
// Does NOT return the actual key, only metadata.
func (s *UserService) GetOpenAIAPIKeyStatus(userID int) (*OpenAIAPIKeyStatus, error) {
	hasKey, err := s.db.HasOpenAIAPIKey(userID)
	if err != nil {
		return nil, err
	}

	if !hasKey {
		return &OpenAIAPIKeyStatus{
			HasKey:    false,
			UpdatedAt: nil,
			MaskedKey: nil,
		}, nil
	}

	updatedAt, err := s.db.GetOpenAIAPIKeyUpdatedAt(userID)
	if err != nil {
		return nil, err
	}

	// Get masked key (decrypt and mask)
	decryptedKey, err := getOpenAIAPIKeyImpl(s.db, userID)
	if err != nil {
		return nil, err
	}

	masked := maskOpenAIAPIKey(decryptedKey)

	return &OpenAIAPIKeyStatus{
		HasKey:    true,
		UpdatedAt: updatedAt,
		MaskedKey: &masked,
	}, nil
}

// OpenAIAPIKeyStatus represents the status of a user's OpenAI API key.
type OpenAIAPIKeyStatus struct {
	HasKey    bool    `json:"has_key"`
	UpdatedAt *string `json:"updated_at,omitempty"`
	MaskedKey *string `json:"masked_key,omitempty"` // e.g., "sk-proj-...xxxx"
}

// maskOpenAIAPIKey returns a masked version of the API key for display.
// Shows first 7 and last 4 characters.
func maskOpenAIAPIKey(apiKey string) string {
	if len(apiKey) <= 11 {
		return "****"
	}
	return apiKey[:7] + "..." + apiKey[len(apiKey)-4:]
}
