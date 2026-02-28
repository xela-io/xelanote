package db

import (
	"database/sql"
	"time"
)

// GetUserPreferences retrieves preferences for a user
// Returns nil if no preferences exist (user should call GetOrCreateUserPreferences)
func (db *DB) GetUserPreferences(userID int) (*UserPreferences, error) {
	var prefs UserPreferences
	var keywordsEnabled, encryptTitles int
	err := db.QueryRow(`
		SELECT id, user_id, theme, editor_mode,
		       COALESCE(keywords_enabled, 0),
		       COALESCE(encrypt_titles, 0),
		       COALESCE(security_level, 'balanced'),
		       COALESCE(auto_lock_timeout, 15),
		       COALESCE(active_ai_provider, 'auto'),
		       COALESCE(claude_model, ''),
		       COALESCE(gemini_model, ''),
		       COALESCE(openai_model, ''),
		       COALESCE(dietary_preference, 'none'),
		       home_dashboard_layout,
		       open_tabs,
		       recovery_key_hash, recovery_key_salt,
		       created_at, updated_at
		FROM user_preferences
		WHERE user_id = ?
	`, userID).Scan(
		&prefs.ID, &prefs.UserID, &prefs.Theme, &prefs.EditorMode,
		&keywordsEnabled, &encryptTitles,
		&prefs.SecurityLevel, &prefs.AutoLockTimeout, &prefs.ActiveAIProvider,
		&prefs.ClaudeModel, &prefs.GeminiModel, &prefs.OpenAIModel,
		&prefs.DietaryPreference, &prefs.HomeDashboardLayout,
		&prefs.OpenTabs,
		&prefs.RecoveryKeyHash, &prefs.RecoveryKeySalt,
		&prefs.CreatedAt, &prefs.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	prefs.KeywordsEnabled = keywordsEnabled == 1
	prefs.EncryptTitles = encryptTitles == 1

	return &prefs, nil
}

// GetOrCreateUserPreferences retrieves preferences for a user, creating defaults if not exist
// Returns preferences and a boolean indicating if the row was newly created
func (db *DB) GetOrCreateUserPreferences(userID int) (*UserPreferences, bool, error) {
	prefs, err := db.GetUserPreferences(userID)
	if err == nil {
		return prefs, false, nil
	}
	if err != ErrNotFound {
		return nil, false, err
	}

	// Create default preferences
	now := time.Now().Format(time.RFC3339)
	result, err := db.Exec(`
		INSERT INTO user_preferences (user_id, theme, editor_mode, keywords_enabled, encrypt_titles, created_at, updated_at)
		VALUES (?, 'default-dark', 'split', 0, 0, ?, ?)
	`, userID, now, now)
	if err != nil {
		return nil, false, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, false, err
	}
	prefsID, err := validateLastInsertID(id, "user preferences id")
	if err != nil {
		return nil, false, err
	}
	return &UserPreferences{
		ID:                prefsID,
		UserID:            userID,
		Theme:             "default-dark",
		EditorMode:        "split",
		KeywordsEnabled:   false,
		EncryptTitles:     false,
		SecurityLevel:     "balanced",
		AutoLockTimeout:   15,
		ActiveAIProvider:  "auto",
		ClaudeModel:       "",
		GeminiModel:       "",
		OpenAIModel:       "",
		DietaryPreference: "none",
		CreatedAt:         now,
		UpdatedAt:         now,
	}, true, nil
}

// UpsertUserPreferences updates or creates user preferences
func (db *DB) UpsertUserPreferences(userID int, theme, editorMode string) (*UserPreferences, error) {
	now := time.Now().Format(time.RFC3339)

	// Try to update first
	result, err := db.Exec(`
		UPDATE user_preferences
		SET theme = ?, editor_mode = ?, updated_at = ?
		WHERE user_id = ?
	`, theme, editorMode, now, userID)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := rowsAffectedCount(result, "")
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		// Insert new row
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, keywords_enabled, encrypt_titles, created_at, updated_at)
			VALUES (?, ?, ?, 0, 0, ?, ?)
		`, userID, theme, editorMode, now, now)
		if err != nil {
			return nil, err
		}
	}

	// Fetch and return the updated preferences
	return db.GetUserPreferences(userID)
}
