package db

import "time"

// UpdateHomeDashboardLayout sets the persisted home dashboard layout JSON for a user.
// Pass nil to clear the stored layout.
func (db *DB) UpdateHomeDashboardLayout(userID int, layoutJSON *string) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE user_preferences
		SET home_dashboard_layout = ?, updated_at = ?
		WHERE user_id = ?
	`, layoutJSON, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := rowsAffectedCount(result, "")
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, home_dashboard_layout, created_at, updated_at)
			VALUES (?, 'default-dark', 'split', ?, ?, ?)
		`, userID, layoutJSON, now, now)
		return err
	}

	return nil
}
