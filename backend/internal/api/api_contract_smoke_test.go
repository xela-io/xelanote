//go:build fts5

package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xela-io/xelanote/internal/service"
)

func TestAPIContractSmoke_ConfigResponse(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(ServerConfig{
		TurnstileService: service.NewTurnstileService("", "", logger),
		Logger:           logger,
		JWTSecret:        []byte("test-secret-key-that-is-at-least-64-chars-long-for-testing-purposes!!"),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()
	srv.getConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp ConfigResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if resp.Version == "" {
		t.Fatalf("expected non-empty version")
	}
	// Contract field exists and is a boolean.
	_ = resp.ErrorReportingEnabled
}

func TestAPIContractSmoke_DueDatesRequiresAuth(t *testing.T) {
	ts := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/due-dates", nil)
	rec := httptest.NewRecorder()
	ts.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated due-dates access, got %d", rec.Code)
	}
}

func TestAPIContractSmoke_WebAuthnQueryContracts(t *testing.T) {
	srv := &Server{}

	tests := []struct {
		name    string
		method  string
		handler func(http.ResponseWriter, *http.Request)
		path    string
	}{
		{
			name:    "delete requires credential_id query parameter",
			method:  http.MethodDelete,
			handler: srv.deleteWebAuthnCredential,
			path:    "/api/users/webauthn/credentials",
		},
		{
			name:    "touch requires credential_id query parameter",
			method:  http.MethodPatch,
			handler: srv.touchWebAuthnCredential,
			path:    "/api/users/webauthn/credentials/touch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req = req.WithContext(context.WithValue(req.Context(), userIDKey, 1))
			rec := httptest.NewRecorder()

			tt.handler(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}

			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}
			if body["error"] != "credential_id query parameter required" {
				t.Fatalf("unexpected error message: %q", body["error"])
			}
		})
	}
}
