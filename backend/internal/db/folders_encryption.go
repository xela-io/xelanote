package db

import (
	"database/sql"
	"fmt"
)

// ============================================================================
// Encryption Default Functions
// ============================================================================

// UpdateFolderEncryptionDefault sets the encryption_default flag for a folder.
// New notes created in this folder will inherit this setting.
func (d *DB) UpdateFolderEncryptionDefault(userID int, folderID int, encrypted bool) error {
	val := 0
	if encrypted {
		val = 1
	}

	result, err := d.Exec(`
		UPDATE folders
		SET encryption_default = ?, updated_at = datetime('now')
		WHERE id = ? AND user_id = ?
	`, val, folderID, userID)
	if err != nil {
		return fmt.Errorf("failed to update encryption_default: %w", err)
	}
	return ensureRowsAffectedWithContext(result, "failed to get rows affected")
}

// GetFolderEncryptionDefault returns the encryption_default for a folder.
func (d *DB) GetFolderEncryptionDefault(userID int, folderID int) (bool, error) {
	var encryptionDefault int
	err := d.QueryRow(`
		SELECT encryption_default FROM folders
		WHERE id = ? AND user_id = ?
	`, folderID, userID).Scan(&encryptionDefault)

	if err == sql.ErrNoRows {
		return true, ErrNotFound
	}
	if err != nil {
		return true, fmt.Errorf("failed to get encryption_default: %w", err)
	}

	return encryptionDefault == 1, nil
}

// GetFolderEncryptionDefaultByPath returns the encryption_default for a folder by path.
// Returns true (encrypted) as safe default if folder not found.
func (d *DB) GetFolderEncryptionDefaultByPath(userID int, folderPath string) (bool, error) {
	var encryptionDefault int
	err := d.QueryRow(`
		SELECT encryption_default FROM folders
		WHERE path = ? AND user_id = ?
	`, folderPath, userID).Scan(&encryptionDefault)

	if err == sql.ErrNoRows {
		// Folder not found - return true (encrypted, safe default)
		return true, nil
	}
	if err != nil {
		return true, fmt.Errorf("failed to get encryption_default by path: %w", err)
	}

	return encryptionDefault == 1, nil
}
