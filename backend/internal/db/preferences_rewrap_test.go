package db

import (
	"database/sql"
	"testing"
)

func TestInvalidateRecoveryKey_Simple(t *testing.T) {
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

	// Set recovery key
	err = db.SetRecoveryKey(userID, "hash", []byte("salt"))
	if err != nil {
		t.Fatalf("Failed to set recovery key: %v", err)
	}

	// Verify it was set
	var hash sql.NullString
	err = db.QueryRow(`SELECT recovery_key_hash FROM user_preferences WHERE user_id = ?`, userID).Scan(&hash)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if !hash.Valid || hash.String != "hash" {
		t.Errorf("Recovery key not set correctly")
	}

	// Invalidate
	err = db.InvalidateRecoveryKey(userID)
	if err != nil {
		t.Fatalf("InvalidateRecoveryKey failed: %v", err)
	}

	// Verify it was cleared
	err = db.QueryRow(`SELECT recovery_key_hash FROM user_preferences WHERE user_id = ?`, userID).Scan(&hash)
	if err != nil {
		t.Fatalf("Failed to query after invalidation: %v", err)
	}
	if hash.Valid {
		t.Errorf("Recovery key should be NULL, got: %s", hash.String)
	}
}
