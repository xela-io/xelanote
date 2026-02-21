package db

import (
	"testing"
)

func TestGetAllTags_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	tags, err := db.GetAllTags(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(tags))
	}
}

func TestGetOrCreateTag_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	tag, err := db.GetOrCreateTag(1, "Work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag.Name != "Work" {
		t.Errorf("expected name 'Work', got %q", tag.Name)
	}
	if tag.NameNorm != "work" {
		t.Errorf("expected normalized name 'work', got %q", tag.NameNorm)
	}
	if tag.UserID != 1 {
		t.Errorf("expected user_id 1, got %d", tag.UserID)
	}
}

func TestGetOrCreateTag_GetExisting(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	tag1, err := db.GetOrCreateTag(1, "Work")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Getting same tag (case-insensitive) should return same ID
	tag2, err := db.GetOrCreateTag(1, "work")
	if err != nil {
		t.Fatalf("get existing failed: %v", err)
	}
	if tag1.ID != tag2.ID {
		t.Errorf("expected same tag ID, got %d and %d", tag1.ID, tag2.ID)
	}
}

func TestGetOrCreateTag_EmptyName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	_, err := db.GetOrCreateTag(1, "")
	if err == nil {
		t.Error("expected error for empty tag name")
	}
}

func TestSetNoteTags_AndGetNoteTags(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	// Create a note
	note, err := db.CreateNote(1, "Tagged Note", "content", "/")
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	// Set tags
	if err := db.SetNoteTags(note.ID, 1, []string{"alpha", "beta", "gamma"}); err != nil {
		t.Fatalf("failed to set tags: %v", err)
	}

	// Get tags
	tags, err := db.GetNoteTags(note.ID)
	if err != nil {
		t.Fatalf("failed to get tags: %v", err)
	}
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(tags))
	}
}

func TestSetNoteTags_ReplacesExisting(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	note, err := db.CreateNote(1, "Replace Tags", "c", "/")
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	// Set initial tags
	if err := db.SetNoteTags(note.ID, 1, []string{"old1", "old2"}); err != nil {
		t.Fatalf("failed to set initial tags: %v", err)
	}

	// Replace with new tags
	if err := db.SetNoteTags(note.ID, 1, []string{"new1"}); err != nil {
		t.Fatalf("failed to replace tags: %v", err)
	}

	tags, err := db.GetNoteTags(note.ID)
	if err != nil {
		t.Fatalf("failed to get tags: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag after replace, got %d", len(tags))
	}
	if tags[0].NameNorm != "new1" {
		t.Errorf("expected tag 'new1', got %q", tags[0].NameNorm)
	}
}

func TestSetNoteTags_ClearAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	note, err := db.CreateNote(1, "Clear Tags", "c", "/")
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	if err := db.SetNoteTags(note.ID, 1, []string{"tag1"}); err != nil {
		t.Fatalf("failed to set tags: %v", err)
	}

	// Clear all tags
	if err := db.SetNoteTags(note.ID, 1, []string{}); err != nil {
		t.Fatalf("failed to clear tags: %v", err)
	}

	tags, err := db.GetNoteTags(note.ID)
	if err != nil {
		t.Fatalf("failed to get tags: %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected 0 tags after clear, got %d", len(tags))
	}
}

func TestDeleteTag(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	tag, err := db.GetOrCreateTag(1, "ToDelete")
	if err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}

	if err := db.DeleteTag(1, tag.ID); err != nil {
		t.Fatalf("failed to delete tag: %v", err)
	}

	// Verify gone
	tags, err := db.GetAllTags(1)
	if err != nil {
		t.Fatalf("failed to get tags: %v", err)
	}
	for _, tg := range tags {
		if tg.ID == tag.ID {
			t.Error("tag should have been deleted")
		}
	}
}

func TestDeleteTag_WrongUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)
	createTestUser2(t, db, 2)

	tag, err := db.GetOrCreateTag(1, "Private")
	if err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}

	// User 2 should not be able to delete user 1's tag
	err = db.DeleteTag(2, tag.ID)
	if err == nil {
		t.Error("expected error when deleting another user's tag")
	}
}

func TestGetAllTags_UserIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)
	createTestUser2(t, db, 2)

	if _, err := db.GetOrCreateTag(1, "user1tag"); err != nil {
		t.Fatalf("failed to create tag for user1: %v", err)
	}
	if _, err := db.GetOrCreateTag(2, "user2tag"); err != nil {
		t.Fatalf("failed to create tag for user2: %v", err)
	}

	tags1, err := db.GetAllTags(1)
	if err != nil {
		t.Fatalf("failed to get user1 tags: %v", err)
	}
	tags2, err := db.GetAllTags(2)
	if err != nil {
		t.Fatalf("failed to get user2 tags: %v", err)
	}

	if len(tags1) != 1 {
		t.Errorf("expected 1 tag for user1, got %d", len(tags1))
	}
	if len(tags2) != 1 {
		t.Errorf("expected 1 tag for user2, got %d", len(tags2))
	}
}

// Helper to create a second test user (since createTestUser hardcodes username)
func createTestUser2(t *testing.T, db *DB, userID int) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (?, 'testuser2', 'test2@example.com', 'hash', datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create test user 2: %v", err)
	}
}
