package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/xela-io/xelanote/internal/service"
)

// refresh issues new access token using refresh token
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	// Cookie has priority
	refreshToken := getRefreshTokenFromCookie(r)

	// Fallback to body for backwards compatibility
	if refreshToken == "" {
		var req RefreshRequest
		if err := decodeJSON(w, r, &req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		refreshToken = req.RefreshToken
	}

	if refreshToken == "" {
		respondError(w, http.StatusBadRequest, "refresh_token required")
		return
	}

	// Refresh tokens (implements token rotation)
	newAccessToken, newRefreshToken, err := s.authService.RefreshAccessToken(r.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, service.ErrRefreshTokenReuseDetected) {
			s.logger().Warn("refresh_token_reuse_detected",
				slog.String("event", "refresh_token_reuse"),
				securityIPAttr(r))
		}
		respondError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	// Set cookies for cookie-based auth
	setAccessTokenCookie(w, newAccessToken)
	setRefreshTokenCookie(w, newRefreshToken)

	// Generate and set CSRF token
	csrfToken, err := generateCSRFToken()
	if err != nil {
		s.logger().Error("failed to generate CSRF token", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	setCSRFTokenCookie(w, csrfToken)

	// SEC-001: Tokens only in body for desktop clients (OS keyring storage)
	resp := TokenResponse{}
	if isDesktopClient(r) {
		resp.AccessToken = newAccessToken
		resp.RefreshToken = newRefreshToken
	}
	respondJSON(w, http.StatusOK, resp)
}

// logout revokes refresh token
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	// Cookie has priority
	refreshToken := getRefreshTokenFromCookie(r)

	// Fallback to body
	if refreshToken == "" {
		var req RefreshRequest
		if err := decodeJSON(w, r, &req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		refreshToken = req.RefreshToken
	}

	if refreshToken == "" {
		respondError(w, http.StatusBadRequest, "refresh_token required")
		return
	}

	// Revoke refresh token
	err := s.authService.Logout(r.Context(), refreshToken)
	if err != nil {
		// Even if token doesn't exist, logout is successful
		// This is idempotent behavior
	}

	// Clear cookies
	clearAuthCookies(w)
	clearCSRFTokenCookie(w)

	// Return 204 No Content (successful logout)
	w.WriteHeader(http.StatusNoContent)
}
