package db

import (
	"database/sql"
	"fmt"
)

// DeleteFolder deletes a folder (CASCADE handled by DB).
func (d *DB) DeleteFolder(userID int, folderID int) error {
	// Get folder first to check if it exists
	folder, err := d.GetFolderByID(userID, folderID)
	if err != nil {
		return err
	}

	// Start transaction for atomic operation
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Soft-delete all notes in this folder and its subfolders recursively
	if err := d.softDeleteNotesInFolderRecursive(tx, userID, folder.Path); err != nil {
		return fmt.Errorf("failed to soft-delete notes: %w", err)
	}

	// Delete all child folders recursively (bottom-up)
	if err := d.deleteFolderRecursive(tx, userID, folderID); err != nil {
		return fmt.Errorf("failed to delete subfolders: %w", err)
	}

	// Delete the folder itself
	_, err = tx.Exec(`DELETE FROM folders WHERE id = ? AND user_id = ?`, folderID, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// softDeleteNotesInFolderRecursive soft-deletes all notes in a folder and its subfolders
func (d *DB) softDeleteNotesInFolderRecursive(tx *sql.Tx, userID int, folderPath string) error {
	// Soft-delete all notes in this folder
	_, err := tx.Exec(`
		UPDATE notes
		SET is_deleted = 1, deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
		  AND folder_path = ?
		  AND is_deleted = 0
	`, userID, folderPath)
	if err != nil {
		return err
	}

	// Find all direct subfolders
	rows, err := tx.Query(`
		SELECT path
		FROM folders
		WHERE user_id = ?
		  AND path LIKE ?
		  AND path != ?
	`, userID, folderPath+"/%", folderPath)
	if err != nil {
		return err
	}
	defer rows.Close()

	var subfolderPaths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return err
		}
		subfolderPaths = append(subfolderPaths, path)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// Soft-delete notes in all subfolders
	for _, subPath := range subfolderPaths {
		_, err := tx.Exec(`
			UPDATE notes
			SET is_deleted = 1, deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE user_id = ?
			  AND folder_path = ?
			  AND is_deleted = 0
		`, userID, subPath)
		if err != nil {
			return err
		}
	}

	return nil
}

// deleteFolderRecursive deletes a folder and all its subfolders recursively
func (d *DB) deleteFolderRecursive(tx *sql.Tx, userID int, folderID int) error {
	// Find all child folders
	rows, err := tx.Query(`SELECT id FROM folders WHERE parent_id = ? AND user_id = ?`, folderID, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var childIDs []int
	for rows.Next() {
		var childID int
		if err := rows.Scan(&childID); err != nil {
			return err
		}
		childIDs = append(childIDs, childID)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// Recursively delete children first (bottom-up)
	for _, childID := range childIDs {
		if err := d.deleteFolderRecursive(tx, userID, childID); err != nil {
			return err
		}
		// Delete child folder
		_, err := tx.Exec(`DELETE FROM folders WHERE id = ? AND user_id = ?`, childID, userID)
		if err != nil {
			return err
		}
	}

	return nil
}
