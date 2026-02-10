package db

import (
	"testing"
)

// Simplified tests for DEK re-wrapping functions

func TestGetAllEncryptedNotesForUser_Simple(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create test user first (foreign key requirement)
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (1, 'test', 'test@example.com', 'hash', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	userID := 1

	// Create plaintext note using CreateNote (which handles title_norm)
	_, err = db.CreateNote(userID, "Plain Note", "Content", "/")
	if err != nil {
		t.Fatalf("Failed to create plaintext note: %v", err)
	}

	// Create encrypted notes with direct insert (need title_norm)
	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
		                   content_encrypted, wrapped_dek, encryption_version)
		VALUES ('enc1', 'Encrypted 1', 'encrypted 1', '', '/', ?, datetime('now'), datetime('now'), 1, 'dek1', 2)
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create encrypted note: %v", err)
	}

	// Get encrypted notes
	notes, err := db.GetAllEncryptedNotesForUser(userID)
	if err != nil {
		t.Fatalf("GetAllEncryptedNotesForUser failed: %v", err)
	}

	// Should return only 1 encrypted note
	if len(notes) != 1 {
		t.Errorf("Expected 1 encrypted note, got %d", len(notes))
	}

	if len(notes) > 0 && notes[0].ID != "enc1" {
		t.Errorf("Expected note ID 'enc1', got '%s'", notes[0].ID)
	}
}

func TestBulkUpdateWrappedDEKs_Simple(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create test user first
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (1, 'test', 'test@example.com', 'hash', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	userID := 1

	// Create encrypted note
	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
		                   content_encrypted, wrapped_dek, encryption_version)
		VALUES ('note1', 'Note 1', 'note 1', '', '/', ?, datetime('now'), datetime('now'), 1, 'old_dek', 2)
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	// Create encrypted version
	_, err = db.Exec(`
		INSERT INTO note_versions (id, note_id, user_id, version, title, content, snapshot_at,
		                           content_encrypted, wrapped_dek, encryption_version)
		VALUES (1, 'note1', ?, 1, 'V1', '', datetime('now'), 1, 'old_v_dek', 2)
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}

	// Update DEKs
	noteUpdates := map[string]string{"note1": "new_dek"}
	versionUpdates := map[string]string{"1": "new_v_dek"}

	err = db.BulkUpdateWrappedDEKs(userID, noteUpdates, versionUpdates)
	if err != nil {
		t.Fatalf("BulkUpdateWrappedDEKs failed: %v", err)
	}

	// Verify note was updated
	var wrappedDEK string
	err = db.QueryRow(`SELECT wrapped_dek FROM notes WHERE id = 'note1'`).Scan(&wrappedDEK)
	if err != nil {
		t.Fatalf("Failed to query note: %v", err)
	}
	if wrappedDEK != "new_dek" {
		t.Errorf("Expected 'new_dek', got '%s'", wrappedDEK)
	}

	// Verify version was updated
	err = db.QueryRow(`SELECT wrapped_dek FROM note_versions WHERE id = 1`).Scan(&wrappedDEK)
	if err != nil {
		t.Fatalf("Failed to query version: %v", err)
	}
	if wrappedDEK != "new_v_dek" {
		t.Errorf("Expected 'new_v_dek', got '%s'", wrappedDEK)
	}
}

func TestGetAllEncryptedVersionsForUser_Simple(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create test user first
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (1, 'test', 'test@example.com', 'hash', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	userID := 1

	// Create a note first (required for foreign key)
	_, err = db.CreateNote(userID, "Test Note", "Content", "/")
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	// Get the note ID
	var noteID string
	err = db.QueryRow(`SELECT id FROM notes WHERE user_id = ? LIMIT 1`, userID).Scan(&noteID)
	if err != nil {
		t.Fatalf("Failed to get note ID: %v", err)
	}

	// Create encrypted version
	_, err = db.Exec(`
		INSERT INTO note_versions (note_id, user_id, version, title, content, snapshot_at,
		                           content_encrypted, wrapped_dek, encryption_version)
		VALUES (?, ?, 1, 'V1', '', datetime('now'), 1, 'dek1', 2)
	`, noteID, userID)
	if err != nil {
		t.Fatalf("Failed to create encrypted version: %v", err)
	}

	// Get encrypted versions
	versions, err := db.GetAllEncryptedVersionsForUser(userID)
	if err != nil {
		t.Fatalf("GetAllEncryptedVersionsForUser failed: %v", err)
	}

	// Should return 1 encrypted version
	if len(versions) != 1 {
		t.Errorf("Expected 1 encrypted version, got %d", len(versions))
	}

	if len(versions) > 0 && !versions[0].ContentEncrypted {
		t.Errorf("Version should be encrypted")
	}
}
