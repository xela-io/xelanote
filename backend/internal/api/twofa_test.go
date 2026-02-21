//go:build fts5

package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
)

func TestTwoFactorAuth_E2E(t *testing.T) {
	ts := newTestServer(t)

	// Auth-protected router for 2FA endpoints
	r := ts.testRouter()
	r.Post("/api/2fa/setup", ts.setupTwoFactor)
	r.Post("/api/2fa/verify", ts.verifyTwoFactor)
	r.Get("/api/2fa/status", ts.getTwoFactorStatus)
	r.Delete("/api/2fa", ts.disableTwoFactor)

	// Public router for login (no auth required)
	pub := chi.NewRouter()
	pub.Post("/api/auth/login", ts.login)

	user := ts.createUser(t, "testuser", "test@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	t.Run("Setup 2FA", func(t *testing.T) {
		rec := doJSON(t, r, http.MethodPost, "/api/2fa/setup", nil, token)
		assert.Equal(t, http.StatusOK, rec.Code)
		var resp TwoFactorSetupResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NotEmpty(t, resp.Secret)
		assert.NotEmpty(t, resp.BackupCodes)
		user.TOTPSecret = resp.Secret
		user.BackupCodes = resp.BackupCodes
	})

	t.Run("Verify 2FA", func(t *testing.T) {
		code, _ := totp.GenerateCode(user.TOTPSecret, time.Now())
		rec := doJSON(t, r, http.MethodPost, "/api/2fa/verify", map[string]string{"code": code}, token)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Login with 2FA", func(t *testing.T) {
		code, _ := totp.GenerateCode(user.TOTPSecret, time.Now())
		rec := doJSON(t, pub, http.MethodPost, "/api/auth/login", map[string]string{
			"username_or_email": user.Email,
			"password":          user.Password,
			"totp_code":         code,
		}, "")
		assert.Equal(t, http.StatusOK, rec.Code)
		var resp AuthResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NotEmpty(t, resp.AccessToken)
		token = resp.AccessToken
	})

	t.Run("Disable 2FA", func(t *testing.T) {
		// Use backup code instead of TOTP to avoid replay protection blocking
		// (Login_with_2FA and Disable_2FA run within same 30s window)
		rec := doJSON(t, r, http.MethodDelete, "/api/2fa", map[string]string{
			"password":    user.Password,
			"backup_code": user.BackupCodes[0],
		}, token)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Get Status Disabled", func(t *testing.T) {
		rec := doJSON(t, r, http.MethodGet, "/api/2fa/status", nil, token)
		assert.Equal(t, http.StatusOK, rec.Code)
		var resp TwoFactorStatusResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.False(t, resp.Enabled)
	})
}
