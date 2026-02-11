package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/xela-io/xelanote/internal/utils"
)

// CreateNoteShare creates a new sharing record.
// owner_user_id is derived from notes.user_id, NOT from the client.
func (db *DB) CreateNoteShare(ownerUserID int, noteID string, sharedWithUserID int, role string) (*NoteShare, error) {
	if role != "viewer" && role != "editor" {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	result, err := db.Exec(`
		INSERT INTO note_shares (note_id, owner_user_id, shared_with_user_id, role)
		VALUES (?, ?, ?, ?)
	`, noteID, ownerUserID, sharedWithUserID, role)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("failed to create note share: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get note share id: %w", err)
	}
	shareID, err := validateLastInsertID(id, "note share id")
	if err != nil {
		return nil, err
	}
	return db.getNoteShareByID(shareID)
}

// DeleteNoteShare removes a sharing record and cleans up any placements.
func (db *DB) DeleteNoteShare(ownerUserID int, noteID string, sharedWithUserID int) error {
	result, err := db.Exec(`
		DELETE FROM note_shares
		WHERE note_id = ? AND owner_user_id = ? AND shared_with_user_id = ?
	`, noteID, ownerUserID, sharedWithUserID)
	if err != nil {
		return fmt.Errorf("failed to delete note share: %w", err)
	}
	if err := ensureRowsAffectedWithContext(result, "failed to check rows affected"); err != nil {
		return err
	}

	// Placement cleanup: remove placement for this note+user
	// Only if the user doesn't still have access via a folder share
	db.Exec(`
		DELETE FROM shared_note_placements
		WHERE note_id = ? AND user_id = ?
		  AND NOT EXISTS (
		    SELECT 1 FROM folder_shares fs
		    JOIN folders f ON f.id = fs.folder_id
		    JOIN notes n ON n.folder_path = f.path AND n.id = ?
		    WHERE fs.shared_with_user_id = ? AND n.is_deleted = 0
		  )
	`, noteID, sharedWithUserID, noteID, sharedWithUserID)

	return nil
}

// GetNoteShares returns all shares for a specific note.
func (db *DB) GetNoteShares(ownerUserID int, noteID string) ([]NoteShare, error) {
	rows, err := db.Query(`
		SELECT ns.id, ns.note_id, ns.owner_user_id, ou.username,
		       ns.shared_with_user_id, su.username, ns.role,
		       ns.created_at, ns.updated_at
		FROM note_shares ns
		JOIN users ou ON ou.id = ns.owner_user_id
		JOIN users su ON su.id = ns.shared_with_user_id
		WHERE ns.note_id = ? AND ns.owner_user_id = ?
		ORDER BY ns.created_at DESC
	`, noteID, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get note shares: %w", err)
	}
	defer rows.Close()

	var shares []NoteShare
	for rows.Next() {
		var s NoteShare
		var createdAt, updatedAt string
		if err := rows.Scan(
			&s.ID, &s.NoteID, &s.OwnerUserID, &s.OwnerUsername,
			&s.SharedWithUserID, &s.SharedWithUsername, &s.Role,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan note share: %w", err)
		}
		s.CreatedAt = parseTime(createdAt)
		s.UpdatedAt = parseTime(updatedAt)
		shares = append(shares, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating note shares: %w", err)
	}

	return shares, nil
}

// GetSharedNotesForUser returns all notes shared with the given user.
// Includes notes shared directly (note_shares) AND notes in shared folders (folder_shares).
// Deduplication: if a note has both, the direct note_share takes priority.
func (db *DB) GetSharedNotesForUser(userID int) ([]SharedNote, error) {
	rows, err := db.Query(`
		SELECT n.id, n.title, n.content, n.folder_path, n.version,
		       n.created_at, n.updated_at,
		       COALESCE(n.note_type, 'note'), -- backwards compat: NULL → 'note' for old data
		       ou.username, ns.role, ns.id
		FROM note_shares ns
		JOIN notes n ON n.id = ns.note_id
		JOIN users ou ON ou.id = ns.owner_user_id
		WHERE ns.shared_with_user_id = ? AND n.is_deleted = 0

		UNION ALL

		SELECT n.id, n.title, n.content, n.folder_path, n.version,
		       n.created_at, n.updated_at,
		       COALESCE(n.note_type, 'note'),
		       ou.username, fs.role, fs.id
		FROM folder_shares fs
		JOIN folders f ON f.id = fs.folder_id
		JOIN notes n ON n.folder_path = f.path AND n.is_deleted = 0 AND n.content_encrypted = 0
		JOIN users ou ON ou.id = fs.owner_user_id
		WHERE fs.shared_with_user_id = ?
		  AND NOT EXISTS (
		    SELECT 1 FROM note_shares ns2
		    WHERE ns2.note_id = n.id AND ns2.shared_with_user_id = ?
		  )

		ORDER BY 7 DESC
	`, userID, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared notes: %w", err)
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
			return nil, fmt.Errorf("failed to scan shared note: %w", err)
		}
		sn.CreatedAt = parseTime(createdAt)
		sn.UpdatedAt = parseTime(updatedAt)
		if content.Valid {
			sn.Content = content.String
		}
		notes = append(notes, sn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shared notes: %w", err)
	}

	return notes, nil
}

// GetSharedNote loads a single shared note for a user.
// Checks note_shares first, then folder_shares.
func (db *DB) GetSharedNote(userID int, noteID string) (*SharedNote, error) {
	var sn SharedNote
	var createdAt, updatedAt string
	var content sql.NullString

	// Try direct note share first
	err := db.QueryRow(`
		SELECT n.id, n.title, n.content, n.folder_path, n.version,
		       n.created_at, n.updated_at,
		       COALESCE(n.note_type, 'note'),
		       ou.username, ns.role, ns.id
		FROM note_shares ns
		JOIN notes n ON n.id = ns.note_id
		JOIN users ou ON ou.id = ns.owner_user_id
		WHERE ns.shared_with_user_id = ? AND ns.note_id = ? AND n.is_deleted = 0
	`, userID, noteID).Scan(
		&sn.ID, &sn.Title, &content, &sn.FolderPath, &sn.Version,
		&createdAt, &updatedAt,
		&sn.NoteType,
		&sn.SharedBy, &sn.ShareRole, &sn.ShareID,
	)

	if err == nil {
		sn.CreatedAt = parseTime(createdAt)
		sn.UpdatedAt = parseTime(updatedAt)
		if content.Valid {
			sn.Content = content.String
		}
		return &sn, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get shared note: %w", err)
	}

	// Fallback to folder share
	err = db.QueryRow(`
		SELECT n.id, n.title, n.content, n.folder_path, n.version,
		       n.created_at, n.updated_at,
		       COALESCE(n.note_type, 'note'),
		       ou.username, fs.role, fs.id
		FROM folder_shares fs
		JOIN folders f ON f.id = fs.folder_id
		JOIN notes n ON n.folder_path = f.path AND n.id = ? AND n.is_deleted = 0 AND n.content_encrypted = 0
		JOIN users ou ON ou.id = fs.owner_user_id
		WHERE fs.shared_with_user_id = ?
		LIMIT 1
	`, noteID, userID).Scan(
		&sn.ID, &sn.Title, &content, &sn.FolderPath, &sn.Version,
		&createdAt, &updatedAt,
		&sn.NoteType,
		&sn.SharedBy, &sn.ShareRole, &sn.ShareID,
	)

	if err == sql.ErrNoRows {
		// Fallback to collection share
		err = db.QueryRow(`
			SELECT n.id, n.title, n.content, n.folder_path, n.version,
			       n.created_at, n.updated_at,
			       COALESCE(n.note_type, 'note'),
			       ou.username, rcs.role, rcs.id
			FROM recipe_collection_shares rcs
			JOIN recipe_collection_items rci ON rci.collection_id = rcs.collection_id
			JOIN notes n ON n.id = rci.note_id AND n.id = ? AND n.is_deleted = 0
			JOIN users ou ON ou.id = rcs.owner_user_id
			WHERE rcs.shared_with_user_id = ?
			LIMIT 1
		`, noteID, userID).Scan(
			&sn.ID, &sn.Title, &content, &sn.FolderPath, &sn.Version,
			&createdAt, &updatedAt,
			&sn.NoteType,
			&sn.SharedBy, &sn.ShareRole, &sn.ShareID,
		)

		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get shared note via collection: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get shared note via folder: %w", err)
	}

	sn.CreatedAt = parseTime(createdAt)
	sn.UpdatedAt = parseTime(updatedAt)
	if content.Valid {
		sn.Content = content.String
	}

	return &sn, nil
}

// GetSharePermission returns the role ("viewer", "editor") for a user on a note,
// or empty string if no share exists.
// Checks note_shares first (highest priority), then folder_shares.
func (db *DB) GetSharePermission(userID int, noteID string) (string, error) {
	// 1. Direct note share (highest priority)
	var noteRole string
	err := db.QueryRow(`
		SELECT role FROM note_shares
		WHERE shared_with_user_id = ? AND note_id = ?
	`, userID, noteID).Scan(&noteRole)
	if err == nil {
		return noteRole, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to get share permission: %w", err)
	}

	// 2. Folder share (implicit inheritance)
	var folderRole string
	err = db.QueryRow(`
		SELECT fs.role FROM folder_shares fs
		JOIN folders f ON f.id = fs.folder_id
		WHERE fs.shared_with_user_id = ?
		  AND f.path = (SELECT folder_path FROM notes WHERE id = ? AND is_deleted = 0)
	`, userID, noteID).Scan(&folderRole)
	if err == nil {
		return folderRole, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to get folder share permission: %w", err)
	}

	// 3. Collection share (lowest priority — only if 1+2 found nothing)
	var collRole string
	err = db.QueryRow(`
		SELECT rcs.role FROM recipe_collection_shares rcs
		JOIN recipe_collection_items rci ON rci.collection_id = rcs.collection_id
		WHERE rcs.shared_with_user_id = ? AND rci.note_id = ?
		LIMIT 1
	`, userID, noteID).Scan(&collRole)
	if err == nil {
		return collRole, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to get collection share permission: %w", err)
	}

	return "", nil // No access
}

// UpdateNoteShareRole updates the role for a share record.
func (db *DB) UpdateNoteShareRole(ownerUserID int, noteID string, sharedWithUserID int, role string) error {
	if role != "viewer" && role != "editor" {
		return fmt.Errorf("invalid role: %s", role)
	}

	result, err := db.Exec(`
		UPDATE note_shares
		SET role = ?, updated_at = datetime('now')
		WHERE note_id = ? AND owner_user_id = ? AND shared_with_user_id = ?
	`, role, noteID, ownerUserID, sharedWithUserID)
	if err != nil {
		return fmt.Errorf("failed to update note share role: %w", err)
	}
	return ensureRowsAffectedWithContext(result, "failed to check rows affected")
}

// DeleteAllSharesForNote removes all shares for a note.
func (db *DB) DeleteAllSharesForNote(noteID string) error {
	// Also clean up placements for all users who had note-level shares
	db.Exec(`
		DELETE FROM shared_note_placements
		WHERE note_id = ?
		  AND NOT EXISTS (
		    SELECT 1 FROM folder_shares fs
		    JOIN folders f ON f.id = fs.folder_id
		    JOIN notes n ON n.folder_path = f.path AND n.id = ?
		    WHERE fs.shared_with_user_id = shared_note_placements.user_id AND n.is_deleted = 0
		  )
	`, noteID, noteID)

	_, err := db.Exec(`DELETE FROM note_shares WHERE note_id = ?`, noteID)
	return err
}

// GetNoteOwnerUserID returns the user_id of the note owner.
func (db *DB) GetNoteOwnerUserID(noteID string) (int, error) {
	var userID int
	err := db.QueryRow(`SELECT user_id FROM notes WHERE id = ? AND is_deleted = 0`, noteID).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get note owner: %w", err)
	}
	return userID, nil
}

// IsNoteEncrypted returns whether a note has encrypted content.
func (db *DB) IsNoteEncrypted(noteID string) (bool, error) {
	var encrypted bool
	err := db.QueryRow(`SELECT content_encrypted FROM notes WHERE id = ? AND is_deleted = 0`, noteID).Scan(&encrypted)
	if err == sql.ErrNoRows {
		return false, ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("failed to check note encryption: %w", err)
	}
	return encrypted, nil
}

// UpdateSharedNote updates a note via a share (editor role).
// Only updates title, content, and version. Does NOT move the note.
func (db *DB) UpdateSharedNote(noteID string, title, content string, expectedVersion int) (*SharedNote, error) {
	titleNorm := utils.NormalizeTitle(title)
	now := time.Now().UTC()

	result, err := db.Exec(`
		UPDATE notes
		SET title = ?, title_norm = ?, content = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND version = ? AND is_deleted = 0
	`, title, titleNorm, content, now.Format(time.RFC3339), noteID, expectedVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to update shared note: %w", err)
	}

	rows, err := rowsAffectedCount(result, "failed to check rows affected")
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		// Check if note exists
		var exists int
		db.QueryRow(`SELECT 1 FROM notes WHERE id = ? AND is_deleted = 0`, noteID).Scan(&exists)
		if exists == 0 {
			return nil, ErrNotFound
		}
		return nil, ErrVersionMismatch
	}

	// Return updated note data (for any share recipient)
	var sn SharedNote
	var createdAt, updatedAt string
	var contentStr sql.NullString
	err = db.QueryRow(`
		SELECT id, title, content, folder_path, version, created_at, updated_at
		FROM notes WHERE id = ?
	`, noteID).Scan(&sn.ID, &sn.Title, &contentStr, &sn.FolderPath, &sn.Version, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated note: %w", err)
	}

	sn.CreatedAt = parseTime(createdAt)
	sn.UpdatedAt = parseTime(updatedAt)
	if contentStr.Valid {
		sn.Content = contentStr.String
	}

	return &sn, nil
}

// getNoteShareByID retrieves a share by its ID (internal helper).
func (db *DB) getNoteShareByID(id int) (*NoteShare, error) {
	var s NoteShare
	var createdAt, updatedAt string

	err := db.QueryRow(`
		SELECT ns.id, ns.note_id, ns.owner_user_id, ou.username,
		       ns.shared_with_user_id, su.username, ns.role,
		       ns.created_at, ns.updated_at
		FROM note_shares ns
		JOIN users ou ON ou.id = ns.owner_user_id
		JOIN users su ON su.id = ns.shared_with_user_id
		WHERE ns.id = ?
	`, id).Scan(
		&s.ID, &s.NoteID, &s.OwnerUserID, &s.OwnerUsername,
		&s.SharedWithUserID, &s.SharedWithUsername, &s.Role,
		&createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get note share: %w", err)
	}

	s.CreatedAt = parseTime(createdAt)
	s.UpdatedAt = parseTime(updatedAt)

	return &s, nil
}
