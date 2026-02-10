package api

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/xela-io/xelanote/internal/db"
)

// RegisterRequest represents the request body for user registration
type RegisterRequest struct {
	Username     string `json:"username"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	CaptchaToken string `json:"captcha_token,omitempty"`
}

// LoginRequest represents the request body for user login
type LoginRequest struct {
	UsernameOrEmail string `json:"username_or_email"`
	Password        string `json:"password"`
	TOTPCode        string `json:"totp_code,omitempty"`
	BackupCode      string `json:"backup_code,omitempty"`
	CaptchaToken    string `json:"captcha_token,omitempty"`
}

// RefreshRequest represents the request body for token refresh
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RecoverySaltRequest represents the request to get recovery key salt
type RecoverySaltRequest struct {
	Email string `json:"email"`
}

// RecoveryResetPasswordRequest represents the request to reset password with recovery key
type RecoveryResetPasswordRequest struct {
	Email       string `json:"email"`
	RecoveryKey string `json:"recovery_key"`
	NewPassword string `json:"new_password"`
}

// AuthResponse represents the response for successful authentication
type AuthResponse struct {
	AccessToken       string       `json:"access_token,omitempty"`
	RefreshToken      string       `json:"refresh_token,omitempty"`
	User              UserResponse `json:"user,omitempty"`
	RequiresTwoFactor bool         `json:"requires_two_factor,omitempty"`
	TwoFactorMethods  []string     `json:"two_factor_methods,omitempty"`
	PendingLoginToken string       `json:"pending_login_token,omitempty"`
	EncryptionSalt    string       `json:"encryption_salt,omitempty"` // Base64-encoded salt for E2E encryption
}

// TokenResponse represents the response for token refresh
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// UserResponse represents user information (without sensitive data)
type UserResponse struct {
	ID             int     `json:"id"`
	Username       string  `json:"username"`
	Email          string  `json:"email"`
	IsAdmin        bool    `json:"is_admin"`
	EncryptionSalt *string `json:"encryption_salt,omitempty"` // Base64-encoded salt for E2E encryption
}

// Username validation for NEW registrations only
// Existing users with non-conforming names are grandfathered
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)

func validateUsername(username string) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("username must be 3-32 characters (alphanumeric, underscore, hyphen only)")
	}
	return nil
}

// isDesktopClient checks if the request is from a desktop client (Electron/Tauri)
// Desktop clients are identified by the X-Client-Type header AND must come from localhost.
// This prevents attackers from simply adding the header to bypass CAPTCHA.
func isDesktopClient(r *http.Request) bool {
	if r.Header.Get("X-Client-Type") != "desktop" {
		return false
	}

	// Only trust desktop header from localhost connections (Electron/Tauri apps)
	// Use RemoteAddr directly - do NOT trust X-Forwarded-For for this security check
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIP = r.RemoteAddr
	}

	return remoteIP == "127.0.0.1" || remoteIP == "::1"
}

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

	// Verify CAPTCHA token
	// If a token is provided (from web or desktop iframe), always verify it.
	// If no token and not a desktop client, require CAPTCHA (when enabled).
	// Desktop clients without a token get a fallback bypass (offline/iframe failure).
	clientIP := getClientIPSafe(r)
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

	// Register user
	user, err := s.authService.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
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

	// Return tokens, user info, and encryption salt
	respondJSON(w, http.StatusCreated, AuthResponse{
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

// getOrGenerateUserSalt fetches the user's encryption salt or generates one if it doesn't exist
// This handles migration for existing users who don't have a salt yet
func (s *Server) getOrGenerateUserSalt(userID int) ([]byte, error) {
	salt, err := s.authService.GetUserEncryptionSalt(userID)

	// If salt doesn't exist, check if user has encrypted notes
	if err == db.ErrNotFound {
		// CRITICAL: Check if user has encrypted notes before generating new salt
		// If they do, generating a new salt would make all encrypted notes permanently unreadable
		// Skip check if noteService is nil (happens in tests)
		if s.noteService != nil {
			hasEncryptedNotes, checkErr := s.noteService.UserHasEncryptedNotes(userID)
			if checkErr != nil {
				s.logger().Error("Failed to check for encrypted notes",
					slog.Int("user_id", userID),
					slog.String("error", checkErr.Error()))
				return nil, fmt.Errorf("failed to verify encryption status")
			}

			if hasEncryptedNotes {
				// CRITICAL ERROR: User has encrypted notes but salt is missing
				// This should NEVER happen - it means data corruption or migration error
				s.logger().Error("CRITICAL: User has encrypted notes but encryption salt is missing - REFUSING to generate new salt to prevent data loss",
					slog.Int("user_id", userID))
				return nil, fmt.Errorf("encryption salt missing but encrypted notes exist - contact administrator for data recovery")
			}
		}

		// Safe to generate new salt (no encrypted notes exist yet)
		s.logger().Info("Generating new encryption salt for user",
			slog.Int("user_id", userID))
		salt = make([]byte, 16) // 128-bit salt
		if _, err := rand.Read(salt); err != nil {
			return nil, err
		}
		if err := s.authService.SetUserEncryptionSalt(userID, salt); err != nil {
			return nil, err
		}
		return salt, nil
	}

	return salt, err
}

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
		respondError(w, http.StatusUnauthorized, err.Error())
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

	// Return new tokens
	respondJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	})
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

// me returns current authenticated user information
// This is a protected endpoint that requires valid JWT
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (set by authMiddleware)
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get user from database
	user, err := s.authService.GetUserByID(userID)
	if err != nil {
		if err == db.ErrNotFound {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to retrieve user")
		return
	}

	// Return user info (without password hash)
	respondJSON(w, http.StatusOK, UserResponse{
		ID:             user.ID,
		Username:       user.Username,
		Email:          user.Email,
		IsAdmin:        user.IsAdmin,
		EncryptionSalt: user.EncryptionSalt, // Include encryption salt for E2E encryption setup
	})
}

// getRecoveryKeySaltByEmail retrieves the recovery key salt for password recovery (public endpoint)
func (s *Server) getRecoveryKeySaltByEmail(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	// Ensure minimum response time to prevent timing attacks
	defer func() {
		elapsed := time.Since(startTime)
		if elapsed < 100*time.Millisecond {
			time.Sleep(100*time.Millisecond - elapsed)
		}
	}()

	var req RecoverySaltRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "email is required")
		return
	}

	salt, err := s.userService.GetRecoveryKeySaltByEmail(req.Email)
	if err != nil {
		// Return generic error to avoid user enumeration
		respondError(w, http.StatusNotFound, "recovery key not available")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"salt": base64.StdEncoding.EncodeToString(salt),
	})
}

// resetPasswordWithRecoveryKey resets a user's password using recovery key (public endpoint)
func (s *Server) resetPasswordWithRecoveryKey(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	// Ensure minimum response time to prevent timing attacks
	defer func() {
		elapsed := time.Since(startTime)
		if elapsed < 200*time.Millisecond {
			time.Sleep(200*time.Millisecond - elapsed)
		}
	}()

	var req RecoveryResetPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.RecoveryKey == "" {
		respondError(w, http.StatusBadRequest, "recovery_key is required")
		return
	}
	if req.NewPassword == "" {
		respondError(w, http.StatusBadRequest, "new_password is required")
		return
	}

	// Attempt password recovery
	err := s.userService.RecoverPasswordWithRecoveryKeyByEmail(req.Email, req.RecoveryKey, req.NewPassword)
	if err != nil {
		// Return generic error message for security
		s.logger().Warn("password recovery attempt failed", "email", req.Email, "error", err.Error())
		respondError(w, http.StatusUnauthorized, "invalid email, recovery key, or password requirements not met")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "password reset successfully, please login with your new password",
	})
}
