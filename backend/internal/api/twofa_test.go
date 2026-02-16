//go:build fts5

package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"

	"github.com/xela-io/xelanote/internal/auth"
	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/service"
)

type testUser struct {
	*db.User
	Password    string
	TOTPSecret  string
	BackupCodes []string
}

type testServer struct {
	*Server
	db        *db.DB
	jwtSecret []byte
}

func newTestServer(t *testing.T) *testServer {
	database, err := db.Open(":memory:", "")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}
	if err := database.SetSetting("registration_enabled", "true"); err != nil {
		t.Fatalf("failed to enable registration in test: %v", err)
	}

	jwtSecret := make([]byte, 32)
	_, err = rand.Read(jwtSecret)
	if err != nil {
		t.Fatalf("failed to generate jwt secret: %v", err)
	}

	// Create test logger (discard output)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tfaService := service.NewTwoFactorService(database, logger)
	authService := service.NewAuthService(database, jwtSecret, tfaService)

	server := NewServer(ServerConfig{
		AuthService: authService,
		TFAService:  tfaService,
		Logger:      logger,
		JWTSecret:   jwtSecret,
	})

	return &testServer{
		Server:    server,
		db:        database,
		jwtSecret: jwtSecret,
	}
}

func (ts *testServer) createUser(t *testing.T, username, email, password string) *testUser {
	user, err := ts.Server.authService.Register(context.Background(), username, email, password)
	if err != nil {
		t.Fatalf("failed to create user via auth service: %v", err)
	}

	return &testUser{
		User:     user,
		Password: password,
	}
}

func (ts *testServer) getAuthToken(t *testing.T, user *db.User) string {
	token, err := auth.GenerateAccessToken(user.ID, user.Username, ts.jwtSecret)
	if err != nil {
		t.Fatalf("failed to generate jwt: %v", err)
	}
	return token
}

func TestTwoFactorAuth_E2E(t *testing.T) {
	ts := newTestServer(t)
	r := chi.NewRouter()

	user := ts.createUser(t, "testuser", "test@example.com", "password123")
	token := ts.getAuthToken(t, user.User)

	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondError(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}
			claims, err := auth.ValidateAccessToken(parts[1], ts.jwtSecret)
			if err != nil {
				respondError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	r.Post("/api/auth/login", ts.login)
	r.With(authMiddleware).Post("/api/2fa/setup", ts.setupTwoFactor)
	r.With(authMiddleware).Post("/api/2fa/verify", ts.verifyTwoFactor)
	r.With(authMiddleware).Get("/api/2fa/status", ts.getTwoFactorStatus)
	r.With(authMiddleware).Delete("/api/2fa", ts.disableTwoFactor)

	t.Run("Setup 2FA", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/2fa/setup", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
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
		body, _ := json.Marshal(map[string]string{"code": code})
		req := httptest.NewRequest(http.MethodPost, "/api/2fa/verify", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Login with 2FA", func(t *testing.T) {
		code, _ := totp.GenerateCode(user.TOTPSecret, time.Now())
		body, _ := json.Marshal(map[string]string{
			"username_or_email": user.Email,
			"password":          user.Password,
			"totp_code":         code,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// SEC-001: Mark as desktop client so tokens appear in response body
		req.Header.Set("X-Client-Type", "desktop")
		req.RemoteAddr = "127.0.0.1:12345"
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		var resp AuthResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NotEmpty(t, resp.AccessToken)
		token = resp.AccessToken
	})

	t.Run("Disable 2FA", func(t *testing.T) {
		// Use backup code instead of TOTP to avoid replay protection blocking
		// (Login_with_2FA and Disable_2FA run within same 30s window)
		body, _ := json.Marshal(map[string]string{
			"password":    user.Password,
			"backup_code": user.BackupCodes[0],
		})
		req := httptest.NewRequest(http.MethodDelete, "/api/2fa", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Get Status Disabled", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/2fa/status", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		var resp TwoFactorStatusResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.False(t, resp.Enabled)
	})
}
