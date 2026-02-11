package db

import (
	"database/sql"
	"time"
)

// --- Claude API Key Management (BYOK) ---

// SetClaudeAPIKey stores the encrypted Claude API key for a user.
// The key should be encrypted with AES-256-GCM before calling this function.
func (db *DB) SetClaudeAPIKey(userID int, encryptedKey string) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE user_preferences
		SET encrypted_claude_api_key = ?, claude_api_key_updated_at = ?, updated_at = ?
		WHERE user_id = ?
	`, encryptedKey, now, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := rowsAffectedCount(result, "")
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		// User preferences don't exist, create them with the API key
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, encrypted_claude_api_key, claude_api_key_updated_at, created_at, updated_at)
			VALUES (?, 'default-dark', 'split', ?, ?, ?, ?)
		`, userID, encryptedKey, now, now, now)
		return err
	}

	return nil
}

// GetClaudeAPIKey retrieves the encrypted Claude API key for a user.
// Returns ErrNotFound if no key is stored.
func (db *DB) GetClaudeAPIKey(userID int) (string, error) {
	var encryptedKey sql.NullString
	err := db.QueryRow(`
		SELECT encrypted_claude_api_key
		FROM user_preferences
		WHERE user_id = ?
	`, userID).Scan(&encryptedKey)

	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if !encryptedKey.Valid || encryptedKey.String == "" {
		return "", ErrNotFound
	}

	return encryptedKey.String, nil
}

// DeleteClaudeAPIKey removes the Claude API key for a user.
func (db *DB) DeleteClaudeAPIKey(userID int) error {
	now := time.Now().Format(time.RFC3339)

	_, err := db.Exec(`
		UPDATE user_preferences
		SET encrypted_claude_api_key = NULL, claude_api_key_updated_at = NULL, updated_at = ?
		WHERE user_id = ?
	`, now, userID)
	return err
}

// HasClaudeAPIKey checks if a user has a Claude API key stored.
func (db *DB) HasClaudeAPIKey(userID int) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM user_preferences
		WHERE user_id = ? AND encrypted_claude_api_key IS NOT NULL AND encrypted_claude_api_key != ''
	`, userID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetClaudeAPIKeyUpdatedAt retrieves the timestamp when the Claude API key was last updated.
// Returns nil if no key is stored.
func (db *DB) GetClaudeAPIKeyUpdatedAt(userID int) (*string, error) {
	var updatedAt sql.NullString
	err := db.QueryRow(`
		SELECT claude_api_key_updated_at
		FROM user_preferences
		WHERE user_id = ?
	`, userID).Scan(&updatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !updatedAt.Valid {
		return nil, nil
	}

	return &updatedAt.String, nil
}

// --- Gemini API Key Management (BYOK) ---

// SetGeminiAPIKey stores the encrypted Gemini API key for a user.
// The key should be encrypted with AES-256-GCM before calling this function.
func (db *DB) SetGeminiAPIKey(userID int, encryptedKey string) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE user_preferences
		SET encrypted_gemini_api_key = ?, gemini_api_key_updated_at = ?, updated_at = ?
		WHERE user_id = ?
	`, encryptedKey, now, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := rowsAffectedCount(result, "")
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		// User preferences don't exist, create them with the API key
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, encrypted_gemini_api_key, gemini_api_key_updated_at, created_at, updated_at)
			VALUES (?, 'default-dark', 'split', ?, ?, ?, ?)
		`, userID, encryptedKey, now, now, now)
		return err
	}

	return nil
}

// GetGeminiAPIKey retrieves the encrypted Gemini API key for a user.
// Returns ErrNotFound if no key is stored.
func (db *DB) GetGeminiAPIKey(userID int) (string, error) {
	var encryptedKey sql.NullString
	err := db.QueryRow(`
		SELECT encrypted_gemini_api_key
		FROM user_preferences
		WHERE user_id = ?
	`, userID).Scan(&encryptedKey)

	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if !encryptedKey.Valid || encryptedKey.String == "" {
		return "", ErrNotFound
	}

	return encryptedKey.String, nil
}

// DeleteGeminiAPIKey removes the Gemini API key for a user.
func (db *DB) DeleteGeminiAPIKey(userID int) error {
	now := time.Now().Format(time.RFC3339)

	_, err := db.Exec(`
		UPDATE user_preferences
		SET encrypted_gemini_api_key = NULL, gemini_api_key_updated_at = NULL, updated_at = ?
		WHERE user_id = ?
	`, now, userID)
	return err
}

// HasGeminiAPIKey checks if a user has a Gemini API key stored.
func (db *DB) HasGeminiAPIKey(userID int) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM user_preferences
		WHERE user_id = ? AND encrypted_gemini_api_key IS NOT NULL AND encrypted_gemini_api_key != ''
	`, userID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetGeminiAPIKeyUpdatedAt retrieves the timestamp when the Gemini API key was last updated.
// Returns nil if no key is stored.
func (db *DB) GetGeminiAPIKeyUpdatedAt(userID int) (*string, error) {
	var updatedAt sql.NullString
	err := db.QueryRow(`
		SELECT gemini_api_key_updated_at
		FROM user_preferences
		WHERE user_id = ?
	`, userID).Scan(&updatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !updatedAt.Valid {
		return nil, nil
	}

	return &updatedAt.String, nil
}
