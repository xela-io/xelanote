package db

import (
	"database/sql"
	"strconv"
)

// SystemSetting represents a system setting key-value pair
type SystemSetting struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt string `json:"updated_at"`
}

// GetSetting retrieves a single setting by key
func (db *DB) GetSetting(key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM system_settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// GetAllSettings retrieves all system settings
func (db *DB) GetAllSettings() (map[string]string, error) {
	rows, err := db.Query("SELECT key, value FROM system_settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}

	return settings, nil
}

// SetSetting sets a setting value (inserts or updates)
func (db *DB) SetSetting(key, value string) error {
	_, err := db.Exec(`
		INSERT INTO system_settings (key, value, updated_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = datetime('now')
	`, key, value)
	return err
}

// SetSettings sets multiple settings at once
func (db *DB) SetSettings(settings map[string]string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO system_settings (key, value, updated_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = datetime('now')
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for key, value := range settings {
		if _, err := stmt.Exec(key, value); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// IsRegistrationEnabled checks if registration is enabled
func (db *DB) IsRegistrationEnabled() (bool, error) {
	value, err := db.GetSetting("registration_enabled")
	if err == ErrNotFound {
		return true, nil // Default to enabled
	}
	if err != nil {
		return false, err
	}
	return value == "true", nil
}

// IsMaintenanceMode checks if maintenance mode is enabled
func (db *DB) IsMaintenanceMode() (bool, error) {
	value, err := db.GetSetting("maintenance_mode")
	if err == ErrNotFound {
		return false, nil // Default to disabled
	}
	if err != nil {
		return false, err
	}
	return value == "true", nil
}

// GetMaxNotesPerUser returns the maximum number of notes per user (0 = unlimited)
func (db *DB) GetMaxNotesPerUser() (int, error) {
	value, err := db.GetSetting("max_notes_per_user")
	if err == ErrNotFound {
		return 0, nil // Default to unlimited
	}
	if err != nil {
		return 0, err
	}

	limit, err := strconv.Atoi(value)
	if err != nil {
		return 0, nil // Invalid value, treat as unlimited
	}
	return limit, nil
}

// GetMaxStorageMBPerUser returns the maximum storage per user in MB (0 = unlimited)
func (db *DB) GetMaxStorageMBPerUser() (int, error) {
	value, err := db.GetSetting("max_storage_mb_per_user")
	if err == ErrNotFound {
		return 0, nil // Default to unlimited
	}
	if err != nil {
		return 0, err
	}

	limit, err := strconv.Atoi(value)
	if err != nil {
		return 0, nil // Invalid value, treat as unlimited
	}
	return limit, nil
}

// GetActivityRetentionDays returns the activity log retention period in days (0 = unlimited)
func (db *DB) GetActivityRetentionDays() (int, error) {
	value, err := db.GetSetting("activity_retention_days")
	if err == ErrNotFound {
		return 90, nil // Default to 90 days
	}
	if err != nil {
		return 0, err
	}

	days, err := strconv.Atoi(value)
	if err != nil {
		return 90, nil // Invalid value, use default
	}
	return days, nil
}

// DeleteSetting removes a setting
func (db *DB) DeleteSetting(key string) error {
	_, err := db.Exec("DELETE FROM system_settings WHERE key = ?", key)
	return err
}
