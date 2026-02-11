package api

import (
	"encoding/base64"
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

	// Verify CAPTCHA token only on first login step (without 2FA code)
	// When 2FA code is provided, user already passed CAPTCHA in the first step
	// If a token is provided (from web or desktop iframe), always verify it.
	// Desktop clients without a token get a fallback bypass (offline/iframe failure).
	if req.TOTPCode == "" && req.BackupCode == "" {
		if req.CaptchaToken != "" {
			if err := s.turnstileService.Verify(r.Context(), req.CaptchaToken, clientIP); err != nil {
				respondError(w, http.StatusBadRequest, err.Error())
				return
			}
		} else if !isDesktopClient(r) {
			if err := s.turnstileService.Verify(r.Context(), "", clientIP); err != nil {
				respondError(w, http.StatusBadRequest, err.Error())
				return
			}
		}
	}

	// Check if this is a 2FA login attempt
	if req.TOTPCode != "" || req.BackupCode != "" {
		// Apply backup code rate limiter if backup code is provided
		if req.BackupCode != "" {
			if !s.backupCodeLimiter.Allow(clientIP) {
				respondError(w, http.StatusTooManyRequests, "Too many backup code attempts, please try again later")
				return
			}
		}

		// Authenticate with 2FA
		accessToken, refreshToken, err := s.authService.LoginWithTwoFactor(
			r.Context(),
			req.UsernameOrEmail,
			req.Password,
			req.TOTPCode,
			req.BackupCode,
		)
		if err != nil {
			// Record failed login attempt for account lockout
			s.accountLockout.RecordFailure(req.UsernameOrEmail, clientIP)
			respondError(w, http.StatusUnauthorized, err.Error())
			return
		}

		// Clear lockout counter on successful login
		s.accountLockout.RecordSuccess(req.UsernameOrEmail)

		// Get user info for response
		user, err := s.authService.GetUserByUsernameOrEmail(req.UsernameOrEmail)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to retrieve user information")
			return
		}

		// Fetch encryption salt (or generate if missing)
		salt, err := s.getOrGenerateUserSalt(user.ID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to fetch encryption salt")
			return
		}

		// Set cookies for cookie-based auth (only after successful 2FA)
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

		// Return tokens, user info, and encryption salt
		respondJSON(w, http.StatusOK, AuthResponse{
			AccessToken:    accessToken,
			RefreshToken:   refreshToken,
			EncryptionSalt: base64.StdEncoding.EncodeToString(salt),
			User: UserResponse{
				ID:       user.ID,
				Username: user.Username,
				Email:    user.Email,
				IsAdmin:  user.IsAdmin,
			},
		})
		return
	}

	// Normal login (first step)
	accessToken, refreshToken, requiresTwoFactor, methods, err := s.authService.Login(r.Context(), req.UsernameOrEmail, req.Password)
	if err != nil {
		// Record failed login attempt for account lockout
		s.accountLockout.RecordFailure(req.UsernameOrEmail, clientIP)

		// Log failed login attempt
		s.logger().Warn("login_failed",
			slog.String("identifier_hash", hashIdentifier(req.UsernameOrEmail)),
			slog.String("event", "login_failed"),
			slog.String("reason", "invalid_credentials"),
			slog.String("remote_ip", getClientIPSafe(r)))

		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// If 2FA is required, return with methods and pending login token
	if requiresTwoFactor {
		// Get user to generate pending login token
		user, err := s.authService.GetUserByUsernameOrEmail(req.UsernameOrEmail)
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
		return
	}

	// Clear lockout counter on successful login (without 2FA)
	s.accountLockout.RecordSuccess(req.UsernameOrEmail)

	// Log successful login
	s.logger().Info("user_logged_in",
		slog.String("identifier_hash", hashIdentifier(req.UsernameOrEmail)),
		slog.String("event", "login_success"),
		slog.String("remote_ip", getClientIPSafe(r)))

	// Get user info for response
	user, err := s.authService.GetUserByUsernameOrEmail(req.UsernameOrEmail)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to retrieve user information")
		return
	}

	// Fetch encryption salt (or generate if missing)
	salt, err := s.getOrGenerateUserSalt(user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch encryption salt")
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

	// Return tokens, user info, and encryption salt
	respondJSON(w, http.StatusOK, AuthResponse{
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		EncryptionSalt: base64.StdEncoding.EncodeToString(salt),
		User: UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  user.IsAdmin,
		},
	})
}
