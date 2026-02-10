package api

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateEncryptedTitle(t *testing.T) {
	// Helper: build a valid encrypted title JSON
	validCiphertext := base64.StdEncoding.EncodeToString(make([]byte, 32))
	validIV := base64.StdEncoding.EncodeToString(make([]byte, 24))

	makeTitle := func(ct, iv string, includeCT, includeIV bool) string {
		m := map[string]interface{}{}
		if includeCT {
			m["ciphertext"] = ct
		}
		if includeIV {
			m["iv"] = iv
		}
		b, _ := json.Marshal(m)
		return string(b)
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
			input:   makeTitle(validCiphertext, validIV, true, true),
			wantErr: "",
		},
		{
			name:    "invalid JSON",
			input:   "not-json",
			wantErr: "invalid JSON format",
		},
		{
			name:    "missing ciphertext field",
			input:   makeTitle("", validIV, false, true),
			wantErr: "missing 'ciphertext' field",
		},
		{
			name:    "empty ciphertext",
			input:   makeTitle("", validIV, true, true),
			wantErr: "missing 'ciphertext' field",
		},
		{
			name:    "invalid ciphertext base64",
			input:   makeTitle("!!!not-base64!!!", validIV, true, true),
			wantErr: "ciphertext is not valid base64",
		},
		{
			name:    "ciphertext too short",
			input:   makeTitle(base64.StdEncoding.EncodeToString(make([]byte, 8)), validIV, true, true),
			wantErr: "encrypted title too short",
		},
		{
			name:    "missing iv field",
			input:   makeTitle(validCiphertext, "", true, false),
			wantErr: "missing 'iv' field",
		},
		{
			name:    "empty iv",
			input:   makeTitle(validCiphertext, "", true, true),
			wantErr: "missing 'iv' field",
		},
		{
			name:    "invalid iv base64",
			input:   makeTitle(validCiphertext, "!!!not-base64!!!", true, true),
			wantErr: "iv is not valid base64",
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
