package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

// CreateFolderShare creates a new folder sharing record.
func (db *DB) CreateFolderShare(ownerUserID, folderID, sharedWithUserID int, role string) (*FolderShare, error) {
	if role != "viewer" && role != "editor" {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	result, err := db.Exec(`
		INSERT INTO folder_shares (folder_id, owner_user_id, shared_with_user_id, role)
		VALUES (?, ?, ?, ?)
	`, folderID, ownerUserID, sharedWithUserID, role)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("failed to create folder share: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get folder share id: %w", err)
	}
	shareID, err := validateLastInsertID(id, "folder share id")
	if err != nil {
		return nil, err
	}
	return db.getFolderShareByID(shareID)
}

// DeleteFolderShare removes a folder sharing record and cleans up placements.
func (db *DB) DeleteFolderShare(ownerUserID, folderID, sharedWithUserID int) error {
	result, err := db.Exec(`
		DELETE FROM folder_shares
		WHERE folder_id = ? AND owner_user_id = ? AND shared_with_user_id = ?
	`, folderID, ownerUserID, sharedWithUserID)
	if err != nil {
		return fmt.Errorf("failed to delete folder share: %w", err)
	}
	if err := ensureRowsAffectedWithContext(result, "failed to check rows affected"); err != nil {
		return err
	}

	// Placement cleanup: remove placements for notes in this folder
	// Only if the user doesn't still have access via a direct note_share
	if _, err := db.Exec(`
		DELETE FROM shared_note_placements
		WHERE user_id = ? AND note_id IN (
			SELECT n.id FROM notes n
			JOIN folders f ON f.path = n.folder_path
			WHERE f.id = ? AND n.is_deleted = 0
		)
		AND NOT EXISTS (
			SELECT 1 FROM note_shares ns
			WHERE ns.note_id = shared_note_placements.note_id
			  AND ns.shared_with_user_id = ?
		)
	`, sharedWithUserID, folderID, sharedWithUserID); err != nil {
		slog.Warn("failed to cleanup folder share placements", slog.Int("folderID", folderID), slog.String("error", err.Error()))
	}

	return nil
}

// GetFolderShares returns all shares for a specific folder (owner view).
func (db *DB) GetFolderShares(ownerUserID, folderID int) ([]FolderShare, error) {
	rows, err := db.Query(`
		SELECT fs.id, fs.folder_id, f.path, f.name,
		       fs.owner_user_id, ou.username,
		       fs.shared_with_user_id, su.username, fs.role,
		       fs.created_at, fs.updated_at
		FROM folder_shares fs
		JOIN folders f ON f.id = fs.folder_id
		JOIN users ou ON ou.id = fs.owner_user_id
		JOIN users su ON su.id = fs.shared_with_user_id
		WHERE fs.folder_id = ? AND fs.owner_user_id = ?
		ORDER BY fs.created_at DESC
	`, folderID, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get folder shares: %w", err)
	}
	defer rows.Close()

	var shares []FolderShare
	for rows.Next() {
		var s FolderShare
		var createdAt, updatedAt string
		if err := rows.Scan(
			&s.ID, &s.FolderID, &s.FolderPath, &s.FolderName,
			&s.OwnerUserID, &s.OwnerUsername,
			&s.SharedWithUserID, &s.SharedWithUsername, &s.Role,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan folder share: %w", err)
		}
		s.CreatedAt = parseTime(createdAt)
		s.UpdatedAt = parseTime(updatedAt)
		shares = append(shares, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating folder shares: %w", err)
	}

	return shares, nil
}

// GetSharedFoldersForUser returns all folders shared with a user.
// NoteCount only counts unencrypted, non-deleted notes.
func (db *DB) GetSharedFoldersForUser(userID int) ([]SharedFolder, error) {
	rows, err := db.Query(`
		SELECT f.id, f.path, f.name,
		       (SELECT COUNT(*) FROM notes n WHERE n.folder_path = f.path
		        AND n.is_deleted = 0 AND n.content_encrypted = 0) as note_count,
		       ou.username, fs.role, fs.id,
		       f.created_at, f.updated_at
		FROM folder_shares fs
		JOIN folders f ON f.id = fs.folder_id
		JOIN users ou ON ou.id = fs.owner_user_id
		WHERE fs.shared_with_user_id = ?
		ORDER BY ou.username, f.path
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared folders: %w", err)
	}
	defer rows.Close()

	var folders []SharedFolder
	for rows.Next() {
		var sf SharedFolder
		var createdAt, updatedAt string
		if err := rows.Scan(
			&sf.ID, &sf.Path, &sf.Name, &sf.NoteCount,
			&sf.SharedBy, &sf.ShareRole, &sf.ShareID,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan shared folder: %w", err)
		}
		sf.CreatedAt = parseTime(createdAt)
		sf.UpdatedAt = parseTime(updatedAt)
		folders = append(folders, sf)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shared folders: %w", err)
	}

	return folders, nil
}

// GetSharedFolderNotes returns notes in a shared folder visible to the user.
// Only returns unencrypted, non-deleted notes.
func (db *DB) GetSharedFolderNotes(userID, folderID int) ([]SharedNote, error) {
	rows, err := db.Query(`
		SELECT n.id, n.title, n.content, n.folder_path, n.version,
		       n.created_at, n.updated_at,
		       COALESCE(n.note_type, 'note'),
		       ou.username, fs.role, fs.id
		FROM folder_shares fs
		JOIN folders f ON f.id = fs.folder_id
		JOIN notes n ON n.folder_path = f.path AND n.is_deleted = 0 AND n.content_encrypted = 0
		JOIN users ou ON ou.id = fs.owner_user_id
		WHERE fs.shared_with_user_id = ? AND fs.folder_id = ?
		ORDER BY n.title ASC
	`, userID, folderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared folder notes: %w", err)
	}
	defer rows.Close()

	var notes []SharedNote
	for rows.Next() {
		var sn SharedNote
		var createdAt, updatedAt string
		var content sql.NullString
		if err := rows.Scan(
			&sn.ID, &sn.Title, &content, &sn.FolderPath, &sn.Version,
			&createdAt, &updatedAt,
			&sn.NoteType,
			&sn.SharedBy, &sn.ShareRole, &sn.ShareID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan shared folder note: %w", err)
		}
		sn.CreatedAt = parseTime(createdAt)
		sn.UpdatedAt = parseTime(updatedAt)
		if content.Valid {
			sn.Content = content.String
		}
		notes = append(notes, sn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shared folder notes: %w", err)
	}

	return notes, nil
}

// UpdateFolderShareRole updates the role for a folder share record.
func (db *DB) UpdateFolderShareRole(ownerUserID, folderID, sharedWithUserID int, role string) error {
	if role != "viewer" && role != "editor" {
		return fmt.Errorf("invalid role: %s", role)
	}

	result, err := db.Exec(`
		UPDATE folder_shares
		SET role = ?, updated_at = datetime('now')
		WHERE folder_id = ? AND owner_user_id = ? AND shared_with_user_id = ?
	`, role, folderID, ownerUserID, sharedWithUserID)
	if err != nil {
		return fmt.Errorf("failed to update folder share role: %w", err)
	}
	return ensureRowsAffectedWithContext(result, "failed to check rows affected")
}

// GetFolderOwnerUserID returns the owner user_id for a folder.
func (db *DB) GetFolderOwnerUserID(folderID int) (int, error) {
	var userID int
	err := db.QueryRow(`SELECT user_id FROM folders WHERE id = ?`, folderID).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get folder owner: %w", err)
	}
	return userID, nil
}

// FolderHasEncryptedNotes returns true if any non-deleted note in the folder is encrypted.
func (db *DB) FolderHasEncryptedNotes(folderID int) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM notes n
		JOIN folders f ON f.path = n.folder_path
		WHERE f.id = ? AND n.is_deleted = 0 AND n.content_encrypted = 1
	`, folderID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check folder encrypted notes: %w", err)
	}
	return count > 0, nil
}

// GetFolderSharesByFolderID returns all shares for a folder (any owner).
// Used by the encryption guard to check if a folder is shared.
func (db *DB) GetFolderSharesByFolderID(folderID int) ([]FolderShare, error) {
	rows, err := db.Query(`
		SELECT fs.id, fs.folder_id, f.path, f.name,
		       fs.owner_user_id, ou.username,
		       fs.shared_with_user_id, su.username, fs.role,
		       fs.created_at, fs.updated_at
		FROM folder_shares fs
		JOIN folders f ON f.id = fs.folder_id
		JOIN users ou ON ou.id = fs.owner_user_id
		JOIN users su ON su.id = fs.shared_with_user_id
		WHERE fs.folder_id = ?
	`, folderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get folder shares by folder ID: %w", err)
	}
	defer rows.Close()

	var shares []FolderShare
	for rows.Next() {
		var s FolderShare
		var createdAt, updatedAt string
		if err := rows.Scan(
			&s.ID, &s.FolderID, &s.FolderPath, &s.FolderName,
			&s.OwnerUserID, &s.OwnerUsername,
			&s.SharedWithUserID, &s.SharedWithUsername, &s.Role,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan folder share: %w", err)
		}
		s.CreatedAt = parseTime(createdAt)
		s.UpdatedAt = parseTime(updatedAt)
		shares = append(shares, s)
	}

	return shares, rows.Err()
}

// getFolderShareByID retrieves a folder share by its ID (internal helper).
func (db *DB) getFolderShareByID(id int) (*FolderShare, error) {
	var s FolderShare
	var createdAt, updatedAt string

	err := db.QueryRow(`
		SELECT fs.id, fs.folder_id, f.path, f.name,
		       fs.owner_user_id, ou.username,
		       fs.shared_with_user_id, su.username, fs.role,
		       fs.created_at, fs.updated_at
		FROM folder_shares fs
		JOIN folders f ON f.id = fs.folder_id
		JOIN users ou ON ou.id = fs.owner_user_id
		JOIN users su ON su.id = fs.shared_with_user_id
		WHERE fs.id = ?
	`, id).Scan(
		&s.ID, &s.FolderID, &s.FolderPath, &s.FolderName,
		&s.OwnerUserID, &s.OwnerUsername,
		&s.SharedWithUserID, &s.SharedWithUsername, &s.Role,
		&createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get folder share: %w", err)
	}

	s.CreatedAt = parseTime(createdAt)
	s.UpdatedAt = parseTime(updatedAt)

	return &s, nil
}
