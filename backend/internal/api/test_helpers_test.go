//go:build fts5

package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/auth"
	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/service"
	"github.com/xela-io/xelanote/internal/websocket"
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
	t.Cleanup(func() { _ = database.Close() })

	jwtSecret := make([]byte, 64)
	_, err = rand.Read(jwtSecret)
	if err != nil {
		t.Fatalf("failed to generate jwt secret: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tfaService := service.NewTwoFactorService(database, logger)
	authService := service.NewAuthService(database, jwtSecret, tfaService)
	noteService := service.NewNoteService(database)
	adminService := service.NewAdminService(database, t.TempDir())
	settingsService := service.NewSettingsService(database)
	activityService := service.NewActivityService(database)
	userService := service.NewUserService(database)
	sharingService := service.NewSharingService(database)
	templateService := service.NewTemplateService(database)
	snippetService := service.NewSnippetService(database)

	server := NewServer(ServerConfig{
		AuthService:      authService,
		TFAService:       tfaService,
		NoteService:      noteService,
		AdminService:     adminService,
		SettingsService:  settingsService,
		ActivityService:  activityService,
		UserService:      userService,
		SharingService:   sharingService,
		TemplateService:  templateService,
		SnippetService:   snippetService,
		TurnstileService: service.NewTurnstileService("", "", logger),
		WSManager:        websocket.NewManager(logger),
		Logger:           logger,
		JWTSecret:        jwtSecret,
		DBPing:           func() error { return nil },
	})

	return &testServer{
		Server:    server,
		db:        database,
		jwtSecret: jwtSecret,
	}
}

func (ts *testServer) createUser(t *testing.T, username, email, password string) *testUser {
	t.Helper()
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
	t.Helper()
	token, err := auth.GenerateAccessToken(user.ID, user.Username, ts.jwtSecret)
	if err != nil {
		t.Fatalf("failed to generate jwt: %v", err)
	}
	return token
}

// testRouter creates a chi router with auth middleware for handler testing.
func (ts *testServer) testRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
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
	})
	return r
}

// enableJournal enables the journal feature for a user.
func (ts *testServer) enableJournal(t *testing.T, userID int) {
	t.Helper()
	if err := ts.Server.noteService.SetUserFeature(userID, "journal", true, nil); err != nil {
		t.Fatalf("failed to enable journal feature: %v", err)
	}
}

// createNoteDirectly creates a note via the service layer (bypasses API).
func (ts *testServer) createNoteDirectly(t *testing.T, userID int, title, content, folder string) *service.Note {
	t.Helper()
	if folder == "" {
		folder = "/"
	}
	note, err := ts.Server.noteService.CreateNote(userID, title, content, folder)
	if err != nil {
		t.Fatalf("failed to create note directly: %v", err)
	}
	return note
}

// doJSON sends a JSON request and returns the recorder.
func doJSON(t *testing.T, handler http.Handler, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	// Desktop client mode to get tokens in response body
	req.Header.Set("X-Client-Type", "desktop")
	req.RemoteAddr = "127.0.0.1:12345"

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// doJSONWithHeaders sends a JSON request with extra headers and returns the recorder.
func doJSONWithHeaders(t *testing.T, handler http.Handler, method, path string, body interface{}, token string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("X-Client-Type", "desktop")
	req.RemoteAddr = "127.0.0.1:12345"
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// decodeResponse decodes a JSON response body into dest.
func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder, dest interface{}) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dest); err != nil {
		t.Fatalf("failed to decode response (status %d, body %q): %v", rec.Code, rec.Body.String(), err)
	}
}

// itoa converts an int to string (convenience for URL construction).
func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
