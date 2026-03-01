//go:build fts5

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_DevelopmentMode_EchoesOrigin(t *testing.T) {
	ts := newTestServer(t)
	// allowedOrigins defaults to empty (dev mode) in newTestServer

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://evil.example.com" {
		t.Errorf("dev mode should echo origin, got %q", got)
	}
}

func TestCORS_ProductionMode_ValidOrigin(t *testing.T) {
	ts := newTestServer(t)
	ts.allowedOrigins = []string{"https://xelanote.com"}

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", "https://xelanote.com")
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://xelanote.com" {
		t.Errorf("expected allowed origin in header, got %q", got)
	}
}

func TestCORS_ProductionMode_InvalidOrigin_NoAllowOrigin(t *testing.T) {
	ts := newTestServer(t)
	ts.allowedOrigins = []string{"https://xelanote.com"}

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no Access-Control-Allow-Origin for invalid origin, got %q", got)
	}
}

func TestCORS_ProductionMode_Preflight_InvalidOrigin_403(t *testing.T) {
	ts := newTestServer(t)
	ts.allowedOrigins = []string{"https://xelanote.com"}

	req := httptest.NewRequest(http.MethodOptions, "/api/health", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for preflight from invalid origin, got %d", rec.Code)
	}
}

func TestCORS_ProductionMode_Preflight_ValidOrigin_204(t *testing.T) {
	ts := newTestServer(t)
	ts.allowedOrigins = []string{"https://xelanote.com"}

	req := httptest.NewRequest(http.MethodOptions, "/api/health", nil)
	req.Header.Set("Origin", "https://xelanote.com")
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for valid preflight, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://xelanote.com" {
		t.Errorf("expected Allow-Origin header on preflight, got %q", got)
	}
}

func TestCORS_PreflightAllowsPATCH(t *testing.T) {
	ts := newTestServer(t)
	// Dev mode: echo origin

	req := httptest.NewRequest(http.MethodOptions, "/api/users/preferences", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "PATCH")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, X-CSRF-Token")
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for PATCH preflight, got %d", rec.Code)
	}

	methods := rec.Header().Get("Access-Control-Allow-Methods")
	if methods == "" {
		t.Fatal("missing Access-Control-Allow-Methods header")
	}

	// PATCH must be in the allowed methods
	found := false
	for _, m := range []string{"PATCH"} {
		if containsMethod(methods, m) {
			found = true
		}
	}
	if !found {
		t.Errorf("Access-Control-Allow-Methods does not include PATCH: %q", methods)
	}
}

func containsMethod(header, method string) bool {
	for _, m := range splitAndTrim(header) {
		if m == method {
			return true
		}
	}
	return false
}

func splitAndTrim(s string) []string {
	parts := make([]string, 0)
	for _, p := range splitByComma(s) {
		trimmed := trimSpaces(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func splitByComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpaces(s string) string {
	i, j := 0, len(s)
	for i < j && s[i] == ' ' {
		i++
	}
	for j > i && s[j-1] == ' ' {
		j--
	}
	return s[i:j]
}

func TestCORS_NoOriginHeader(t *testing.T) {
	ts := newTestServer(t)
	ts.allowedOrigins = []string{"https://xelanote.com"}

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	// No Origin header set
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no Access-Control-Allow-Origin without Origin header, got %q", got)
	}
}
