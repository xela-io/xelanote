package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/service"
)

func setupExportServer(t *testing.T) (*Server, *db.DB, int) {
	t.Helper()

	database, err := db.Open(":memory:", "")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	userID := 1
	if _, err := database.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (?, 'testuser', 'test@example.com', 'hash', datetime('now'))
	`, userID); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	noteService := service.NewNoteService(database)

	return &Server{
		noteService: noteService,
		log:         logger,
	}, database, userID
}

func TestExportMarkdown_WritesZipWithNotes(t *testing.T) {
	server, database, userID := setupExportServer(t)
	defer database.Close()

	if _, err := server.noteService.CreateNote(userID, "First Note", "Hello", "/"); err != nil {
		t.Fatalf("create note: %v", err)
	}
	if _, err := server.noteService.CreateNote(userID, "Second Note", "Body", "/projects"); err != nil {
		t.Fatalf("create note: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/export/markdown", nil)
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, userID))
	rec := httptest.NewRecorder()

	server.exportMarkdown(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/zip" {
		t.Fatalf("expected zip content type, got %q", contentType)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "xelanote-export-") {
		t.Fatalf("unexpected content disposition: %q", cd)
	}
	if rec.Body.Len() == 0 {
		t.Fatalf("expected response body")
	}
}
