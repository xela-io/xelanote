package db

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CreateCanvasNote creates a new plaintext canvas note.
func (db *DB) CreateCanvasNote(userID int, title, content, folderPath string) (*Note, error) {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(`
		INSERT INTO notes (id, user_id, title, title_norm, content, folder_path, note_type, version, created_at, updated_at)
		VALUES (?, ?, ?, LOWER(?), ?, ?, 'canvas', 1, ?, ?)
	`, id, userID, title, title, content, folderPath, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create canvas note: %w", err)
	}

	return db.GetNote(userID, id)
}

// CreateEncryptedCanvasNote creates a new encrypted canvas note.
func (db *DB) CreateEncryptedCanvasNote(
	userID int,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	keywords []string,
	folderPath string,
) (*Note, error) {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	titleEnc := 0
	if titleEncrypted {
		titleEnc = 1
	}

	_, err := db.Exec(`
		INSERT INTO notes (id, user_id, title, title_norm, content, encrypted_content, content_encrypted,
			encrypted_title, title_encrypted, wrapped_dek, encryption_version, encryption_metadata,
			folder_path, note_type, version, created_at, updated_at)
		VALUES (?, ?, ?, LOWER(?), '', ?, 1, ?, ?, ?, 1, ?, ?, 'canvas', 1, ?, ?)
	`, id, userID, title, title, encryptedContent, encryptedTitle, titleEnc,
		wrappedDEK, encryptionMetadata, folderPath, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create encrypted canvas note: %w", err)
	}

	// Store keywords if provided
	if len(keywords) > 0 {
		for _, kw := range keywords {
			if err := db.InsertNoteKeyword(id, kw); err != nil {
				fmt.Printf("WARNING: Failed to insert keyword for canvas note %s: %v\n", id, err)
			}
		}
	}

	return db.GetNote(userID, id)
}

// ListCanvasNotes returns a slim list of canvas notes for a user.
func (db *DB) ListCanvasNotes(userID int) ([]Note, error) {
	rows, err := db.Query(`
		SELECT id, title, folder_path, created_at, updated_at, content_encrypted,
			encrypted_title, title_encrypted, wrapped_dek, encryption_version, encryption_metadata
		FROM notes
		WHERE user_id = ? AND note_type = 'canvas' AND is_deleted = 0
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list canvas notes: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		var createdAt, updatedAt string
		var encryptedTitle, wrappedDEK, encryptionMetadata interface{}

		if err := rows.Scan(
			&note.ID, &note.Title, &note.FolderPath,
			&createdAt, &updatedAt, &note.ContentEncrypted,
			&encryptedTitle, &note.TitleEncrypted,
			&wrappedDEK, &note.EncryptionVersion, &encryptionMetadata,
		); err != nil {
			return nil, fmt.Errorf("failed to scan canvas note: %w", err)
		}

		parsedCreatedAt, err := parseRFC3339Timestamp(createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at for canvas note %s: %w", note.ID, err)
		}
		parsedUpdatedAt, err := parseRFC3339Timestamp(updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated_at for canvas note %s: %w", note.ID, err)
		}
		note.CreatedAt = parsedCreatedAt
		note.UpdatedAt = parsedUpdatedAt
		note.NoteType = NoteTypeCanvas
		note.UserID = userID

		if s, ok := encryptedTitle.(*string); ok && s != nil {
			note.EncryptedTitle = s
		} else if s, ok := encryptedTitle.(string); ok && s != "" {
			note.EncryptedTitle = &s
		}
		if s, ok := wrappedDEK.(string); ok {
			note.WrappedDEK = s
		}
		if s, ok := encryptionMetadata.(string); ok {
			note.EncryptionMetadata = s
		}

		notes = append(notes, note)
	}

	return notes, rows.Err()
}
