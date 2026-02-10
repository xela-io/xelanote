package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	// Create a simple handler that does nothing
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap it with the security headers middleware
	wrappedHandler := securityHeadersMiddleware(handler)

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	// Execute the request
	wrappedHandler.ServeHTTP(rec, req)

	// Test Content-Security-Policy
	t.Run("Content-Security-Policy", func(t *testing.T) {
		csp := rec.Header().Get("Content-Security-Policy")
		if csp == "" {
			t.Error("Content-Security-Policy header is missing")
		}

		expectedDirectives := []string{
			"default-src 'self'",
			"script-src 'self' 'wasm-unsafe-eval'",
			"style-src 'self' 'unsafe-inline'",
			"img-src 'self' data: blob:",
			"connect-src 'self' ws: wss:",
			"worker-src 'self' blob:",
			"frame-ancestors 'none'",
			"base-uri 'self'",
			"form-action 'self'",
		}

		for _, directive := range expectedDirectives {
			if !strings.Contains(csp, directive) {
				t.Errorf("CSP missing directive: %s", directive)
			}
		}
	})

	// Test X-Frame-Options
	t.Run("X-Frame-Options", func(t *testing.T) {
		xfo := rec.Header().Get("X-Frame-Options")
		if xfo != "DENY" {
			t.Errorf("X-Frame-Options: expected DENY, got %s", xfo)
		}
	})

	// Test X-Content-Type-Options
	t.Run("X-Content-Type-Options", func(t *testing.T) {
		xcto := rec.Header().Get("X-Content-Type-Options")
		if xcto != "nosniff" {
			t.Errorf("X-Content-Type-Options: expected nosniff, got %s", xcto)
		}
	})

	// Test Referrer-Policy
	t.Run("Referrer-Policy", func(t *testing.T) {
		rp := rec.Header().Get("Referrer-Policy")
		if rp != "strict-origin-when-cross-origin" {
			t.Errorf("Referrer-Policy: expected strict-origin-when-cross-origin, got %s", rp)
		}
	})

	// Test Permissions-Policy
	t.Run("Permissions-Policy", func(t *testing.T) {
		pp := rec.Header().Get("Permissions-Policy")
		if pp != "geolocation=(), microphone=(), camera=()" {
			t.Errorf("Permissions-Policy: expected geolocation=(), microphone=(), camera=(), got %s", pp)
		}
	})

	// Test Strict-Transport-Security (HSTS)
	t.Run("Strict-Transport-Security", func(t *testing.T) {
		hsts := rec.Header().Get("Strict-Transport-Security")
		if hsts == "" {
			t.Error("Strict-Transport-Security header is missing")
		}
		expectedHSTS := "max-age=31536000; includeSubDomains; preload"
		if hsts != expectedHSTS {
			t.Errorf("Strict-Transport-Security: expected %s, got %s", expectedHSTS, hsts)
		}
	})
}

func TestSecurityHeadersMiddlewarePassesRequest(t *testing.T) {
	// Verify that the middleware correctly passes the request to the next handler
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})

	wrappedHandler := securityHeadersMiddleware(handler)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if !called {
		t.Error("Next handler was not called")
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}
