package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// Encryption validation constants
const (
	// MinAESGCMCiphertext is the minimum size for AES-GCM encrypted content.
	// AES-GCM uses 12-byte IV + 16-byte authentication tag = 28 bytes minimum.
	// We allow 16 bytes minimum for flexibility.
	MinAESGCMCiphertext = 16

	// MaxEncryptedContentBase64 is the maximum allowed length of the Base64-encoded
	// encrypted content string. 10MB plaintext + ~33% Base64 overhead ≈ 14MB.
	MaxEncryptedContentBase64 = 14 * 1024 * 1024

	// MinWrappedDEKSize is the minimum size for a wrapped DEK.
	// 256-bit DEK (32 bytes) + AES-GCM overhead (12-byte IV + 16-byte tag) = 60 bytes.
	// We allow 32 bytes minimum for flexibility.
	MinWrappedDEKSize = 32
)

// Validation errors
var (
	ErrInvalidBase64            = errors.New("invalid base64 encoding")
	ErrInvalidJSON              = errors.New("invalid JSON format")
	ErrEncryptedContentTooShort = errors.New("encrypted content too short (min 16 bytes)")
	ErrEncryptedContentTooLong  = errors.New("encrypted content too long")
	ErrWrappedDEKTooShort       = errors.New("wrapped DEK too short (min 32 bytes)")
	ErrEncryptedTitleTooShort   = errors.New("encrypted title too short")
)

// ValidateEncryptedContent validates base64-encoded encrypted content
// Minimum length: 16 bytes (AES-GCM minimum: 12-byte IV + 16-byte tag + at least some ciphertext)
func ValidateEncryptedContent(base64Content string) ([]byte, error) {
	if base64Content == "" {
		return nil, errors.New("encrypted content is required")
	}

	// Check maximum Base64 string length before decoding to prevent memory exhaustion
	if len(base64Content) > MaxEncryptedContentBase64 {
		return nil, ErrEncryptedContentTooLong
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(base64Content)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBase64, err)
	}

	// Check minimum length (IV + tag + minimal ciphertext)
	if len(decoded) < MinAESGCMCiphertext {
		return nil, ErrEncryptedContentTooShort
	}

	return decoded, nil
}

// ValidateWrappedDEK validates base64-encoded wrapped DEK
// Minimum length: 32 bytes (256-bit DEK wrapped with AES-GCM)
func ValidateWrappedDEK(base64DEK string) error {
	if base64DEK == "" {
		return errors.New("wrapped DEK is required")
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(base64DEK)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidBase64, err)
	}

	// Check minimum length
	if len(decoded) < MinWrappedDEKSize {
		return ErrWrappedDEKTooShort
	}

	return nil
}

// ValidateEncryptionMetadata validates JSON-encoded encryption metadata
func ValidateEncryptionMetadata(jsonMetadata string) error {
	if jsonMetadata == "" {
		// Metadata is optional
		return nil
	}

	// Validate JSON structure
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(jsonMetadata), &metadata); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}

	return nil
}

// ValidateEncryptedTitle validates JSON-encoded encrypted title
func ValidateEncryptedTitle(jsonTitle string) error {
	if jsonTitle == "" {
		// Encrypted title is optional (title can be plaintext)
		return nil
	}

	// Validate JSON structure
	var titleData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonTitle), &titleData); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}

	// Check for required fields in encrypted title structure
	// Expected format: {"ciphertext": "base64", "iv": "base64"}
	ciphertext, hasCiphertext := titleData["ciphertext"].(string)
	if !hasCiphertext || ciphertext == "" {
		return errors.New("encrypted title missing 'ciphertext' field")
	}

	// Validate ciphertext is valid base64
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return fmt.Errorf("encrypted title ciphertext is not valid base64: %v", err)
	}

	if len(decoded) < 16 {
		return ErrEncryptedTitleTooShort
	}

	// Validate IV field exists, is non-empty, and is valid base64
	iv, hasIV := titleData["iv"].(string)
	if !hasIV || iv == "" {
		return errors.New("encrypted title missing 'iv' field")
	}
	if _, err := base64.StdEncoding.DecodeString(iv); err != nil {
		return fmt.Errorf("encrypted title iv is not valid base64: %v", err)
	}

	return nil
}

// ValidateEncryptedNoteRequest validates all encryption-related fields in a note request
func ValidateEncryptedNoteRequest(encryptedContent, wrappedDEK, encryptionMetadata string, encryptedTitle *string) error {
	// Validate encrypted content
	if _, err := ValidateEncryptedContent(encryptedContent); err != nil {
		return fmt.Errorf("encrypted_content validation failed: %w", err)
	}

	// Validate wrapped DEK
	if err := ValidateWrappedDEK(wrappedDEK); err != nil {
		return fmt.Errorf("wrapped_dek validation failed: %w", err)
	}

	// Validate encryption metadata (optional)
	if err := ValidateEncryptionMetadata(encryptionMetadata); err != nil {
		return fmt.Errorf("encryption_metadata validation failed: %w", err)
	}

	// Validate encrypted title (optional)
	if encryptedTitle != nil && *encryptedTitle != "" {
		if err := ValidateEncryptedTitle(*encryptedTitle); err != nil {
			return fmt.Errorf("encrypted_title validation failed: %w", err)
		}
	}

	return nil
}
