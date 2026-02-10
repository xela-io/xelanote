package db

import (
	"testing"

	"github.com/xela-io/xelanote/internal/utils"
)

// Helper to create a named test user
func createNamedTestUser(t *testing.T, db *DB, id int, username, email string) {
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (?, ?, ?, 'hash', datetime('now'))
	`, id, username, email)
	if err != nil {
		t.Fatalf("Failed to create test user %s: %v", username, err)
	}
}

// Helper to create a test note
func createTestNote(t *testing.T, db *DB, noteID string, userID int, title string) {
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at)
		VALUES (?, ?, ?, 'test content', '/', ?, datetime('now'), datetime('now'))
	`, noteID, title, utils.NormalizeTitle(title), userID)
	if err != nil {
		t.Fatalf("Failed to create test note %s: %v", noteID, err)
	}
}

func TestSharingCreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createTestNote(t, db, "note1", 1, "Test Note")

	// Create share
	share, err := db.CreateNoteShare(1, "note1", 2, "viewer")
	if err != nil {
		t.Fatalf("CreateNoteShare failed: %v", err)
	}
	if share.Role != "viewer" {
		t.Errorf("Expected role 'viewer', got '%s'", share.Role)
	}
	if share.SharedWithUsername != "bob" {
		t.Errorf("Expected shared_with_username 'bob', got '%s'", share.SharedWithUsername)
	}
	if share.OwnerUsername != "alice" {
		t.Errorf("Expected owner_username 'alice', got '%s'", share.OwnerUsername)
	}

	// Get shares
	shares, err := db.GetNoteShares(1, "note1")
	if err != nil {
		t.Fatalf("GetNoteShares failed: %v", err)
	}
	if len(shares) != 1 {
		t.Fatalf("Expected 1 share, got %d", len(shares))
	}
}

func TestSharingDuplicateFails(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createTestNote(t, db, "note1", 1, "Test Note")

	_, err := db.CreateNoteShare(1, "note1", 2, "viewer")
	if err != nil {
		t.Fatalf("First share failed: %v", err)
	}

	// Duplicate should fail
	_, err = db.CreateNoteShare(1, "note1", 2, "editor")
	if err != ErrDuplicate {
		t.Errorf("Expected ErrDuplicate, got: %v", err)
	}
}

func TestSharingInvalidRole(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createTestNote(t, db, "note1", 1, "Test Note")

	_, err := db.CreateNoteShare(1, "note1", 2, "admin")
	if err == nil {
		t.Error("Expected error for invalid role 'admin', got nil")
	}
}

func TestSharingDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createTestNote(t, db, "note1", 1, "Test Note")

	_, err := db.CreateNoteShare(1, "note1", 2, "viewer")
	if err != nil {
		t.Fatalf("CreateNoteShare failed: %v", err)
	}

	// Delete share
	err = db.DeleteNoteShare(1, "note1", 2)
	if err != nil {
		t.Fatalf("DeleteNoteShare failed: %v", err)
	}

	// Verify deleted
	shares, err := db.GetNoteShares(1, "note1")
	if err != nil {
		t.Fatalf("GetNoteShares failed: %v", err)
	}
	if len(shares) != 0 {
		t.Errorf("Expected 0 shares after delete, got %d", len(shares))
	}
}

func TestSharingDeleteNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createTestNote(t, db, "note1", 1, "Test Note")

	err := db.DeleteNoteShare(1, "note1", 2)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestSharingGetSharedNotesForUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createTestNote(t, db, "note1", 1, "Shared Note")
	createTestNote(t, db, "note2", 1, "Unshared Note")

	_, err := db.CreateNoteShare(1, "note1", 2, "editor")
	if err != nil {
		t.Fatalf("CreateNoteShare failed: %v", err)
	}

	// Bob should see 1 shared note
	notes, err := db.GetSharedNotesForUser(2)
	if err != nil {
		t.Fatalf("GetSharedNotesForUser failed: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("Expected 1 shared note, got %d", len(notes))
	}
	if notes[0].Title != "Shared Note" {
		t.Errorf("Expected title 'Shared Note', got '%s'", notes[0].Title)
	}
	if notes[0].SharedBy != "alice" {
		t.Errorf("Expected shared_by 'alice', got '%s'", notes[0].SharedBy)
	}
	if notes[0].ShareRole != "editor" {
		t.Errorf("Expected share_role 'editor', got '%s'", notes[0].ShareRole)
	}
}

func TestSharingGetSharePermission(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createTestNote(t, db, "note1", 1, "Test Note")

	_, err := db.CreateNoteShare(1, "note1", 2, "viewer")
	if err != nil {
		t.Fatalf("CreateNoteShare failed: %v", err)
	}

	// Bob has viewer permission
	role, err := db.GetSharePermission(2, "note1")
	if err != nil {
		t.Fatalf("GetSharePermission failed: %v", err)
	}
	if role != "viewer" {
		t.Errorf("Expected 'viewer', got '%s'", role)
	}

	// Alice has no share permission (she's the owner, not a shared user)
	role, err = db.GetSharePermission(1, "note1")
	if err != nil {
		t.Fatalf("GetSharePermission for owner failed: %v", err)
	}
	if role != "" {
		t.Errorf("Expected empty string for owner, got '%s'", role)
	}
}

func TestSharingUpdateRole(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createTestNote(t, db, "note1", 1, "Test Note")

	_, err := db.CreateNoteShare(1, "note1", 2, "viewer")
	if err != nil {
		t.Fatalf("CreateNoteShare failed: %v", err)
	}

	// Update role
	err = db.UpdateNoteShareRole(1, "note1", 2, "editor")
	if err != nil {
		t.Fatalf("UpdateNoteShareRole failed: %v", err)
	}

	// Verify updated
	role, err := db.GetSharePermission(2, "note1")
	if err != nil {
		t.Fatalf("GetSharePermission failed: %v", err)
	}
	if role != "editor" {
		t.Errorf("Expected 'editor', got '%s'", role)
	}
}

func TestSharingSearchUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createNamedTestUser(t, db, 3, "bobby", "bobby@test.com")

	// Search for "bob" excluding alice (user 1)
	results, err := db.SearchUserByUsernameOrEmail("bob", 1)
	if err != nil {
		t.Fatalf("SearchUserByUsernameOrEmail failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// Search too short
	_, err = db.SearchUserByUsernameOrEmail("ab", 1)
	if err == nil {
		t.Error("Expected error for short query, got nil")
	}

	// Search excluding self (bob)
	results, err = db.SearchUserByUsernameOrEmail("bob", 2)
	if err != nil {
		t.Fatalf("SearchUserByUsernameOrEmail failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result (bobby), got %d", len(results))
	}
	if results[0].Username != "bobby" {
		t.Errorf("Expected 'bobby', got '%s'", results[0].Username)
	}
}

func TestSharingGetNoteOwnerAndEncryption(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createTestNote(t, db, "note1", 1, "Test Note")

	// Owner check
	ownerID, err := db.GetNoteOwnerUserID("note1")
	if err != nil {
		t.Fatalf("GetNoteOwnerUserID failed: %v", err)
	}
	if ownerID != 1 {
		t.Errorf("Expected owner ID 1, got %d", ownerID)
	}

	// Non-existent note
	_, err = db.GetNoteOwnerUserID("nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}

	// Encryption check (default false)
	encrypted, err := db.IsNoteEncrypted("note1")
	if err != nil {
		t.Fatalf("IsNoteEncrypted failed: %v", err)
	}
	if encrypted {
		t.Error("Expected note to not be encrypted")
	}
}

func TestSharingCascadeDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createTestNote(t, db, "note1", 1, "Test Note")

	_, err := db.CreateNoteShare(1, "note1", 2, "viewer")
	if err != nil {
		t.Fatalf("CreateNoteShare failed: %v", err)
	}

	// Delete all shares for note
	err = db.DeleteAllSharesForNote("note1")
	if err != nil {
		t.Fatalf("DeleteAllSharesForNote failed: %v", err)
	}

	// Verify deleted
	shares, err := db.GetNoteShares(1, "note1")
	if err != nil {
		t.Fatalf("GetNoteShares failed: %v", err)
	}
	if len(shares) != 0 {
		t.Errorf("Expected 0 shares, got %d", len(shares))
	}
}
