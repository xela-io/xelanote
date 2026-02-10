package auth

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateUploadSignature(t *testing.T) {
	secret := []byte("test-secret-key-with-at-least-64-characters-for-security-testing-purposes")
	userID := 42
	filename := "test-image.png"

	sig, expires, err := GenerateUploadSignature(userID, filename, secret)
	if err != nil {
		t.Fatalf("GenerateUploadSignature failed: %v", err)
	}

	if sig == "" {
		t.Error("Expected non-empty signature")
	}

	if expires <= time.Now().Unix() {
		t.Error("Expected expiry in the future")
	}

	// Signature should be approximately 7 days from now
	expectedExpiry := time.Now().Add(SignatureExpiry).Unix()
	if expires < expectedExpiry-5 || expires > expectedExpiry+5 {
		t.Errorf("Expected expiry around %d, got %d", expectedExpiry, expires)
	}
}

func TestGenerateUploadSignature_EmptySecret(t *testing.T) {
	_, _, err := GenerateUploadSignature(42, "test.png", []byte{})
	if err == nil {
		t.Error("Expected error for empty secret")
	}
}

func TestValidateUploadSignature_Valid(t *testing.T) {
	secret := []byte("test-secret-key-with-at-least-64-characters-for-security-testing-purposes")
	userID := 42
	filename := "test-image.png"

	sig, expires, err := GenerateUploadSignature(userID, filename, secret)
	if err != nil {
		t.Fatalf("GenerateUploadSignature failed: %v", err)
	}

	// Validate the signature
	err = ValidateUploadSignature(userID, filename, sig, expires, secret)
	if err != nil {
		t.Errorf("ValidateUploadSignature failed for valid signature: %v", err)
	}
}

func TestValidateUploadSignature_Expired(t *testing.T) {
	secret := []byte("test-secret-key-with-at-least-64-characters-for-security-testing-purposes")
	userID := 42
	filename := "test-image.png"

	// Create signature with expiry in the past
	expires := time.Now().Add(-1 * time.Hour).Unix()
	sig := signUploadURL(userID, filename, expires, secret)

	err := ValidateUploadSignature(userID, filename, sig, expires, secret)
	if err == nil {
		t.Error("Expected error for expired signature")
	}

	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("Expected 'expired' error, got: %v", err)
	}
}

func TestValidateUploadSignature_Tampered(t *testing.T) {
	secret := []byte("test-secret-key-with-at-least-64-characters-for-security-testing-purposes")
	userID := 42
	filename := "test-image.png"

	sig, expires, err := GenerateUploadSignature(userID, filename, secret)
	if err != nil {
		t.Fatalf("GenerateUploadSignature failed: %v", err)
	}

	// Tamper with signature
	tamperedSig := sig + "x"

	err = ValidateUploadSignature(userID, filename, tamperedSig, expires, secret)
	if err == nil {
		t.Error("Expected error for tampered signature")
	}

	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("Expected 'invalid' error, got: %v", err)
	}
}

func TestValidateUploadSignature_WrongUser(t *testing.T) {
	secret := []byte("test-secret-key-with-at-least-64-characters-for-security-testing-purposes")
	userID := 42
	filename := "test-image.png"

	sig, expires, err := GenerateUploadSignature(userID, filename, secret)
	if err != nil {
		t.Fatalf("GenerateUploadSignature failed: %v", err)
	}

	// Try to validate with different user ID
	wrongUserID := 99
	err = ValidateUploadSignature(wrongUserID, filename, sig, expires, secret)
	if err == nil {
		t.Error("Expected error when validating with wrong user ID")
	}

	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("Expected 'invalid' error, got: %v", err)
	}
}

func TestValidateUploadSignature_WrongFilename(t *testing.T) {
	secret := []byte("test-secret-key-with-at-least-64-characters-for-security-testing-purposes")
	userID := 42
	filename := "test-image.png"

	sig, expires, err := GenerateUploadSignature(userID, filename, secret)
	if err != nil {
		t.Fatalf("GenerateUploadSignature failed: %v", err)
	}

	// Try to validate with different filename
	wrongFilename := "hacked.png"
	err = ValidateUploadSignature(userID, wrongFilename, sig, expires, secret)
	if err == nil {
		t.Error("Expected error when validating with wrong filename")
	}

	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("Expected 'invalid' error, got: %v", err)
	}
}

func TestValidateUploadSignature_EmptySecret(t *testing.T) {
	err := ValidateUploadSignature(42, "test.png", "somesig", time.Now().Unix(), []byte{})
	if err == nil {
		t.Error("Expected error for empty secret")
	}
}

func TestSignatureConsistency(t *testing.T) {
	secret := []byte("test-secret-key-with-at-least-64-characters-for-security-testing-purposes")
	userID := 42
	filename := "test-image.png"
	expires := time.Now().Add(1 * time.Hour).Unix()

	// Generate same signature twice
	sig1 := signUploadURL(userID, filename, expires, secret)
	sig2 := signUploadURL(userID, filename, expires, secret)

	if sig1 != sig2 {
		t.Error("Expected consistent signatures for same input")
	}
}

func TestSignatureDifferentForDifferentInputs(t *testing.T) {
	secret := []byte("test-secret-key-with-at-least-64-characters-for-security-testing-purposes")
	expires := time.Now().Add(1 * time.Hour).Unix()

	sig1 := signUploadURL(1, "file1.png", expires, secret)
	sig2 := signUploadURL(2, "file1.png", expires, secret)
	sig3 := signUploadURL(1, "file2.png", expires, secret)
	sig4 := signUploadURL(1, "file1.png", expires+100, secret)

	// All signatures should be different
	if sig1 == sig2 || sig1 == sig3 || sig1 == sig4 {
		t.Error("Expected different signatures for different inputs")
	}
}
