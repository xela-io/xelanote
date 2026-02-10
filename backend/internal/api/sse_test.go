package api

import (
	"encoding/json"
	"testing"
)

func TestSSEPayloadEscaping(t *testing.T) {
	tests := []struct{ name, input string }{
		{"newlines", "line1\nline2\nline3"},
		{"quotes", `he said "hello"`},
		{"backslash", `path\to\file`},
		{"json_injection", "\"}\n\nevent: token\ndata: injected"},
		{"unicode", "emoji \U0001F600 and umlauts äöü"},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test token event (json.Marshal on string)
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("json.Marshal(string) failed: %v", err)
			}
			var roundtrip string
			if err := json.Unmarshal(data, &roundtrip); err != nil {
				t.Fatalf("json.Unmarshal(string) failed: %v", err)
			}
			if roundtrip != tt.input {
				t.Errorf("string roundtrip mismatch: got %q, want %q", roundtrip, tt.input)
			}

			// Test error/cached event (json.Marshal on map)
			payload, err := json.Marshal(map[string]string{"value": tt.input})
			if err != nil {
				t.Fatalf("json.Marshal(map) failed: %v", err)
			}
			var result map[string]string
			if err := json.Unmarshal(payload, &result); err != nil {
				t.Fatalf("json.Unmarshal(map) failed: %v", err)
			}
			if result["value"] != tt.input {
				t.Errorf("map roundtrip mismatch: got %q, want %q", result["value"], tt.input)
			}

			// Verify no SSE event injection (no unescaped newlines in output)
			if containsNewline(string(data)) {
				t.Errorf("marshaled string contains unescaped newline")
			}
			if containsNewline(string(payload)) {
				t.Errorf("marshaled map contains unescaped newline")
			}
		})
	}
}

func containsNewline(s string) bool {
	for _, c := range s {
		if c == '\n' {
			return true
		}
	}
	return false
}
