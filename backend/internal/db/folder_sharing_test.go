package db

import (
	"testing"
)

// Helper: create a test folder belonging to a user
func createTestFolder(t *testing.T, db *DB, userID int, path string) int {
	t.Helper()
	folder, err := db.CreateFolder(userID, path, nil)
	if err != nil {
		t.Fatalf("Failed to create test folder %s: %v", path, err)
	}
	return folder.ID
}

// Helper: create a test note in a specific folder
func createTestNoteInFolder(t *testing.T, db *DB, noteID string, userID int, title, folderPath string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at)
		VALUES (?, ?, ?, 'test content', ?, ?, datetime('now'), datetime('now'))
	`, noteID, title, NormalizeTitle(title), folderPath, userID)
	if err != nil {
		t.Fatalf("Failed to create test note %s: %v", noteID, err)
	}
}

// ============================================================================
// Folder Share CRUD
// ============================================================================

func TestCreateFolderShare(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")

	share, err := db.CreateFolderShare(1, folderID, 2, "viewer")
	if err != nil {
		t.Fatalf("CreateFolderShare failed: %v", err)
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
	if share.FolderPath != "/Recipes" {
		t.Errorf("Expected folder_path '/Recipes', got '%s'", share.FolderPath)
	}
}

func TestDeleteFolderShare(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")

	_, err := db.CreateFolderShare(1, folderID, 2, "viewer")
	if err != nil {
		t.Fatalf("CreateFolderShare failed: %v", err)
	}

	err = db.DeleteFolderShare(1, folderID, 2)
	if err != nil {
		t.Fatalf("DeleteFolderShare failed: %v", err)
	}

	// Verify deleted
	shares, err := db.GetFolderShares(1, folderID)
	if err != nil {
		t.Fatalf("GetFolderShares failed: %v", err)
	}
	if len(shares) != 0 {
		t.Errorf("Expected 0 shares after delete, got %d", len(shares))
	}
}

func TestDeleteFolderShare_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")

	err := db.DeleteFolderShare(1, folderID, 2)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestFolderShare_DuplicateFails(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")

	_, err := db.CreateFolderShare(1, folderID, 2, "viewer")
	if err != nil {
		t.Fatalf("First share failed: %v", err)
	}

	_, err = db.CreateFolderShare(1, folderID, 2, "editor")
	if err != ErrDuplicate {
		t.Errorf("Expected ErrDuplicate, got: %v", err)
	}
}

func TestGetSharedFoldersForUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")

	// Add some notes to the folder
	createTestNoteInFolder(t, db, "n1", 1, "Pasta", "/Recipes")
	createTestNoteInFolder(t, db, "n2", 1, "Salad", "/Recipes")

	_, err := db.CreateFolderShare(1, folderID, 2, "editor")
	if err != nil {
		t.Fatalf("CreateFolderShare failed: %v", err)
	}

	folders, err := db.GetSharedFoldersForUser(2)
	if err != nil {
		t.Fatalf("GetSharedFoldersForUser failed: %v", err)
	}
	if len(folders) != 1 {
		t.Fatalf("Expected 1 shared folder, got %d", len(folders))
	}
	if folders[0].Name != "Recipes" {
		t.Errorf("Expected folder name 'Recipes', got '%s'", folders[0].Name)
	}
	if folders[0].SharedBy != "alice" {
		t.Errorf("Expected shared_by 'alice', got '%s'", folders[0].SharedBy)
	}
	if folders[0].ShareRole != "editor" {
		t.Errorf("Expected share_role 'editor', got '%s'", folders[0].ShareRole)
	}
	if folders[0].NoteCount != 2 {
		t.Errorf("Expected 2 notes, got %d", folders[0].NoteCount)
	}
}

func TestGetSharedFolderNotes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")

	createTestNoteInFolder(t, db, "n1", 1, "Pasta", "/Recipes")
	createTestNoteInFolder(t, db, "n2", 1, "Salad", "/Recipes")

	_, err := db.CreateFolderShare(1, folderID, 2, "viewer")
	if err != nil {
		t.Fatalf("CreateFolderShare failed: %v", err)
	}

	notes, err := db.GetSharedFolderNotes(2, folderID)
	if err != nil {
		t.Fatalf("GetSharedFolderNotes failed: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("Expected 2 notes, got %d", len(notes))
	}
}

func TestEncryptedNoteExcludedFromFolderShare(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")

	createTestNoteInFolder(t, db, "n1", 1, "Pasta", "/Recipes")
	createTestNoteInFolder(t, db, "n2", 1, "Secret Recipe", "/Recipes")

	// Encrypt one note
	_, err := db.Exec(`UPDATE notes SET content_encrypted = 1 WHERE id = 'n2'`)
	if err != nil {
		t.Fatalf("Failed to encrypt note: %v", err)
	}

	_, err = db.CreateFolderShare(1, folderID, 2, "viewer")
	if err != nil {
		t.Fatalf("CreateFolderShare failed: %v", err)
	}

	notes, err := db.GetSharedFolderNotes(2, folderID)
	if err != nil {
		t.Fatalf("GetSharedFolderNotes failed: %v", err)
	}
	// Only the non-encrypted note should be visible
	if len(notes) != 1 {
		t.Fatalf("Expected 1 non-encrypted note, got %d", len(notes))
	}
	if notes[0].Title != "Pasta" {
		t.Errorf("Expected 'Pasta', got '%s'", notes[0].Title)
	}
}

func TestUpdateFolderShareRole(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")

	_, err := db.CreateFolderShare(1, folderID, 2, "viewer")
	if err != nil {
		t.Fatalf("CreateFolderShare failed: %v", err)
	}

	err = db.UpdateFolderShareRole(1, folderID, 2, "editor")
	if err != nil {
		t.Fatalf("UpdateFolderShareRole failed: %v", err)
	}

	shares, err := db.GetFolderShares(1, folderID)
	if err != nil {
		t.Fatalf("GetFolderShares failed: %v", err)
	}
	if len(shares) != 1 || shares[0].Role != "editor" {
		t.Errorf("Expected role 'editor', got '%s'", shares[0].Role)
	}
}

// ============================================================================
// Permission Chain
// ============================================================================

func TestSharePermission_FolderShareFallback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")
	createTestNoteInFolder(t, db, "n1", 1, "Pasta", "/Recipes")

	// Share the folder (not the individual note)
	_, err := db.CreateFolderShare(1, folderID, 2, "editor")
	if err != nil {
		t.Fatalf("CreateFolderShare failed: %v", err)
	}

	// Bob should have permission via folder share
	role, err := db.GetSharePermission(2, "n1")
	if err != nil {
		t.Fatalf("GetSharePermission failed: %v", err)
	}
	if role != "editor" {
		t.Errorf("Expected 'editor' via folder share, got '%s'", role)
	}
}

func TestSharePermission_NoteSharePriority(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")
	createTestNoteInFolder(t, db, "n1", 1, "Pasta", "/Recipes")

	// Share both folder and note individually
	_, err := db.CreateFolderShare(1, folderID, 2, "editor")
	if err != nil {
		t.Fatalf("CreateFolderShare failed: %v", err)
	}
	_, err = db.CreateNoteShare(1, "n1", 2, "viewer")
	if err != nil {
		t.Fatalf("CreateNoteShare failed: %v", err)
	}

	// Note share should take priority over folder share
	role, err := db.GetSharePermission(2, "n1")
	if err != nil {
		t.Fatalf("GetSharePermission failed: %v", err)
	}
	if role != "viewer" {
		t.Errorf("Expected 'viewer' (note share priority), got '%s'", role)
	}
}

func TestGetSharedNotesForUser_Dedupe(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")
	createTestNoteInFolder(t, db, "n1", 1, "Pasta", "/Recipes")

	// Share both folder and note individually
	_, err := db.CreateFolderShare(1, folderID, 2, "viewer")
	if err != nil {
		t.Fatalf("CreateFolderShare failed: %v", err)
	}
	_, err = db.CreateNoteShare(1, "n1", 2, "editor")
	if err != nil {
		t.Fatalf("CreateNoteShare failed: %v", err)
	}

	// Should not return duplicate
	notes, err := db.GetSharedNotesForUser(2)
	if err != nil {
		t.Fatalf("GetSharedNotesForUser failed: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("Expected 1 note (deduplicated), got %d", len(notes))
	}
	// Should use note_share role (editor), not folder_share role (viewer)
	if notes[0].ShareRole != "editor" {
		t.Errorf("Expected note share role 'editor', got '%s'", notes[0].ShareRole)
	}
}

func TestGetSharedNotesForUser_IncludesFolderNotes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")
	createTestNoteInFolder(t, db, "n1", 1, "Pasta", "/Recipes")

	// Only share the folder (not the note directly)
	_, err := db.CreateFolderShare(1, folderID, 2, "viewer")
	if err != nil {
		t.Fatalf("CreateFolderShare failed: %v", err)
	}

	notes, err := db.GetSharedNotesForUser(2)
	if err != nil {
		t.Fatalf("GetSharedNotesForUser failed: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("Expected 1 note from folder share, got %d", len(notes))
	}
	if notes[0].Title != "Pasta" {
		t.Errorf("Expected title 'Pasta', got '%s'", notes[0].Title)
	}
}

// ============================================================================
// Placements
// ============================================================================

func TestPlacement_RequiresActiveShare(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createTestNote(t, db, "n1", 1, "Test Note")
	bobFolderID := createTestFolder(t, db, 2, "/MyNotes")

	// Bob tries to place without any share → should fail
	err := db.CreateOrUpdatePlacement(2, "n1", bobFolderID)
	if err == nil {
		t.Fatal("Expected error when placing without share, got nil")
	}
}

func TestPlacement_WithNoteShare(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createTestNote(t, db, "n1", 1, "Test Note")
	bobFolderID := createTestFolder(t, db, 2, "/MyNotes")

	// Share note with Bob
	_, err := db.CreateNoteShare(1, "n1", 2, "viewer")
	if err != nil {
		t.Fatalf("CreateNoteShare failed: %v", err)
	}

	// Bob places it
	err = db.CreateOrUpdatePlacement(2, "n1", bobFolderID)
	if err != nil {
		t.Fatalf("CreateOrUpdatePlacement failed: %v", err)
	}

	// Verify placement exists
	folderID, err := db.GetPlacement(2, "n1")
	if err != nil {
		t.Fatalf("GetPlacement failed: %v", err)
	}
	if folderID == nil {
		t.Fatal("Expected placement, got nil")
	}
	if *folderID != bobFolderID {
		t.Errorf("Expected folder ID %d, got %d", bobFolderID, *folderID)
	}
}

func TestPlacement_WithFolderShare(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")
	createTestNoteInFolder(t, db, "n1", 1, "Pasta", "/Recipes")
	bobFolderID := createTestFolder(t, db, 2, "/MyRecipes")

	// Share folder with Bob
	_, err := db.CreateFolderShare(1, folderID, 2, "viewer")
	if err != nil {
		t.Fatalf("CreateFolderShare failed: %v", err)
	}

	// Bob places a note from the shared folder into his own folder
	err = db.CreateOrUpdatePlacement(2, "n1", bobFolderID)
	if err != nil {
		t.Fatalf("CreateOrUpdatePlacement failed: %v", err)
	}

	fID, err := db.GetPlacement(2, "n1")
	if err != nil {
		t.Fatalf("GetPlacement failed: %v", err)
	}
	if fID == nil || *fID != bobFolderID {
		t.Error("Placement not created or wrong folder")
	}
}

func TestPlacement_DeletePlacement(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createTestNote(t, db, "n1", 1, "Test Note")
	bobFolderID := createTestFolder(t, db, 2, "/MyNotes")

	_, err := db.CreateNoteShare(1, "n1", 2, "viewer")
	if err != nil {
		t.Fatalf("CreateNoteShare failed: %v", err)
	}

	err = db.CreateOrUpdatePlacement(2, "n1", bobFolderID)
	if err != nil {
		t.Fatalf("CreateOrUpdatePlacement failed: %v", err)
	}

	err = db.DeletePlacement(2, "n1")
	if err != nil {
		t.Fatalf("DeletePlacement failed: %v", err)
	}

	fID, err := db.GetPlacement(2, "n1")
	if err != nil {
		t.Fatalf("GetPlacement failed: %v", err)
	}
	if fID != nil {
		t.Error("Expected nil placement after delete")
	}
}

// ============================================================================
// Share Revocation → Placement Cleanup
// ============================================================================

func TestUnshareNote_RemovesPlacement(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createTestNote(t, db, "n1", 1, "Test Note")
	bobFolderID := createTestFolder(t, db, 2, "/MyNotes")

	// Share and place
	_, err := db.CreateNoteShare(1, "n1", 2, "viewer")
	if err != nil {
		t.Fatalf("CreateNoteShare failed: %v", err)
	}
	err = db.CreateOrUpdatePlacement(2, "n1", bobFolderID)
	if err != nil {
		t.Fatalf("CreateOrUpdatePlacement failed: %v", err)
	}

	// Unshare
	err = db.DeleteNoteShare(1, "n1", 2)
	if err != nil {
		t.Fatalf("DeleteNoteShare failed: %v", err)
	}

	// Placement should be cleaned up
	fID, err := db.GetPlacement(2, "n1")
	if err != nil {
		t.Fatalf("GetPlacement failed: %v", err)
	}
	if fID != nil {
		t.Error("Expected placement to be removed after unshare")
	}
}

func TestUnshareFolder_RemovesPlacements(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")
	createTestNoteInFolder(t, db, "n1", 1, "Pasta", "/Recipes")
	createTestNoteInFolder(t, db, "n2", 1, "Salad", "/Recipes")
	bobFolderID := createTestFolder(t, db, 2, "/MyRecipes")

	// Share folder with Bob
	_, err := db.CreateFolderShare(1, folderID, 2, "viewer")
	if err != nil {
		t.Fatalf("CreateFolderShare failed: %v", err)
	}

	// Bob places notes into his folder
	err = db.CreateOrUpdatePlacement(2, "n1", bobFolderID)
	if err != nil {
		t.Fatalf("CreateOrUpdatePlacement n1 failed: %v", err)
	}
	err = db.CreateOrUpdatePlacement(2, "n2", bobFolderID)
	if err != nil {
		t.Fatalf("CreateOrUpdatePlacement n2 failed: %v", err)
	}

	// Unshare folder
	err = db.DeleteFolderShare(1, folderID, 2)
	if err != nil {
		t.Fatalf("DeleteFolderShare failed: %v", err)
	}

	// All placements should be cleaned up
	fID1, _ := db.GetPlacement(2, "n1")
	fID2, _ := db.GetPlacement(2, "n2")
	if fID1 != nil || fID2 != nil {
		t.Error("Expected placements to be removed after folder unshare")
	}
}

// ============================================================================
// Folder Owner + Encryption Checks
// ============================================================================

func TestGetFolderOwnerUserID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")

	ownerID, err := db.GetFolderOwnerUserID(folderID)
	if err != nil {
		t.Fatalf("GetFolderOwnerUserID failed: %v", err)
	}
	if ownerID != 1 {
		t.Errorf("Expected owner ID 1, got %d", ownerID)
	}
}

func TestFolderHasEncryptedNotes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")
	createTestNoteInFolder(t, db, "n1", 1, "Pasta", "/Recipes")

	// No encrypted notes
	has, err := db.FolderHasEncryptedNotes(folderID)
	if err != nil {
		t.Fatalf("FolderHasEncryptedNotes failed: %v", err)
	}
	if has {
		t.Error("Expected no encrypted notes")
	}

	// Encrypt a note
	_, err = db.Exec(`UPDATE notes SET content_encrypted = 1 WHERE id = 'n1'`)
	if err != nil {
		t.Fatalf("Failed to encrypt note: %v", err)
	}

	has, err = db.FolderHasEncryptedNotes(folderID)
	if err != nil {
		t.Fatalf("FolderHasEncryptedNotes failed: %v", err)
	}
	if !has {
		t.Error("Expected encrypted notes detected")
	}
}

func TestGetFolderSharesByFolderID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")

	_, err := db.CreateFolderShare(1, folderID, 2, "viewer")
	if err != nil {
		t.Fatalf("CreateFolderShare failed: %v", err)
	}

	shares, err := db.GetFolderSharesByFolderID(folderID)
	if err != nil {
		t.Fatalf("GetFolderSharesByFolderID failed: %v", err)
	}
	if len(shares) != 1 {
		t.Fatalf("Expected 1 share, got %d", len(shares))
	}
}

// ============================================================================
// Shared Note via GetSharedNote (extended to check folder_shares fallback)
// ============================================================================

func TestGetSharedNote_ViaFolderShare(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	folderID := createTestFolder(t, db, 1, "/Recipes")
	createTestNoteInFolder(t, db, "n1", 1, "Pasta", "/Recipes")

	_, err := db.CreateFolderShare(1, folderID, 2, "editor")
	if err != nil {
		t.Fatalf("CreateFolderShare failed: %v", err)
	}

	sn, err := db.GetSharedNote(2, "n1")
	if err != nil {
		t.Fatalf("GetSharedNote via folder share failed: %v", err)
	}
	if sn == nil {
		t.Fatal("Expected shared note, got nil")
	}
	if sn.Title != "Pasta" {
		t.Errorf("Expected title 'Pasta', got '%s'", sn.Title)
	}
	if sn.ShareRole != "editor" {
		t.Errorf("Expected role 'editor', got '%s'", sn.ShareRole)
	}
}
