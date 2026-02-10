package db

import (
	"testing"

	"github.com/xela-io/xelanote/internal/utils"
)

// Helper to create an encrypted test note
func createEncryptedTestNote(t *testing.T, db *DB, noteID string, userID int, title string) {
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id,
			content_encrypted, encrypted_content, encrypted_title, title_encrypted,
			wrapped_dek, encryption_version, encryption_metadata,
			encrypted_summary, summary_encrypted,
			created_at, updated_at)
		VALUES (?, ?, ?, '', '/', ?,
			1, X'DEADBEEF', 'enc_title', 1,
			'wrapped_key_data', 1, '{"alg":"test"}',
			'enc_summary', 1,
			datetime('now'), datetime('now'))
	`, noteID, title, utils.NormalizeTitle(title), userID)
	if err != nil {
		t.Fatalf("Failed to create encrypted test note %s: %v", noteID, err)
	}
}

func TestDecryptNote(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)
	createEncryptedTestNote(t, db, "enc1", userID, "Encrypted Note")

	// Verify note is encrypted
	note, err := db.GetNote(userID, "enc1")
	if err != nil {
		t.Fatalf("GetNote failed: %v", err)
	}
	if !note.ContentEncrypted {
		t.Fatal("Expected note to be encrypted")
	}

	// Decrypt the note
	decrypted, err := db.DecryptNote(userID, "enc1", "Decrypted Title", "Decrypted Content", note.Version)
	if err != nil {
		t.Fatalf("DecryptNote failed: %v", err)
	}

	// Verify all fields are correct
	if decrypted.Title != "Decrypted Title" {
		t.Errorf("Expected title 'Decrypted Title', got '%s'", decrypted.Title)
	}
	if decrypted.Content != "Decrypted Content" {
		t.Errorf("Expected content 'Decrypted Content', got '%s'", decrypted.Content)
	}
	if decrypted.ContentEncrypted {
		t.Error("Expected content_encrypted to be false")
	}
	if decrypted.TitleEncrypted {
		t.Error("Expected title_encrypted to be false")
	}
	if decrypted.EncryptionVersion != 0 {
		t.Errorf("Expected encryption_version 0, got %d", decrypted.EncryptionVersion)
	}
	if decrypted.WrappedDEK != "" {
		t.Errorf("Expected empty wrapped_dek, got '%s'", decrypted.WrappedDEK)
	}
	if decrypted.EncryptionMetadata != "" {
		t.Errorf("Expected empty encryption_metadata, got '%s'", decrypted.EncryptionMetadata)
	}
	if decrypted.EncryptedTitle != nil {
		t.Errorf("Expected nil encrypted_title, got '%v'", decrypted.EncryptedTitle)
	}
	if len(decrypted.EncryptedContent) != 0 {
		t.Errorf("Expected empty encrypted_content, got %d bytes", len(decrypted.EncryptedContent))
	}
	if decrypted.SummaryEncrypted {
		t.Error("Expected summary_encrypted to be false")
	}
	if decrypted.EncryptedSummary != nil {
		t.Errorf("Expected nil encrypted_summary, got '%v'", decrypted.EncryptedSummary)
	}
	// Version should be incremented
	if decrypted.Version != note.Version+1 {
		t.Errorf("Expected version %d, got %d", note.Version+1, decrypted.Version)
	}
}

func TestDecryptNotePlaintextError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)
	createTestNote(t, db, "plain1", userID, "Plain Note")

	// Get version
	note, _ := db.GetNote(userID, "plain1")

	// Attempt to decrypt a plaintext note should fail
	_, err := db.DecryptNote(userID, "plain1", "Title", "Content", note.Version)
	if err == nil {
		t.Fatal("Expected error when decrypting plaintext note, got nil")
	}
	if err.Error() != "note is not encrypted" {
		t.Errorf("Expected 'note is not encrypted' error, got: %v", err)
	}
}

func TestDecryptNoteVersionMismatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)
	createEncryptedTestNote(t, db, "enc2", userID, "Encrypted Note 2")

	// Get version
	note, _ := db.GetNote(userID, "enc2")

	// Attempt decrypt with wrong version
	_, err := db.DecryptNote(userID, "enc2", "Title", "Content", note.Version+999)
	if err != ErrVersionMismatch {
		t.Errorf("Expected ErrVersionMismatch, got: %v", err)
	}
}

func TestEncryptNoteRemovesShares(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createTestNote(t, db, "note1", 1, "Shared Note")

	// Create a share
	_, err := db.CreateNoteShare(1, "note1", 2, "viewer")
	if err != nil {
		t.Fatalf("CreateNoteShare failed: %v", err)
	}

	// Verify share exists
	shares, err := db.GetNoteShares(1, "note1")
	if err != nil {
		t.Fatalf("GetNoteShares failed: %v", err)
	}
	if len(shares) != 1 {
		t.Fatalf("Expected 1 share, got %d", len(shares))
	}

	// Simulate encryption by calling DeleteAllSharesForNote (as the service layer does)
	err = db.DeleteAllSharesForNote("note1")
	if err != nil {
		t.Fatalf("DeleteAllSharesForNote failed: %v", err)
	}

	// Verify shares are gone
	shares, err = db.GetNoteShares(1, "note1")
	if err != nil {
		t.Fatalf("GetNoteShares failed: %v", err)
	}
	if len(shares) != 0 {
		t.Errorf("Expected 0 shares after encryption, got %d", len(shares))
	}
}

func TestFolderEncryptionDefault(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create a folder
	folder, err := db.CreateFolder(userID, "/TestFolder", nil)
	if err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}

	// Default should be true (encrypted)
	encrypted, err := db.GetFolderEncryptionDefault(userID, folder.ID)
	if err != nil {
		t.Fatalf("GetFolderEncryptionDefault failed: %v", err)
	}
	if !encrypted {
		t.Error("Expected default encryption_default to be true")
	}

	// Also check via struct
	if !folder.EncryptionDefault {
		t.Error("Expected folder.EncryptionDefault to be true by default")
	}

	// Set to false (unencrypted)
	err = db.UpdateFolderEncryptionDefault(userID, folder.ID, false)
	if err != nil {
		t.Fatalf("UpdateFolderEncryptionDefault failed: %v", err)
	}

	// Verify
	encrypted, err = db.GetFolderEncryptionDefault(userID, folder.ID)
	if err != nil {
		t.Fatalf("GetFolderEncryptionDefault failed: %v", err)
	}
	if encrypted {
		t.Error("Expected encryption_default to be false after update")
	}

	// Set back to true
	err = db.UpdateFolderEncryptionDefault(userID, folder.ID, true)
	if err != nil {
		t.Fatalf("UpdateFolderEncryptionDefault failed: %v", err)
	}
	encrypted, err = db.GetFolderEncryptionDefault(userID, folder.ID)
	if err != nil {
		t.Fatalf("GetFolderEncryptionDefault failed: %v", err)
	}
	if !encrypted {
		t.Error("Expected encryption_default to be true after re-enabling")
	}
}

func TestFolderEncryptionDefaultByPath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create a folder
	_, err := db.CreateFolder(userID, "/TestFolder2", nil)
	if err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}

	// Default by path should be true
	encrypted, err := db.GetFolderEncryptionDefaultByPath(userID, "/TestFolder2")
	if err != nil {
		t.Fatalf("GetFolderEncryptionDefaultByPath failed: %v", err)
	}
	if !encrypted {
		t.Error("Expected encryption_default to be true")
	}

	// Non-existent folder should return true (safe default)
	encrypted, err = db.GetFolderEncryptionDefaultByPath(userID, "/NonExistent")
	if err != nil {
		t.Fatalf("GetFolderEncryptionDefaultByPath for non-existent failed: %v", err)
	}
	if !encrypted {
		t.Error("Expected true (safe default) for non-existent folder")
	}
}

func TestFolderEncryptionDefaultSurvivesRename(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create folder and set encryption_default to false
	folder, err := db.CreateFolder(userID, "/OriginalName", nil)
	if err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}

	err = db.UpdateFolderEncryptionDefault(userID, folder.ID, false)
	if err != nil {
		t.Fatalf("UpdateFolderEncryptionDefault failed: %v", err)
	}

	// Rename the folder
	err = db.RenameFolder(userID, folder.ID, "RenamedFolder")
	if err != nil {
		t.Fatalf("RenameFolder failed: %v", err)
	}

	// Encryption default should still be false (persisted on folder ID, not path)
	encrypted, err := db.GetFolderEncryptionDefault(userID, folder.ID)
	if err != nil {
		t.Fatalf("GetFolderEncryptionDefault after rename failed: %v", err)
	}
	if encrypted {
		t.Error("Expected encryption_default to remain false after rename")
	}
}
