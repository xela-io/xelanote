package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// NoteVersion represents a snapshot of a note at a specific point in time.
type NoteVersion struct {
	ID                int       `json:"id"`
	NoteID            string    `json:"note_id"`
	UserID            int       `json:"user_id"`
	Version           int       `json:"version"`
	Title             string    `json:"title"`
	Content           string    `json:"content"`
	SnapshotAt        time.Time `json:"snapshot_at"`
	EncryptedContent  []byte    `json:"encrypted_content,omitempty"`
	WrappedDEK        string    `json:"wrapped_dek,omitempty"`
	ContentEncrypted  bool      `json:"content_encrypted"`
	TitleEncrypted    bool      `json:"title_encrypted"`
	EncryptedTitle    *string   `json:"encrypted_title,omitempty"`
	EncryptionVersion int       `json:"encryption_version"`
}

// CreateNoteVersion creates a new version snapshot for a note.
func (db *DB) CreateNoteVersion(userID int, noteID string, version int, title, content string) error {
	snapshotAt := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(`
		INSERT INTO note_versions (note_id, user_id, version, title, content, snapshot_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, noteID, userID, version, title, content, snapshotAt)

	if err != nil {
		return fmt.Errorf("failed to create note version: %w", err)
	}

	return nil
}

// CreateEncryptedNoteVersion creates a new version snapshot for an encrypted note.
func (db *DB) CreateEncryptedNoteVersion(
	userID int,
	noteID string,
	version int,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionVersion int,
) error {
	snapshotAt := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(`
		INSERT INTO note_versions (
			note_id, user_id, version, title, content, snapshot_at,
			encrypted_content, wrapped_dek, content_encrypted,
			title_encrypted, encrypted_title, encryption_version
		)
		VALUES (?, ?, ?, ?, '', ?, ?, ?, 1, ?, ?, ?)
	`, noteID, userID, version, title, snapshotAt,
		encryptedContent, wrappedDEK, titleEncrypted, encryptedTitle, encryptionVersion)

	if err != nil {
		return fmt.Errorf("failed to create encrypted note version: %w", err)
	}

	return nil
}

// GetNoteVersions returns a paginated list of versions for a note.
// Cursor format: "snapshot_at|id" for stable pagination.
// Sorted by snapshot_at DESC, id DESC.
func (db *DB) GetNoteVersions(userID int, noteID string, limit int, cursor string) ([]NoteVersion, string, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	// Get total count
	var total int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM note_versions
		WHERE note_id = ? AND user_id = ?
	`, noteID, userID).Scan(&total)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to count versions: %w", err)
	}

	var rows *sql.Rows

	if cursor == "" {
		rows, err = db.Query(`
			SELECT id, note_id, user_id, version, title, content, snapshot_at,
			       encrypted_content, wrapped_dek, content_encrypted,
			       title_encrypted, encrypted_title, encryption_version
			FROM note_versions
			WHERE note_id = ? AND user_id = ?
			ORDER BY snapshot_at DESC, id DESC
			LIMIT ?
		`, noteID, userID, limit+1)
	} else {
		// Parse cursor (format: timestamp|id)
		parts := strings.SplitN(cursor, "|", 2)
		if len(parts) != 2 {
			return nil, "", 0, fmt.Errorf("invalid cursor format")
		}
		cursorTime, cursorID := parts[0], parts[1]
		rows, err = db.Query(`
			SELECT id, note_id, user_id, version, title, content, snapshot_at,
			       encrypted_content, wrapped_dek, content_encrypted,
			       title_encrypted, encrypted_title, encryption_version
			FROM note_versions
			WHERE note_id = ? AND user_id = ?
			  AND (snapshot_at < ? OR (snapshot_at = ? AND id < ?))
			ORDER BY snapshot_at DESC, id DESC
			LIMIT ?
		`, noteID, userID, cursorTime, cursorTime, cursorID, limit+1)
	}

	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to list versions: %w", err)
	}
	defer rows.Close()

	var versions []NoteVersion
	for rows.Next() {
		var v NoteVersion
		var snapshotAt string
		var wrappedDEK sql.NullString
		if err := rows.Scan(&v.ID, &v.NoteID, &v.UserID, &v.Version, &v.Title, &v.Content, &snapshotAt,
			&v.EncryptedContent, &wrappedDEK, &v.ContentEncrypted,
			&v.TitleEncrypted, &v.EncryptedTitle, &v.EncryptionVersion); err != nil {
			return nil, "", 0, fmt.Errorf("failed to scan version: %w", err)
		}
		v.WrappedDEK = wrappedDEK.String
		v.SnapshotAt, _ = time.Parse(time.RFC3339, snapshotAt)
		versions = append(versions, v)
	}

	if err := rows.Err(); err != nil {
		return nil, "", 0, fmt.Errorf("error iterating versions: %w", err)
	}

	// Check for next page
	var nextCursor string
	if len(versions) > limit {
		lastVersion := versions[limit-1]
		nextCursor = fmt.Sprintf("%s|%d", lastVersion.SnapshotAt.Format(time.RFC3339), lastVersion.ID)
		versions = versions[:limit]
	}

	return versions, nextCursor, total, nil
}

// GetNoteVersion retrieves a specific version of a note.
func (db *DB) GetNoteVersion(userID int, noteID string, version int) (*NoteVersion, error) {
	var v NoteVersion
	var snapshotAt string
	var wrappedDEK sql.NullString

	err := db.QueryRow(`
		SELECT id, note_id, user_id, version, title, content, snapshot_at,
		       encrypted_content, wrapped_dek, content_encrypted,
		       title_encrypted, encrypted_title, encryption_version
		FROM note_versions
		WHERE note_id = ? AND user_id = ? AND version = ?
	`, noteID, userID, version).Scan(&v.ID, &v.NoteID, &v.UserID, &v.Version, &v.Title, &v.Content, &snapshotAt,
		&v.EncryptedContent, &wrappedDEK, &v.ContentEncrypted,
		&v.TitleEncrypted, &v.EncryptedTitle, &v.EncryptionVersion)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	v.WrappedDEK = wrappedDEK.String
	v.SnapshotAt, _ = time.Parse(time.RFC3339, snapshotAt)
	return &v, nil
}

// GetLatestVersionSnapshot returns the most recent snapshot for a note.
// Sorted by snapshot_at DESC LIMIT 1 for robustness with version gaps.
func (db *DB) GetLatestVersionSnapshot(userID int, noteID string) (*NoteVersion, error) {
	var v NoteVersion
	var snapshotAt string
	var wrappedDEK sql.NullString

	err := db.QueryRow(`
		SELECT id, note_id, user_id, version, title, content, snapshot_at,
		       encrypted_content, wrapped_dek, content_encrypted,
		       title_encrypted, encrypted_title, encryption_version
		FROM note_versions
		WHERE note_id = ? AND user_id = ?
		ORDER BY snapshot_at DESC
		LIMIT 1
	`, noteID, userID).Scan(&v.ID, &v.NoteID, &v.UserID, &v.Version, &v.Title, &v.Content, &snapshotAt,
		&v.EncryptedContent, &wrappedDEK, &v.ContentEncrypted,
		&v.TitleEncrypted, &v.EncryptedTitle, &v.EncryptionVersion)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version snapshot: %w", err)
	}

	v.WrappedDEK = wrappedDEK.String
	v.SnapshotAt, _ = time.Parse(time.RFC3339, snapshotAt)
	return &v, nil
}

// PruneVersions removes old versions for a specific note, keeping the most recent keepCount versions.
// Returns the number of deleted versions.
func (db *DB) PruneVersions(userID int, noteID string, keepCount int) (int, error) {
	if keepCount <= 0 {
		keepCount = 100
	}

	result, err := db.Exec(`
		DELETE FROM note_versions
		WHERE note_id = ? AND user_id = ?
		  AND id NOT IN (
			SELECT id FROM note_versions
			WHERE note_id = ? AND user_id = ?
			ORDER BY snapshot_at DESC
			LIMIT ?
		  )
	`, noteID, userID, noteID, userID, keepCount)

	if err != nil {
		return 0, fmt.Errorf("failed to prune versions: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rows), nil
}

// PruneAllUserVersions removes old versions for all notes of a user, keeping the most recent keepCount versions per note.
// Returns the total number of deleted versions.
func (db *DB) PruneAllUserVersions(userID int, keepCount int) (int, error) {
	if keepCount <= 0 {
		keepCount = 100
	}

	// Get all note IDs that have versions for this user
	rows, err := db.Query(`
		SELECT DISTINCT note_id FROM note_versions WHERE user_id = ?
	`, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get notes with versions: %w", err)
	}
	defer rows.Close()

	var noteIDs []string
	for rows.Next() {
		var noteID string
		if err := rows.Scan(&noteID); err != nil {
			return 0, fmt.Errorf("failed to scan note_id: %w", err)
		}
		noteIDs = append(noteIDs, noteID)
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating note IDs: %w", err)
	}

	// Prune each note
	totalPruned := 0
	for _, noteID := range noteIDs {
		pruned, err := db.PruneVersions(userID, noteID, keepCount)
		if err != nil {
			// Log but continue with other notes
			continue
		}
		totalPruned += pruned
	}

	return totalPruned, nil
}

// CountVersions returns the number of versions for a specific note.
func (db *DB) CountVersions(userID int, noteID string) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM note_versions
		WHERE note_id = ? AND user_id = ?
	`, noteID, userID).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to count versions: %w", err)
	}

	return count, nil
}

// GetUsersWithVersions returns a list of user IDs that have note versions.
// Used by the pruning job to iterate over all users.
func (db *DB) GetUsersWithVersions() ([]int, error) {
	rows, err := db.Query(`
		SELECT DISTINCT user_id FROM note_versions
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get users with versions: %w", err)
	}
	defer rows.Close()

	var userIDs []int
	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan user_id: %w", err)
		}
		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user IDs: %w", err)
	}

	return userIDs, nil
}

// GetAllEncryptedVersionsForUser retrieves all encrypted versions for a user.
// Returns versions where content_encrypted is true.
func (db *DB) GetAllEncryptedVersionsForUser(userID int) ([]NoteVersion, error) {
	rows, err := db.Query(`
		SELECT id, note_id, user_id, version, title, content, snapshot_at,
		       encrypted_content, wrapped_dek, content_encrypted,
		       title_encrypted, encrypted_title, encryption_version
		FROM note_versions
		WHERE user_id = ? AND content_encrypted = 1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query encrypted versions: %w", err)
	}
	defer rows.Close()

	var versions []NoteVersion
	for rows.Next() {
		var v NoteVersion
		var snapshotAt string
		var wrappedDEK sql.NullString
		if err := rows.Scan(&v.ID, &v.NoteID, &v.UserID, &v.Version, &v.Title, &v.Content, &snapshotAt,
			&v.EncryptedContent, &wrappedDEK, &v.ContentEncrypted,
			&v.TitleEncrypted, &v.EncryptedTitle, &v.EncryptionVersion); err != nil {
			return nil, fmt.Errorf("failed to scan encrypted version: %w", err)
		}
		v.WrappedDEK = wrappedDEK.String
		v.SnapshotAt, _ = time.Parse(time.RFC3339, snapshotAt)
		versions = append(versions, v)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating encrypted versions: %w", err)
	}

	return versions, nil
}
