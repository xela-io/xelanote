package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Note represents a note in the database.
type Note struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	FolderPath   string     `json:"folder_path"`
	Version      int        `json:"version"`
	DisplayOrder int        `json:"display_order"`
	Color        *string    `json:"color,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	// Encryption fields
	EncryptedContent   []byte  `json:"encrypted_content,omitempty"`
	ContentEncrypted   bool    `json:"content_encrypted"`
	EncryptedTitle     *string `json:"encrypted_title,omitempty"`
	TitleEncrypted     bool    `json:"title_encrypted"`
	WrappedDEK         string  `json:"wrapped_dek,omitempty"`
	EncryptionVersion  int     `json:"encryption_version"`
	EncryptionMetadata string  `json:"encryption_metadata,omitempty"`
	// Summary fields (LLM-generated)
	Summary            *string    `json:"summary,omitempty"`
	EncryptedSummary   *string    `json:"encrypted_summary,omitempty"`
	SummaryEncrypted   bool       `json:"summary_encrypted"`
	ContentHash        *string    `json:"content_hash,omitempty"`
	SummaryGeneratedAt *time.Time `json:"summary_generated_at,omitempty"`
	// Journal fields
	NoteType    string  `json:"note_type,omitempty"`    // "note" or "journal"
	JournalDate *string `json:"journal_date,omitempty"` // YYYY-MM-DD for journal notes
	// AI/Claude API fields
	AIEnabled bool `json:"ai_enabled"` // true = Cloud-KI (Claude) erlaubt
	UserID    int  `json:"-"`          // Not exported to JSON, used internally
	// Sharing fields (populated by ListNotesByFolder UNION)
	IsShared bool `json:"is_shared,omitempty"` // true if this is a placed shared note
}

// NoteWithBacklinks extends Note with backlink information.
type NoteWithBacklinks struct {
	Note
	BacklinkCount int `json:"backlink_count"`
}

// CreateNote creates a new note and returns it.
// ai_enabled is inherited from folder's ai_enabled_default, except for root notes (always false).
func (db *DB) CreateNote(userID int, title, content, folderPath string) (*Note, error) {
	id := uuid.New().String()
	titleNorm := NormalizeTitle(title)
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
	var content, encryptedTitle, wrappedDEK, encryptionMetadata sql.NullString
	var encryptedContent []byte
	// Summary fields
	var summary, encryptedSummary, contentHash, summaryGeneratedAt sql.NullString
	// Journal fields
	var noteType, journalDate sql.NullString

	err := db.QueryRow(`
		SELECT id, title, content, folder_path, version, color, created_at, updated_at,
		       encrypted_content, content_encrypted, encrypted_title, title_encrypted,
		       wrapped_dek, encryption_version, encryption_metadata,
		       summary, encrypted_summary, summary_encrypted, content_hash, summary_generated_at,
		       note_type, journal_date, ai_enabled
		FROM notes
		WHERE id = ? AND user_id = ? AND is_deleted = 0
	`, id, userID).Scan(
		&note.ID, &note.Title, &content, &note.FolderPath, &note.Version, &note.Color,
		&createdAt, &updatedAt,
		&encryptedContent, &note.ContentEncrypted, &encryptedTitle, &note.TitleEncrypted,
		&wrappedDEK, &note.EncryptionVersion, &encryptionMetadata,
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
	if encryptionMetadata.Valid {
		note.EncryptionMetadata = encryptionMetadata.String
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
	titleNorm := NormalizeTitle(title)
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
	titleNorm := NormalizeTitle(title)
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
	titleNorm := NormalizeTitle(title)

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
	titleNorm := NormalizeTitle(title)
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

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
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

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// ListNotes returns a paginated list of notes (including journal notes).
// Journal notes appear in the /Journal folder in the tree view.
func (db *DB) ListNotes(userID int, limit int, cursor string) ([]Note, string, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	var rows *sql.Rows
	var err error

	// Cursor is the last note's updated_at + id for stable pagination
	if cursor == "" {
		rows, err = db.Query(`
			SELECT id, title, content, folder_path, version, display_order, color, created_at, updated_at,
			       encrypted_content, content_encrypted, encrypted_title, title_encrypted,
			       wrapped_dek, encryption_version, encryption_metadata,
			       note_type, journal_date, ai_enabled
			FROM notes
			WHERE user_id = ? AND is_deleted = 0 AND COALESCE(note_type, 'note') = 'note'
			ORDER BY updated_at DESC, id DESC
			LIMIT ?
		`, userID, limit+1)
	} else {
		// Parse cursor (format: timestamp|id)
		parts := strings.SplitN(cursor, "|", 2)
		if len(parts) != 2 {
			return nil, "", fmt.Errorf("invalid cursor format")
		}
		cursorTime, cursorID := parts[0], parts[1]
		rows, err = db.Query(`
			SELECT id, title, content, folder_path, version, display_order, color, created_at, updated_at,
			       encrypted_content, content_encrypted, encrypted_title, title_encrypted,
			       wrapped_dek, encryption_version, encryption_metadata,
			       note_type, journal_date, ai_enabled
			FROM notes
			WHERE user_id = ? AND is_deleted = 0 AND COALESCE(note_type, 'note') = 'note'
			  AND (updated_at < ? OR (updated_at = ? AND id < ?))
			ORDER BY updated_at DESC, id DESC
			LIMIT ?
		`, userID, cursorTime, cursorTime, cursorID, limit+1)
	}

	if err != nil {
		return nil, "", fmt.Errorf("failed to list notes: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		var createdAt, updatedAt string
		var content, encryptedTitle, wrappedDEK, encryptionMetadata sql.NullString
		var encryptedContent []byte
		var noteType, journalDate sql.NullString

		if err := rows.Scan(
			&note.ID, &note.Title, &content, &note.FolderPath, &note.Version, &note.DisplayOrder, &note.Color, &createdAt, &updatedAt,
			&encryptedContent, &note.ContentEncrypted, &encryptedTitle, &note.TitleEncrypted,
			&wrappedDEK, &note.EncryptionVersion, &encryptionMetadata,
			&noteType, &journalDate, &note.AIEnabled,
		); err != nil {
			return nil, "", fmt.Errorf("failed to scan note: %w", err)
		}
		parsedCreatedAt, err := parseRFC3339Timestamp(createdAt)
		if err != nil {
			return nil, "", fmt.Errorf("failed to parse created_at for note %s: %w", note.ID, err)
		}
		parsedUpdatedAt, err := parseRFC3339Timestamp(updatedAt)
		if err != nil {
			return nil, "", fmt.Errorf("failed to parse updated_at for note %s: %w", note.ID, err)
		}
		note.CreatedAt = parsedCreatedAt
		note.UpdatedAt = parsedUpdatedAt
		note.Content = content.String
		note.EncryptedContent = encryptedContent
		if encryptedTitle.Valid {
			note.EncryptedTitle = &encryptedTitle.String
		}
		note.WrappedDEK = wrappedDEK.String
		note.EncryptionMetadata = encryptionMetadata.String
		if noteType.Valid {
			note.NoteType = noteType.String
		} else {
			note.NoteType = NoteTypeNote
		}
		if journalDate.Valid {
			note.JournalDate = &journalDate.String
		}
		note.UserID = userID
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("error iterating notes: %w", err)
	}

	// Check for next page
	var nextCursor string
	if len(notes) > limit {
		lastNote := notes[limit-1]
		nextCursor = fmt.Sprintf("%s|%s", lastNote.UpdatedAt.Format(time.RFC3339), lastNote.ID)
		notes = notes[:limit]
	}

	return notes, nextCursor, nil
}

// ListNotesByFolder returns notes in a specific folder.
// For the /Journal folder, returns journal notes sorted by date (newest first).
// For other folders, excludes journal notes from the list.
// Also includes shared notes that the user has placed into this folder (with active share check).
func (db *DB) ListNotesByFolder(userID int, folderPath string) ([]Note, error) {
	var query string

	if folderPath == "/Journal" {
		// Journal folder: show journal notes, sorted by journal_date descending
		// No shared placements in Journal folder
		query = `
			SELECT id, title, content, folder_path, version, display_order, color, created_at, updated_at,
			       encrypted_content, content_encrypted, encrypted_title, title_encrypted,
			       wrapped_dek, encryption_version, encryption_metadata,
			       note_type, journal_date, ai_enabled, 0 as is_shared
			FROM notes
			WHERE folder_path = ? AND user_id = ? AND is_deleted = 0
			  AND note_type = 'journal'
			ORDER BY journal_date DESC`
	} else if folderPath == "/Rezepte" {
		// Rezepte folder: show recipe notes, sorted by updated_at descending
		// No shared placements in Rezepte folder
		query = `
			SELECT id, title, content, folder_path, version, display_order, color, created_at, updated_at,
			       encrypted_content, content_encrypted, encrypted_title, title_encrypted,
			       wrapped_dek, encryption_version, encryption_metadata,
			       note_type, journal_date, ai_enabled, 0 as is_shared
			FROM notes
			WHERE user_id = ? AND note_type = 'recipe' AND is_deleted = 0
			ORDER BY updated_at DESC`
	} else {
		// Other folders: own notes + placed shared notes with active share check
		query = `
			SELECT id, title, content, folder_path, version, display_order, color, created_at, updated_at,
			       encrypted_content, content_encrypted, encrypted_title, title_encrypted,
			       wrapped_dek, encryption_version, encryption_metadata,
			       note_type, journal_date, ai_enabled, is_shared
			FROM (
				SELECT id, title, content, folder_path, version, display_order, color, created_at, updated_at,
				       encrypted_content, content_encrypted, encrypted_title, title_encrypted,
				       wrapped_dek, encryption_version, encryption_metadata,
				       note_type, journal_date, ai_enabled, 0 as is_shared
				FROM notes
				WHERE folder_path = ? AND user_id = ? AND is_deleted = 0
				  AND COALESCE(note_type, 'note') = 'note'

				UNION ALL

				SELECT n.id, n.title, n.content, n.folder_path, n.version, 99999 as display_order,
				       n.color, n.created_at, n.updated_at,
				       n.encrypted_content, n.content_encrypted, n.encrypted_title, n.title_encrypted,
				       n.wrapped_dek, n.encryption_version, n.encryption_metadata,
				       n.note_type, n.journal_date, n.ai_enabled, 1 as is_shared
				FROM notes n
				JOIN shared_note_placements p ON p.note_id = n.id AND p.user_id = ?
				JOIN folders pf ON pf.id = p.folder_id
				WHERE pf.path = ? AND n.is_deleted = 0 AND n.content_encrypted = 0
				  AND (
				    EXISTS (SELECT 1 FROM note_shares ns WHERE ns.note_id = n.id AND ns.shared_with_user_id = ?)
				    OR EXISTS (SELECT 1 FROM folder_shares fs
				               JOIN folders ff ON ff.id = fs.folder_id
				               WHERE fs.shared_with_user_id = ? AND ff.path = n.folder_path)
				  )
			)
			ORDER BY display_order ASC, title ASC`
	}

	var rows *sql.Rows
	var err error
	if folderPath == "/Journal" {
		rows, err = db.Query(query, folderPath, userID)
	} else if folderPath == "/Rezepte" {
		rows, err = db.Query(query, userID)
	} else {
		rows, err = db.Query(query, folderPath, userID, userID, folderPath, userID, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list notes by folder: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		var createdAt, updatedAt string
		var content, encryptedTitle, wrappedDEK, encryptionMetadata sql.NullString
		var encryptedContent []byte
		var noteType, journalDate sql.NullString
		var isShared int

		if err := rows.Scan(
			&note.ID, &note.Title, &content, &note.FolderPath, &note.Version, &note.DisplayOrder, &note.Color, &createdAt, &updatedAt,
			&encryptedContent, &note.ContentEncrypted, &encryptedTitle, &note.TitleEncrypted,
			&wrappedDEK, &note.EncryptionVersion, &encryptionMetadata,
			&noteType, &journalDate, &note.AIEnabled, &isShared,
		); err != nil {
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
		note.Content = content.String
		note.EncryptedContent = encryptedContent
		if encryptedTitle.Valid {
			note.EncryptedTitle = &encryptedTitle.String
		}
		note.WrappedDEK = wrappedDEK.String
		note.EncryptionMetadata = encryptionMetadata.String
		if noteType.Valid {
			note.NoteType = noteType.String
		} else {
			note.NoteType = NoteTypeNote
		}
		if journalDate.Valid {
			note.JournalDate = &journalDate.String
		}
		note.UserID = userID
		note.IsShared = isShared == 1
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notes: %w", err)
	}

	return notes, nil
}

// GetFolders returns all unique folder paths.
func (db *DB) GetFolders(userID int) ([]string, error) {
	rows, err := db.Query(`
		SELECT DISTINCT folder_path
		FROM notes
		WHERE user_id = ? AND is_deleted = 0
		ORDER BY folder_path ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get folders: %w", err)
	}
	defer rows.Close()

	var folders []string
	for rows.Next() {
		var folder string
		if err := rows.Scan(&folder); err != nil {
			return nil, fmt.Errorf("failed to scan folder: %w", err)
		}
		folders = append(folders, folder)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating folders: %w", err)
	}

	return folders, nil
}

// FolderInfo contains folder path and note count.
type FolderInfo struct {
	Path      string `json:"path"`
	NoteCount int    `json:"note_count"`
}

// GetFoldersWithCounts returns all unique folder paths with note counts.
func (db *DB) GetFoldersWithCounts(userID int) ([]FolderInfo, error) {
	rows, err := db.Query(`
		SELECT folder_path, COUNT(*) as note_count
		FROM notes
		WHERE user_id = ? AND is_deleted = 0
		GROUP BY folder_path
		ORDER BY folder_path ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get folders with counts: %w", err)
	}
	defer rows.Close()

	var folders []FolderInfo
	for rows.Next() {
		var f FolderInfo
		if err := rows.Scan(&f.Path, &f.NoteCount); err != nil {
			return nil, fmt.Errorf("failed to scan folder info: %w", err)
		}
		folders = append(folders, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating folder info: %w", err)
	}

	return folders, nil
}

// UpdateNoteTitle updates only the title of a note
func (db *DB) UpdateNoteTitle(userID int, id, newTitle string, expectedVersion int) (*Note, error) {
	titleNorm := NormalizeTitle(newTitle)
	now := time.Now().UTC()

	result, err := db.Exec(`
		UPDATE notes
		SET title = ?, title_norm = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND user_id = ? AND version = ? AND is_deleted = 0
	`, newTitle, titleNorm, now.Format(time.RFC3339), id, userID, expectedVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to update note title: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
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

// --- Trash Management Functions ---

// ListDeletedNotes returns a paginated list of soft-deleted notes.
// Notes are sorted by deleted_at DESC (most recently deleted first).
func (db *DB) ListDeletedNotes(userID int, limit int, cursor string) ([]Note, string, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	var rows *sql.Rows
	var err error

	// Cursor is the last note's deleted_at + id for stable pagination
	if cursor == "" {
		rows, err = db.Query(`
			SELECT id, title, content, folder_path, version, color, created_at, updated_at, deleted_at
			FROM notes
			WHERE user_id = ? AND is_deleted = 1
			ORDER BY deleted_at DESC, id DESC
			LIMIT ?
		`, userID, limit+1)
	} else {
		// Parse cursor (format: timestamp|id)
		parts := strings.SplitN(cursor, "|", 2)
		if len(parts) != 2 {
			return nil, "", fmt.Errorf("invalid cursor format")
		}
		cursorTime, cursorID := parts[0], parts[1]
		rows, err = db.Query(`
			SELECT id, title, content, folder_path, version, color, created_at, updated_at, deleted_at
			FROM notes
			WHERE user_id = ? AND is_deleted = 1
			  AND (deleted_at < ? OR (deleted_at = ? AND id < ?))
			ORDER BY deleted_at DESC, id DESC
			LIMIT ?
		`, userID, cursorTime, cursorTime, cursorID, limit+1)
	}

	if err != nil {
		return nil, "", fmt.Errorf("failed to list deleted notes: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		var createdAt, updatedAt string
		var deletedAt sql.NullString
		if err := rows.Scan(&note.ID, &note.Title, &note.Content, &note.FolderPath, &note.Version, &note.Color, &createdAt, &updatedAt, &deletedAt); err != nil {
			return nil, "", fmt.Errorf("failed to scan note: %w", err)
		}
		parsedCreatedAt, err := parseRFC3339Timestamp(createdAt)
		if err != nil {
			return nil, "", fmt.Errorf("failed to parse created_at for note %s: %w", note.ID, err)
		}
		parsedUpdatedAt, err := parseRFC3339Timestamp(updatedAt)
		if err != nil {
			return nil, "", fmt.Errorf("failed to parse updated_at for note %s: %w", note.ID, err)
		}
		note.CreatedAt = parsedCreatedAt
		note.UpdatedAt = parsedUpdatedAt
		if deletedAt.Valid && deletedAt.String != "" {
			parsedDeletedAt, err := parseRFC3339Timestamp(deletedAt.String)
			if err != nil {
				return nil, "", fmt.Errorf("failed to parse deleted_at for note %s: %w", note.ID, err)
			}
			note.DeletedAt = &parsedDeletedAt
		}
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("error iterating deleted notes: %w", err)
	}

	// Check for next page
	var nextCursor string
	if len(notes) > limit {
		lastNote := notes[limit-1]
		if lastNote.DeletedAt != nil {
			nextCursor = fmt.Sprintf("%s|%s", lastNote.DeletedAt.Format(time.RFC3339), lastNote.ID)
		}
		notes = notes[:limit]
	}

	return notes, nextCursor, nil
}

// RestoreNote restores a soft-deleted note.
func (db *DB) RestoreNote(userID int, id string) (*Note, error) {
	result, err := db.Exec(`
		UPDATE notes
		SET is_deleted = 0, deleted_at = NULL, updated_at = ?
		WHERE id = ? AND user_id = ? AND is_deleted = 1
	`, time.Now().UTC().Format(time.RFC3339), id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to restore note: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return nil, ErrNotFound
	}

	return db.GetNote(userID, id)
}

// PermanentlyDeleteNote performs a hard delete on a note.
// Safety: Only deletes notes that are already soft-deleted (is_deleted = 1).
func (db *DB) PermanentlyDeleteNote(userID int, id string) error {
	result, err := db.Exec(`
		DELETE FROM notes
		WHERE id = ? AND user_id = ? AND is_deleted = 1
	`, id, userID)
	if err != nil {
		return fmt.Errorf("failed to permanently delete note: %w", err)
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

// GetDeletedNotesCount returns the count of soft-deleted notes for a user.
func (db *DB) GetDeletedNotesCount(userID int) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM notes
		WHERE user_id = ? AND is_deleted = 1
	`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get deleted notes count: %w", err)
	}
	return count, nil
}

// EmptyTrash permanently deletes all soft-deleted notes for a user.
func (db *DB) EmptyTrash(userID int) (int, error) {
	result, err := db.Exec(`
		DELETE FROM notes
		WHERE user_id = ? AND is_deleted = 1
	`, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to empty trash: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to check rows affected: %w", err)
	}

	return int(rows), nil
}

// CreateEncryptedNote creates a new encrypted note with wrapped DEK
func (db *DB) CreateEncryptedNote(
	userID int,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	folderPath string,
) (*Note, error) {
	id := uuid.New().String()
	titleNorm := NormalizeTitle(title)
	now := time.Now().UTC()

	_, err := db.Exec(`
		INSERT INTO notes (
			id, title, title_norm, encrypted_title, title_encrypted,
			content, encrypted_content, content_encrypted,
			wrapped_dek, encryption_version, encryption_metadata,
			folder_path, user_id, version, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, '', ?, 1, ?, 1, ?, ?, ?, 1, ?, ?)
	`, id, title, titleNorm, encryptedTitle, titleEncrypted,
		encryptedContent, wrappedDEK, encryptionMetadata,
		folderPath, userID, now.Format(time.RFC3339), now.Format(time.RFC3339))

	if err != nil {
		return nil, fmt.Errorf("failed to create encrypted note: %w", err)
	}

	return db.GetNote(userID, id)
}

// CreateJournalNote creates a new plaintext journal note for a specific date.
// Automatically creates the /Journal folder if it doesn't exist.
func (db *DB) CreateJournalNote(userID int, title, content, folderPath, journalDate string) (*Note, error) {
	// Ensure Journal folder exists (idempotent)
	if folderPath == "/Journal" {
		_, err := db.CreateFolder(userID, "/Journal", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create journal folder: %w", err)
		}
	}

	id := uuid.New().String()
	titleNorm := NormalizeTitle(title)
	now := time.Now().UTC()

	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id,
		                  note_type, journal_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'journal', ?, ?, ?)
	`, id, title, titleNorm, content, folderPath, userID,
		journalDate, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("failed to create journal note: %w", err)
	}

	jd := journalDate
	return &Note{
		ID:          id,
		Title:       title,
		Content:     content,
		FolderPath:  folderPath,
		Version:     1,
		NoteType:    NoteTypeJournal,
		JournalDate: &jd,
		CreatedAt:   now,
		UpdatedAt:   now,
		UserID:      userID,
	}, nil
}

// CreateEncryptedJournalNote creates a new encrypted journal note for a specific date.
// Automatically creates the /Journal folder if it doesn't exist.
func (db *DB) CreateEncryptedJournalNote(
	userID int,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	folderPath string,
	journalDate string,
) (*Note, error) {
	// Ensure Journal folder exists (idempotent)
	if folderPath == "/Journal" {
		_, err := db.CreateFolder(userID, "/Journal", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create journal folder: %w", err)
		}
	}

	id := uuid.New().String()
	titleNorm := NormalizeTitle(title)
	now := time.Now().UTC()

	_, err := db.Exec(`
		INSERT INTO notes (
			id, title, title_norm, encrypted_title, title_encrypted,
			content, encrypted_content, content_encrypted,
			wrapped_dek, encryption_version, encryption_metadata,
			folder_path, user_id, version,
			note_type, journal_date,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, '', ?, 1, ?, 1, ?, ?, ?, 1, 'journal', ?, ?, ?)
	`, id, title, titleNorm, encryptedTitle, titleEncrypted,
		encryptedContent, wrappedDEK, encryptionMetadata,
		folderPath, userID, journalDate,
		now.Format(time.RFC3339), now.Format(time.RFC3339))

	if err != nil {
		return nil, fmt.Errorf("failed to create encrypted journal note: %w", err)
	}

	return db.GetNote(userID, id)
}

// UpdateEncryptedNote updates an existing note with encrypted content
func (db *DB) UpdateEncryptedNote(
	userID int,
	id string,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	folderPath string,
	expectedVersion int,
) (*Note, error) {
	titleNorm := NormalizeTitle(title)
	now := time.Now().UTC()

	// Build SQL based on whether folderPath is provided
	var result sql.Result
	var err error

	if folderPath != "" {
		// Update with folder_path
		result, err = db.Exec(`
			UPDATE notes
			SET title = ?, title_norm = ?, encrypted_title = ?, title_encrypted = ?,
			    content = '', encrypted_content = ?, content_encrypted = 1,
			    wrapped_dek = ?, encryption_version = 1, encryption_metadata = ?,
			    folder_path = ?, version = version + 1, updated_at = ?
			WHERE id = ? AND user_id = ? AND version = ? AND is_deleted = 0
		`, title, titleNorm, encryptedTitle, titleEncrypted,
			encryptedContent, wrappedDEK, encryptionMetadata,
			folderPath, now.Format(time.RFC3339), id, userID, expectedVersion)
	} else {
		// Update without changing folder_path
		result, err = db.Exec(`
			UPDATE notes
			SET title = ?, title_norm = ?, encrypted_title = ?, title_encrypted = ?,
			    content = '', encrypted_content = ?, content_encrypted = 1,
			    wrapped_dek = ?, encryption_version = 1, encryption_metadata = ?,
			    version = version + 1, updated_at = ?
			WHERE id = ? AND user_id = ? AND version = ? AND is_deleted = 0
		`, title, titleNorm, encryptedTitle, titleEncrypted,
			encryptedContent, wrappedDEK, encryptionMetadata,
			now.Format(time.RFC3339), id, userID, expectedVersion)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to update encrypted note: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
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

// DecryptNote atomically clears all encryption fields and sets plaintext content.
// Requires the note to be currently encrypted (content_encrypted = 1).
// Uses optimistic locking with version field.
func (db *DB) DecryptNote(userID int, id, title, content string, expectedVersion int) (*Note, error) {
	titleNorm := NormalizeTitle(title)
	now := time.Now().UTC()

	result, err := db.Exec(`
		UPDATE notes
		SET title = ?, title_norm = ?, content = ?,
		    content_encrypted = 0, encrypted_content = NULL,
		    encrypted_title = NULL, title_encrypted = 0,
		    wrapped_dek = NULL, encryption_version = 0, encryption_metadata = NULL,
		    encrypted_summary = NULL, summary_encrypted = 0,
		    version = version + 1, updated_at = ?
		WHERE id = ? AND user_id = ? AND version = ? AND is_deleted = 0 AND content_encrypted = 1
	`, title, titleNorm, content, now.Format(time.RFC3339), id, userID, expectedVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt note: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		existing, err := db.GetNote(userID, id)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, ErrNotFound
		}
		if !existing.ContentEncrypted {
			return nil, fmt.Errorf("note is not encrypted")
		}
		return nil, ErrVersionMismatch
	}

	return db.GetNote(userID, id)
}

// ClearPlaintextContent clears the plaintext content column after encryption
func (db *DB) ClearPlaintextContent(noteID string) error {
	_, err := db.Exec("UPDATE notes SET content = NULL WHERE id = ?", noteID)
	return err
}

// InsertNoteKeyword adds a keyword to a note (for opt-in searchable keywords)
func (db *DB) InsertNoteKeyword(noteID, keyword string) error {
	_, err := db.Exec(`
		INSERT INTO note_keywords (note_id, keyword)
		VALUES (?, ?)
		ON CONFLICT(note_id, keyword) DO NOTHING
	`, noteID, keyword)
	return err
}

// DeleteNoteKeywords removes all keywords for a note
func (db *DB) DeleteNoteKeywords(noteID string) error {
	_, err := db.Exec("DELETE FROM note_keywords WHERE note_id = ?", noteID)
	return err
}

// GetAllEncryptedNotesForUser retrieves all encrypted notes for a user.
// Returns notes where either content_encrypted or title_encrypted is true.
func (db *DB) GetAllEncryptedNotesForUser(userID int) ([]Note, error) {
	rows, err := db.Query(`
		SELECT id, title, content, folder_path, version, color, created_at, updated_at,
		       encrypted_content, content_encrypted, encrypted_title, title_encrypted,
		       wrapped_dek, encryption_version, encryption_metadata
		FROM notes
		WHERE user_id = ? AND is_deleted = 0 AND (content_encrypted = 1 OR title_encrypted = 1)
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query encrypted notes: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		var createdAt, updatedAt string
		var content, encryptedTitle, wrappedDEK, encryptionMetadata sql.NullString
		var encryptedContent []byte

		err := rows.Scan(
			&note.ID, &note.Title, &content, &note.FolderPath, &note.Version, &note.Color,
			&createdAt, &updatedAt,
			&encryptedContent, &note.ContentEncrypted, &encryptedTitle, &note.TitleEncrypted,
			&wrappedDEK, &note.EncryptionVersion, &encryptionMetadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan encrypted note: %w", err)
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
		if encryptionMetadata.Valid {
			note.EncryptionMetadata = encryptionMetadata.String
		}

		notes = append(notes, note)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating encrypted notes: %w", err)
	}

	return notes, nil
}

// UpdateNoteColor updates the color of a note.
// Pass nil to remove the color.
func (db *DB) UpdateNoteColor(userID int, noteID string, color *string) error {
	// Validate color format if provided
	if color != nil {
		if err := validateHexColor(*color); err != nil {
			return err
		}
	}

	result, err := db.Exec(`
		UPDATE notes
		SET color = ?, updated_at = ?
		WHERE id = ? AND user_id = ? AND is_deleted = 0
	`, color, time.Now().UTC().Format(time.RFC3339), noteID, userID)
	if err != nil {
		return fmt.Errorf("failed to update note color: %w", err)
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

// BulkUpdateWrappedDEKs updates wrapped_dek fields for multiple notes and versions in a transaction.
// noteUpdates maps noteID -> new_wrapped_dek
// versionUpdates maps versionID (as string) -> new_wrapped_dek
func (db *DB) BulkUpdateWrappedDEKs(userID int, noteUpdates map[string]string, versionUpdates map[string]string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update notes
	for noteID, newWrappedDEK := range noteUpdates {
		_, err := tx.Exec(`
			UPDATE notes
			SET wrapped_dek = ?
			WHERE id = ? AND user_id = ?
		`, newWrappedDEK, noteID, userID)
		if err != nil {
			return fmt.Errorf("failed to update wrapped_dek for note %s: %w", noteID, err)
		}
	}

	// Update note_versions
	for versionIDStr, newWrappedDEK := range versionUpdates {
		_, err := tx.Exec(`
			UPDATE note_versions
			SET wrapped_dek = ?
			WHERE id = ? AND user_id = ?
		`, newWrappedDEK, versionIDStr, userID)
		if err != nil {
			return fmt.Errorf("failed to update wrapped_dek for version %s: %w", versionIDStr, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BulkUpdateWrappedDEKsTx updates wrapped_dek fields within an existing transaction.
// Use this when atomically updating DEKs along with other operations (e.g., password change).
func (tx *Tx) BulkUpdateWrappedDEKsTx(userID int, noteUpdates map[string]string, versionUpdates map[string]string) error {
	// Update notes
	for noteID, newWrappedDEK := range noteUpdates {
		_, err := tx.Exec(`
			UPDATE notes
			SET wrapped_dek = ?
			WHERE id = ? AND user_id = ?
		`, newWrappedDEK, noteID, userID)
		if err != nil {
			return fmt.Errorf("failed to update wrapped_dek for note %s: %w", noteID, err)
		}
	}

	// Update note_versions
	for versionIDStr, newWrappedDEK := range versionUpdates {
		_, err := tx.Exec(`
			UPDATE note_versions
			SET wrapped_dek = ?
			WHERE id = ? AND user_id = ?
		`, newWrappedDEK, versionIDStr, userID)
		if err != nil {
			return fmt.Errorf("failed to update wrapped_dek for version %s: %w", versionIDStr, err)
		}
	}

	return nil
}

// --- Summary Functions ---

// ComputeContentHash computes a SHA256 hash of the content and returns the first 16 characters.
// Used for change detection to determine if content has been modified since last summary.
func ComputeContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])[:16]
}

// UpdateNoteSummary updates the summary fields for a note.
// For plaintext notes: sets summary, clears encrypted_summary, sets summary_encrypted=false
// For encrypted notes: sets encrypted_summary, clears summary, sets summary_encrypted=true
func (db *DB) UpdateNoteSummary(userID int, noteID, summary string, encrypted bool, generatedAt time.Time) error {
	var result sql.Result
	var err error

	if encrypted {
		// Encrypted note: store in encrypted_summary field
		result, err = db.Exec(`
			UPDATE notes
			SET summary = NULL, encrypted_summary = ?, summary_encrypted = 1, summary_generated_at = ?
			WHERE id = ? AND user_id = ? AND is_deleted = 0
		`, summary, generatedAt.Format(time.RFC3339), noteID, userID)
	} else {
		// Plaintext note: store in summary field
		result, err = db.Exec(`
			UPDATE notes
			SET summary = ?, encrypted_summary = NULL, summary_encrypted = 0, summary_generated_at = ?
			WHERE id = ? AND user_id = ? AND is_deleted = 0
		`, summary, generatedAt.Format(time.RFC3339), noteID, userID)
	}

	if err != nil {
		return fmt.Errorf("failed to update note summary: %w", err)
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

// UpdateNoteContentHash updates the content_hash for a note.
// Called when content is updated to track changes for summary regeneration.
func (db *DB) UpdateNoteContentHash(userID int, noteID, contentHash string) error {
	result, err := db.Exec(`
		UPDATE notes
		SET content_hash = ?
		WHERE id = ? AND user_id = ? AND is_deleted = 0
	`, contentHash, noteID, userID)
	if err != nil {
		return fmt.Errorf("failed to update content hash: %w", err)
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

// NoteNeedingSummary represents a note that needs summary generation.
type NoteNeedingSummary struct {
	ID          string
	UserID      int
	Content     string
	ContentHash *string
	SummaryHash *string // Hash when summary was last generated
}

// GetNotesNeedingSummary returns unencrypted notes that need summary generation.
// A note needs a summary if:
// - It has no summary (summary IS NULL AND summary_generated_at IS NULL)
// - OR its content has changed (content_hash differs from when summary was generated)
// Only returns unencrypted notes (content_encrypted = 0) as encrypted notes require frontend-triggered generation.
func (db *DB) GetNotesNeedingSummary(limit int) ([]NoteNeedingSummary, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.Query(`
		SELECT n.id, n.user_id, n.content, n.content_hash
		FROM notes n
		WHERE n.is_deleted = 0
		  AND n.content_encrypted = 0
		  AND (
		      n.summary_generated_at IS NULL
		      OR n.content_hash IS NULL
		      OR n.summary IS NULL
		  )
		ORDER BY n.updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query notes needing summary: %w", err)
	}
	defer rows.Close()

	var notes []NoteNeedingSummary
	for rows.Next() {
		var note NoteNeedingSummary
		var contentHash sql.NullString

		if err := rows.Scan(&note.ID, &note.UserID, &note.Content, &contentHash); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}

		if contentHash.Valid {
			note.ContentHash = &contentHash.String
		}

		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notes: %w", err)
	}

	return notes, nil
}

// ClearNoteSummary clears the summary for a note (used when content changes significantly).
func (db *DB) ClearNoteSummary(userID int, noteID string) error {
	_, err := db.Exec(`
		UPDATE notes
		SET summary = NULL, encrypted_summary = NULL, summary_generated_at = NULL
		WHERE id = ? AND user_id = ? AND is_deleted = 0
	`, noteID, userID)
	return err
}

// ============================================================================
// AI-Enabled Functions (Claude API Integration)
// ============================================================================

// UpdateNoteAIEnabled sets the ai_enabled flag for a note.
// ai_enabled=true means the note can be processed by Claude API (cloud).
// ai_enabled=false means only Ollama (local) can be used.
func (db *DB) UpdateNoteAIEnabled(userID int, noteID string, enabled bool) error {
	result, err := db.Exec(`
		UPDATE notes
		SET ai_enabled = ?, updated_at = ?
		WHERE id = ? AND user_id = ? AND is_deleted = 0
	`, boolToInt(enabled), time.Now().UTC().Format(time.RFC3339), noteID, userID)
	if err != nil {
		return fmt.Errorf("failed to update ai_enabled: %w", err)
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

// GetNoteAIEnabled returns the ai_enabled status for a note.
func (db *DB) GetNoteAIEnabled(userID int, noteID string) (bool, error) {
	var aiEnabled int
	err := db.QueryRow(`
		SELECT ai_enabled FROM notes
		WHERE id = ? AND user_id = ? AND is_deleted = 0
	`, noteID, userID).Scan(&aiEnabled)

	if err == sql.ErrNoRows {
		return false, ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("failed to get ai_enabled: %w", err)
	}
	return aiEnabled == 1, nil
}

// GetNoteTitlesAIEnabled returns titles of all ai_enabled notes for a user.
// Used for Link-Suggestions when using Claude API (only cross-reference enabled notes).
func (db *DB) GetNoteTitlesAIEnabled(userID int) ([]string, error) {
	rows, err := db.Query(`
		SELECT title FROM notes
		WHERE user_id = ? AND is_deleted = 0 AND ai_enabled = 1
		  AND COALESCE(note_type, 'note') = 'note'
		ORDER BY title
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query ai_enabled note titles: %w", err)
	}
	defer rows.Close()

	var titles []string
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			return nil, fmt.Errorf("failed to scan title: %w", err)
		}
		titles = append(titles, title)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating titles: %w", err)
	}

	return titles, nil
}

// ReorderNotes updates the display_order for notes within a folder.
// noteIDs is a list of note IDs (UUIDs) in the desired order.
func (d *DB) ReorderNotes(userID int, folderPath string, noteIDs []string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, noteID := range noteIDs {
		newOrder := i * 100
		_, err := tx.Exec(`
			UPDATE notes
			SET display_order = ?, updated_at = datetime('now')
			WHERE id = ? AND user_id = ? AND folder_path = ? AND is_deleted = 0
		`, newOrder, noteID, userID, folderPath)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// boolToInt converts a bool to SQLite integer (0 or 1).
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
