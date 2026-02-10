package db

import (
	"database/sql"
	"encoding/json"
)

// Feature represents a user-specific feature configuration.
type Feature struct {
	UserID    int             `json:"user_id"`
	Feature   string          `json:"feature"`
	Enabled   bool            `json:"enabled"`
	Settings  json.RawMessage `json:"settings,omitempty"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

// GetUserFeature retrieves a specific feature for a user.
// Returns a disabled feature with default values if not found.
func (db *DB) GetUserFeature(userID int, feature string) (*Feature, error) {
	var f Feature
	var settings sql.NullString

	err := db.QueryRow(`
		SELECT user_id, feature, enabled, settings, created_at, updated_at
		FROM user_features
		WHERE user_id = ? AND feature = ?
	`, userID, feature).Scan(&f.UserID, &f.Feature, &f.Enabled, &settings, &f.CreatedAt, &f.UpdatedAt)

	if err == sql.ErrNoRows {
		// Features enabled by default when not explicitly set
		defaultEnabled := feature == "journal" || feature == "recipe"
		return &Feature{UserID: userID, Feature: feature, Enabled: defaultEnabled}, nil
	}
	if err != nil {
		return nil, err
	}

	if settings.Valid {
		f.Settings = json.RawMessage(settings.String)
	}
	return &f, nil
}

// SetUserFeature enables or disables a feature for a user.
// Uses UPSERT to create or update the feature setting.
func (db *DB) SetUserFeature(userID int, feature string, enabled bool, settings json.RawMessage) error {
	var settingsStr *string
	if len(settings) > 0 {
		s := string(settings)
		settingsStr = &s
	}

	_, err := db.Exec(`
		INSERT INTO user_features (user_id, feature, enabled, settings)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (user_id, feature) DO UPDATE SET
			enabled = excluded.enabled,
			settings = COALESCE(excluded.settings, user_features.settings),
			updated_at = datetime('now')
	`, userID, feature, enabled, settingsStr)
	return err
}

// ListUserFeatures retrieves all feature configurations for a user.
func (db *DB) ListUserFeatures(userID int) ([]Feature, error) {
	rows, err := db.Query(`
		SELECT user_id, feature, enabled, settings, created_at, updated_at
		FROM user_features WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []Feature
	for rows.Next() {
		var f Feature
		var settings sql.NullString
		if err := rows.Scan(&f.UserID, &f.Feature, &f.Enabled, &settings, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		if settings.Valid {
			f.Settings = json.RawMessage(settings.String)
		}
		features = append(features, f)
	}
	return features, rows.Err()
}
