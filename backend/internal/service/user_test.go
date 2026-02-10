package service

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/xela-io/xelanote/internal/db"
)

// Helper to setup test database and service
func setupUserServiceTest(t *testing.T) (*db.DB, *UserService) {
	testDB, err := db.Open(":memory:", "") // Empty key for tests
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Apply schema and migrations
	if err := testDB.Migrate(); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	userService := NewUserService(testDB)
	return testDB, userService
}

// Helper to create test user for password tests
func createTestUserForPasswordTests(t *testing.T, testDB *db.DB, username, email, password string) int {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	result, err := testDB.Exec(`
		INSERT INTO users (username, email, password_hash, created_at)
		VALUES (?, ?, ?, datetime('now'))
	`, username, email, string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	userID, _ := result.LastInsertId()
	return int(userID)
}

func TestChangePasswordWithDEKRewrap_NoEncryptedNotes(t *testing.T) {
	testDB, userService := setupUserServiceTest(t)
	defer testDB.Close()

	// Create test user
	userID := createTestUserForPasswordTests(t, testDB, "testuser", "test@example.com", "oldpassword123")

	// Test password change without encrypted notes (backwards-compatible)
	err := userService.ChangePasswordWithDEKRewrap(
		userID,
		"oldpassword123",
		"newpassword456",
		nil, // No re-wrapped notes
		nil, // No re-wrapped versions
		"",  // No refresh token
	)

	if err != nil {
		t.Fatalf("ChangePasswordWithDEKRewrap failed for user without encrypted notes: %v", err)
	}

	// Verify password was changed
	user, err := testDB.GetUserByID(userID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("newpassword456"))
	if err != nil {
		t.Errorf("Password was not updated correctly")
	}
}

func TestChangePasswordWithDEKRewrap_WithEncryptedNotes_MissingDEKs(t *testing.T) {
	testDB, userService := setupUserServiceTest(t)
	defer testDB.Close()

	userID := createTestUserForPasswordTests(t, testDB, "testuser", "test@example.com", "oldpassword123")

	// Create encrypted note
	_, err := testDB.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
		                   content_encrypted, wrapped_dek, encryption_version)
		VALUES ('note1', 'Encrypted Note', 'encrypted note', '', '/', ?, datetime('now'), datetime('now'), 1, 'old_dek', 2)
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create encrypted note: %v", err)
	}

	// Test password change WITHOUT providing re-wrapped DEKs (should fail)
	err = userService.ChangePasswordWithDEKRewrap(
		userID,
		"oldpassword123",
		"newpassword456",
		nil, // Missing re-wrapped notes!
		nil,
		"",
	)

	if err == nil {
		t.Fatal("ChangePasswordWithDEKRewrap should fail when encrypted notes exist but no re-wrapped DEKs provided")
	}

	if err.Error() != "DEK re-wrapping required: user has encrypted notes or versions" {
		t.Errorf("Expected 'DEK re-wrapping required' error, got: %v", err)
	}

	// Verify password was NOT changed (operation should have failed early)
	user, _ := testDB.GetUserByID(userID)
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("oldpassword123"))
	if err != nil {
		t.Errorf("Password should not have been changed after failed re-wrapping validation")
	}
}

func TestChangePasswordWithDEKRewrap_WithEncryptedNotes_Success(t *testing.T) {
	testDB, userService := setupUserServiceTest(t)
	defer testDB.Close()

	userID := createTestUserForPasswordTests(t, testDB, "testuser", "test@example.com", "oldpassword123")

	// Create encrypted notes
	_, err := testDB.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
		                   content_encrypted, wrapped_dek, encryption_version)
		VALUES
			('note1', 'Note 1', 'note 1', '', '/', ?, datetime('now'), datetime('now'), 1, 'old_dek_1', 2),
			('note2', 'Note 2', 'note 2', '', '/', ?, datetime('now'), datetime('now'), 1, 'old_dek_2', 2)
	`, userID, userID)
	if err != nil {
		t.Fatalf("Failed to create encrypted notes: %v", err)
	}

	// Create encrypted versions
	_, err = testDB.Exec(`
		INSERT INTO note_versions (note_id, user_id, version, title, content, snapshot_at,
		                           content_encrypted, wrapped_dek, encryption_version)
		VALUES
			('note1', ?, 1, 'V1', '', datetime('now'), 1, 'old_v_dek_1', 2),
			('note1', ?, 2, 'V2', '', datetime('now'), 1, 'old_v_dek_2', 2)
	`, userID, userID)
	if err != nil {
		t.Fatalf("Failed to create encrypted versions: %v", err)
	}

	// Prepare re-wrapped DEKs
	reWrappedNotes := map[string]string{
		"note1": "new_dek_1",
		"note2": "new_dek_2",
	}
	reWrappedVersions := map[string]string{
		"1": "new_v_dek_1",
		"2": "new_v_dek_2",
	}

	// Test password change with re-wrapped DEKs
	err = userService.ChangePasswordWithDEKRewrap(
		userID,
		"oldpassword123",
		"newpassword456",
		reWrappedNotes,
		reWrappedVersions,
		"",
	)

	if err != nil {
		t.Fatalf("ChangePasswordWithDEKRewrap failed: %v", err)
	}

	// Verify password was changed
	user, _ := testDB.GetUserByID(userID)
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("newpassword456"))
	if err != nil {
		t.Errorf("Password was not updated correctly")
	}

	// Verify DEKs were updated
	var wrappedDEK string
	err = testDB.QueryRow(`SELECT wrapped_dek FROM notes WHERE id = 'note1'`).Scan(&wrappedDEK)
	if err != nil {
		t.Fatalf("Failed to query note1: %v", err)
	}
	if wrappedDEK != "new_dek_1" {
		t.Errorf("note1 wrapped_dek not updated: expected 'new_dek_1', got '%s'", wrappedDEK)
	}

	err = testDB.QueryRow(`SELECT wrapped_dek FROM note_versions WHERE id = 1`).Scan(&wrappedDEK)
	if err != nil {
		t.Fatalf("Failed to query version 1: %v", err)
	}
	if wrappedDEK != "new_v_dek_1" {
		t.Errorf("version 1 wrapped_dek not updated: expected 'new_v_dek_1', got '%s'", wrappedDEK)
	}
}

func TestChangePasswordWithDEKRewrap_MissingNoteInMap(t *testing.T) {
	testDB, userService := setupUserServiceTest(t)
	defer testDB.Close()

	userID := createTestUserForPasswordTests(t, testDB, "testuser", "test@example.com", "oldpassword123")

	// Create 2 encrypted notes
	_, err := testDB.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
		                   content_encrypted, wrapped_dek, encryption_version)
		VALUES
			('note1', 'Note 1', 'note 1', '', '/', ?, datetime('now'), datetime('now'), 1, 'old_dek_1', 2),
			('note2', 'Note 2', 'note 2', '', '/', ?, datetime('now'), datetime('now'), 1, 'old_dek_2', 2)
	`, userID, userID)
	if err != nil {
		t.Fatalf("Failed to create encrypted notes: %v", err)
	}

	// Only provide re-wrapped DEK for note1 (missing note2!)
	reWrappedNotes := map[string]string{
		"note1": "new_dek_1",
		// Missing "note2"!
	}

	// Test - should fail validation
	err = userService.ChangePasswordWithDEKRewrap(
		userID,
		"oldpassword123",
		"newpassword456",
		reWrappedNotes,
		nil,
		"",
	)

	if err == nil {
		t.Fatal("ChangePasswordWithDEKRewrap should fail when not all notes are in re-wrapped map")
	}

	if err.Error() != "missing re-wrapped DEK for note note2" {
		t.Errorf("Expected 'missing re-wrapped DEK' error, got: %v", err)
	}

	// Verify password was NOT changed
	user, _ := testDB.GetUserByID(userID)
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("oldpassword123"))
	if err != nil {
		t.Errorf("Password should not have been changed after validation failure")
	}
}

func TestChangePasswordWithDEKRewrap_WrongCurrentPassword(t *testing.T) {
	testDB, userService := setupUserServiceTest(t)
	defer testDB.Close()

	userID := createTestUserForPasswordTests(t, testDB, "testuser", "test@example.com", "oldpassword123")

	// Test with wrong current password
	err := userService.ChangePasswordWithDEKRewrap(
		userID,
		"wrongpassword",
		"newpassword456",
		nil,
		nil,
		"",
	)

	if err == nil {
		t.Fatal("ChangePasswordWithDEKRewrap should fail with wrong current password")
	}

	if err != ErrInvalidPassword {
		t.Errorf("Expected ErrInvalidPassword, got: %v", err)
	}
}

func TestChangePasswordWithDEKRewrap_ShortNewPassword(t *testing.T) {
	testDB, userService := setupUserServiceTest(t)
	defer testDB.Close()

	userID := createTestUserForPasswordTests(t, testDB, "testuser", "test@example.com", "oldpassword123")

	// Test with too short new password
	err := userService.ChangePasswordWithDEKRewrap(
		userID,
		"oldpassword123",
		"short", // Only 5 characters
		nil,
		nil,
		"",
	)

	if err == nil {
		t.Fatal("ChangePasswordWithDEKRewrap should fail with short new password")
	}

	if err != ErrPasswordTooShort {
		t.Errorf("Expected ErrPasswordTooShort, got: %v", err)
	}
}

func TestChangePasswordWithDEKRewrap_RecoveryKeyInvalidated(t *testing.T) {
	testDB, userService := setupUserServiceTest(t)
	defer testDB.Close()

	userID := createTestUserForPasswordTests(t, testDB, "testuser", "test@example.com", "oldpassword123")

	// Set recovery key
	err := testDB.SetRecoveryKey(userID, "recovery_hash", []byte("recovery_salt"))
	if err != nil {
		t.Fatalf("Failed to set recovery key: %v", err)
	}

	// Change password (no encrypted notes, so no re-wrapping)
	err = userService.ChangePasswordWithDEKRewrap(
		userID,
		"oldpassword123",
		"newpassword456",
		nil,
		nil,
		"",
	)

	if err != nil {
		t.Fatalf("ChangePasswordWithDEKRewrap failed: %v", err)
	}

	// Verify recovery key was invalidated
	prefs, err := testDB.GetUserPreferences(userID)
	if err != nil {
		t.Fatalf("Failed to get preferences: %v", err)
	}

	if prefs.RecoveryKeyHash != nil {
		t.Errorf("Recovery key hash should be NULL after password change")
	}
	if prefs.RecoveryKeySalt != nil {
		t.Errorf("Recovery key salt should be NULL after password change")
	}
}

func TestChangePassword_BackwardsCompatibility(t *testing.T) {
	testDB, userService := setupUserServiceTest(t)
	defer testDB.Close()

	userID := createTestUserForPasswordTests(t, testDB, "testuser", "test@example.com", "oldpassword123")

	// Test old ChangePassword function (should delegate to ChangePasswordWithDEKRewrap)
	err := userService.ChangePassword(userID, "oldpassword123", "newpassword456", "")
	if err != nil {
		t.Fatalf("ChangePassword (old API) failed: %v", err)
	}

	// Verify password was changed
	user, _ := testDB.GetUserByID(userID)
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("newpassword456"))
	if err != nil {
		t.Errorf("Password was not updated via old ChangePassword API")
	}
}
