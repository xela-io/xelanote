package db

import (
	"fmt"
	"strings"

	"github.com/xela-io/xelanote/internal/utils"
)

// MoveFolder moves a folder to a new parent (updates path hierarchy).
func (d *DB) MoveFolder(userID int, folderID int, newParentPath string) error {
	newParentPath = utils.NormalizeFolderPath(newParentPath)

	// Get folder to move
	folder, err := d.GetFolderByID(userID, folderID)
	if err != nil {
		return err
	}

	// GUARD: Cannot move folder into itself or its descendants
	if newParentPath == folder.Path || strings.HasPrefix(newParentPath, folder.Path+"/") {
		return fmt.Errorf("cannot move folder into itself or its descendants")
	}

	// Get new parent folder
	var newParentID *int
	if newParentPath == "/" {
		// Moving to top level - virtual root (parent_id = NULL)
		newParentID = nil
	} else {
		// Moving to another folder - get its ID
		parent, err := d.GetFolderByPath(userID, newParentPath)
		if err != nil {
			return fmt.Errorf("parent folder not found: %w", err)
		}
		newParentID = &parent.ID
	}

	// Calculate new path
	oldPath := folder.Path
	newPath := newParentPath
	if newPath == "/" {
		newPath = "/" + folder.Name
	} else {
		newPath = newPath + "/" + folder.Name
	}

	// Check if target path already exists
	existing, err := d.GetFolderByPath(userID, newPath)
	if err == nil && existing != nil && existing.ID != folderID {
		return fmt.Errorf("folder already exists at target path: %s", newPath)
	}

	// Start transaction
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update folder itself
	_, err = tx.Exec(`
		UPDATE folders
		SET path = ?, parent_id = ?, updated_at = datetime('now')
		WHERE id = ? AND user_id = ?
	`, newPath, newParentID, folderID, userID)
	if err != nil {
		return err
	}

	// Update all child folders (recursive path update)
	// FIXED: Use explicit path = oldPath OR path LIKE oldPath || '/%'
	// to avoid matching /A2 when moving /A
	_, err = tx.Exec(`
		UPDATE folders
		SET path = ? || substr(path, length(?) + 1),
			updated_at = datetime('now')
		WHERE (path = ? OR path LIKE ? || '/%') AND id != ? AND user_id = ?
	`, newPath, oldPath, oldPath, oldPath, folderID, userID)
	if err != nil {
		return err
	}

	// Update all notes in this folder and subfolders
	// FIXED: Same precise pattern
	_, err = tx.Exec(`
		UPDATE notes
		SET folder_path = ? || substr(folder_path, length(?) + 1),
			updated_at = datetime('now')
		WHERE (folder_path = ? OR folder_path LIKE ? || '/%') AND user_id = ?
	`, newPath, oldPath, oldPath, oldPath, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
