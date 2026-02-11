package db

import (
	"database/sql"
	"fmt"
)

// ============================================================================
// AI-Enabled Functions (Claude API Integration)
// ============================================================================

// UpdateFolderAIEnabledDefault sets the ai_enabled_default flag for a folder.
// New notes created in this folder will inherit this setting.
func (d *DB) UpdateFolderAIEnabledDefault(userID int, folderID int, enabled bool) error {
	aiEnabled := 0
	if enabled {
		aiEnabled = 1
	}

	result, err := d.Exec(`
		UPDATE folders
		SET ai_enabled_default = ?, updated_at = datetime('now')
		WHERE id = ? AND user_id = ?
	`, aiEnabled, folderID, userID)
	if err != nil {
		return fmt.Errorf("failed to update ai_enabled_default: %w", err)
	}
	return ensureRowsAffectedWithContext(result, "failed to get rows affected")
}

// GetFolderAIEnabledDefault returns the ai_enabled_default for a folder.
// Returns false for root folder (path="/") as root notes are always ai_enabled=false.
func (d *DB) GetFolderAIEnabledDefault(userID int, folderID int) (bool, error) {
	var aiEnabledDefault int
	var path string
	err := d.QueryRow(`
		SELECT ai_enabled_default, path FROM folders
		WHERE id = ? AND user_id = ?
	`, folderID, userID).Scan(&aiEnabledDefault, &path)

	if err == sql.ErrNoRows {
		return false, ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("failed to get ai_enabled_default: %w", err)
	}

	// Root folder always returns false (root notes are never ai_enabled by default)
	if path == "/" {
		return false, nil
	}

	return aiEnabledDefault == 1, nil
}

// GetFolderAIEnabledDefaultByPath returns the ai_enabled_default for a folder by path.
// Returns false for root folder (path="/") as root notes are always ai_enabled=false.
func (d *DB) GetFolderAIEnabledDefaultByPath(userID int, folderPath string) (bool, error) {
	// Root folder: always false
	if folderPath == "/" {
		return false, nil
	}

	var aiEnabledDefault int
	err := d.QueryRow(`
		SELECT ai_enabled_default FROM folders
		WHERE path = ? AND user_id = ?
	`, folderPath, userID).Scan(&aiEnabledDefault)

	if err == sql.ErrNoRows {
		// Folder not found - return false (safe default)
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get ai_enabled_default by path: %w", err)
	}

	return aiEnabledDefault == 1, nil
}
