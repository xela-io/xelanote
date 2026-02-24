package service

import (
	"errors"

	"github.com/xela-io/xelanote/internal/crypto"
	"github.com/xela-io/xelanote/internal/db"
)

// setClaudeAPIKeyImpl validates, encrypts, and stores a Claude API key.
// This is separated to avoid import cycle issues with the crypto package.
func setClaudeAPIKeyImpl(database *db.DB, userID int, apiKey string) error {
	// Validate API key format
	if err := crypto.ValidateClaudeAPIKey(apiKey); err != nil {
		return ErrInvalidClaudeAPIKey
	}

	// Encrypt the API key
	encryptedKey, err := crypto.EncryptAPIKey(apiKey)
	if err != nil {
		return err
	}

	// Store encrypted key
	return database.SetClaudeAPIKey(userID, encryptedKey)
}

// getClaudeAPIKeyImpl retrieves and decrypts the Claude API key.
// This is separated to avoid import cycle issues with the crypto package.
func getClaudeAPIKeyImpl(database *db.DB, userID int) (string, error) {
	// Get encrypted key from database
	encryptedKey, err := database.GetClaudeAPIKey(userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return "", ErrNoClaudeAPIKey
		}
		return "", err
	}

	// Decrypt the API key
	apiKey, err := crypto.DecryptAPIKey(encryptedKey)
	if err != nil {
		return "", err
	}

	return apiKey, nil
}

// setGeminiAPIKeyImpl validates, encrypts, and stores a Gemini API key.
// This is separated to avoid import cycle issues with the crypto package.
func setGeminiAPIKeyImpl(database *db.DB, userID int, apiKey string) error {
	// Validate API key format
	if err := crypto.ValidateGeminiAPIKey(apiKey); err != nil {
		return ErrInvalidGeminiAPIKey
	}

	// Encrypt the API key
	encryptedKey, err := crypto.EncryptAPIKey(apiKey)
	if err != nil {
		return err
	}

	// Store encrypted key
	return database.SetGeminiAPIKey(userID, encryptedKey)
}

// getGeminiAPIKeyImpl retrieves and decrypts the Gemini API key.
// This is separated to avoid import cycle issues with the crypto package.
func getGeminiAPIKeyImpl(database *db.DB, userID int) (string, error) {
	// Get encrypted key from database
	encryptedKey, err := database.GetGeminiAPIKey(userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return "", ErrNoGeminiAPIKey
		}
		return "", err
	}

	// Decrypt the API key
	apiKey, err := crypto.DecryptAPIKey(encryptedKey)
	if err != nil {
		return "", err
	}

	return apiKey, nil
}

// setOpenAIAPIKeyImpl validates, encrypts, and stores an OpenAI API key.
// This is separated to avoid import cycle issues with the crypto package.
func setOpenAIAPIKeyImpl(database *db.DB, userID int, apiKey string) error {
	// Validate API key format
	if err := crypto.ValidateOpenAIAPIKey(apiKey); err != nil {
		return ErrInvalidOpenAIAPIKey
	}

	// Encrypt the API key
	encryptedKey, err := crypto.EncryptAPIKey(apiKey)
	if err != nil {
		return err
	}

	// Store encrypted key
	return database.SetOpenAIAPIKey(userID, encryptedKey)
}

// getOpenAIAPIKeyImpl retrieves and decrypts the OpenAI API key.
// This is separated to avoid import cycle issues with the crypto package.
func getOpenAIAPIKeyImpl(database *db.DB, userID int) (string, error) {
	// Get encrypted key from database
	encryptedKey, err := database.GetOpenAIAPIKey(userID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return "", ErrNoOpenAIAPIKey
		}
		return "", err
	}

	// Decrypt the API key
	apiKey, err := crypto.DecryptAPIKey(encryptedKey)
	if err != nil {
		return "", err
	}

	return apiKey, nil
}
