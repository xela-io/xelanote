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
