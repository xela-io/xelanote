package api

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xela-io/xelanote/internal/service"
)

func TestHealth_OK(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(ServerConfig{
		TurnstileService: service.NewTurnstileService("", "", logger),
		Logger:           logger,
		JWTSecret:        []byte("test-secret-key-that-is-at-least-64-chars-long-for-testing-purposes!!"),
		DBPing:           func() error { return nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("expected body 'ok', got %q", rec.Body.String())
	}
}

func TestHealth_DBError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(ServerConfig{
		TurnstileService: service.NewTurnstileService("", "", logger),
		Logger:           logger,
		JWTSecret:        []byte("test-secret-key-that-is-at-least-64-chars-long-for-testing-purposes!!"),
		DBPing:           func() error { return errors.New("connection refused") },
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if rec.Body.String() != "db_error" {
		t.Fatalf("expected body 'db_error', got %q", rec.Body.String())
	}
}

func TestHealth_NoPing(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(ServerConfig{
		TurnstileService: service.NewTurnstileService("", "", logger),
		Logger:           logger,
		JWTSecret:        []byte("test-secret-key-that-is-at-least-64-chars-long-for-testing-purposes!!"),
		// DBPing not set — should still return ok
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
