package db

import (
	"strings"
	"testing"
	"time"
)

// Helper function to create a test database
func setupTestDB(t *testing.T) *DB {
	db, err := Open(":memory:", "") // Empty key for tests
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Apply schema and migrations
	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// Helper function to create a test user
func createTestUser(t *testing.T, db *DB, userID int) {
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (?, 'testuser', 'test@example.com', 'hash', datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
}

func TestGetAllEncryptedNotesForUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create plaintext notes (should not be returned)
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at)
		VALUES ('note1', 'Plain Note 1', 'plain note 1', 'Content 1', '/', ?, datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create plaintext note: %v", err)
	}

	// Create encrypted notes (should be returned)
	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
		                   content_encrypted, wrapped_dek, encryption_version)
		VALUES ('note2', 'Encrypted Note 1', 'encrypted note 1', '', '/', ?, datetime('now'), datetime('now'), 1, 'wrapped_dek_1', 2)
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create encrypted note 1: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
		                   content_encrypted, wrapped_dek, encryption_version)
		VALUES ('note3', 'Encrypted Note 2', 'encrypted note 2', '', '/', ?, datetime('now'), datetime('now'), 1, 'wrapped_dek_2', 2)
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create encrypted note 2: %v", err)
	}

	// Create encrypted note for different user (should not be returned)
	// First create the other user
	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (999, 'otheruser', 'other@example.com', 'hash', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to create other user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
		                   content_encrypted, wrapped_dek, encryption_version)
		VALUES ('note4', 'Other User Note', 'other user note', '', '/', 999, datetime('now'), datetime('now'), 1, 'wrapped_dek_3', 2)
	`)
	if err != nil {
		t.Fatalf("Failed to create note for other user: %v", err)
	}

	// Test GetAllEncryptedNotesForUser
	notes, err := db.GetAllEncryptedNotesForUser(userID)
	if err != nil {
		t.Fatalf("GetAllEncryptedNotesForUser failed: %v", err)
	}

	// Should return only 2 encrypted notes for this user
	if len(notes) != 2 {
		t.Errorf("Expected 2 encrypted notes, got %d", len(notes))
	}

	// Verify the notes are the correct ones
	noteIDs := make(map[string]bool)
	for _, note := range notes {
		noteIDs[note.ID] = true
		if !note.ContentEncrypted {
			t.Errorf("Note %s should have content_encrypted = true", note.ID)
		}
		if note.WrappedDEK == "" {
			t.Errorf("Note %s should have wrapped_dek", note.ID)
		}
	}

	if !noteIDs["note2"] || !noteIDs["note3"] {
		t.Errorf("Expected notes note2 and note3, got: %v", noteIDs)
	}
}

func TestBulkUpdateWrappedDEKs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create encrypted notes
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
		                   content_encrypted, wrapped_dek, encryption_version)
		VALUES
			('note1', 'Note 1', 'note 1', '', '/', ?, datetime('now'), datetime('now'), 1, 'old_dek_1', 2),
			('note2', 'Note 2', 'note 2', '', '/', ?, datetime('now'), datetime('now'), 1, 'old_dek_2', 2)
	`, userID, userID)
	if err != nil {
		t.Fatalf("Failed to create notes: %v", err)
	}

	// Create note versions
	_, err = db.Exec(`
		INSERT INTO note_versions (id, note_id, user_id, version, title, content, snapshot_at,
		                           content_encrypted, wrapped_dek, encryption_version)
		VALUES
			(1, 'note1', ?, 1, 'Version 1', '', datetime('now'), 1, 'old_version_dek_1', 2),
			(2, 'note1', ?, 2, 'Version 2', '', datetime('now'), 1, 'old_version_dek_2', 2)
	`, userID, userID)
	if err != nil {
		t.Fatalf("Failed to create versions: %v", err)
	}

	// Prepare update maps
	noteUpdates := map[string]string{
		"note1": "new_dek_1",
		"note2": "new_dek_2",
	}
	versionUpdates := map[string]string{
		"1": "new_version_dek_1",
		"2": "new_version_dek_2",
	}

	// Test BulkUpdateWrappedDEKs
	err = db.BulkUpdateWrappedDEKs(userID, noteUpdates, versionUpdates)
	if err != nil {
		t.Fatalf("BulkUpdateWrappedDEKs failed: %v", err)
	}

	// Verify notes were updated
	var wrappedDEK string
	err = db.QueryRow(`SELECT wrapped_dek FROM notes WHERE id = ?`, "note1").Scan(&wrappedDEK)
	if err != nil {
		t.Fatalf("Failed to query note1: %v", err)
	}
	if wrappedDEK != "new_dek_1" {
		t.Errorf("note1 wrapped_dek: expected 'new_dek_1', got '%s'", wrappedDEK)
	}

	err = db.QueryRow(`SELECT wrapped_dek FROM notes WHERE id = ?`, "note2").Scan(&wrappedDEK)
	if err != nil {
		t.Fatalf("Failed to query note2: %v", err)
	}
	if wrappedDEK != "new_dek_2" {
		t.Errorf("note2 wrapped_dek: expected 'new_dek_2', got '%s'", wrappedDEK)
	}

	// Verify versions were updated
	err = db.QueryRow(`SELECT wrapped_dek FROM note_versions WHERE id = 1`).Scan(&wrappedDEK)
	if err != nil {
		t.Fatalf("Failed to query version 1: %v", err)
	}
	if wrappedDEK != "new_version_dek_1" {
		t.Errorf("version 1 wrapped_dek: expected 'new_version_dek_1', got '%s'", wrappedDEK)
	}

	err = db.QueryRow(`SELECT wrapped_dek FROM note_versions WHERE id = 2`).Scan(&wrappedDEK)
	if err != nil {
		t.Fatalf("Failed to query version 2: %v", err)
	}
	if wrappedDEK != "new_version_dek_2" {
		t.Errorf("version 2 wrapped_dek: expected 'new_version_dek_2', got '%s'", wrappedDEK)
	}
}

func TestBulkUpdateWrappedDEKs_EmptyMaps(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Test with empty maps (should succeed but do nothing)
	err := db.BulkUpdateWrappedDEKs(userID, map[string]string{}, map[string]string{})
	if err != nil {
		t.Errorf("BulkUpdateWrappedDEKs with empty maps should not fail: %v", err)
	}
}

func TestBulkUpdateWrappedDEKs_Atomicity(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create a note
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
		                   content_encrypted, wrapped_dek, encryption_version)
		VALUES ('note1', 'Note 1', 'note 1', '', '/', ?, datetime('now'), datetime('now'), 1, 'old_dek', 2)
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	// Try to update with invalid note ID (should fail)
	noteUpdates := map[string]string{
		"note1":       "new_dek_1",
		"nonexistent": "new_dek_2", // This will succeed but affect 0 rows
	}

	// This should succeed (SQLite doesn't fail on UPDATE with no matches)
	err = db.BulkUpdateWrappedDEKs(userID, noteUpdates, map[string]string{})
	if err != nil {
		t.Fatalf("BulkUpdateWrappedDEKs failed: %v", err)
	}

	// Verify note1 was still updated
	var wrappedDEK string
	err = db.QueryRow(`SELECT wrapped_dek FROM notes WHERE id = ?`, "note1").Scan(&wrappedDEK)
	if err != nil {
		t.Fatalf("Failed to query note1: %v", err)
	}
	if wrappedDEK != "new_dek_1" {
		t.Errorf("note1 should have been updated despite nonexistent note")
	}
}

// --- Tests for folder-scoped note titles (Migration 024) ---

// TestCreateNote_SameTitleDifferentFolders verifies that notes with the same title
// can be created in different folders.
func TestCreateNote_SameTitleDifferentFolders(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create note "Daily Log" in /work
	note1, err := db.CreateNote(userID, "Daily Log", "Work content", "/work")
	if err != nil {
		t.Fatalf("Failed to create note in /work: %v", err)
	}
	if note1.FolderPath != "/work" {
		t.Errorf("Expected folder_path /work, got %s", note1.FolderPath)
	}

	// Create note "Daily Log" in /personal (should succeed)
	note2, err := db.CreateNote(userID, "Daily Log", "Personal content", "/personal")
	if err != nil {
		t.Fatalf("Failed to create note in /personal: %v", err)
	}
	if note2.FolderPath != "/personal" {
		t.Errorf("Expected folder_path /personal, got %s", note2.FolderPath)
	}

	// Verify both notes exist with different IDs
	if note1.ID == note2.ID {
		t.Errorf("Notes should have different IDs, both got %s", note1.ID)
	}

	// Verify content is correct
	if note1.Content != "Work content" {
		t.Errorf("Expected 'Work content', got %s", note1.Content)
	}
	if note2.Content != "Personal content" {
		t.Errorf("Expected 'Personal content', got %s", note2.Content)
	}
}

// TestCreateNote_SameTitleSameFolder_ShouldFail verifies that duplicate titles
// within the same folder are rejected.
func TestCreateNote_SameTitleSameFolder_ShouldFail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create note "Meeting Notes" in /work
	_, err := db.CreateNote(userID, "Meeting Notes", "First meeting", "/work")
	if err != nil {
		t.Fatalf("Failed to create first note: %v", err)
	}

	// Try to create another note "Meeting Notes" in /work (should fail)
	_, err = db.CreateNote(userID, "Meeting Notes", "Second meeting", "/work")
	if err == nil {
		t.Fatal("Expected UNIQUE constraint error, got nil")
	}

	// Verify error message contains UNIQUE constraint
	errMsg := err.Error()
	if !strings.Contains(errMsg, "UNIQUE constraint failed") && !strings.Contains(errMsg, "UNIQUE") {
		t.Errorf("Expected UNIQUE constraint error, got: %s", errMsg)
	}
}

// TestGetNoteByTitleInFolder verifies folder-scoped note retrieval.
func TestGetNoteByTitleInFolder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create note "Project" in /work and /personal
	_, err := db.CreateNote(userID, "Project", "Work project", "/work")
	if err != nil {
		t.Fatalf("Failed to create note in /work: %v", err)
	}
	_, err = db.CreateNote(userID, "Project", "Personal project", "/personal")
	if err != nil {
		t.Fatalf("Failed to create note in /personal: %v", err)
	}

	// Get note from /work
	noteWork, err := db.GetNoteByTitleInFolder(userID, "Project", "/work")
	if err != nil {
		t.Fatalf("Failed to get note from /work: %v", err)
	}
	if noteWork.Content != "Work project" {
		t.Errorf("Expected 'Work project', got %s", noteWork.Content)
	}
	if noteWork.FolderPath != "/work" {
		t.Errorf("Expected folder_path /work, got %s", noteWork.FolderPath)
	}

	// Get note from /personal
	notePersonal, err := db.GetNoteByTitleInFolder(userID, "Project", "/personal")
	if err != nil {
		t.Fatalf("Failed to get note from /personal: %v", err)
	}
	if notePersonal.Content != "Personal project" {
		t.Errorf("Expected 'Personal project', got %s", notePersonal.Content)
	}
	if notePersonal.FolderPath != "/personal" {
		t.Errorf("Expected folder_path /personal, got %s", notePersonal.FolderPath)
	}

	// Try to get non-existent note
	_, err = db.GetNoteByTitleInFolder(userID, "NonExistent", "/work")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestGetNotesByTitle_MultipleResults verifies that all notes with the same title
// across folders are returned.
func TestGetNotesByTitle_MultipleResults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create note "Project" in multiple folders
	_, err := db.CreateNote(userID, "Project", "Work", "/work")
	if err != nil {
		t.Fatalf("Failed to create note in /work: %v", err)
	}
	_, err = db.CreateNote(userID, "Project", "Personal", "/personal")
	if err != nil {
		t.Fatalf("Failed to create note in /personal: %v", err)
	}
	_, err = db.CreateNote(userID, "Project", "Side", "/side")
	if err != nil {
		t.Fatalf("Failed to create note in /side: %v", err)
	}

	// Get all notes with title "Project"
	notes, err := db.GetNotesByTitle(userID, "Project")
	if err != nil {
		t.Fatalf("Failed to get notes by title: %v", err)
	}

	if len(notes) != 3 {
		t.Fatalf("Expected 3 notes, got %d", len(notes))
	}

	// Verify all notes have title "Project" but different folders
	folderMap := make(map[string]bool)
	for _, note := range notes {
		if note.Title != "Project" {
			t.Errorf("Expected title 'Project', got %s", note.Title)
		}
		folderMap[note.FolderPath] = true
	}

	expectedFolders := []string{"/work", "/personal", "/side"}
	for _, folder := range expectedFolders {
		if !folderMap[folder] {
			t.Errorf("Expected folder %s not found in results", folder)
		}
	}
}

// TestGetNoteByTitle_Deterministic verifies that GetNoteByTitle returns
// the most recently updated note when duplicates exist (with ORDER BY).
func TestGetNoteByTitle_Deterministic(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create note "Test" in different folders
	note1, err := db.CreateNote(userID, "Test", "Old", "/folder1")
	if err != nil {
		t.Fatalf("Failed to create note1: %v", err)
	}

	note2, err := db.CreateNote(userID, "Test", "New", "/folder2")
	if err != nil {
		t.Fatalf("Failed to create note2: %v", err)
	}

	// Set deterministic timestamps to control ordering
	ts1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ts2 := ts1.Add(2 * time.Second)
	if _, err := db.Exec(`UPDATE notes SET updated_at = ? WHERE id = ?`, ts1.Format(time.RFC3339), note1.ID); err != nil {
		t.Fatalf("Failed to update note1 timestamp: %v", err)
	}
	if _, err := db.Exec(`UPDATE notes SET updated_at = ? WHERE id = ?`, ts2.Format(time.RFC3339), note2.ID); err != nil {
		t.Fatalf("Failed to update note2 timestamp: %v", err)
	}

	// GetNoteByTitle should return the most recently updated note (note2)
	result, err := db.GetNoteByTitle(userID, "Test")
	if err != nil {
		t.Fatalf("Failed to get note by title: %v", err)
	}

	if result.ID != note2.ID {
		t.Errorf("Expected note2 (ID %s), got note1 (ID %s)", note2.ID, result.ID)
	}
	if result.Content != "New" {
		t.Errorf("Expected content 'New', got %s", result.Content)
	}

	// Update note1 to make it the most recent
	_, err = db.UpdateNote(userID, note1.ID, "Test", "Updated", "/folder1", note1.Version)
	if err != nil {
		t.Fatalf("Failed to update note1: %v", err)
	}

	// Now GetNoteByTitle should return note1
	result, err = db.GetNoteByTitle(userID, "Test")
	if err != nil {
		t.Fatalf("Failed to get note by title after update: %v", err)
	}

	if result.ID != note1.ID {
		t.Errorf("Expected note1 (ID %s) after update, got %s", note1.ID, result.ID)
	}
	if result.Content != "Updated" {
		t.Errorf("Expected content 'Updated', got %s", result.Content)
	}
}

func TestUpdateNoteTitle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	note, err := db.CreateNote(userID, "Old Title", "Content", "/")
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	updated, err := db.UpdateNoteTitle(userID, note.ID, "New Title", note.Version)
	if err != nil {
		t.Fatalf("UpdateNoteTitle failed: %v", err)
	}

	if updated.Title != "New Title" {
		t.Errorf("Expected updated title 'New Title', got %s", updated.Title)
	}
	if updated.Version != note.Version+1 {
		t.Errorf("Expected version %d, got %d", note.Version+1, updated.Version)
	}

	// Title normalization should update lookup
	byNewTitle, err := db.GetNoteByTitle(userID, "new title")
	if err != nil {
		t.Fatalf("GetNoteByTitle failed for updated title: %v", err)
	}
	if byNewTitle.ID != note.ID {
		t.Errorf("Expected note ID %s for updated title, got %s", note.ID, byNewTitle.ID)
	}

	_, err = db.UpdateNoteTitle(userID, note.ID, "Another Title", note.Version)
	if err != ErrVersionMismatch {
		t.Errorf("Expected ErrVersionMismatch, got %v", err)
	}
}

func TestGetNotesByIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	note1, err := db.CreateNote(userID, "First", "One", "/")
	if err != nil {
		t.Fatalf("Failed to create note1: %v", err)
	}
	note2, err := db.CreateNote(userID, "Second", "Two", "/")
	if err != nil {
		t.Fatalf("Failed to create note2: %v", err)
	}

	result, err := db.GetNotesByIDs(userID, []string{note1.ID, note2.ID, "missing"})
	if err != nil {
		t.Fatalf("GetNotesByIDs failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 notes, got %d", len(result))
	}
	if result[note1.ID] == nil || result[note2.ID] == nil {
		t.Errorf("Expected both note IDs in result map")
	}

	empty, err := db.GetNotesByIDs(userID, []string{})
	if err != nil {
		t.Fatalf("GetNotesByIDs with empty list failed: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("Expected empty result for empty ID list, got %d", len(empty))
	}
}
