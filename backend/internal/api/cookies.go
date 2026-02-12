package api

import (
	"net/http"
)

const (
	AccessTokenCookie  = "access_token"
	RefreshTokenCookie = "refresh_token"
)

// setAccessTokenCookie sets the access token as an HttpOnly cookie
func setAccessTokenCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     AccessTokenCookie,
		Value:    token,
		Path:     "/api",
		MaxAge:   900, // 15 minutes
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode, // SEC-L04: Strict mode for full CSRF protection (signed URLs handle uploads)
	}

	if isDevelopment() {
		cookie.Secure = false
	}

	http.SetCookie(w, cookie)
}

// setRefreshTokenCookie sets the refresh token as an HttpOnly cookie
func setRefreshTokenCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    token,
		Path:     "/api",
		MaxAge:   2592000, // 30 days
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode, // SEC-L04: Strict mode for full CSRF protection
	}

	if isDevelopment() {
		cookie.Secure = false
	}

	http.SetCookie(w, cookie)
}

// clearAuthCookies removes both access and refresh token cookies
func clearAuthCookies(w http.ResponseWriter) {
	accessCookie := &http.Cookie{
		Name:     AccessTokenCookie,
		Value:    "",
		Path:     "/api",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode, // SEC-L04: Match cookie creation mode
	}

	refreshCookie := &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    "",
		Path:     "/api",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode, // SEC-L04: Match cookie creation mode
	}

	if isDevelopment() {
		accessCookie.Secure = false
		refreshCookie.Secure = false
	}

	http.SetCookie(w, accessCookie)
	http.SetCookie(w, refreshCookie)
}

// getAccessTokenFromCookie retrieves the access token from the request cookie
func getAccessTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(AccessTokenCookie)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// getRefreshTokenFromCookie retrieves the refresh token from the request cookie
func getRefreshTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(RefreshTokenCookie)
	if err != nil {
		return ""
	}
	return cookie.Value
}
