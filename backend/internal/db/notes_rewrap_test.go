package db

import (
	"database/sql"
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
		                   content_encrypted, wrapped_dek, wrapped_dek_recovery, encryption_version)
		VALUES ('enc1', 'Encrypted 1', 'encrypted 1', '', '/', ?, datetime('now'), datetime('now'), 1, 'dek1', 'recovery_dek1', 2)
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
	if len(notes) > 0 && notes[0].WrappedDEKRecovery != "recovery_dek1" {
		t.Errorf("Expected wrapped_dek_recovery 'recovery_dek1', got '%s'", notes[0].WrappedDEKRecovery)
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
		                           content_encrypted, wrapped_dek, wrapped_dek_recovery, encryption_version)
		VALUES (?, ?, 1, 'V1', '', datetime('now'), 1, 'dek1', 'recovery_dek1', 2)
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
	if len(versions) > 0 && versions[0].WrappedDEKRecovery != "recovery_dek1" {
		t.Errorf("Expected wrapped_dek_recovery 'recovery_dek1', got '%s'", versions[0].WrappedDEKRecovery)
	}
}

func TestBulkUpdateRecoveryWrappedDEKs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (1, 'test', 'test@example.com', 'hash', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	userID := 1

	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
		                   content_encrypted, wrapped_dek, encryption_version)
		VALUES ('note1', 'Note 1', 'note 1', '', '/', ?, datetime('now'), datetime('now'), 1, 'old_dek', 2)
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO note_versions (id, note_id, user_id, version, title, content, snapshot_at,
		                           content_encrypted, wrapped_dek, encryption_version)
		VALUES (1, 'note1', ?, 1, 'V1', '', datetime('now'), 1, 'old_v_dek', 2)
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}

	noteUpdates := map[string]string{"note1": "recovery_dek_note1"}
	versionUpdates := map[string]string{"1": "recovery_dek_version1"}

	err = db.BulkUpdateRecoveryWrappedDEKs(userID, noteUpdates, versionUpdates)
	if err != nil {
		t.Fatalf("BulkUpdateRecoveryWrappedDEKs failed: %v", err)
	}

	var wrappedDEKRecovery string
	err = db.QueryRow(`SELECT wrapped_dek_recovery FROM notes WHERE id = 'note1'`).Scan(&wrappedDEKRecovery)
	if err != nil {
		t.Fatalf("Failed to query note recovery wrapped DEK: %v", err)
	}
	if wrappedDEKRecovery != "recovery_dek_note1" {
		t.Errorf("Expected 'recovery_dek_note1', got '%s'", wrappedDEKRecovery)
	}

	err = db.QueryRow(`SELECT wrapped_dek_recovery FROM note_versions WHERE id = 1`).Scan(&wrappedDEKRecovery)
	if err != nil {
		t.Fatalf("Failed to query version recovery wrapped DEK: %v", err)
	}
	if wrappedDEKRecovery != "recovery_dek_version1" {
		t.Errorf("Expected 'recovery_dek_version1', got '%s'", wrappedDEKRecovery)
	}
}

func TestClearRecoveryWrappedDEKs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (1, 'test', 'test@example.com', 'hash', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	userID := 1

	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
		                   content_encrypted, wrapped_dek, wrapped_dek_recovery, encryption_version)
		VALUES ('note1', 'Note 1', 'note 1', '', '/', ?, datetime('now'), datetime('now'), 1, 'old_dek', 'recovery_dek_note1', 2)
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO note_versions (id, note_id, user_id, version, title, content, snapshot_at,
		                           content_encrypted, wrapped_dek, wrapped_dek_recovery, encryption_version)
		VALUES (1, 'note1', ?, 1, 'V1', '', datetime('now'), 1, 'old_v_dek', 'recovery_dek_version1', 2)
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}

	err = db.ClearRecoveryWrappedDEKs(userID)
	if err != nil {
		t.Fatalf("ClearRecoveryWrappedDEKs failed: %v", err)
	}

	var noteRecovery sql.NullString
	err = db.QueryRow(`SELECT wrapped_dek_recovery FROM notes WHERE id = 'note1'`).Scan(&noteRecovery)
	if err != nil {
		t.Fatalf("Failed to query note recovery wrapped DEK: %v", err)
	}
	if noteRecovery.Valid {
		t.Errorf("Expected note wrapped_dek_recovery to be NULL, got '%s'", noteRecovery.String)
	}

	var versionRecovery sql.NullString
	err = db.QueryRow(`SELECT wrapped_dek_recovery FROM note_versions WHERE id = 1`).Scan(&versionRecovery)
	if err != nil {
		t.Fatalf("Failed to query version recovery wrapped DEK: %v", err)
	}
	if versionRecovery.Valid {
		t.Errorf("Expected version wrapped_dek_recovery to be NULL, got '%s'", versionRecovery.String)
	}
}
