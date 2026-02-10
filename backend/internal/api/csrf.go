package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
)

const (
	csrfTokenLength = 32
	csrfCookieName  = "csrf_token"
	csrfHeaderName  = "X-CSRF-Token"
	csrfLegacyPath  = "/api" // Legacy path from before 2026-01-28 fix
	csrfCurrentPath = "/"    // Current path for JavaScript accessibility
)

// generateCSRFToken creates a cryptographically secure random token
func generateCSRFToken() (string, error) {
	bytes := make([]byte, csrfTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// clearLegacyCSRFCookie removes the legacy CSRF cookie with Path=/api
// This is needed because cookies with different paths are treated as separate cookies
func clearLegacyCSRFCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   csrfCookieName,
		Value:  "",
		Path:   csrfLegacyPath,
		MaxAge: -1,
	})
}

// setCSRFTokenCookie sets the CSRF token cookie
func setCSRFTokenCookie(w http.ResponseWriter, token string) {
	// First, clear any legacy cookie with Path=/api to prevent mismatch
	clearLegacyCSRFCookie(w)

	// Set new cookie with Path=/
	cookie := &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     csrfCurrentPath, // Must be "/" so JavaScript can read it from any page
		MaxAge:   900,             // 15 minutes (sync with access token)
		HttpOnly: false,           // MUST be readable by JavaScript!
		Secure:   true,
		SameSite: http.SameSiteStrictMode, // SEC-L04: Match auth cookies for consistency
	}

	if isDevelopment() {
		cookie.Secure = false
	}

	http.SetCookie(w, cookie)
}

// clearCSRFTokenCookie removes both current and legacy CSRF token cookies
func clearCSRFTokenCookie(w http.ResponseWriter) {
	// Clear legacy cookie (Path=/api)
	clearLegacyCSRFCookie(w)

	// Clear current cookie (Path=/)
	http.SetCookie(w, &http.Cookie{
		Name:   csrfCookieName,
		Value:  "",
		Path:   csrfCurrentPath,
		MaxAge: -1,
	})
}

// csrfValidationResult contains validation result and diagnostic info
type csrfValidationResult struct {
	err             error
	multipleCookies bool
	cookieCount     int
}

// validateCSRF validates the CSRF token using double-submit pattern
// Returns validation result with diagnostic info for telemetry
func validateCSRF(r *http.Request) csrfValidationResult {
	result := csrfValidationResult{}

	// Get ALL cookies with the CSRF name to detect duplicates
	var csrfCookies []*http.Cookie
	for _, c := range r.Cookies() {
		if c.Name == csrfCookieName {
			csrfCookies = append(csrfCookies, c)
		}
	}

	result.cookieCount = len(csrfCookies)
	result.multipleCookies = len(csrfCookies) > 1

	if len(csrfCookies) == 0 {
		result.err = errors.New("CSRF token cookie missing")
		return result
	}

	// Get token from header
	headerToken := r.Header.Get(csrfHeaderName)
	if headerToken == "" {
		result.err = errors.New("CSRF token header missing")
		return result
	}

	// If multiple cookies exist, try to find one that matches the header
	// This handles the legacy Path=/api vs new Path=/ situation
	for _, cookie := range csrfCookies {
		if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(headerToken)) == 1 {
			// Found a matching cookie
			return result
		}
	}

	// No cookie matched
	result.err = errors.New("CSRF token mismatch")
	return result
}

// csrfMiddleware validates CSRF tokens for state-changing requests
func (s *Server) csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip validation for safe methods
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip CSRF validation if request uses ONLY Bearer token (CLI clients, API-only clients)
		// CSRF is only a risk when browser automatically sends cookies
		authHeader := r.Header.Get("Authorization")
		_, cookieErr := r.Cookie(AccessTokenCookie)
		hasCookie := cookieErr == nil

		// Skip CSRF check if:
		// 1. Authorization header present (Bearer token)
		// 2. AND no access_token cookie (pure API client, not browser)
		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " && !hasCookie {
			// Pure Bearer token auth without cookies → no CSRF risk
			next.ServeHTTP(w, r)
			return
		}

		// Validate CSRF token for cookie-based auth
		result := validateCSRF(r)

		if result.err != nil {
			// Enhanced telemetry: log if multiple cookies were present
			logFields := []any{
				slog.String("error", result.err.Error()),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_ip", getClientIPSafe(r)),
				slog.Int("csrf_cookie_count", result.cookieCount),
				slog.Bool("multiple_csrf_cookies", result.multipleCookies),
			}
			s.logger().Warn("CSRF validation failed", logFields...)

			// Hotfix: If validation failed and multiple cookies exist,
			// clear the legacy cookie to help the session self-repair on next refresh
			if result.multipleCookies {
				clearLegacyCSRFCookie(w)
			}

			respondError(w, http.StatusForbidden, "CSRF token validation failed")
			return
		}

		// If validation succeeded but multiple cookies exist, clean up legacy cookie
		if result.multipleCookies {
			clearLegacyCSRFCookie(w)
			s.logger().Info("Cleaned up legacy CSRF cookie",
				slog.String("path", r.URL.Path),
				slog.String("remote_ip", getClientIPSafe(r)))
		}

		next.ServeHTTP(w, r)
	})
}
