package api

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/xela-io/xelanote/internal/service"
)

// changeEmail updates user email with password verification
func (s *Server) changeEmail(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req changeEmailRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.NewEmail == "" {
		respondError(w, http.StatusBadRequest, "new email is required")
		return
	}

	if req.CurrentPassword == "" {
		respondError(w, http.StatusBadRequest, "current password is required")
		return
	}

	// Get current refresh token from cookie or request body
	currentRefreshToken := getRefreshTokenFromCookie(r)

	err := s.userService.ChangeEmail(userID, req.NewEmail, req.CurrentPassword, currentRefreshToken)
	if err != nil {
		switch err {
		case service.ErrInvalidPassword:
			respondError(w, http.StatusUnauthorized, "incorrect password")
		case service.ErrEmailInUse:
			respondError(w, http.StatusConflict, "email already in use")
		case service.ErrInvalidEmail:
			respondError(w, http.StatusBadRequest, "invalid email format")
		default:
			s.logger().Error("failed to change email", "error", err)
			respondError(w, http.StatusInternalServerError, "failed to change email")
		}
		return
	}

	// Log email change event (PII protection: email not logged)
	s.logger().Info("email_changed",
		slog.Int("user_id", userID),
		slog.String("event", "email_change"),
		securityIPAttr(r))

	respondJSON(w, http.StatusOK, map[string]string{"message": "email changed successfully"})
}

// changePassword updates user password with verification
func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req changePasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CurrentPassword == "" {
		respondError(w, http.StatusBadRequest, "current password is required")
		return
	}

	if req.NewPassword == "" {
		respondError(w, http.StatusBadRequest, "new password is required")
		return
	}

	// Get current refresh token from cookie
	currentRefreshToken := getRefreshTokenFromCookie(r)

	err := s.userService.ChangePasswordWithDEKRewrap(
		userID,
		req.CurrentPassword,
		req.NewPassword,
		req.ReWrappedNoteDEKs,
		req.ReWrappedVersionDEKs,
		currentRefreshToken,
	)
	if err != nil {
		switch err {
		case service.ErrInvalidPassword:
			respondError(w, http.StatusUnauthorized, "incorrect password")
		case service.ErrPasswordTooShort:
			respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		default:
			// Check if it's a DEK re-wrapping error
			errMsg := err.Error()
			if errMsg == "DEK re-wrapping required: user has encrypted notes or versions" {
				respondError(w, http.StatusBadRequest, "DEK re-wrapping required")
			} else if strings.HasPrefix(errMsg, "missing") {
				// "missing re-wrapped DEK for note X"
				respondError(w, http.StatusBadRequest, errMsg)
			} else {
				s.logger().Error("failed to change password", "error", err)
				respondError(w, http.StatusInternalServerError, "failed to change password")
			}
		}
		return
	}

	// Log password change event
	s.logger().Info("password_changed",
		slog.Int("user_id", userID),
		slog.String("event", "password_change"),
		securityIPAttr(r))

	respondJSON(w, http.StatusOK, map[string]string{
		"message":                  "password changed successfully",
		"recovery_key_invalidated": "true",
	})
}
