package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/xela-io/xelanote/internal/service"
)

// register handles user registration endpoint
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Username == "" || req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "username, email, and password are required")
		return
	}

	// Validate username format (only for new registrations, existing users are grandfathered)
	if err := validateUsername(req.Username); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Verify() returns nil when CAPTCHA is disabled, so a single call suffices.
	clientIP := getClientIPSafe(r)
	if err := s.turnstileService.Verify(r.Context(), req.CaptchaToken, clientIP); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Register user
	user, err := s.authService.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		if err == service.ErrRegistrationDisabled {
			bootstrapSecret := os.Getenv("XELANOTE_BOOTSTRAP_TOKEN")
			if bootstrapSecret == "" {
				respondError(w, http.StatusForbidden, "registration is disabled")
				return
			}
			if subtle.ConstantTimeCompare([]byte(req.BootstrapToken), []byte(bootstrapSecret)) != 1 {
				respondError(w, http.StatusForbidden, "invalid bootstrap token")
				return
			}
			user, err = s.authService.BootstrapAdmin(r.Context(), req.Username, req.Email, req.Password)
			if err != nil {
				respondRegistrationError(s, w, err)
				return
			}
		} else {
			respondRegistrationError(s, w, err)
			return
		}
	}

	// Generate encryption salt for E2E encryption
	salt := make([]byte, 16) // 128-bit salt
	if _, err := rand.Read(salt); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate encryption salt")
		return
	}

	if err := s.authService.SetUserEncryptionSalt(user.ID, salt); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to store encryption salt")
		return
	}

	// Auto-login after successful registration
	accessToken, refreshToken, requiresTwoFactor, _, err := s.authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		// Registration succeeded but login failed (should not happen)
		respondError(w, http.StatusInternalServerError, "registration succeeded but login failed")
		return
	}

	// New users never have 2FA enabled, but check just in case
	if requiresTwoFactor {
		respondError(w, http.StatusInternalServerError, "unexpected 2FA requirement for new user")
		return
	}

	// Set cookies for cookie-based auth
	setAccessTokenCookie(w, accessToken)
	setRefreshTokenCookie(w, refreshToken)

	// Generate and set CSRF token
	csrfToken, err := generateCSRFToken()
	if err != nil {
		s.logger().Error("failed to generate CSRF token", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	setCSRFTokenCookie(w, csrfToken)

	// Return user info and encryption salt
	// SEC-001: Tokens only in body for desktop clients (OS keyring storage)
	resp := AuthResponse{
		EncryptionSalt: base64.StdEncoding.EncodeToString(salt),
		User: UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  user.IsAdmin,
		},
	}
	if isDesktopClient(r) {
		resp.AccessToken = accessToken
		resp.RefreshToken = refreshToken
	}
	respondJSON(w, http.StatusCreated, resp)
}

// respondRegistrationError returns validation errors as 400 with the original
// message, and internal errors (DB, bcrypt) as 500 with a generic message.
func respondRegistrationError(s *Server, w http.ResponseWriter, err error) {
	var ve *service.ValidationError
	if errors.As(err, &ve) {
		respondError(w, http.StatusBadRequest, ve.Message)
		return
	}
	s.respondInternalErr(w, "registration failed", err)
}
