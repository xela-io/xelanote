//go:build fts5

package api

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister_Success(t *testing.T) {
	ts := newTestServer(t)
	pub := chi.NewRouter()
	pub.Post("/api/auth/register", ts.register)

	rec := doJSON(t, pub, http.MethodPost, "/api/auth/register", RegisterRequest{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "securePassword123",
	}, "")

	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp AuthResponse
	decodeResponse(t, rec, &resp)
	assert.Equal(t, "newuser", resp.User.Username)
	assert.Equal(t, "new@example.com", resp.User.Email)
	assert.NotEmpty(t, resp.AccessToken, "desktop client should receive access token in body")
	assert.NotEmpty(t, resp.RefreshToken, "desktop client should receive refresh token in body")
	assert.NotEmpty(t, resp.EncryptionSalt, "registration should return encryption salt")
}

func TestRegister_MissingFields(t *testing.T) {
	ts := newTestServer(t)
	pub := chi.NewRouter()
	pub.Post("/api/auth/register", ts.register)

	tests := []struct {
		name string
		req  RegisterRequest
	}{
		{"missing username", RegisterRequest{Email: "a@b.com", Password: "pass1234"}},
		{"missing email", RegisterRequest{Username: "user", Password: "pass1234"}},
		{"missing password", RegisterRequest{Username: "user", Email: "a@b.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doJSON(t, pub, http.MethodPost, "/api/auth/register", tt.req, "")
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestRegister_DuplicateUser(t *testing.T) {
	ts := newTestServer(t)
	pub := chi.NewRouter()
	pub.Post("/api/auth/register", ts.register)

	// First registration succeeds
	rec := doJSON(t, pub, http.MethodPost, "/api/auth/register", RegisterRequest{
		Username: "dupuser",
		Email:    "dup@example.com",
		Password: "password123",
	}, "")
	require.Equal(t, http.StatusCreated, rec.Code)

	// Second registration with same username fails
	rec = doJSON(t, pub, http.MethodPost, "/api/auth/register", RegisterRequest{
		Username: "dupuser",
		Email:    "other@example.com",
		Password: "password123",
	}, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_InvalidUsername(t *testing.T) {
	ts := newTestServer(t)
	pub := chi.NewRouter()
	pub.Post("/api/auth/register", ts.register)

	rec := doJSON(t, pub, http.MethodPost, "/api/auth/register", RegisterRequest{
		Username: "no spaces allowed",
		Email:    "test@example.com",
		Password: "password123",
	}, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_Success(t *testing.T) {
	ts := newTestServer(t)
	ts.createUser(t, "loginuser", "login@example.com", "testpass123")

	pub := chi.NewRouter()
	pub.Post("/api/auth/login", ts.login)

	rec := doJSON(t, pub, http.MethodPost, "/api/auth/login", LoginRequest{
		UsernameOrEmail: "loginuser",
		Password:        "testpass123",
	}, "")

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp AuthResponse
	decodeResponse(t, rec, &resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "loginuser", resp.User.Username)
}

func TestLogin_ByEmail(t *testing.T) {
	ts := newTestServer(t)
	ts.createUser(t, "emailuser", "email@example.com", "testpass123")

	pub := chi.NewRouter()
	pub.Post("/api/auth/login", ts.login)

	rec := doJSON(t, pub, http.MethodPost, "/api/auth/login", LoginRequest{
		UsernameOrEmail: "email@example.com",
		Password:        "testpass123",
	}, "")

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp AuthResponse
	decodeResponse(t, rec, &resp)
	assert.Equal(t, "emailuser", resp.User.Username)
}

func TestLogin_WrongPassword(t *testing.T) {
	ts := newTestServer(t)
	ts.createUser(t, "wrongpw", "wrong@example.com", "correctPassword")

	pub := chi.NewRouter()
	pub.Post("/api/auth/login", ts.login)

	rec := doJSON(t, pub, http.MethodPost, "/api/auth/login", LoginRequest{
		UsernameOrEmail: "wrongpw",
		Password:        "incorrectPassword",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLogin_NonexistentUser(t *testing.T) {
	ts := newTestServer(t)
	pub := chi.NewRouter()
	pub.Post("/api/auth/login", ts.login)

	rec := doJSON(t, pub, http.MethodPost, "/api/auth/login", LoginRequest{
		UsernameOrEmail: "ghost",
		Password:        "password123",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRefresh_Success(t *testing.T) {
	ts := newTestServer(t)
	ts.createUser(t, "refreshuser", "refresh@example.com", "testpass123")

	pub := chi.NewRouter()
	pub.Post("/api/auth/login", ts.login)
	pub.Post("/api/auth/refresh", ts.refresh)

	// Login first to get refresh token
	rec := doJSON(t, pub, http.MethodPost, "/api/auth/login", LoginRequest{
		UsernameOrEmail: "refreshuser",
		Password:        "testpass123",
	}, "")
	require.Equal(t, http.StatusOK, rec.Code)
	var loginResp AuthResponse
	decodeResponse(t, rec, &loginResp)
	require.NotEmpty(t, loginResp.RefreshToken)

	// Use refresh token to get new access token
	rec = doJSON(t, pub, http.MethodPost, "/api/auth/refresh", RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	}, "")
	assert.Equal(t, http.StatusOK, rec.Code)
	var tokenResp TokenResponse
	decodeResponse(t, rec, &tokenResp)
	assert.NotEmpty(t, tokenResp.AccessToken)
	assert.NotEmpty(t, tokenResp.RefreshToken)
}

func TestRefresh_InvalidToken(t *testing.T) {
	ts := newTestServer(t)
	pub := chi.NewRouter()
	pub.Post("/api/auth/refresh", ts.refresh)

	rec := doJSON(t, pub, http.MethodPost, "/api/auth/refresh", RefreshRequest{
		RefreshToken: "invalid-token",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRefresh_MissingToken(t *testing.T) {
	ts := newTestServer(t)
	pub := chi.NewRouter()
	pub.Post("/api/auth/refresh", ts.refresh)

	rec := doJSON(t, pub, http.MethodPost, "/api/auth/refresh", RefreshRequest{}, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogout_Success(t *testing.T) {
	ts := newTestServer(t)
	ts.createUser(t, "logoutuser", "logout@example.com", "testpass123")

	pub := chi.NewRouter()
	pub.Post("/api/auth/login", ts.login)
	pub.Post("/api/auth/logout", ts.logout)
	pub.Post("/api/auth/refresh", ts.refresh)

	// Login to get tokens
	rec := doJSON(t, pub, http.MethodPost, "/api/auth/login", LoginRequest{
		UsernameOrEmail: "logoutuser",
		Password:        "testpass123",
	}, "")
	require.Equal(t, http.StatusOK, rec.Code)
	var loginResp AuthResponse
	decodeResponse(t, rec, &loginResp)

	// Logout
	rec = doJSON(t, pub, http.MethodPost, "/api/auth/logout", RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	}, "")
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Refresh with revoked token should fail
	rec = doJSON(t, pub, http.MethodPost, "/api/auth/refresh", RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	}, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMe_Success(t *testing.T) {
	ts := newTestServer(t)
	r := ts.testRouter()
	r.Get("/api/auth/me", ts.me)

	user := ts.createUser(t, "meuser", "me@example.com", "testpass123")
	token := ts.getAuthToken(t, user.User)

	rec := doJSON(t, r, http.MethodGet, "/api/auth/me", nil, token)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp UserResponse
	decodeResponse(t, rec, &resp)
	assert.Equal(t, "meuser", resp.Username)
	assert.Equal(t, "me@example.com", resp.Email)
}

func TestMe_Unauthorized(t *testing.T) {
	ts := newTestServer(t)
	r := ts.testRouter()
	r.Get("/api/auth/me", ts.me)

	rec := doJSON(t, r, http.MethodGet, "/api/auth/me", nil, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRegister_WhenDisabled(t *testing.T) {
	ts := newTestServer(t)
	// Disable registration
	err := ts.db.SetSetting("registration_enabled", "false")
	require.NoError(t, err)

	pub := chi.NewRouter()
	pub.Post("/api/auth/register", ts.register)

	rec := doJSON(t, pub, http.MethodPost, "/api/auth/register", RegisterRequest{
		Username: "blocked",
		Email:    "blocked@example.com",
		Password: "password123",
	}, "")
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
