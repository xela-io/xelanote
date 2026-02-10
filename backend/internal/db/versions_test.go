package db

import (
	"testing"
)

func TestGetAllEncryptedVersionsForUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create a note first
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at)
		VALUES ('note1', 'Test Note', 'test note', 'Content', '/', ?, datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	// Create plaintext versions (should not be returned)
	_, err = db.Exec(`
		INSERT INTO note_versions (note_id, user_id, version, title, content, snapshot_at)
		VALUES ('note1', ?, 1, 'Plain Version', 'Content', datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create plaintext version: %v", err)
	}

	// Create encrypted versions (should be returned)
	_, err = db.Exec(`
		INSERT INTO note_versions (note_id, user_id, version, title, content, snapshot_at,
		                           content_encrypted, wrapped_dek, encryption_version)
		VALUES
			('note1', ?, 2, 'Encrypted V1', '', datetime('now'), 1, 'wrapped_dek_v1', 2),
			('note1', ?, 3, 'Encrypted V2', '', datetime('now'), 1, 'wrapped_dek_v2', 2)
	`, userID, userID)
	if err != nil {
		t.Fatalf("Failed to create encrypted versions: %v", err)
	}

	// Create encrypted version for different user (should not be returned)
	// First create the other user
	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (999, 'otheruser', 'other@example.com', 'hash', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to create other user: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO note_versions (note_id, user_id, version, title, content, snapshot_at,
		                           content_encrypted, wrapped_dek, encryption_version)
		VALUES ('note1', 999, 4, 'Other User Version', '', datetime('now'), 1, 'wrapped_dek_v3', 2)
	`)
	if err != nil {
		t.Fatalf("Failed to create version for other user: %v", err)
	}

	// Test GetAllEncryptedVersionsForUser
	versions, err := db.GetAllEncryptedVersionsForUser(userID)
	if err != nil {
		t.Fatalf("GetAllEncryptedVersionsForUser failed: %v", err)
	}

	// Should return only 2 encrypted versions for this user
	if len(versions) != 2 {
		t.Errorf("Expected 2 encrypted versions, got %d", len(versions))
	}

	// Verify the versions are encrypted
	for _, v := range versions {
		if !v.ContentEncrypted {
			t.Errorf("Version %d should have content_encrypted = true", v.ID)
		}
		if v.WrappedDEK == "" {
			t.Errorf("Version %d should have wrapped_dek", v.ID)
		}
		if v.Version < 2 {
			t.Errorf("Version %d should be version 2 or higher (encrypted), got version %d", v.ID, v.Version)
		}
	}
}

func TestGetAllEncryptedVersionsForUser_NoVersions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Test with no versions
	versions, err := db.GetAllEncryptedVersionsForUser(userID)
	if err != nil {
		t.Fatalf("GetAllEncryptedVersionsForUser failed: %v", err)
	}

	if len(versions) != 0 {
		t.Errorf("Expected 0 versions, got %d", len(versions))
	}
}

func TestGetAllEncryptedVersionsForUser_OnlyPlaintext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create a note
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at)
		VALUES ('note1', 'Test Note', 'test note', 'Content', '/', ?, datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create note: %v", err)
	}

	// Create only plaintext versions
	_, err = db.Exec(`
		INSERT INTO note_versions (note_id, user_id, version, title, content, snapshot_at)
		VALUES
			('note1', ?, 1, 'Plain V1', 'Content 1', datetime('now')),
			('note1', ?, 2, 'Plain V2', 'Content 2', datetime('now'))
	`, userID, userID)
	if err != nil {
		t.Fatalf("Failed to create plaintext versions: %v", err)
	}

	// Test - should return 0 encrypted versions
	versions, err := db.GetAllEncryptedVersionsForUser(userID)
	if err != nil {
		t.Fatalf("GetAllEncryptedVersionsForUser failed: %v", err)
	}

	if len(versions) != 0 {
		t.Errorf("Expected 0 encrypted versions (all plaintext), got %d", len(versions))
	}
}
