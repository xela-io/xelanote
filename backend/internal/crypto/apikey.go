// Package crypto provides cryptographic utilities for xelanote.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

var (
	// ErrInvalidCiphertext is returned when the ciphertext is too short or malformed
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	// ErrNoEncryptionKey is returned when the encryption key is not configured
	ErrNoEncryptionKey = errors.New("encryption key not configured")
)

const (
	// NonceSize is the size of the GCM nonce (12 bytes)
	NonceSize = 12
	// KeySize is the size of the AES-256 key (32 bytes)
	KeySize = 32
)

var (
	// encryptionKey is derived from XELANOTE_API_KEY_SECRET environment variable
	encryptionKey []byte
	keyOnce       sync.Once
	keyErr        error
)

// initKey derives the encryption key from the environment variable.
// Uses SHA-256 to derive a 32-byte key from any length secret.
func initKey() {
	keyOnce.Do(func() {
		secret := os.Getenv("XELANOTE_API_KEY_SECRET")
		if secret == "" {
			// Fallback: derive from JWT_SECRET if API_KEY_SECRET not set
			secret = os.Getenv("JWT_SECRET")
			if secret == "" {
				keyErr = ErrNoEncryptionKey
				return
			}
		}

		// Derive 32-byte key using SHA-256
		hash := sha256.Sum256([]byte(secret + ":api-key-encryption"))
		encryptionKey = hash[:]
	})
}

// EncryptAPIKey encrypts an API key using AES-256-GCM.
// Returns base64-encoded ciphertext: nonce (12 bytes) || ciphertext || tag (16 bytes)
func EncryptAPIKey(apiKey string) (string, error) {
	initKey()
	if keyErr != nil {
		return "", keyErr
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt: ciphertext includes the authentication tag
	ciphertext := gcm.Seal(nonce, nonce, []byte(apiKey), nil)

	// Return base64-encoded result
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAPIKey decrypts a base64-encoded AES-256-GCM ciphertext.
// Returns the original API key.
func DecryptAPIKey(encryptedKey string) (string, error) {
	initKey()
	if keyErr != nil {
		return "", keyErr
	}

	// Decode base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Validate minimum length: nonce (12) + at least 1 byte + tag (16) = 29
	if len(ciphertext) < NonceSize+1+16 {
		return "", ErrInvalidCiphertext
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce from the beginning
	nonce := ciphertext[:NonceSize]
	ciphertext = ciphertext[NonceSize:]

	// Decrypt and verify authentication tag
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintext), nil
}

// ValidateClaudeAPIKey performs basic validation of a Claude API key format.
// Claude API keys start with "sk-ant-" prefix.
func ValidateClaudeAPIKey(apiKey string) error {
	if len(apiKey) < 20 {
		return errors.New("API key too short")
	}

	// Claude API keys should start with "sk-ant-"
	if len(apiKey) >= 7 && apiKey[:7] != "sk-ant-" {
		return errors.New("invalid Claude API key format (should start with sk-ant-)")
	}

	return nil
}

// ValidateGeminiAPIKey performs basic validation of a Gemini API key format.
// Gemini API keys start with "AIza" prefix.
func ValidateGeminiAPIKey(apiKey string) error {
	if len(apiKey) < 20 {
		return errors.New("API key too short")
	}

	// Gemini API keys should start with "AIza"
	if len(apiKey) >= 4 && apiKey[:4] != "AIza" {
		return errors.New("invalid Gemini API key format (should start with AIza)")
	}

	return nil
}

// ValidateOpenAIAPIKey performs basic validation of an OpenAI API key format.
// OpenAI API keys generally start with "sk-".
func ValidateOpenAIAPIKey(apiKey string) error {
	if len(apiKey) < 20 {
		return errors.New("API key too short")
	}

	// OpenAI API keys should start with "sk-"
	if len(apiKey) >= 3 && apiKey[:3] != "sk-" {
		return errors.New("invalid OpenAI API key format (should start with sk-)")
	}

	return nil
}

// MaskAPIKey returns a masked version of the API key for display.
// Shows first 10 and last 4 characters.
func MaskAPIKey(apiKey string) string {
	if len(apiKey) <= 14 {
		return "****"
	}
	return apiKey[:10] + "..." + apiKey[len(apiKey)-4:]
}
