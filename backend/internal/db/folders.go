package db

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/xela-io/xelanote/internal/utils"
)

// hexColorRegex validates hex color format (#RRGGBB)
var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// validateHexColor validates that a color string is a valid hex color
func validateHexColor(color string) error {
	if color != "" && !hexColorRegex.MatchString(color) {
		return fmt.Errorf("invalid color format, expected #RRGGBB")
	}
	return nil
}

// Folder represents a folder in the hierarchy.
type Folder struct {
	ID                int       `json:"id"`
	Path              string    `json:"path"`
	ParentID          *int      `json:"parent_id"` // Removed omitempty - need to serialize null for virtual root (Migration 025)
	Name              string    `json:"name"`
	NoteCount         int       `json:"note_count"`
	DisplayOrder      int       `json:"display_order"`
	Color             *string   `json:"color,omitempty"`
	AIEnabledDefault  bool      `json:"ai_enabled_default"` // Default for new notes in this folder
	EncryptionDefault bool      `json:"encryption_default"` // Default encryption for new notes (true=encrypted)
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// GetAllFolders returns all folders with note counts from the folders table.
// Note count: Journal folder counts journal notes, other folders count regular notes.
func (d *DB) GetAllFolders(userID int) ([]Folder, error) {
	query := `
		SELECT
			f.id, f.path, f.parent_id, f.name, f.display_order, f.color, f.ai_enabled_default, f.encryption_default,
			f.created_at, f.updated_at,
			COUNT(n.note_rowid) as note_count
		FROM folders f
		LEFT JOIN notes n ON n.folder_path = f.path AND n.is_deleted = 0 AND n.user_id = ?
			AND COALESCE(n.note_type, 'note') = CASE WHEN f.path = '/Journal' THEN 'journal' ELSE 'note' END
		WHERE f.user_id = ?
		GROUP BY f.id
		ORDER BY f.display_order, f.path
	`

	rows, err := d.Query(query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []Folder
	for rows.Next() {
		var f Folder
		var createdAt, updatedAt string
		var aiEnabledDefault, encryptionDefault int
		err := rows.Scan(&f.ID, &f.Path, &f.ParentID, &f.Name, &f.DisplayOrder, &f.Color, &aiEnabledDefault, &encryptionDefault,
			&createdAt, &updatedAt, &f.NoteCount)
		if err != nil {
			return nil, err
		}

		// Parse timestamps
		f.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		f.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
		f.AIEnabledDefault = aiEnabledDefault == 1
		f.EncryptionDefault = encryptionDefault == 1

		folders = append(folders, f)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return folders, nil
}

// GetFolderByPath retrieves folder by path.
// Note count: Journal folder counts journal notes, other folders count regular notes.
func (d *DB) GetFolderByPath(userID int, path string) (*Folder, error) {
	path = utils.NormalizeFolderPath(path)

	query := `
		SELECT
			f.id, f.path, f.parent_id, f.name, f.display_order, f.color, f.ai_enabled_default, f.encryption_default,
			f.created_at, f.updated_at,
			COUNT(n.note_rowid) as note_count
		FROM folders f
		LEFT JOIN notes n ON n.folder_path = f.path AND n.is_deleted = 0 AND n.user_id = ?
			AND COALESCE(n.note_type, 'note') = CASE WHEN f.path = '/Journal' THEN 'journal' ELSE 'note' END
		WHERE f.path = ? AND f.user_id = ?
		GROUP BY f.id
	`

	var f Folder
	var createdAt, updatedAt string
	var aiEnabledDefault, encryptionDefault int
	err := d.QueryRow(query, userID, path, userID).Scan(
		&f.ID, &f.Path, &f.ParentID, &f.Name, &f.DisplayOrder, &f.Color, &aiEnabledDefault, &encryptionDefault,
		&createdAt, &updatedAt, &f.NoteCount,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Parse timestamps
	f.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	f.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	f.AIEnabledDefault = aiEnabledDefault == 1
	f.EncryptionDefault = encryptionDefault == 1

	return &f, nil
}

// GetFolderByID retrieves folder by ID.
// Note count: Journal folder counts journal notes, other folders count regular notes.
func (d *DB) GetFolderByID(userID int, id int) (*Folder, error) {
	query := `
		SELECT
			f.id, f.path, f.parent_id, f.name, f.display_order, f.color, f.ai_enabled_default, f.encryption_default,
			f.created_at, f.updated_at,
			COUNT(n.note_rowid) as note_count
		FROM folders f
		LEFT JOIN notes n ON n.folder_path = f.path AND n.is_deleted = 0 AND n.user_id = ?
			AND COALESCE(n.note_type, 'note') = CASE WHEN f.path = '/Journal' THEN 'journal' ELSE 'note' END
		WHERE f.id = ? AND f.user_id = ?
		GROUP BY f.id
	`

	var f Folder
	var createdAt, updatedAt string
	var aiEnabledDefault, encryptionDefault int
	err := d.QueryRow(query, userID, id, userID).Scan(
		&f.ID, &f.Path, &f.ParentID, &f.Name, &f.DisplayOrder, &f.Color, &aiEnabledDefault, &encryptionDefault,
		&createdAt, &updatedAt, &f.NoteCount,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Parse timestamps
	f.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	f.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	f.AIEnabledDefault = aiEnabledDefault == 1
	f.EncryptionDefault = encryptionDefault == 1

	return &f, nil
}

// CreateFolder creates a new folder.
func (d *DB) CreateFolder(userID int, path string, parentID *int) (*Folder, error) {
	path = utils.NormalizeFolderPath(path)
	name := utils.GetFolderName(path)

	// Defensive check: Prevent root folder creation (final safety layer)
	if path == "/" {
		return nil, fmt.Errorf("cannot create root folder - root is virtual")
	}

	// Check if folder already exists
	existing, err := d.GetFolderByPath(userID, path)
	if err == nil && existing != nil {
		return existing, nil // Idempotent
	}

	query := `
		INSERT INTO folders (user_id, path, parent_id, name)
		VALUES (?, ?, ?, ?)
	`

	result, err := d.Exec(query, userID, path, parentID, name)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return d.GetFolderByID(userID, int(id))
}

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

// ReorderFolders updates the display_order for folders within a parent.
// items is a list of folder IDs in the desired order.
func (d *DB) ReorderFolders(userID int, parentID *int, items []int) error {
	// Start transaction
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update each folder's display_order
	// Use spacing of 100 to allow for future insertions
	for i, folderID := range items {
		newOrder := i * 100
		_, err := tx.Exec(`
			UPDATE folders
			SET display_order = ?, updated_at = datetime('now')
			WHERE id = ? AND user_id = ?
		`, newOrder, folderID, userID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpdateFolderColor updates the color of a folder.
// Pass nil to remove the color.
func (d *DB) UpdateFolderColor(userID int, folderID int, color *string) error {
	// Validate color format if provided
	if color != nil {
		if err := validateHexColor(*color); err != nil {
			return err
		}
	}

	result, err := d.Exec(`
		UPDATE folders
		SET color = ?, updated_at = datetime('now')
		WHERE id = ? AND user_id = ?
	`, color, folderID, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

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

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
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

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
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
