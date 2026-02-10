package service

import (
	"errors"
	"testing"

	"github.com/xela-io/xelanote/internal/db"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()

	database, err := db.Open(":memory:", "")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := database.Migrate(); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	t.Cleanup(func() {
		_ = database.Close()
	})

	return database
}

func createTestUser(t *testing.T, database *db.DB, username string) *db.User {
	t.Helper()

	user, err := database.CreateUser(username, username+"@example.com", "hash")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func TestNoteCacheUserIsolation(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)

	user1 := createTestUser(t, database, "user1")
	user2 := createTestUser(t, database, "user2")

	note, err := service.CreateNote(user1.ID, "Title", "content", "/")
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	if _, err := service.GetNote(user1.ID, note.ID); err != nil {
		t.Fatalf("failed to get note for user1: %v", err)
	}

	noteUser2, err := service.GetNote(user2.ID, note.ID)
	if !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for user2, got: %v", err)
	}
	if noteUser2 != nil {
		t.Fatalf("expected nil note for user2, got %v", noteUser2.ID)
	}
}

func TestNotesByFolderCacheInvalidationOnMove(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)

	user := createTestUser(t, database, "user")

	note, err := service.CreateNote(user.ID, "Title", "content", "/")
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	rootNotes, err := service.GetNotesByFolder(user.ID, "/")
	if err != nil {
		t.Fatalf("failed to list root notes: %v", err)
	}
	if len(rootNotes) != 1 {
		t.Fatalf("expected 1 root note, got %d", len(rootNotes))
	}

	updated, err := service.UpdateNote(user.ID, note.ID, note.Title, note.Content, "/projects", note.Version)
	if err != nil {
		t.Fatalf("failed to move note: %v", err)
	}

	rootNotes, err = service.GetNotesByFolder(user.ID, "/")
	if err != nil {
		t.Fatalf("failed to list root notes after move: %v", err)
	}
	if len(rootNotes) != 0 {
		t.Fatalf("expected 0 root notes after move, got %d", len(rootNotes))
	}

	projectNotes, err := service.GetNotesByFolder(user.ID, "/projects")
	if err != nil {
		t.Fatalf("failed to list project notes: %v", err)
	}
	if len(projectNotes) != 1 {
		t.Fatalf("expected 1 project note after move, got %d", len(projectNotes))
	}
	if projectNotes[0].ID != updated.ID {
		t.Fatalf("expected moved note %s, got %s", updated.ID, projectNotes[0].ID)
	}
}
