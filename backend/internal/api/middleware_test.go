package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xela-io/xelanote/internal/service"
)

func TestPanicRecoveryMiddleware_ReportsAndReturns500(t *testing.T) {
	// Create a disabled error report service that still tracks EnqueueReport calls.
	// We use a real (but enabled) service with a buffered channel.
	svc := service.NewErrorReportService("https://forgejo.example.com", "owner", "repo", "token",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	// Don't start the background worker — we just want to read from the channel.

	s := &Server{
		errorReportService: svc,
	}

	// Handler that panics
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic value")
	})

	handler := s.panicRecoveryMiddleware(panicking)

	req := httptest.NewRequest(http.MethodGet, "/api/notes/42", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 500
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	// Should have enqueued a report
	select {
	case report := <-svc.ReportChan():
		if report.ErrorType != "Panic" {
			t.Fatalf("expected ErrorType=Panic, got %q", report.ErrorType)
		}
		if report.Message != "test panic value" {
			t.Fatalf("expected Message='test panic value', got %q", report.Message)
		}
		if report.Component != "backend" {
			t.Fatalf("expected Component=backend, got %q", report.Component)
		}
		if report.Stack == "" {
			t.Fatal("expected non-empty stack trace")
		}
		if report.Fingerprint == "" {
			t.Fatal("expected non-empty fingerprint")
		}
	default:
		t.Fatal("expected a report to be enqueued, but channel was empty")
	}
}

func TestPanicRecoveryMiddleware_NoPanicPassesThrough(t *testing.T) {
	s := &Server{}

	normal := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	handler := s.panicRecoveryMiddleware(normal)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}
