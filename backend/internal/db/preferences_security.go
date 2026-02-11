package db

import "time"

// UpdateSecurityPreferences updates security-related preferences (security_level and auto_lock_timeout)
func (db *DB) UpdateSecurityPreferences(userID int, securityLevel string, autoLockTimeout int) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE user_preferences
		SET security_level = ?, auto_lock_timeout = ?, updated_at = ?
		WHERE user_id = ?
	`, securityLevel, autoLockTimeout, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		// User preferences don't exist, create them
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, security_level, auto_lock_timeout, created_at, updated_at)
			VALUES (?, 'default-dark', 'split', ?, ?, ?, ?)
		`, userID, securityLevel, autoLockTimeout, now, now)
		return err
	}

	return nil
}
