package db

import (
	"testing"
)

// --- Tests for Virtual Root (Migration 025) ---

// TestCreateFolder_TopLevel verifies that top-level folders have parent_id = NULL (virtual root)
func TestCreateFolder_TopLevel(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create a top-level folder
	folder, err := db.CreateFolder(userID, "/Projects", nil)
	if err != nil {
		t.Fatalf("Failed to create top-level folder: %v", err)
	}

	// Verify parent_id is NULL (virtual root)
	if folder.ParentID != nil {
		t.Errorf("Expected parent_id to be nil (virtual root), got %d", *folder.ParentID)
	}

	// Verify path and name
	if folder.Path != "/Projects" {
		t.Errorf("Expected path '/Projects', got '%s'", folder.Path)
	}
	if folder.Name != "Projects" {
		t.Errorf("Expected name 'Projects', got '%s'", folder.Name)
	}
}

// TestCreateFolder_Nested verifies that nested folders have correct parent_id
func TestCreateFolder_Nested(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create parent folder first
	parent, err := db.CreateFolder(userID, "/Projects", nil)
	if err != nil {
		t.Fatalf("Failed to create parent folder: %v", err)
	}

	// Create nested folder with parent_id
	child, err := db.CreateFolder(userID, "/Projects/Backend", &parent.ID)
	if err != nil {
		t.Fatalf("Failed to create nested folder: %v", err)
	}

	// Verify parent_id points to parent folder
	if child.ParentID == nil {
		t.Error("Expected parent_id to be set, got nil")
	} else if *child.ParentID != parent.ID {
		t.Errorf("Expected parent_id %d, got %d", parent.ID, *child.ParentID)
	}

	// Verify path and name
	if child.Path != "/Projects/Backend" {
		t.Errorf("Expected path '/Projects/Backend', got '%s'", child.Path)
	}
	if child.Name != "Backend" {
		t.Errorf("Expected name 'Backend', got '%s'", child.Name)
	}
}

// TestCreateFolder_RootForbidden verifies that creating "/" folder is rejected
func TestCreateFolder_RootForbidden(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Attempt to create root folder (should fail)
	_, err := db.CreateFolder(userID, "/", nil)
	if err == nil {
		t.Fatal("Expected error when creating root folder, got nil")
	}

	// Verify error message
	expectedMsg := "cannot create root folder - root is virtual"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestGetAllFolders_NoGlobalRoot verifies that no folder with id=1 and path="/" exists
func TestGetAllFolders_NoGlobalRoot(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create some folders
	_, err := db.CreateFolder(userID, "/Folder1", nil)
	if err != nil {
		t.Fatalf("Failed to create folder1: %v", err)
	}
	_, err = db.CreateFolder(userID, "/Folder2", nil)
	if err != nil {
		t.Fatalf("Failed to create folder2: %v", err)
	}

	// Get all folders
	folders, err := db.GetAllFolders(userID)
	if err != nil {
		t.Fatalf("Failed to get all folders: %v", err)
	}

	// Verify no root folder exists
	for _, f := range folders {
		if f.Path == "/" {
			t.Errorf("Found root folder with path='/', this should not exist (virtual root)")
		}
		if f.ID == 1 && f.Name == "Root" {
			t.Errorf("Found hardcoded Root folder with id=1, this should not exist")
		}
	}

	// Verify we got our 2 folders
	if len(folders) != 2 {
		t.Errorf("Expected 2 folders, got %d", len(folders))
	}
}

// TestGetAllFolders_UserIsolation verifies that users only see their own folders
func TestGetAllFolders_UserIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create two users
	user1ID := 1
	user2ID := 2
	createTestUser(t, db, user1ID)
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (?, 'user2', 'user2@example.com', 'hash', datetime('now'))
	`, user2ID)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Create folders for user1
	_, err = db.CreateFolder(user1ID, "/User1Folder", nil)
	if err != nil {
		t.Fatalf("Failed to create folder for user1: %v", err)
	}

	// Create folders for user2
	_, err = db.CreateFolder(user2ID, "/User2Folder", nil)
	if err != nil {
		t.Fatalf("Failed to create folder for user2: %v", err)
	}

	// Get folders for user1
	folders1, err := db.GetAllFolders(user1ID)
	if err != nil {
		t.Fatalf("Failed to get folders for user1: %v", err)
	}

	// Verify user1 only sees their folder
	if len(folders1) != 1 {
		t.Errorf("User1 should see 1 folder, got %d", len(folders1))
	}
	if folders1[0].Path != "/User1Folder" {
		t.Errorf("User1 should see '/User1Folder', got '%s'", folders1[0].Path)
	}

	// Get folders for user2
	folders2, err := db.GetAllFolders(user2ID)
	if err != nil {
		t.Fatalf("Failed to get folders for user2: %v", err)
	}

	// Verify user2 only sees their folder
	if len(folders2) != 1 {
		t.Errorf("User2 should see 1 folder, got %d", len(folders2))
	}
	if folders2[0].Path != "/User2Folder" {
		t.Errorf("User2 should see '/User2Folder', got '%s'", folders2[0].Path)
	}
}

// TestMoveFolder_ToVirtualRoot verifies moving a folder to top-level (virtual root)
func TestMoveFolder_ToVirtualRoot(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create parent folder
	parent, err := db.CreateFolder(userID, "/Parent", nil)
	if err != nil {
		t.Fatalf("Failed to create parent folder: %v", err)
	}

	// Create nested folder
	child, err := db.CreateFolder(userID, "/Parent/Child", &parent.ID)
	if err != nil {
		t.Fatalf("Failed to create child folder: %v", err)
	}

	// Verify child has parent_id set
	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Fatalf("Child should have parent_id=%d before move", parent.ID)
	}

	// Move child to virtual root (top level)
	err = db.MoveFolder(userID, child.ID, "/")
	if err != nil {
		t.Fatalf("Failed to move folder to root: %v", err)
	}

	// Get updated folder
	movedChild, err := db.GetFolderByID(userID, child.ID)
	if err != nil {
		t.Fatalf("Failed to get moved folder: %v", err)
	}

	// Verify parent_id is now NULL (virtual root)
	if movedChild.ParentID != nil {
		t.Errorf("Expected parent_id to be nil after move to root, got %d", *movedChild.ParentID)
	}

	// Verify path updated
	if movedChild.Path != "/Child" {
		t.Errorf("Expected path '/Child' after move, got '%s'", movedChild.Path)
	}
}

// TestMoveFolder_FromVirtualRootToNested verifies moving a top-level folder into another folder
func TestMoveFolder_FromVirtualRootToNested(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create two top-level folders
	folder1, err := db.CreateFolder(userID, "/Folder1", nil)
	if err != nil {
		t.Fatalf("Failed to create folder1: %v", err)
	}
	folder2, err := db.CreateFolder(userID, "/Folder2", nil)
	if err != nil {
		t.Fatalf("Failed to create folder2: %v", err)
	}

	// Both should have parent_id = NULL
	if folder1.ParentID != nil {
		t.Fatalf("Folder1 should have parent_id=nil, got %d", *folder1.ParentID)
	}
	if folder2.ParentID != nil {
		t.Fatalf("Folder2 should have parent_id=nil, got %d", *folder2.ParentID)
	}

	// Move folder2 into folder1
	err = db.MoveFolder(userID, folder2.ID, "/Folder1")
	if err != nil {
		t.Fatalf("Failed to move folder2 into folder1: %v", err)
	}

	// Get updated folder2
	movedFolder2, err := db.GetFolderByID(userID, folder2.ID)
	if err != nil {
		t.Fatalf("Failed to get moved folder2: %v", err)
	}

	// Verify parent_id is now folder1.ID
	if movedFolder2.ParentID == nil {
		t.Error("Expected parent_id to be set after move, got nil")
	} else if *movedFolder2.ParentID != folder1.ID {
		t.Errorf("Expected parent_id=%d, got %d", folder1.ID, *movedFolder2.ParentID)
	}

	// Verify path updated
	if movedFolder2.Path != "/Folder1/Folder2" {
		t.Errorf("Expected path '/Folder1/Folder2', got '%s'", movedFolder2.Path)
	}
}

// TestMoveFolder_WithSubfolders verifies that moving a folder also updates subfolder paths
func TestMoveFolder_WithSubfolders(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create hierarchy: /A, /A/B, /A/B/C
	folderA, err := db.CreateFolder(userID, "/A", nil)
	if err != nil {
		t.Fatalf("Failed to create /A: %v", err)
	}
	folderB, err := db.CreateFolder(userID, "/A/B", &folderA.ID)
	if err != nil {
		t.Fatalf("Failed to create /A/B: %v", err)
	}
	_, err = db.CreateFolder(userID, "/A/B/C", &folderB.ID)
	if err != nil {
		t.Fatalf("Failed to create /A/B/C: %v", err)
	}

	// Create target folder /Target
	_, err = db.CreateFolder(userID, "/Target", nil)
	if err != nil {
		t.Fatalf("Failed to create /Target: %v", err)
	}

	// Move /A into /Target
	err = db.MoveFolder(userID, folderA.ID, "/Target")
	if err != nil {
		t.Fatalf("Failed to move /A to /Target: %v", err)
	}

	// Verify all paths updated
	folders, err := db.GetAllFolders(userID)
	if err != nil {
		t.Fatalf("Failed to get folders: %v", err)
	}

	expectedPaths := map[string]bool{
		"/Target":       true,
		"/Target/A":     true,
		"/Target/A/B":   true,
		"/Target/A/B/C": true,
	}

	for _, f := range folders {
		if !expectedPaths[f.Path] {
			t.Errorf("Unexpected folder path: %s", f.Path)
		}
		delete(expectedPaths, f.Path)
	}

	for path := range expectedPaths {
		t.Errorf("Expected folder path not found: %s", path)
	}
}

// TestDeleteFolder_UserIsolation verifies that a user cannot delete another user's folder
func TestDeleteFolder_UserIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create two users
	user1ID := 1
	user2ID := 2
	createTestUser(t, db, user1ID)
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (?, 'user2', 'user2@example.com', 'hash', datetime('now'))
	`, user2ID)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Create folder for user1
	folder, err := db.CreateFolder(user1ID, "/User1Folder", nil)
	if err != nil {
		t.Fatalf("Failed to create folder for user1: %v", err)
	}

	// Try to delete user1's folder as user2 (should fail)
	err = db.DeleteFolder(user2ID, folder.ID)
	if err == nil {
		t.Error("Expected error when deleting another user's folder, got nil")
	}
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}

	// Verify folder still exists for user1
	existing, err := db.GetFolderByID(user1ID, folder.ID)
	if err != nil {
		t.Errorf("Folder should still exist for user1: %v", err)
	}
	if existing == nil {
		t.Error("Folder should still exist for user1")
	}
}

// TestRenameFolder_TopLevel verifies renaming a top-level folder
func TestRenameFolder_TopLevel(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create top-level folder
	folder, err := db.CreateFolder(userID, "/OldName", nil)
	if err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}

	// Rename folder
	err = db.RenameFolder(userID, folder.ID, "NewName")
	if err != nil {
		t.Fatalf("Failed to rename folder: %v", err)
	}

	// Get renamed folder
	renamed, err := db.GetFolderByID(userID, folder.ID)
	if err != nil {
		t.Fatalf("Failed to get renamed folder: %v", err)
	}

	// Verify name and path updated
	if renamed.Name != "NewName" {
		t.Errorf("Expected name 'NewName', got '%s'", renamed.Name)
	}
	if renamed.Path != "/NewName" {
		t.Errorf("Expected path '/NewName', got '%s'", renamed.Path)
	}

	// Verify parent_id still NULL (top-level)
	if renamed.ParentID != nil {
		t.Errorf("Expected parent_id to remain nil, got %d", *renamed.ParentID)
	}
}

// TestFolderColor verifies setting and clearing folder color
func TestFolderColor(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create folder
	folder, err := db.CreateFolder(userID, "/ColorTest", nil)
	if err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}

	// Verify no color initially
	if folder.Color != nil {
		t.Errorf("Expected no color initially, got '%s'", *folder.Color)
	}

	// Set color
	color := "#FF5733"
	err = db.UpdateFolderColor(userID, folder.ID, &color)
	if err != nil {
		t.Fatalf("Failed to set color: %v", err)
	}

	// Verify color set
	updated, err := db.GetFolderByID(userID, folder.ID)
	if err != nil {
		t.Fatalf("Failed to get folder: %v", err)
	}
	if updated.Color == nil || *updated.Color != color {
		t.Errorf("Expected color '%s', got '%v'", color, updated.Color)
	}

	// Clear color
	err = db.UpdateFolderColor(userID, folder.ID, nil)
	if err != nil {
		t.Fatalf("Failed to clear color: %v", err)
	}

	// Verify color cleared
	cleared, err := db.GetFolderByID(userID, folder.ID)
	if err != nil {
		t.Fatalf("Failed to get folder: %v", err)
	}
	if cleared.Color != nil {
		t.Errorf("Expected color to be cleared, got '%s'", *cleared.Color)
	}
}

// TestFolderColor_InvalidFormat verifies that invalid color formats are rejected
func TestFolderColor_InvalidFormat(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create folder
	folder, err := db.CreateFolder(userID, "/ColorTest", nil)
	if err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}

	// Try invalid color formats
	invalidColors := []string{
		"red",        // named color
		"#FFF",       // short hex
		"#GGGGGG",    // invalid hex chars
		"rgb(1,2,3)", // rgb format
		"FF5733",     // missing #
	}

	for _, color := range invalidColors {
		c := color
		err = db.UpdateFolderColor(userID, folder.ID, &c)
		if err == nil {
			t.Errorf("Expected error for invalid color '%s', got nil", color)
		}
	}
}

// TestGetFolderByPath_UserIsolation verifies that users cannot access folders by path from other users
func TestGetFolderByPath_UserIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create two users
	user1ID := 1
	user2ID := 2
	createTestUser(t, db, user1ID)
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (?, 'user2', 'user2@example.com', 'hash', datetime('now'))
	`, user2ID)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Create folder with same path for both users
	_, err = db.CreateFolder(user1ID, "/SharedPath", nil)
	if err != nil {
		t.Fatalf("Failed to create folder for user1: %v", err)
	}
	_, err = db.CreateFolder(user2ID, "/SharedPath", nil)
	if err != nil {
		t.Fatalf("Failed to create folder for user2: %v", err)
	}

	// Get folder by path for each user
	folder1, err := db.GetFolderByPath(user1ID, "/SharedPath")
	if err != nil {
		t.Fatalf("Failed to get folder for user1: %v", err)
	}
	folder2, err := db.GetFolderByPath(user2ID, "/SharedPath")
	if err != nil {
		t.Fatalf("Failed to get folder for user2: %v", err)
	}

	// Verify different folder IDs (separate folders)
	if folder1.ID == folder2.ID {
		t.Error("Users should have separate folders with same path")
	}
}
