package db

import "time"

// GetDietaryPreference returns the dietary preference for a user.
// Returns "none" when no preference row exists yet.
func (db *DB) GetDietaryPreference(userID int) (string, error) {
	prefs, err := db.GetUserPreferences(userID)
	if err == ErrNotFound {
		return "none", nil
	}
	if err != nil {
		return "", err
	}
	if prefs.DietaryPreference == "" {
		return "none", nil
	}
	return prefs.DietaryPreference, nil
}

// SetDietaryPreference sets the dietary preference for a user.
func (db *DB) SetDietaryPreference(userID int, preference string) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE user_preferences
		SET dietary_preference = ?, updated_at = ?
		WHERE user_id = ?
	`, preference, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := rowsAffectedCount(result, "")
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, dietary_preference, created_at, updated_at)
			VALUES (?, 'default-dark', 'split', ?, ?, ?)
		`, userID, preference, now, now)
		return err
	}

	return nil
}
