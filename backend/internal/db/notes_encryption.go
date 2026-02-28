package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/xela-io/xelanote/internal/utils"
)

// CreateEncryptedNote creates a new encrypted note with wrapped DEK.
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
	return db.CreateEncryptedNoteWithID(
		userID,
		"",
		title,
		encryptedTitle,
		titleEncrypted,
		encryptedContent,
		wrappedDEK,
		encryptionMetadata,
		folderPath,
	)
}

// CreateEncryptedNoteWithID creates a new encrypted note with optional client-provided ID.
func (db *DB) CreateEncryptedNoteWithID(
	userID int,
	noteID string,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	folderPath string,
) (*Note, error) {
	id := noteID
	if id == "" {
		id = uuid.New().String()
	}
	titleNorm := utils.NormalizeTitle(title)
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
	titleNorm := utils.NormalizeTitle(title)
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
	return db.CreateEncryptedJournalNoteWithID(
		userID,
		"",
		title,
		encryptedTitle,
		titleEncrypted,
		encryptedContent,
		wrappedDEK,
		encryptionMetadata,
		folderPath,
		journalDate,
	)
}

// CreateEncryptedJournalNoteWithID creates a new encrypted journal note for a specific date
// with optional client-provided ID.
func (db *DB) CreateEncryptedJournalNoteWithID(
	userID int,
	noteID string,
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

	id := noteID
	if id == "" {
		id = uuid.New().String()
	}
	titleNorm := utils.NormalizeTitle(title)
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

// UpdateEncryptedNote updates an existing note with encrypted content.
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
	titleNorm := utils.NormalizeTitle(title)
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

// DecryptNote atomically clears all encryption fields and sets plaintext content.
// Requires the note to be currently encrypted (content_encrypted = 1).
// Uses optimistic locking with version field.
func (db *DB) DecryptNote(userID int, id, title, content string, expectedVersion int) (*Note, error) {
	titleNorm := utils.NormalizeTitle(title)
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
		if !existing.ContentEncrypted {
			return nil, fmt.Errorf("note is not encrypted")
		}
		return nil, ErrVersionMismatch
	}

	return db.GetNote(userID, id)
}

// ClearPlaintextContent clears the plaintext content column after encryption.
func (db *DB) ClearPlaintextContent(noteID string) error {
	_, err := db.Exec("UPDATE notes SET content = NULL WHERE id = ?", noteID)
	return err
}

// InsertNoteKeyword adds a keyword to a note (for opt-in searchable keywords).
func (db *DB) InsertNoteKeyword(noteID, keyword string) error {
	_, err := db.Exec(`
		INSERT INTO note_keywords (note_id, keyword)
		VALUES (?, ?)
		ON CONFLICT(note_id, keyword) DO NOTHING
	`, noteID, keyword)
	return err
}

// InsertNoteKeywords adds multiple keywords to a note in a single batch INSERT.
func (db *DB) InsertNoteKeywords(noteID string, keywords []string) error {
	if len(keywords) == 0 {
		return nil
	}
	query := "INSERT INTO note_keywords (note_id, keyword) VALUES "
	args := make([]interface{}, 0, len(keywords)*2)
	for i, kw := range keywords {
		if i > 0 {
			query += ","
		}
		query += "(?,?)"
		args = append(args, noteID, kw)
	}
	query += " ON CONFLICT(note_id, keyword) DO NOTHING"
	_, err := db.Exec(query, args...)
	return err
}

// DeleteNoteKeywords removes all keywords for a note.
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
