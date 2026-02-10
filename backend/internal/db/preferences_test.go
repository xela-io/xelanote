package db

import (
	"database/sql"
	"testing"
)

func TestInvalidateRecoveryKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create user preferences with recovery key
	recoveryKeyHash := "hashed_recovery_key"
	recoveryKeySalt := []byte("salt_bytes_here")

	err := db.SetRecoveryKey(userID, recoveryKeyHash, recoveryKeySalt)
	if err != nil {
		t.Fatalf("Failed to set recovery key: %v", err)
	}

	// Verify recovery key was set
	var hash sql.NullString
	var salt []byte
	err = db.QueryRow(`
		SELECT recovery_key_hash, recovery_key_salt
		FROM user_preferences
		WHERE user_id = ?
	`, userID).Scan(&hash, &salt)
	if err != nil {
		t.Fatalf("Failed to query recovery key: %v", err)
	}

	if !hash.Valid || hash.String != recoveryKeyHash {
		t.Errorf("Recovery key hash not set correctly")
	}
	if len(salt) == 0 {
		t.Errorf("Recovery key salt not set correctly")
	}

	// Test InvalidateRecoveryKey
	err = db.InvalidateRecoveryKey(userID)
	if err != nil {
		t.Fatalf("InvalidateRecoveryKey failed: %v", err)
	}

	// Verify recovery key was cleared
	err = db.QueryRow(`
		SELECT recovery_key_hash, recovery_key_salt
		FROM user_preferences
		WHERE user_id = ?
	`, userID).Scan(&hash, &salt)
	if err != nil {
		t.Fatalf("Failed to query recovery key after invalidation: %v", err)
	}

	if hash.Valid {
		t.Errorf("Recovery key hash should be NULL after invalidation, got: %s", hash.String)
	}
	if salt != nil {
		t.Errorf("Recovery key salt should be NULL after invalidation")
	}
}

func TestInvalidateRecoveryKey_NoPreferences(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 999
	createTestUser(t, db, userID) // User with no preferences

	// Test InvalidateRecoveryKey on non-existent preferences (should not error)
	err := db.InvalidateRecoveryKey(userID)
	if err != nil {
		t.Errorf("InvalidateRecoveryKey should not fail for non-existent user preferences: %v", err)
	}
}

func TestInvalidateRecoveryKey_AlreadyNull(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create user preferences without recovery key
	_, err := db.Exec(`
		INSERT INTO user_preferences (user_id, theme, editor_mode, created_at, updated_at)
		VALUES (?, 'default-dark', 'split', datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create preferences: %v", err)
	}

	// Test InvalidateRecoveryKey when already NULL (should succeed)
	err = db.InvalidateRecoveryKey(userID)
	if err != nil {
		t.Errorf("InvalidateRecoveryKey should not fail when recovery key is already NULL: %v", err)
	}

	// Verify still NULL
	var hash sql.NullString
	err = db.QueryRow(`SELECT recovery_key_hash FROM user_preferences WHERE user_id = ?`, userID).Scan(&hash)
	if err != nil {
		t.Fatalf("Failed to query recovery key: %v", err)
	}
	if hash.Valid {
		t.Errorf("Recovery key hash should still be NULL")
	}
}
