package db

import (
	"database/sql"
	"time"
)

// UpdateEncryptionPreferences updates encryption-related preferences
func (db *DB) UpdateEncryptionPreferences(userID int, _ bool, encryptTitles bool) error {
	now := time.Now().Format(time.RFC3339)
	keywordsInt := 0 // forced off: encrypted-note plaintext keyword indexing is deprecated

	encryptTitlesInt := 0
	if encryptTitles {
		encryptTitlesInt = 1
	}

	result, err := db.Exec(`
		UPDATE user_preferences
		SET keywords_enabled = ?, encrypt_titles = ?, updated_at = ?
		WHERE user_id = ?
	`, keywordsInt, encryptTitlesInt, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := rowsAffectedCount(result, "")
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		// User preferences don't exist, create them
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, keywords_enabled, encrypt_titles, created_at, updated_at)
			VALUES (?, 'default-dark', 'split', ?, ?, ?, ?)
		`, userID, keywordsInt, encryptTitlesInt, now, now)
		return err
	}

	return nil
}

// SetRecoveryKey stores the hashed recovery key and salt for a user
func (db *DB) SetRecoveryKey(userID int, recoveryKeyHash string, recoveryKeySalt []byte) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE user_preferences
		SET recovery_key_hash = ?, recovery_key_salt = ?, updated_at = ?
		WHERE user_id = ?
	`, recoveryKeyHash, recoveryKeySalt, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := rowsAffectedCount(result, "")
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		// User preferences don't exist, create them with recovery key
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, recovery_key_hash, recovery_key_salt, created_at, updated_at)
			VALUES (?, 'default-dark', 'split', ?, ?, ?, ?)
		`, userID, recoveryKeyHash, recoveryKeySalt, now, now)
		return err
	}

	return nil
}

// SetRecoveryKeyTx stores recovery key data within an existing transaction.
func (tx *Tx) SetRecoveryKeyTx(userID int, recoveryKeyHash string, recoveryKeySalt []byte) error {
	now := time.Now().Format(time.RFC3339)

	result, err := tx.Exec(`
		UPDATE user_preferences
		SET recovery_key_hash = ?, recovery_key_salt = ?, updated_at = ?
		WHERE user_id = ?
	`, recoveryKeyHash, recoveryKeySalt, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := rowsAffectedCount(result, "")
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		_, err = tx.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, recovery_key_hash, recovery_key_salt, created_at, updated_at)
			VALUES (?, 'default-dark', 'split', ?, ?, ?, ?)
		`, userID, recoveryKeyHash, recoveryKeySalt, now, now)
		return err
	}

	return nil
}

// GetRecoveryKeySalt retrieves the recovery key salt for a user
func (db *DB) GetRecoveryKeySalt(userID int) ([]byte, error) {
	var salt []byte
	err := db.QueryRow(`
		SELECT recovery_key_salt
		FROM user_preferences
		WHERE user_id = ? AND recovery_key_salt IS NOT NULL
	`, userID).Scan(&salt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return salt, nil
}

// InvalidateRecoveryKey clears the recovery key hash and salt for a user.
// This should be called when the user changes their password, as the old recovery key
// is no longer valid (it was derived from the old password).
func (db *DB) InvalidateRecoveryKey(userID int) error {
	now := time.Now().Format(time.RFC3339)

	_, err := db.Exec(`
		UPDATE user_preferences
		SET recovery_key_hash = NULL, recovery_key_salt = NULL, updated_at = ?
		WHERE user_id = ?
	`, now, userID)
	return err
}

// InvalidateRecoveryKeyTx clears recovery key data within an existing transaction.
func (tx *Tx) InvalidateRecoveryKeyTx(userID int) error {
	now := time.Now().Format(time.RFC3339)

	_, err := tx.Exec(`
		UPDATE user_preferences
		SET recovery_key_hash = NULL, recovery_key_salt = NULL, updated_at = ?
		WHERE user_id = ?
	`, now, userID)
	return err
}
