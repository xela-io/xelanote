package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// NoteShare represents a sharing record for a note.
type NoteShare struct {
	ID                 int       `json:"id"`
	NoteID             string    `json:"note_id"`
	OwnerUserID        int       `json:"owner_user_id"`
	OwnerUsername      string    `json:"owner_username"`
	SharedWithUserID   int       `json:"shared_with_user_id"`
	SharedWithUsername string    `json:"shared_with_username"`
	Role               string    `json:"role"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// SharedNote represents a note shared with the current user.
// Flat structure (no embedding of Note) to avoid JSON field conflicts.
type SharedNote struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	FolderPath string    `json:"folder_path"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	// note_type: omitempty for backwards compatibility with old clients
	NoteType string `json:"note_type,omitempty"`
	// Sharing-specific fields
	SharedBy  string `json:"shared_by"`
	ShareRole string `json:"share_role"`
	ShareID   int    `json:"share_id"`
}

// FolderShare represents a sharing record for a folder.
type FolderShare struct {
	ID                 int       `json:"id"`
	FolderID           int       `json:"folder_id"`
	FolderPath         string    `json:"folder_path"`
	FolderName         string    `json:"folder_name"`
	OwnerUserID        int       `json:"owner_user_id"`
	OwnerUsername      string    `json:"owner_username"`
	SharedWithUserID   int       `json:"shared_with_user_id"`
	SharedWithUsername string    `json:"shared_with_username"`
	Role               string    `json:"role"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// SharedFolder represents a folder shared with the current user.
type SharedFolder struct {
	ID        int       `json:"id"`
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	NoteCount int       `json:"note_count"`
	SharedBy  string    `json:"shared_by"`
	ShareRole string    `json:"share_role"`
	ShareID   int       `json:"share_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserSearchResult represents a user found via search (for the share dialog).
type UserSearchResult struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

// parseTime tries RFC3339 first, then SQLite datetime format.
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, _ = time.Parse("2006-01-02 15:04:05", s)
	}
	return t
}

// ============================================================================
// Note Share CRUD
// ============================================================================

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

	id, _ := result.LastInsertId()
	return db.getNoteShareByID(int(id))
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

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
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

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// SearchUserByUsernameOrEmail searches for users by username or email prefix.
// Excludes the requesting user and limits results to 10.
func (db *DB) SearchUserByUsernameOrEmail(query string, excludeUserID int) ([]UserSearchResult, error) {
	if len(query) < 3 {
		return nil, fmt.Errorf("query must be at least 3 characters")
	}

	likeQuery := query + "%"
	rows, err := db.Query(`
		SELECT id, username FROM users
		WHERE id != ? AND (username LIKE ? OR email LIKE ?)
		ORDER BY username ASC
		LIMIT 10
	`, excludeUserID, likeQuery, likeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	var results []UserSearchResult
	for rows.Next() {
		var u UserSearchResult
		if err := rows.Scan(&u.ID, &u.Username); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		results = append(results, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return results, nil
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
	titleNorm := NormalizeTitle(title)
	now := time.Now().UTC()

	result, err := db.Exec(`
		UPDATE notes
		SET title = ?, title_norm = ?, content = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND version = ? AND is_deleted = 0
	`, title, titleNorm, content, now.Format(time.RFC3339), noteID, expectedVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to update shared note: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
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

// ============================================================================
// Folder Share CRUD
// ============================================================================

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

	id, _ := result.LastInsertId()
	return db.getFolderShareByID(int(id))
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

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	// Placement cleanup: remove placements for notes in this folder
	// Only if the user doesn't still have access via a direct note_share
	db.Exec(`
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
	`, sharedWithUserID, folderID, sharedWithUserID)

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

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
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

// ============================================================================
// Placement CRUD (with share validation)
// ============================================================================

// CreateOrUpdatePlacement places a shared note in the user's own folder.
// Uses a subquery to verify the user actually has an active share on the note.
func (db *DB) CreateOrUpdatePlacement(userID int, noteID string, folderID int) error {
	// Verify note owner is NOT the current user (can't place own notes)
	var noteOwnerID int
	err := db.QueryRow(`SELECT user_id FROM notes WHERE id = ? AND is_deleted = 0`, noteID).Scan(&noteOwnerID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to check note owner: %w", err)
	}
	if noteOwnerID == userID {
		return fmt.Errorf("cannot place own note")
	}

	// Insert with share-existence check (defense in depth)
	result, err := db.Exec(`
		INSERT OR REPLACE INTO shared_note_placements (note_id, user_id, folder_id, created_at)
		SELECT ?, ?, ?, datetime('now')
		WHERE EXISTS (
			SELECT 1 FROM note_shares ns
			JOIN notes n ON n.id = ns.note_id
			WHERE ns.note_id = ? AND ns.shared_with_user_id = ? AND n.is_deleted = 0
			UNION ALL
			SELECT 1 FROM folder_shares fs
			JOIN folders f ON f.id = fs.folder_id
			JOIN notes n ON n.folder_path = f.path AND n.id = ?
			WHERE fs.shared_with_user_id = ? AND n.is_deleted = 0
		)
	`, noteID, userID, folderID, noteID, userID, noteID, userID)
	if err != nil {
		return fmt.Errorf("failed to create placement: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("no active share exists for this note")
	}

	return nil
}

// DeletePlacement removes a note placement.
func (db *DB) DeletePlacement(userID int, noteID string) error {
	result, err := db.Exec(`
		DELETE FROM shared_note_placements
		WHERE note_id = ? AND user_id = ?
	`, noteID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete placement: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// GetPlacement returns the folder_id where a shared note is placed for a user.
// Returns nil if no placement exists.
func (db *DB) GetPlacement(userID int, noteID string) (*int, error) {
	var folderID int
	err := db.QueryRow(`
		SELECT folder_id FROM shared_note_placements
		WHERE note_id = ? AND user_id = ?
	`, noteID, userID).Scan(&folderID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get placement: %w", err)
	}
	return &folderID, nil
}

// GetSharedRecipesForUser returns all recipe notes shared with the given user.
// 3-way UNION: note_shares + folder_shares + collection_shares.
// Dedup via NOT EXISTS: highest priority source wins (R2, R3).
func (db *DB) GetSharedRecipesForUser(userID int) ([]SharedNote, error) {
	rows, err := db.Query(`
		-- 1. Direct note shares (highest priority)
		SELECT n.id, n.title, n.content, n.folder_path, n.version,
		       n.created_at, n.updated_at,
		       COALESCE(n.note_type, 'note'),
		       ou.username, ns.role, ns.id
		FROM note_shares ns
		JOIN notes n ON n.id = ns.note_id
		JOIN users ou ON ou.id = ns.owner_user_id
		WHERE ns.shared_with_user_id = ? AND n.is_deleted = 0 AND n.note_type = 'recipe'

		UNION ALL

		-- 2. Folder shares (dedup against note_shares)
		SELECT n.id, n.title, n.content, n.folder_path, n.version,
		       n.created_at, n.updated_at,
		       COALESCE(n.note_type, 'note'),
		       ou.username, fs.role, fs.id
		FROM folder_shares fs
		JOIN folders f ON f.id = fs.folder_id
		JOIN notes n ON n.folder_path = f.path AND n.is_deleted = 0 AND n.content_encrypted = 0
		JOIN users ou ON ou.id = fs.owner_user_id
		WHERE fs.shared_with_user_id = ? AND n.note_type = 'recipe'
		  AND NOT EXISTS (
		    SELECT 1 FROM note_shares ns2
		    WHERE ns2.note_id = n.id AND ns2.shared_with_user_id = ?
		  )

		UNION ALL

		-- 3. Collection shares (dedup against note_shares + folder_shares)
		SELECT n.id, n.title, n.content, n.folder_path, n.version,
		       n.created_at, n.updated_at,
		       COALESCE(n.note_type, 'note'),
		       ou.username, rcs.role, rcs.id
		FROM recipe_collection_shares rcs
		JOIN recipe_collection_items rci ON rci.collection_id = rcs.collection_id
		JOIN notes n ON n.id = rci.note_id AND n.is_deleted = 0 AND n.note_type = 'recipe'
		JOIN users ou ON ou.id = rcs.owner_user_id
		WHERE rcs.shared_with_user_id = ?
		  AND NOT EXISTS (
		    SELECT 1 FROM note_shares ns3
		    WHERE ns3.note_id = n.id AND ns3.shared_with_user_id = ?
		  )
		  AND NOT EXISTS (
		    SELECT 1 FROM folder_shares fs2
		    JOIN folders f2 ON f2.id = fs2.folder_id
		    WHERE fs2.shared_with_user_id = ? AND f2.path = n.folder_path
		  )

		ORDER BY 7 DESC
	`, userID, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared recipes: %w", err)
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
			return nil, fmt.Errorf("failed to scan shared recipe: %w", err)
		}
		sn.CreatedAt = parseTime(createdAt)
		sn.UpdatedAt = parseTime(updatedAt)
		if content.Valid {
			sn.Content = content.String
		}
		notes = append(notes, sn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shared recipes: %w", err)
	}

	return notes, nil
}
