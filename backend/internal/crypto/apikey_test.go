package crypto

import (
	"os"
	"strings"
	"sync"
	"testing"
)

const testAPIKeySecret = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func resetKeyForTest() {
	encryptionKey = nil
	keyErr = nil
	keyOnce = sync.Once{}
}

func TestEncryptDecryptAPIKey_RoundTrip(t *testing.T) {
	resetKeyForTest()

	if err := os.Setenv("XELANOTE_API_KEY_SECRET", testAPIKeySecret); err != nil {
		t.Fatalf("set env: %v", err)
	}
	_ = os.Unsetenv("JWT_SECRET")

	enc, err := EncryptAPIKey("my-api-key")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if enc == "" {
		t.Fatalf("expected ciphertext")
	}

	dec, err := DecryptAPIKey(enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if dec != "my-api-key" {
		t.Fatalf("unexpected plaintext: %q", dec)
	}
}

func TestEncryptAPIKey_NoSecret(t *testing.T) {
	resetKeyForTest()
	_ = os.Unsetenv("XELANOTE_API_KEY_SECRET")
	_ = os.Setenv("JWT_SECRET", testAPIKeySecret)

	if _, err := EncryptAPIKey("key"); err != ErrNoEncryptionKey {
		t.Fatalf("expected ErrNoEncryptionKey, got %v", err)
	}
}

func TestEncryptAPIKey_WeakSecret(t *testing.T) {
	resetKeyForTest()
	_ = os.Setenv("XELANOTE_API_KEY_SECRET", "short-secret")
	_ = os.Unsetenv("JWT_SECRET")

	if _, err := EncryptAPIKey("key"); err != ErrWeakEncryptionKey {
		t.Fatalf("expected ErrWeakEncryptionKey, got %v", err)
	}
}

func TestEncryptAPIKey_RejectsJWTSecretReuse(t *testing.T) {
	resetKeyForTest()
	_ = os.Setenv("XELANOTE_API_KEY_SECRET", testAPIKeySecret)
	_ = os.Setenv("JWT_SECRET", testAPIKeySecret)

	if _, err := EncryptAPIKey("key"); err != ErrKeySeparation {
		t.Fatalf("expected ErrKeySeparation, got %v", err)
	}
}

func TestDecryptAPIKey_InvalidCiphertext(t *testing.T) {
	resetKeyForTest()
	_ = os.Setenv("XELANOTE_API_KEY_SECRET", testAPIKeySecret)
	_ = os.Unsetenv("JWT_SECRET")

	if _, err := DecryptAPIKey("not-base64"); err == nil {
		t.Fatalf("expected error")
	}

	// too short after base64 decode
	short := "AA==" // 1 byte
	if _, err := DecryptAPIKey(short); err != ErrInvalidCiphertext {
		t.Fatalf("expected ErrInvalidCiphertext, got %v", err)
	}
}

func TestValidateClaudeAPIKey(t *testing.T) {
	t.Parallel()

	if err := ValidateClaudeAPIKey("sk-ant-12345678901234567890"); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if err := ValidateClaudeAPIKey("short"); err == nil {
		t.Fatalf("expected error for short key")
	}
	if err := ValidateClaudeAPIKey("sk-foo-12345678901234567890"); err == nil {
		t.Fatalf("expected error for prefix")
	}
}

func TestValidateGeminiAPIKey(t *testing.T) {
	t.Parallel()

	if err := ValidateGeminiAPIKey("AIza12345678901234567890"); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if err := ValidateGeminiAPIKey("short"); err == nil {
		t.Fatalf("expected error for short key")
	}
	if err := ValidateGeminiAPIKey("ABCD12345678901234567890"); err == nil {
		t.Fatalf("expected error for prefix")
	}
}

func TestValidateOpenAIAPIKey(t *testing.T) {
	t.Parallel()

	if err := ValidateOpenAIAPIKey("sk-test-12345678901234567890"); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if err := ValidateOpenAIAPIKey("short"); err == nil {
		t.Fatalf("expected error for short key")
	}
	if err := ValidateOpenAIAPIKey("pk-test-12345678901234567890"); err == nil {
		t.Fatalf("expected error for prefix")
	}
}

func TestMaskAPIKey(t *testing.T) {
	t.Parallel()

	if got := MaskAPIKey("short"); got != "****" {
		t.Fatalf("expected mask for short, got %q", got)
	}
	got := MaskAPIKey("0123456789ABCDEFGH")
	if !strings.HasPrefix(got, "0123456789...") || !strings.HasSuffix(got, "EFGH") {
		t.Fatalf("unexpected mask: %q", got)
	}
}
