package db

import (
	"testing"
)

func TestCreateSnippet_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	snip, err := db.CreateSnippet(1, "Greeting", "A greeting", "Hello, world!", "hw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snip.Name != "Greeting" {
		t.Errorf("expected name 'Greeting', got %q", snip.Name)
	}
	if snip.Shortcut != "hw" {
		t.Errorf("expected shortcut 'hw', got %q", snip.Shortcut)
	}
	if snip.Content != "Hello, world!" {
		t.Errorf("expected content 'Hello, world!', got %q", snip.Content)
	}
}

func TestCreateSnippet_EmptyName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	_, err := db.CreateSnippet(1, "", "", "content", "")
	if err == nil {
		t.Error("expected error for empty snippet name")
	}
}

func TestGetSnippet_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	created, err := db.CreateSnippet(1, "Test", "", "body", "")
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	got, err := db.GetSnippet(1, created.ID)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if got.Name != "Test" {
		t.Errorf("expected name 'Test', got %q", got.Name)
	}
}

func TestGetSnippet_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	_, err := db.GetSnippet(1, 99999)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetSnippet_WrongUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)
	createTestUser2(t, db, 2)

	snip, err := db.CreateSnippet(1, "Private", "", "secret", "")
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	// User 2 should not see user 1's snippet
	_, err = db.GetSnippet(2, snip.ID)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for wrong user, got %v", err)
	}
}

func TestGetAllSnippets(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	if _, err := db.CreateSnippet(1, "A", "", "a", ""); err != nil {
		t.Fatalf("failed to create snippet A: %v", err)
	}
	if _, err := db.CreateSnippet(1, "B", "", "b", ""); err != nil {
		t.Fatalf("failed to create snippet B: %v", err)
	}

	snippets, err := db.GetAllSnippets(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snippets) != 2 {
		t.Errorf("expected 2 snippets, got %d", len(snippets))
	}
}

func TestUpdateSnippet_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	snip, err := db.CreateSnippet(1, "Old", "", "old content", "o")
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	if err := db.UpdateSnippet(1, snip.ID, "New", "desc", "new content", "n"); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	updated, err := db.GetSnippet(1, snip.ID)
	if err != nil {
		t.Fatalf("failed to get updated: %v", err)
	}
	if updated.Name != "New" {
		t.Errorf("expected name 'New', got %q", updated.Name)
	}
	if updated.Content != "new content" {
		t.Errorf("expected content 'new content', got %q", updated.Content)
	}
	if updated.Shortcut != "n" {
		t.Errorf("expected shortcut 'n', got %q", updated.Shortcut)
	}
}

func TestUpdateSnippet_WrongUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)
	createTestUser2(t, db, 2)

	snip, err := db.CreateSnippet(1, "Private", "", "c", "")
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	err = db.UpdateSnippet(2, snip.ID, "Hacked", "", "hacked", "")
	if err == nil {
		t.Error("expected error when updating another user's snippet")
	}
}

func TestDeleteSnippet_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	snip, err := db.CreateSnippet(1, "ToDelete", "", "c", "")
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	if err := db.DeleteSnippet(1, snip.ID); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err = db.GetSnippet(1, snip.ID)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDeleteSnippet_WrongUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)
	createTestUser2(t, db, 2)

	snip, err := db.CreateSnippet(1, "Private", "", "c", "")
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	err = db.DeleteSnippet(2, snip.ID)
	if err == nil {
		t.Error("expected error when deleting another user's snippet")
	}
}
