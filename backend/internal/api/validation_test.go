package api

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateWrappedDEK(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "valid wrapped DEK",
			input:   base64.StdEncoding.EncodeToString(make([]byte, 72)),
			wantErr: "",
		},
		{
			name:    "empty wrapped DEK",
			input:   "",
			wantErr: "wrapped DEK is required",
		},
		{
			name:    "invalid base64",
			input:   "not-base64!!!",
			wantErr: "invalid base64 encoding",
		},
		{
			name:    "wrapped DEK too short",
			input:   base64.StdEncoding.EncodeToString(make([]byte, 8)),
			wantErr: "wrapped DEK too short",
		},
		{
			name:    "wrapped DEK too long by decoded size",
			input:   base64.StdEncoding.EncodeToString(make([]byte, MaxWrappedDEKSize+1)),
			wantErr: "wrapped DEK too long",
		},
		{
			name:    "wrapped DEK too long by base64 size guard",
			input:   strings.Repeat("A", MaxWrappedDEKBase64+1),
			wantErr: "wrapped DEK too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWrappedDEK(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestValidateEncryptedTitle(t *testing.T) {
	// Helper: build a valid encrypted title JSON
	validCiphertext := base64.StdEncoding.EncodeToString(make([]byte, 32))
	validWrappedDEK := base64.StdEncoding.EncodeToString(make([]byte, 32))

	makeTitle := func(ct string, metadata interface{}, includeCT, includeMetadata bool) string {
		m := map[string]interface{}{}
		if includeCT {
			m["ciphertext"] = ct
		}
		if includeMetadata {
			m["metadata"] = metadata
		}
		b, _ := json.Marshal(m)
		return string(b)
	}

	validMetadata := map[string]interface{}{
		"version":      2,
		"algorithm":    "XChaCha20-Poly1305",
		"kdf":          "Argon2id",
		"kdf_strength": "interactive",
		"nonce_bytes":  24,
		"wrapped_dek":  validWrappedDEK,
	}

	tests := []struct {
		name    string
		input   string
		wantErr string // empty = no error expected
	}{
		{
			name:    "empty string is valid (optional)",
			input:   "",
			wantErr: "",
		},
		{
			name:    "valid encrypted title",
			input:   makeTitle(validCiphertext, validMetadata, true, true),
			wantErr: "",
		},
		{
			name:    "invalid JSON",
			input:   "not-json",
			wantErr: "invalid JSON format",
		},
		{
			name:    "missing ciphertext field",
			input:   makeTitle("", validMetadata, false, true),
			wantErr: "missing 'ciphertext' field",
		},
		{
			name:    "empty ciphertext",
			input:   makeTitle("", validMetadata, true, true),
			wantErr: "missing 'ciphertext' field",
		},
		{
			name:    "invalid ciphertext base64",
			input:   makeTitle("!!!not-base64!!!", validMetadata, true, true),
			wantErr: "ciphertext is not valid base64",
		},
		{
			name:    "ciphertext too short",
			input:   makeTitle(base64.StdEncoding.EncodeToString(make([]byte, 8)), validMetadata, true, true),
			wantErr: "encrypted title too short",
		},
		{
			name:    "missing metadata field",
			input:   makeTitle(validCiphertext, nil, true, false),
			wantErr: "missing 'metadata' field",
		},
		{
			name:    "invalid metadata field",
			input:   makeTitle(validCiphertext, "not-an-object", true, true),
			wantErr: "'metadata' field is invalid",
		},
		{
			name: "unsupported metadata version",
			input: makeTitle(validCiphertext, map[string]interface{}{
				"version":      99,
				"algorithm":    "XChaCha20-Poly1305",
				"kdf":          "Argon2id",
				"kdf_strength": "interactive",
				"nonce_bytes":  24,
				"wrapped_dek":  validWrappedDEK,
			}, true, true),
			wantErr: "unsupported 'version'",
		},
		{
			name: "invalid metadata algorithm",
			input: makeTitle(validCiphertext, map[string]interface{}{
				"version":      2,
				"algorithm":    "AES-256-GCM",
				"kdf":          "Argon2id",
				"kdf_strength": "interactive",
				"nonce_bytes":  24,
				"wrapped_dek":  validWrappedDEK,
			}, true, true),
			wantErr: "invalid 'algorithm'",
		},
		{
			name: "invalid metadata kdf",
			input: makeTitle(validCiphertext, map[string]interface{}{
				"version":      2,
				"algorithm":    "XChaCha20-Poly1305",
				"kdf":          "PBKDF2",
				"kdf_strength": "interactive",
				"nonce_bytes":  24,
				"wrapped_dek":  validWrappedDEK,
			}, true, true),
			wantErr: "invalid 'kdf'",
		},
		{
			name: "invalid metadata kdf_strength",
			input: makeTitle(validCiphertext, map[string]interface{}{
				"version":      2,
				"algorithm":    "XChaCha20-Poly1305",
				"kdf":          "Argon2id",
				"kdf_strength": "moderate",
				"nonce_bytes":  24,
				"wrapped_dek":  validWrappedDEK,
			}, true, true),
			wantErr: "invalid 'kdf_strength'",
		},
		{
			name: "invalid metadata nonce_bytes",
			input: makeTitle(validCiphertext, map[string]interface{}{
				"version":      2,
				"algorithm":    "XChaCha20-Poly1305",
				"kdf":          "Argon2id",
				"kdf_strength": "interactive",
				"nonce_bytes":  12,
				"wrapped_dek":  validWrappedDEK,
			}, true, true),
			wantErr: "invalid 'nonce_bytes'",
		},
		{
			name: "missing metadata wrapped_dek",
			input: makeTitle(validCiphertext, map[string]interface{}{
				"version":      2,
				"algorithm":    "XChaCha20-Poly1305",
				"kdf":          "Argon2id",
				"kdf_strength": "interactive",
				"nonce_bytes":  24,
			}, true, true),
			wantErr: "missing 'wrapped_dek'",
		},
		{
			name: "invalid metadata wrapped_dek base64",
			input: makeTitle(validCiphertext, map[string]interface{}{
				"version":      2,
				"algorithm":    "XChaCha20-Poly1305",
				"kdf":          "Argon2id",
				"kdf_strength": "interactive",
				"nonce_bytes":  24,
				"wrapped_dek":  "!!!not-base64!!!",
			}, true, true),
			wantErr: "wrapped_dek invalid",
		},
		{
			name: "metadata wrapped_dek too short",
			input: makeTitle(validCiphertext, map[string]interface{}{
				"version":      2,
				"algorithm":    "XChaCha20-Poly1305",
				"kdf":          "Argon2id",
				"kdf_strength": "interactive",
				"nonce_bytes":  24,
				"wrapped_dek":  base64.StdEncoding.EncodeToString(make([]byte, 8)),
			}, true, true),
			wantErr: "wrapped DEK too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEncryptedTitle(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}
