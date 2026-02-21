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

	if req.Username == "" || req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "username, email, and password are required")
		return
	}
	if err := validateUsername(req.Username); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	clientIP := getClientIPSafe(r)
	if err := s.turnstileService.Verify(r.Context(), req.CaptchaToken, clientIP); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, ok := s.registerOrBootstrapUser(w, r, req)
	if !ok {
		return
	}

	s.respondRegistrationSuccess(w, r, user, req)
}

// registerOrBootstrapUser tries Register, falling back to BootstrapAdmin when
// registration is disabled and a valid bootstrap token is provided.
func (s *Server) registerOrBootstrapUser(w http.ResponseWriter, r *http.Request, req RegisterRequest) (*service.User, bool) {
	user, err := s.authService.Register(r.Context(), req.Username, req.Email, req.Password)
	if err == nil {
		return user, true
	}

	if err != service.ErrRegistrationDisabled {
		respondRegistrationError(s, w, err)
		return nil, false
	}

	bootstrapSecret := os.Getenv("XELANOTE_BOOTSTRAP_TOKEN")
	if bootstrapSecret == "" {
		respondError(w, http.StatusForbidden, "registration is disabled")
		return nil, false
	}
	if subtle.ConstantTimeCompare([]byte(req.BootstrapToken), []byte(bootstrapSecret)) != 1 {
		respondError(w, http.StatusForbidden, "invalid bootstrap token")
		return nil, false
	}

	user, err = s.authService.BootstrapAdmin(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		respondRegistrationError(s, w, err)
		return nil, false
	}
	return user, true
}

// respondRegistrationSuccess handles the post-registration flow: generate salt,
// auto-login, set cookies, and return the response.
func (s *Server) respondRegistrationSuccess(w http.ResponseWriter, r *http.Request, user *service.User, req RegisterRequest) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate encryption salt")
		return
	}
	if err := s.authService.SetUserEncryptionSalt(user.ID, salt); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to store encryption salt")
		return
	}

	accessToken, refreshToken, requiresTwoFactor, _, err := s.authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "registration succeeded but login failed")
		return
	}
	if requiresTwoFactor {
		respondError(w, http.StatusInternalServerError, "unexpected 2FA requirement for new user")
		return
	}

	setAccessTokenCookie(w, accessToken)
	setRefreshTokenCookie(w, refreshToken)

	csrfToken, err := generateCSRFToken()
	if err != nil {
		s.logger().Error("failed to generate CSRF token", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	setCSRFTokenCookie(w, csrfToken)

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
