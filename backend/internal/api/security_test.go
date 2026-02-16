package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xela-io/xelanote/internal/service"
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
			"script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval'",
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

// --- Helpers for SEC-002/SEC-006 integration tests ---

// newSecurityTestServerWithCaptcha creates a test server with configurable CAPTCHA.
// When captchaEnabled=true, requests without a valid CAPTCHA token are rejected.
// Always sets a TurnstileService (even when disabled) to avoid nil pointer panics.
func newSecurityTestServerWithCaptcha(t *testing.T, captchaEnabled bool) *testServer {
	t.Helper()
	ts := newTestServer(t)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if captchaEnabled {
		// Enabled: Verify("", ip) returns "captcha token required"
		ts.turnstileService = service.NewTurnstileService("test-secret", "test-site", logger)
	} else {
		// Disabled: Verify() always returns nil (no CAPTCHA check)
		ts.turnstileService = service.NewTurnstileService("", "", logger)
	}

	return ts
}

// loginAndExtractCookies performs a login via the full router and returns the response cookies.
func loginAndExtractCookies(t *testing.T, ts *testServer) []*http.Cookie {
	t.Helper()

	body, _ := json.Marshal(map[string]string{
		"username_or_email": "testuser",
		"password":          "password123",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", rec.Code, rec.Body.String())
	}

	return rec.Result().Cookies()
}

func cookieValue(cookies []*http.Cookie, name string) string {
	// Return the last non-empty value — clearLegacyCSRFCookie may set an empty
	// cookie with the same name before the real one is written.
	var value string
	for _, c := range cookies {
		if c.Name == name && c.Value != "" {
			value = c.Value
		}
	}
	return value
}

// --- SEC-002: CAPTCHA bypass rejection tests ---

func TestSEC002_LoginDesktopWithoutCaptcha_Rejected(t *testing.T) {
	ts := newSecurityTestServerWithCaptcha(t, true)
	ts.createUser(t, "testuser", "test@example.com", "password123")

	body, _ := json.Marshal(map[string]string{
		"username_or_email": "testuser",
		"password":          "password123",
		// captcha_token intentionally omitted
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Type", "desktop")
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for desktop login without CAPTCHA, got %d", rec.Code)
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "captcha token required" {
		t.Errorf("expected 'captcha token required', got %q", resp["error"])
	}
}

func TestSEC002_RegisterDesktopWithoutCaptcha_Rejected(t *testing.T) {
	ts := newSecurityTestServerWithCaptcha(t, true)

	body, _ := json.Marshal(map[string]string{
		"username": "newuser",
		"email":    "new@example.com",
		"password": "password123",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Type", "desktop")
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for desktop register without CAPTCHA, got %d", rec.Code)
	}
}

func TestSEC002_LoginCaptchaDisabled_Succeeds(t *testing.T) {
	ts := newSecurityTestServerWithCaptcha(t, false)
	ts.createUser(t, "testuser", "test@example.com", "password123")

	body, _ := json.Marshal(map[string]string{
		"username_or_email": "testuser",
		"password":          "password123",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 when CAPTCHA disabled, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

// --- SEC-001: Web clients must NOT receive tokens in response body ---

func TestSEC001_LoginWebClient_NoTokensInBody(t *testing.T) {
	ts := newSecurityTestServerWithCaptcha(t, false)
	ts.createUser(t, "testuser", "test@example.com", "password123")

	body, _ := json.Marshal(map[string]string{
		"username_or_email": "testuser",
		"password":          "password123",
	})

	// Web client: no X-Client-Type header, non-localhost address
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp AuthResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)

	// Tokens must NOT be in body
	if resp.AccessToken != "" {
		t.Error("web client should NOT receive access_token in body")
	}
	if resp.RefreshToken != "" {
		t.Error("web client should NOT receive refresh_token in body")
	}

	// User info must still be present
	if resp.User.ID == 0 {
		t.Fatal("response must contain user info")
	}

	// Cookies must be set
	cookies := rec.Result().Cookies()
	if cookieValue(cookies, AccessTokenCookie) == "" {
		t.Error("access_token cookie must be set")
	}
	if cookieValue(cookies, RefreshTokenCookie) == "" {
		t.Error("refresh_token cookie must be set")
	}
}

func TestSEC001_LoginDesktopClient_TokensInBody(t *testing.T) {
	ts := newSecurityTestServerWithCaptcha(t, false)
	ts.createUser(t, "testuser", "test@example.com", "password123")

	body, _ := json.Marshal(map[string]string{
		"username_or_email": "testuser",
		"password":          "password123",
	})

	// Desktop client: X-Client-Type + localhost
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Type", "desktop")
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp AuthResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)

	// Desktop clients MUST receive tokens in body
	if resp.AccessToken == "" {
		t.Error("desktop client should receive access_token in body")
	}
	if resp.RefreshToken == "" {
		t.Error("desktop client should receive refresh_token in body")
	}
}

func TestSEC001_RefreshWebClient_NoTokensInBody(t *testing.T) {
	ts := newSecurityTestServerWithCaptcha(t, false)
	ts.createUser(t, "testuser", "test@example.com", "password123")

	cookies := loginAndExtractCookies(t, ts)
	refreshToken := cookieValue(cookies, RefreshTokenCookie)
	csrfToken := cookieValue(cookies, "csrf_token")

	if refreshToken == "" || csrfToken == "" {
		t.Fatalf("missing cookies: refresh=%q csrf=%q", refreshToken, csrfToken)
	}

	// Web client refresh (cookies, no desktop header)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: RefreshTokenCookie, Value: refreshToken})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var resp TokenResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)

	// Web client: no tokens in body
	if resp.AccessToken != "" {
		t.Error("web refresh should NOT return access_token in body")
	}
	if resp.RefreshToken != "" {
		t.Error("web refresh should NOT return refresh_token in body")
	}

	// But cookies must be refreshed
	newCookies := rec.Result().Cookies()
	if cookieValue(newCookies, AccessTokenCookie) == "" {
		t.Error("refresh must set new access_token cookie")
	}
	if cookieValue(newCookies, RefreshTokenCookie) == "" {
		t.Error("refresh must set new refresh_token cookie")
	}
}

// --- SEC-006: CSRF on /auth/refresh and /auth/logout ---

func TestSEC006_RefreshWithCookieNoCSRF_Rejected(t *testing.T) {
	ts := newSecurityTestServerWithCaptcha(t, false)
	ts.createUser(t, "testuser", "test@example.com", "password123")

	cookies := loginAndExtractCookies(t, ts)
	refreshToken := cookieValue(cookies, RefreshTokenCookie)
	if refreshToken == "" {
		t.Fatal("no refresh_token cookie from login")
	}

	// Refresh with cookie but WITHOUT CSRF header → must be rejected
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: RefreshTokenCookie, Value: refreshToken})
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for refresh without CSRF, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestSEC006_RefreshWithCookieAndCSRF_Succeeds(t *testing.T) {
	ts := newSecurityTestServerWithCaptcha(t, false)
	ts.createUser(t, "testuser", "test@example.com", "password123")

	cookies := loginAndExtractCookies(t, ts)
	refreshToken := cookieValue(cookies, RefreshTokenCookie)
	csrfToken := cookieValue(cookies, "csrf_token")

	if refreshToken == "" || csrfToken == "" {
		t.Fatalf("missing cookies: refresh=%q csrf=%q", refreshToken, csrfToken)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: RefreshTokenCookie, Value: refreshToken})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for refresh with CSRF, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestSEC006_RefreshBearerOnly_SkipsCSRF(t *testing.T) {
	ts := newSecurityTestServerWithCaptcha(t, false)
	ts.createUser(t, "testuser", "test@example.com", "password123")

	// Login via service to get raw tokens (no cookies)
	accessToken, refreshToken, _, _, err := ts.authService.Login(context.Background(), "testuser", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"refresh_token": refreshToken,
	})

	// Send with Bearer token only, no cookies, no CSRF → should work
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for Bearer-only refresh, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestSEC006_LogoutWithCookieNoCSRF_Rejected(t *testing.T) {
	ts := newSecurityTestServerWithCaptcha(t, false)
	ts.createUser(t, "testuser", "test@example.com", "password123")

	cookies := loginAndExtractCookies(t, ts)
	refreshToken := cookieValue(cookies, RefreshTokenCookie)
	if refreshToken == "" {
		t.Fatal("no refresh_token cookie from login")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: RefreshTokenCookie, Value: refreshToken})
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for logout without CSRF, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestSEC006_LogoutWithCookieAndCSRF_Succeeds(t *testing.T) {
	ts := newSecurityTestServerWithCaptcha(t, false)
	ts.createUser(t, "testuser", "test@example.com", "password123")

	cookies := loginAndExtractCookies(t, ts)
	refreshToken := cookieValue(cookies, RefreshTokenCookie)
	csrfToken := cookieValue(cookies, "csrf_token")

	if refreshToken == "" || csrfToken == "" {
		t.Fatalf("missing cookies: refresh=%q csrf=%q", refreshToken, csrfToken)
	}

	body, _ := json.Marshal(map[string]string{
		"refresh_token": refreshToken,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: RefreshTokenCookie, Value: refreshToken})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for logout with CSRF, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}
