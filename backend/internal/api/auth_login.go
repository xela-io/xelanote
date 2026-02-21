package api

import (
	"fmt"
	"log/slog"
	"net/http"
)

// login handles user authentication endpoint
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	clientIP := getClientIPSafe(r)

	// Check account lockout before attempting login (prevents brute-force attacks)
	if locked, remaining := s.accountLockout.IsLocked(req.UsernameOrEmail, clientIP); locked {
		remainingSeconds := int(remaining.Seconds())
		respondError(w, http.StatusTooManyRequests,
			fmt.Sprintf("Account temporarily locked. Try again in %d seconds", remainingSeconds))
		return
	}

	// Verify CAPTCHA token only on first login step (without 2FA code).
	// Verify() returns nil when CAPTCHA is disabled, so a single call suffices.
	if req.TOTPCode == "" && req.BackupCode == "" {
		if err := s.turnstileService.Verify(r.Context(), req.CaptchaToken, clientIP); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	// 2FA login path
	if req.TOTPCode != "" || req.BackupCode != "" {
		s.handleTwoFactorLogin(w, r, req, clientIP)
		return
	}

	// Normal login (first step)
	accessToken, refreshToken, requiresTwoFactor, methods, err := s.authService.Login(r.Context(), req.UsernameOrEmail, req.Password)
	if err != nil {
		s.accountLockout.RecordFailure(req.UsernameOrEmail, clientIP)
		s.logger().Warn("login_failed",
			slog.String("identifier_hash", hashIdentifier(req.UsernameOrEmail)),
			slog.String("event", "login_failed"),
			slog.String("reason", "invalid_credentials"),
			securityIPAttr(r))
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if requiresTwoFactor {
		s.respondTwoFactorChallenge(w, req.UsernameOrEmail, methods)
		return
	}

	s.accountLockout.RecordSuccess(req.UsernameOrEmail)
	s.logger().Info("user_logged_in",
		slog.String("identifier_hash", hashIdentifier(req.UsernameOrEmail)),
		slog.String("event", "login_success"),
		securityIPAttr(r))

	s.respondLoginSuccess(w, r, req.UsernameOrEmail, accessToken, refreshToken)
}

// handleTwoFactorLogin authenticates with TOTP or backup code.
func (s *Server) handleTwoFactorLogin(w http.ResponseWriter, r *http.Request, req LoginRequest, clientIP string) {
	if req.BackupCode != "" {
		if !s.backupCodeLimiter.Allow(clientIP) {
			respondError(w, http.StatusTooManyRequests, "Too many backup code attempts, please try again later")
			return
		}
	}

	accessToken, refreshToken, err := s.authService.LoginWithTwoFactor(
		r.Context(), req.UsernameOrEmail, req.Password, req.TOTPCode, req.BackupCode,
	)
	if err != nil {
		s.accountLockout.RecordFailure(req.UsernameOrEmail, clientIP)
		respondError(w, http.StatusUnauthorized, "invalid credentials or two-factor code")
		return
	}

	s.accountLockout.RecordSuccess(req.UsernameOrEmail)
	s.respondLoginSuccess(w, r, req.UsernameOrEmail, accessToken, refreshToken)
}

// respondTwoFactorChallenge returns a 2FA challenge with a pending login token.
func (s *Server) respondTwoFactorChallenge(w http.ResponseWriter, identifier string, methods []string) {
	user, err := s.authService.GetUserByUsernameOrEmail(identifier)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to retrieve user information")
		return
	}

	pendingToken, err := s.storePendingLoginToken(user.ID, user.Username)
	if err != nil {
		s.logger().Error("failed to create pending login token", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, AuthResponse{
		RequiresTwoFactor: true,
		TwoFactorMethods:  methods,
		PendingLoginToken: pendingToken,
	})
}
