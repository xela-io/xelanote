package api

import (
	"encoding/base64"
	"log/slog"
	"net/http"
)

func (s *Server) respondLoginSuccess(
	w http.ResponseWriter,
	r *http.Request,
	usernameOrEmail string,
	accessToken string,
	refreshToken string,
) bool {
	user, err := s.authService.GetUserByUsernameOrEmail(usernameOrEmail)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to retrieve user information")
		return false
	}

	salt, err := s.getOrGenerateUserSalt(user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch encryption salt")
		return false
	}

	setAccessTokenCookie(w, accessToken)
	setRefreshTokenCookie(w, refreshToken)

	csrfToken, err := generateCSRFToken()
	if err != nil {
		s.logger().Error("failed to generate CSRF token", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, "internal server error")
		return false
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

	respondJSON(w, http.StatusOK, resp)
	return true
}
