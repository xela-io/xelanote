package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/xela-io/xelanote/internal/db"
)

func setupSummarizeTestDB(t *testing.T) (*db.DB, int) {
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

	return database, userID
}

func TestSummarizeNote_ReturnsExistingSummaryWhenHashMatches(t *testing.T) {
	database, userID := setupSummarizeTestDB(t)
	defer database.Close()

	note, err := database.CreateNote(userID, "Title", "Hello world", "/")
	if err != nil {
		t.Fatalf("create note: %v", err)
	}

	hash := db.ComputeContentHash("Hello world")
	if err := database.UpdateNoteContentHash(userID, note.ID, hash); err != nil {
		t.Fatalf("update hash: %v", err)
	}
	if err := database.UpdateNoteSummary(userID, note.ID, "summary", false, time.Now().UTC()); err != nil {
		t.Fatalf("update summary: %v", err)
	}

	svc := NewSummarizeService(database, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	out, err := svc.SummarizeNote(context.Background(), userID, note.ID, "")
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}
	if out != "summary" {
		t.Fatalf("unexpected summary: %q", out)
	}
}

func TestSummarizeNote_EncryptedRequiresPlaintext(t *testing.T) {
	database, userID := setupSummarizeTestDB(t)
	defer database.Close()

	note, err := database.CreateNote(userID, "Title", "Secret", "/")
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if _, err := database.Exec(`UPDATE notes SET content_encrypted = 1 WHERE id = ?`, note.ID); err != nil {
		t.Fatalf("mark encrypted: %v", err)
	}

	svc := NewSummarizeService(database, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := svc.SummarizeNote(context.Background(), userID, note.ID, ""); err == nil {
		t.Fatalf("expected error for missing plaintext")
	}
}

func TestSummarizeNoteEncrypted_StoresSummary(t *testing.T) {
	database, userID := setupSummarizeTestDB(t)
	defer database.Close()

	note, err := database.CreateNote(userID, "Title", "Secret", "/")
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if _, err := database.Exec(`UPDATE notes SET content_encrypted = 1 WHERE id = ?`, note.ID); err != nil {
		t.Fatalf("mark encrypted: %v", err)
	}

	svc := NewSummarizeService(database, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := svc.SummarizeNoteEncrypted(context.Background(), userID, note.ID, "enc-summary", "hash123"); err != nil {
		t.Fatalf("summarize encrypted: %v", err)
	}

	updated, err := database.GetNote(userID, note.ID)
	if err != nil {
		t.Fatalf("get note: %v", err)
	}
	if updated.EncryptedSummary == nil || *updated.EncryptedSummary != "enc-summary" {
		t.Fatalf("expected encrypted summary stored")
	}
	if updated.ContentHash == nil || *updated.ContentHash != "hash123" {
		t.Fatalf("expected content hash updated")
	}
	if !updated.SummaryEncrypted {
		t.Fatalf("expected summary_encrypted true")
	}
}

func TestParseTagSuggestions(t *testing.T) {
	existing := map[string]bool{"work": true}
	input := "```json\n[{\"name\":\"Work\",\"score\":0.9},{\"name\":\"New Tag\",\"score\":0.2}]\n```"

	suggestions, err := parseTagSuggestions(input, existing)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(suggestions) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(suggestions))
	}
	if suggestions[0].Name != "work" || suggestions[0].IsNew {
		t.Fatalf("expected existing tag work")
	}
	if suggestions[1].Name != "new tag" || !suggestions[1].IsNew {
		t.Fatalf("expected new tag")
	}
}

func TestParseLinkSuggestions(t *testing.T) {
	validTitles := []string{"Alpha", "Beta"}
	existing := []string{"Beta"}
	input := "```json\n[{\"term\":\"alpha\",\"target_title\":\"Alpha\",\"confidence\":0.9},{\"term\":\"beta\",\"target_title\":\"Beta\",\"confidence\":0.5},{\"term\":\"nope\",\"target_title\":\"Gamma\",\"confidence\":0.1}]\n```"

	suggestions, err := parseLinkSuggestions(input, validTitles, existing)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}
	if suggestions[0].TargetTitle != "Alpha" || suggestions[0].Term != "alpha" {
		t.Fatalf("unexpected suggestion: %+v", suggestions[0])
	}
}
