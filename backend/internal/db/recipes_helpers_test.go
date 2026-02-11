package db

import "testing"

// Helper for creating a second test user with a unique username
func createTestUserWithName(t *testing.T, db *DB, userID int, username string) {
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (?, ?, ?, 'hash', datetime('now'))
	`, userID, username, username+"@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user %s: %v", username, err)
	}
}
