package db

import (
	"fmt"
	"strings"
)

// RenameFolder renames a folder and updates all paths transactionally.
func (d *DB) RenameFolder(userID int, folderID int, newName string) error {
	// Get folder to rename
	folder, err := d.GetFolderByID(userID, folderID)
	if err != nil {
		return err
	}

	// GUARD: Validate new name
	if newName == "" {
		return fmt.Errorf("folder name cannot be empty")
	}
	if strings.Contains(newName, "/") {
		return fmt.Errorf("folder name cannot contain /")
	}
	if strings.Contains(newName, "..") {
		return fmt.Errorf("folder name cannot contain ..")
	}

	// Calculate new path
	oldPath := folder.Path
	parentPath := ""
	if folder.ParentID != nil {
		parent, err := d.GetFolderByID(userID, *folder.ParentID)
		if err != nil {
			return fmt.Errorf("parent folder not found: %w", err)
		}
		parentPath = parent.Path
	}

	newPath := parentPath
	if parentPath == "/" {
		newPath = "/" + newName
	} else {
		newPath = parentPath + "/" + newName
	}

	// GUARD: Check if target path already exists
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

	// Update folder itself (name + path)
	_, err = tx.Exec(`
		UPDATE folders
		SET name = ?, path = ?, updated_at = datetime('now')
		WHERE id = ? AND user_id = ?
	`, newName, newPath, folderID, userID)
	if err != nil {
		return err
	}

	// Update all child folders (recursive path update)
	// Pattern: /A/B -> /A/C führt zu /A/B/X -> /A/C/X
	_, err = tx.Exec(`
		UPDATE folders
		SET path = ? || substr(path, length(?) + 1),
			updated_at = datetime('now')
		WHERE (path LIKE ? || '/%') AND id != ? AND user_id = ?
	`, newPath, oldPath, oldPath, folderID, userID)
	if err != nil {
		return err
	}

	// Update all notes in this folder and subfolders
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
