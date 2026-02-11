package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/xela-io/xelanote/internal/utils"
)

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

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return d.GetFolderByID(userID, int(id))
}
