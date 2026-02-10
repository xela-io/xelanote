package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

const (
	// SignatureExpiry defines how long signed URLs remain valid (7 days)
	SignatureExpiry = 7 * 24 * time.Hour
)

// GenerateUploadSignature creates a cryptographically signed URL for upload access.
// Returns the signature, expiry timestamp (Unix seconds), and any error.
//
// The signature is an HMAC-SHA256 hash of: userID|filename|expires
// This ensures that tampering with any of these parameters invalidates the signature.
func GenerateUploadSignature(userID int, filename string, secret []byte) (signature string, expires int64, err error) {
	if len(secret) == 0 {
		return "", 0, fmt.Errorf("secret cannot be empty")
	}

	expires = time.Now().Add(SignatureExpiry).Unix()
	sig := signUploadURL(userID, filename, expires, secret)

	return sig, expires, nil
}

// ValidateUploadSignature verifies that a signed URL is valid and not expired.
// Returns an error if the signature is invalid, tampered with, or expired.
//
// Uses constant-time comparison to prevent timing attacks.
func ValidateUploadSignature(userID int, filename string, signature string, expires int64, secret []byte) error {
	if len(secret) == 0 {
		return fmt.Errorf("secret cannot be empty")
	}

	// Check expiry first (fast path)
	if time.Now().Unix() > expires {
		return fmt.Errorf("signature expired")
	}

	// Compute expected signature
	expectedSig := signUploadURL(userID, filename, expires, secret)

	// Constant-time comparison to prevent timing attacks
	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// signUploadURL computes HMAC-SHA256 signature for upload URL parameters.
// Input format: userID|filename|expires
// Output: Base64 URL-safe encoded signature
func signUploadURL(userID int, filename string, expires int64, secret []byte) string {
	// Create HMAC-SHA256 hasher with JWT_SECRET
	mac := hmac.New(sha256.New, secret)

	// Hash the critical parameters: userID|filename|expires
	// Using fmt.Fprintf for efficient concatenation
	fmt.Fprintf(mac, "%d|%s|%d", userID, filename, expires)

	// Compute signature and encode as Base64 URL-safe
	sigBytes := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sigBytes)
}
