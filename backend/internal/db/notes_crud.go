package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/xela-io/xelanote/internal/utils"
)

// CreateNote creates a new note and returns it.
// ai_enabled is inherited from folder's ai_enabled_default, except for root notes (always false).
func (db *DB) CreateNote(userID int, title, content, folderPath string) (*Note, error) {
	id := uuid.New().String()
	titleNorm := utils.NormalizeTitle(title)
	now := time.Now().UTC()

	// Determine ai_enabled based on folder default
	// Root notes (folderPath == "/") are always ai_enabled=false
	aiEnabled := 0
	if folderPath != "/" {
		folderDefault, err := db.GetFolderAIEnabledDefaultByPath(userID, folderPath)
		if err == nil && folderDefault {
			aiEnabled = 1
		}
	}

	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, ai_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, title, titleNorm, content, folderPath, userID, aiEnabled, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}

	return &Note{
		ID:         id,
		Title:      title,
		Content:    content,
		FolderPath: folderPath,
		Version:    1,
		AIEnabled:  aiEnabled == 1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// GetNote retrieves a note by ID.
func (db *DB) GetNote(userID int, id string) (*Note, error) {
	var note Note
	var createdAt, updatedAt string
	var content, encryptedTitle, wrappedDEK, wrappedDEKRecovery, encryptionMetadata sql.NullString
	var encryptedContent []byte
	var encryptedFolderPath sql.NullString
	// Summary fields
	var summary, encryptedSummary, contentHash, summaryGeneratedAt sql.NullString
	// Journal fields
	var noteType, journalDate sql.NullString

	err := db.QueryRow(`
		SELECT id, title, content, folder_path, version, color, created_at, updated_at,
		       encrypted_content, content_encrypted, encrypted_title, title_encrypted,
		       wrapped_dek, wrapped_dek_recovery, encryption_version, encryption_metadata,
		       encrypted_folder_path,
		       summary, encrypted_summary, summary_encrypted, content_hash, summary_generated_at,
		       note_type, journal_date, ai_enabled
		FROM notes
		WHERE id = ? AND user_id = ? AND is_deleted = 0
	`, id, userID).Scan(
		&note.ID, &note.Title, &content, &note.FolderPath, &note.Version, &note.Color,
		&createdAt, &updatedAt,
		&encryptedContent, &note.ContentEncrypted, &encryptedTitle, &note.TitleEncrypted,
		&wrappedDEK, &wrappedDEKRecovery, &note.EncryptionVersion, &encryptionMetadata,
		&encryptedFolderPath,
		&summary, &encryptedSummary, &note.SummaryEncrypted, &contentHash, &summaryGeneratedAt,
		&noteType, &journalDate, &note.AIEnabled,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get note: %w", err)
	}

	parsedCreatedAt, err := parseRFC3339Timestamp(createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at for note %s: %w", note.ID, err)
	}
	parsedUpdatedAt, err := parseRFC3339Timestamp(updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at for note %s: %w", note.ID, err)
	}
	note.CreatedAt = parsedCreatedAt
	note.UpdatedAt = parsedUpdatedAt

	// Handle nullable fields
	if content.Valid {
		note.Content = content.String
	}
	if encryptedContent != nil {
		note.EncryptedContent = encryptedContent
	}
	if encryptedTitle.Valid {
		title := encryptedTitle.String
		note.EncryptedTitle = &title
	}
	if wrappedDEK.Valid {
		note.WrappedDEK = wrappedDEK.String
	}
	if wrappedDEKRecovery.Valid {
		note.WrappedDEKRecovery = wrappedDEKRecovery.String
	}
	if encryptionMetadata.Valid {
		note.EncryptionMetadata = encryptionMetadata.String
	}
	if encryptedFolderPath.Valid {
		note.EncryptedFolderPath = &encryptedFolderPath.String
	}
	// Summary fields
	if summary.Valid {
		note.Summary = &summary.String
	}
	if encryptedSummary.Valid {
		note.EncryptedSummary = &encryptedSummary.String
	}
	if contentHash.Valid {
		note.ContentHash = &contentHash.String
	}
	if summaryGeneratedAt.Valid {
		parsedSummaryAt, err := parseRFC3339Timestamp(summaryGeneratedAt.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse summary_generated_at for note %s: %w", note.ID, err)
		}
		note.SummaryGeneratedAt = &parsedSummaryAt
	}
	// Journal fields
	if noteType.Valid {
		note.NoteType = noteType.String
	} else {
		note.NoteType = NoteTypeNote // Default for legacy notes
	}
	if journalDate.Valid {
		note.JournalDate = &journalDate.String
	}
	note.UserID = userID

	return &note, nil
}

// GetNoteByTitle retrieves a note by its normalized title.
// If multiple notes with the same title exist (across folders), returns the most recently updated one.
// Uses deterministic ordering (updated_at DESC, id ASC) for consistent wikilink resolution.
func (db *DB) GetNoteByTitle(userID int, title string) (*Note, error) {
	titleNorm := utils.NormalizeTitle(title)
	var note Note
	var createdAt, updatedAt string

	err := db.QueryRow(`
		SELECT id, title, content, folder_path, version, color, created_at, updated_at
		FROM notes
		WHERE title_norm = ? AND user_id = ? AND is_deleted = 0
		ORDER BY updated_at DESC, id ASC
		LIMIT 1
	`, titleNorm, userID).Scan(&note.ID, &note.Title, &note.Content, &note.FolderPath, &note.Version, &note.Color, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get note by title: %w", err)
	}

	parsedCreatedAt, err := parseRFC3339Timestamp(createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at for note %s: %w", note.ID, err)
	}
	parsedUpdatedAt, err := parseRFC3339Timestamp(updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at for note %s: %w", note.ID, err)
	}
	note.CreatedAt = parsedCreatedAt
	note.UpdatedAt = parsedUpdatedAt

	return &note, nil
}

// GetNoteByTitleInFolder retrieves a note by title within a specific folder.
// Uses deterministic ordering in case of legacy duplicates.
func (db *DB) GetNoteByTitleInFolder(userID int, title, folderPath string) (*Note, error) {
	titleNorm := utils.NormalizeTitle(title)
	var note Note
	var createdAt, updatedAt string

	err := db.QueryRow(`
		SELECT id, title, content, folder_path, version, color, created_at, updated_at
		FROM notes
		WHERE title_norm = ? AND user_id = ? AND folder_path = ? AND is_deleted = 0
		ORDER BY updated_at DESC, id ASC
		LIMIT 1
	`, titleNorm, userID, folderPath).Scan(&note.ID, &note.Title, &note.Content, &note.FolderPath, &note.Version, &note.Color, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get note by title in folder: %w", err)
	}

	parsedCreatedAt, err := parseRFC3339Timestamp(createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at for note %s: %w", note.ID, err)
	}
	parsedUpdatedAt, err := parseRFC3339Timestamp(updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at for note %s: %w", note.ID, err)
	}
	note.CreatedAt = parsedCreatedAt
	note.UpdatedAt = parsedUpdatedAt

	return &note, nil
}

// GetNotesByTitle retrieves all notes with a given title across all folders.
func (db *DB) GetNotesByTitle(userID int, title string) ([]Note, error) {
	titleNorm := utils.NormalizeTitle(title)

	rows, err := db.Query(`
		SELECT id, title, content, folder_path, version, color, created_at, updated_at
		FROM notes
		WHERE title_norm = ? AND user_id = ? AND is_deleted = 0
		ORDER BY updated_at DESC
	`, titleNorm, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notes by title: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		var createdAt, updatedAt string
		if err := rows.Scan(&note.ID, &note.Title, &note.Content, &note.FolderPath, &note.Version, &note.Color, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		parsedCreatedAt, err := parseRFC3339Timestamp(createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at for note %s: %w", note.ID, err)
		}
		parsedUpdatedAt, err := parseRFC3339Timestamp(updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at for note %s: %w", note.ID, err)
		}
		note.CreatedAt = parsedCreatedAt
		note.UpdatedAt = parsedUpdatedAt
		notes = append(notes, note)
	}

	return notes, rows.Err()
}

// UpdateNote updates a note's content, title, and optionally folder_path.
// If folderPath is empty, the folder_path is not changed.
// Returns ErrVersionMismatch if the version doesn't match.
func (db *DB) UpdateNote(userID int, id, title, content, folderPath string, expectedVersion int) (*Note, error) {
	titleNorm := utils.NormalizeTitle(title)
	now := time.Now().UTC()

	// If folderPath is provided, update it; otherwise keep the existing value
	var result sql.Result
	var err error
	if folderPath != "" {
		result, err = db.Exec(`
			UPDATE notes
			SET title = ?, title_norm = ?, content = ?, folder_path = ?, version = version + 1, updated_at = ?
			WHERE id = ? AND user_id = ? AND version = ? AND is_deleted = 0
		`, title, titleNorm, content, folderPath, now.Format(time.RFC3339), id, userID, expectedVersion)
	} else {
		result, err = db.Exec(`
			UPDATE notes
			SET title = ?, title_norm = ?, content = ?, version = version + 1, updated_at = ?
			WHERE id = ? AND user_id = ? AND version = ? AND is_deleted = 0
		`, title, titleNorm, content, now.Format(time.RFC3339), id, userID, expectedVersion)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update note: %w", err)
	}

	rows, err := rowsAffectedCount(result, "failed to check rows affected")
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		// Check if note exists
		existing, err := db.GetNote(userID, id)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, ErrNotFound
		}
		return nil, ErrVersionMismatch
	}

	return db.GetNote(userID, id)
}

// UpdateNoteTitle updates only the title of a note
func (db *DB) UpdateNoteTitle(userID int, id, newTitle string, expectedVersion int) (*Note, error) {
	titleNorm := utils.NormalizeTitle(newTitle)
	now := time.Now().UTC()

	result, err := db.Exec(`
		UPDATE notes
		SET title = ?, title_norm = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND user_id = ? AND version = ? AND is_deleted = 0
	`, newTitle, titleNorm, now.Format(time.RFC3339), id, userID, expectedVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to update note title: %w", err)
	}

	rows, err := rowsAffectedCount(result, "failed to check rows affected")
	if err != nil {
		return nil, err
	}

	if rows == 0 {
		existing, err := db.GetNote(userID, id)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, ErrNotFound
		}
		return nil, ErrVersionMismatch
	}

	return db.GetNote(userID, id)
}

// DeleteNote performs a soft delete on a note.
func (db *DB) DeleteNote(userID int, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := db.Exec(`
		UPDATE notes SET is_deleted = 1, deleted_at = ?, updated_at = ?
		WHERE id = ? AND user_id = ? AND is_deleted = 0
	`, now, now, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	rows, err := rowsAffectedCount(result, "failed to check rows affected")
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// GetNotesByIDs retrieves multiple notes by their IDs in a single query.
// This is more efficient than calling GetNote multiple times (avoids N+1 query problem).
// Returns a map of note ID to Note for easy lookup.
func (db *DB) GetNotesByIDs(userID int, ids []string) (map[string]*Note, error) {
	if len(ids) == 0 {
		return make(map[string]*Note), nil
	}

	// Create args slice: userID + all IDs
	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, userID)
	for _, id := range ids {
		args = append(args, id)
	}

	// Build the query with proper placeholders for SQLite
	// SQLite needs separate ? for each ID in the IN clause
	query := `
		SELECT id, title, content, folder_path, version, created_at, updated_at
		FROM notes
		WHERE user_id = ? AND is_deleted = 0 AND id IN (`
	for i := range ids {
		if i > 0 {
			query += ", "
		}
		query += "?"
	}
	query += ")"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get notes by IDs: %w", err)
	}
	defer rows.Close()

	notes := make(map[string]*Note)
	for rows.Next() {
		var note Note
		var createdAt, updatedAt string
		if err := rows.Scan(&note.ID, &note.Title, &note.Content, &note.FolderPath, &note.Version, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		parsedCreatedAt, err := parseRFC3339Timestamp(createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at for note %s: %w", note.ID, err)
		}
		parsedUpdatedAt, err := parseRFC3339Timestamp(updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at for note %s: %w", note.ID, err)
		}
		note.CreatedAt = parsedCreatedAt
		note.UpdatedAt = parsedUpdatedAt
		notes[note.ID] = &note
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notes: %w", err)
	}

	return notes, nil
}
