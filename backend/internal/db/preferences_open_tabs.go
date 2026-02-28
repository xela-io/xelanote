package db

import "time"

// UpdateOpenTabs sets the persisted open tabs JSON for a user.
// Pass nil to clear the stored tabs.
func (db *DB) UpdateOpenTabs(userID int, tabsJSON *string) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE user_preferences
		SET open_tabs = ?, updated_at = ?
		WHERE user_id = ?
	`, tabsJSON, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := rowsAffectedCount(result, "")
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, open_tabs, created_at, updated_at)
			VALUES (?, 'default-dark', 'split', ?, ?, ?)
		`, userID, tabsJSON, now, now)
		return err
	}

	return nil
}
