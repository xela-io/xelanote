package api

import (
	"archive/zip"
	"bytes"
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

func TestExportMarkdown_EncryptedNotesAreMarkedInExport(t *testing.T) {
	server, database, userID := setupExportServer(t)
	defer database.Close()

	if _, err := server.noteService.CreateEncryptedNote(
		userID,
		"",
		nil,
		false,
		[]byte("ciphertext"),
		"wrapped-dek",
		`{"version":2}`,
		nil,
		"/",
	); err != nil {
		t.Fatalf("create encrypted note: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/export/markdown", nil)
	req = req.WithContext(context.WithValue(req.Context(), userIDKey, userID))
	rec := httptest.NewRecorder()

	server.exportMarkdown(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("failed to read zip: %v", err)
	}
	if len(zr.File) == 0 {
		t.Fatalf("expected at least one file in zip")
	}

	foundMarkedEncrypted := false
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("failed to open zip entry: %v", err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("failed to read zip entry: %v", err)
		}
		content := string(data)
		if strings.Contains(content, "x-xelanote-encrypted: true") &&
			strings.Contains(content, "[Encrypted note omitted in server-side export.]") {
			foundMarkedEncrypted = true
			break
		}
	}

	if !foundMarkedEncrypted {
		t.Fatalf("expected marked encrypted export entry with placeholder content")
	}
}
