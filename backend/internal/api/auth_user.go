package api

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/xela-io/xelanote/internal/db"
)

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
