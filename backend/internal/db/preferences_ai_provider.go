package db

import "time"

// GetActiveAIProvider returns the currently selected AI provider for a user.
// Returns "auto" when no preference row exists yet.
func (db *DB) GetActiveAIProvider(userID int) (string, error) {
	prefs, err := db.GetUserPreferences(userID)
	if err == ErrNotFound {
		return "auto", nil
	}
	if err != nil {
		return "", err
	}
	if prefs.ActiveAIProvider == "" {
		return "auto", nil
	}
	return prefs.ActiveAIProvider, nil
}

// SetActiveAIProvider sets the preferred AI provider for a user.
func (db *DB) SetActiveAIProvider(userID int, provider string) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE user_preferences
		SET active_ai_provider = ?, updated_at = ?
		WHERE user_id = ?
	`, provider, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := rowsAffectedCount(result, "")
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, active_ai_provider, created_at, updated_at)
			VALUES (?, 'default-dark', 'split', ?, ?, ?)
		`, userID, provider, now, now)
		return err
	}

	return nil
}
