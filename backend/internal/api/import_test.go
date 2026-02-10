package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/service"
	"github.com/xela-io/xelanote/internal/websocket"
)

func setupImportServer(t *testing.T) (*Server, *db.DB, int) {
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
	wsManager := websocket.NewManager(logger)

	return &Server{
		noteService: noteService,
		wsManager:   wsManager,
		log:         logger,
	}, database, userID
}

func doImportRequest(t *testing.T, s *Server, userID int, req ImportRequest) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/api/import", bytes.NewReader(body))
	r = r.WithContext(context.WithValue(r.Context(), userIDKey, userID))
	w := httptest.NewRecorder()

	s.importMarkdown(w, r)
	return w
}

func TestImportMarkdown_RejectsEmptyFiles(t *testing.T) {
	server, database, userID := setupImportServer(t)
	defer database.Close()
	rec := doImportRequest(t, server, userID, ImportRequest{Files: []ImportFile{}})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestImportMarkdown_SkipsEmptyAndImportsValid(t *testing.T) {
	server, database, userID := setupImportServer(t)
	defer database.Close()

	req := ImportRequest{
		Files: []ImportFile{
			{Filename: "empty.md", Content: ""},
			{Filename: "note.md", Content: "Hello world"},
		},
	}

	rec := doImportRequest(t, server, userID, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp ImportResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Imported != 1 || resp.Skipped != 1 || resp.Failed != 0 {
		t.Fatalf("unexpected counts: %+v", resp)
	}

	note, err := server.noteService.GetNoteByTitleInFolder(userID, "note", "/")
	if err != nil || note == nil {
		t.Fatalf("expected imported note, got err=%v", err)
	}
}

func TestImportMarkdown_PreserveStructureAndDedup(t *testing.T) {
	server, database, userID := setupImportServer(t)
	defer database.Close()

	content := "---\n" +
		"title: My Note\n" +
		"---\n" +
		"\nBody"

	req := ImportRequest{
		PreserveStructure: true,
		Files: []ImportFile{
			{Path: "folder/sub/note.md", Filename: "note.md", Content: content},
			{Path: "folder/sub/note.md", Filename: "note.md", Content: content},
		},
	}

	rec := doImportRequest(t, server, userID, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp ImportResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Imported != 1 || resp.Skipped != 1 {
		t.Fatalf("unexpected counts: %+v", resp)
	}
	if resp.FoldersCreated == 0 {
		t.Fatalf("expected folders to be created")
	}

	note, err := server.noteService.GetNoteByTitleInFolder(userID, "My Note", "/folder/sub")
	if err != nil || note == nil {
		t.Fatalf("expected imported note, got err=%v", err)
	}
}

func TestImportMarkdown_AppliesFrontmatterTags(t *testing.T) {
	server, database, userID := setupImportServer(t)
	defer database.Close()

	content := "---\n" +
		"title: Tagged Note\n" +
		"tags:\n" +
		"  - Work\n" +
		"  - Personal\n" +
		"---\n" +
		"\nBody"

	req := ImportRequest{
		Files: []ImportFile{
			{Filename: "note.md", Content: content},
		},
	}

	rec := doImportRequest(t, server, userID, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	note, err := server.noteService.GetNoteByTitleInFolder(userID, "Tagged Note", "/")
	if err != nil || note == nil {
		t.Fatalf("expected imported note, got err=%v", err)
	}

	tags, err := server.noteService.GetNoteTags(note.ID)
	if err != nil {
		t.Fatalf("expected tags, got err=%v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}

	names := []string{tags[0].Name, tags[1].Name}
	if !containsAll(names, []string{"Work", "Personal"}) {
		t.Fatalf("unexpected tags: %+v", names)
	}
}

func containsAll(haystack, needles []string) bool {
	for _, needle := range needles {
		found := false
		for _, item := range haystack {
			if item == needle {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
