package api

import "testing"

func TestParseBearerToken(t *testing.T) {
	t.Run("should parse token when header uses canonical bearer scheme", func(t *testing.T) {
		token, ok := parseBearerToken("Bearer abc.def.ghi")
		if !ok {
			t.Fatalf("expected token to parse successfully")
		}
		if token != "abc.def.ghi" {
			t.Fatalf("expected parsed token to match, got %q", token)
		}
	})

	t.Run("should parse token when scheme case differs", func(t *testing.T) {
		token, ok := parseBearerToken("bEaReR token123")
		if !ok {
			t.Fatalf("expected case-insensitive bearer scheme to parse")
		}
		if token != "token123" {
			t.Fatalf("expected parsed token to match, got %q", token)
		}
	})

	t.Run("should parse token when header has extra whitespace", func(t *testing.T) {
		token, ok := parseBearerToken("   Bearer    spaced-token   ")
		if !ok {
			t.Fatalf("expected whitespace-tolerant header parsing")
		}
		if token != "spaced-token" {
			t.Fatalf("expected parsed token to match, got %q", token)
		}
	})

	t.Run("should reject header when scheme is missing", func(t *testing.T) {
		if _, ok := parseBearerToken("just-a-token"); ok {
			t.Fatalf("expected missing scheme to be rejected")
		}
	})

	t.Run("should reject header when scheme is not bearer", func(t *testing.T) {
		if _, ok := parseBearerToken("Basic abc123"); ok {
			t.Fatalf("expected non-bearer scheme to be rejected")
		}
	})

	t.Run("should reject header when token is missing", func(t *testing.T) {
		if _, ok := parseBearerToken("Bearer"); ok {
			t.Fatalf("expected empty bearer token to be rejected")
		}
	})
}

func TestHasBearerAuthorizationHeader(t *testing.T) {
	t.Run("should return true when bearer header is valid", func(t *testing.T) {
		if !hasBearerAuthorizationHeader("Bearer valid-token") {
			t.Fatalf("expected valid bearer header")
		}
	})

	t.Run("should return false when header is invalid", func(t *testing.T) {
		if hasBearerAuthorizationHeader("Token invalid") {
			t.Fatalf("expected invalid header to be rejected")
		}
	})
}
