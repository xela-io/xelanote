package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

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
