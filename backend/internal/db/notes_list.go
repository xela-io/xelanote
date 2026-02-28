package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Full SELECT columns (default)
const fullSelectColumns = `id, title, content, folder_path, version, display_order, color, created_at, updated_at,
	encrypted_content, content_encrypted, encrypted_title, title_encrypted,
	wrapped_dek, wrapped_dek_recovery, encryption_version, encryption_metadata,
	note_type, journal_date, ai_enabled, is_deleted`

// Slim SELECT columns — excludes content, encrypted_content, summary fields, content_hash, summary_generated_at.
// Keeps all encryption metadata (wrapped_dek, encryption_version, encryption_metadata, encrypted_title, title_encrypted)
// because the frontend needs them for title decryption.
const slimSelectColumns = `id, title, '' as content, folder_path, version, display_order, color, created_at, updated_at,
	NULL as encrypted_content, content_encrypted, encrypted_title, title_encrypted,
	wrapped_dek, wrapped_dek_recovery, encryption_version, encryption_metadata,
	note_type, journal_date, ai_enabled, is_deleted`

// selectColumns returns the appropriate SELECT columns based on the slim flag.
func selectColumns(slim bool) string {
	if slim {
		return slimSelectColumns
	}
	return fullSelectColumns
}

// scanNoteBase scans the common note columns from a SQL row into a Note struct.
// extraDests are additional scan destinations appended after the standard columns.
func scanNoteBase(rows *sql.Rows, slim bool, userID int, extraDests ...any) (Note, error) {
	var note Note
	var createdAt, updatedAt string
	var content, encryptedTitle, wrappedDEK, wrappedDEKRecovery, encryptionMetadata sql.NullString
	var encryptedContent []byte
	var noteType, journalDate sql.NullString
	var isDeleted int

	dests := []any{
		&note.ID, &note.Title, &content, &note.FolderPath, &note.Version, &note.DisplayOrder, &note.Color, &createdAt, &updatedAt,
		&encryptedContent, &note.ContentEncrypted, &encryptedTitle, &note.TitleEncrypted,
		&wrappedDEK, &wrappedDEKRecovery, &note.EncryptionVersion, &encryptionMetadata,
		&noteType, &journalDate, &note.AIEnabled, &isDeleted,
	}
	dests = append(dests, extraDests...)

	if err := rows.Scan(dests...); err != nil {
		return Note{}, fmt.Errorf("failed to scan note: %w", err)
	}
	parsedCreatedAt, err := parseRFC3339Timestamp(createdAt)
	if err != nil {
		return Note{}, fmt.Errorf("failed to parse created_at for note %s: %w", note.ID, err)
	}
	parsedUpdatedAt, err := parseRFC3339Timestamp(updatedAt)
	if err != nil {
		return Note{}, fmt.Errorf("failed to parse updated_at for note %s: %w", note.ID, err)
	}
	note.CreatedAt = parsedCreatedAt
	note.UpdatedAt = parsedUpdatedAt
	if !slim {
		note.Content = content.String
		note.EncryptedContent = encryptedContent
	}
	if encryptedTitle.Valid {
		note.EncryptedTitle = &encryptedTitle.String
	}
	note.WrappedDEK = wrappedDEK.String
	note.WrappedDEKRecovery = wrappedDEKRecovery.String
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
	note.IsDeleted = isDeleted == 1
	return note, nil
}

// scanNote scans a single row from a notes query into a Note struct.
// The slim parameter controls whether content/encrypted_content are populated.
func scanNote(rows *sql.Rows, slim bool, userID int) (Note, error) {
	return scanNoteBase(rows, slim, userID)
}

// scanNoteWithShared scans a single row that includes an is_shared column (after is_deleted).
func scanNoteWithShared(rows *sql.Rows, slim bool, userID int) (Note, error) {
	var isShared int
	note, err := scanNoteBase(rows, slim, userID, &isShared)
	if err != nil {
		return Note{}, err
	}
	note.IsShared = isShared == 1
	return note, nil
}

// ListNotesOptions holds parameters for the ListNotes query.
type ListNotesOptions struct {
	Fields       string // "slim" or "" (full)
	UpdatedSince string // Sync token: "RFC3339Nano|UUID" (for delta-sync)
}

// ParseSyncToken parses a sync token of the format "RFC3339Nano|UUID" into a timestamp and ID.
func ParseSyncToken(token string) (time.Time, string, error) {
	parts := strings.SplitN(token, "|", 2)
	if len(parts) != 2 || parts[1] == "" {
		return time.Time{}, "", fmt.Errorf("invalid sync token format: expected timestamp|id")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid sync token: %w", err)
	}
	return t, parts[1], nil
}

// HighWatermark computes the sync_token from a result set.
// It returns the newest (updated_at, id) tuple regardless of sort direction.
func HighWatermark(notes []Note, ascending bool) string {
	if len(notes) == 0 {
		return ""
	}
	var hw Note
	if ascending {
		hw = notes[len(notes)-1] // ASC: last = newest
	} else {
		hw = notes[0] // DESC: first = newest
	}
	return fmt.Sprintf("%s|%s", hw.UpdatedAt.Format(time.RFC3339Nano), hw.ID)
}

// ListNotes returns a paginated list of notes.
// Supports slim field projection and delta-sync via updated_since.
func (db *DB) ListNotes(userID int, limit int, cursor string, opts ListNotesOptions) ([]Note, string, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	slim := opts.Fields == "slim"
	cols := selectColumns(slim)
	isDelta := opts.UpdatedSince != ""

	var rows *sql.Rows
	var err error

	if isDelta {
		// Delta-sync mode: ORDER BY updated_at ASC, id ASC
		sinceTime, sinceID, parseErr := ParseSyncToken(opts.UpdatedSince)
		if parseErr != nil {
			return nil, "", fmt.Errorf("invalid updated_since: %w", parseErr)
		}
		sinceTimeStr := sinceTime.Format(time.RFC3339Nano)

		if cursor == "" {
			rows, err = db.Query(fmt.Sprintf(`
				SELECT %s
				FROM notes
				WHERE user_id = ?
				  AND COALESCE(note_type, 'note') IN ('note', 'canvas')
				  AND (is_deleted = 0 OR updated_at >= ?)
				  AND (updated_at > ? OR (updated_at = ? AND id > ?))
				ORDER BY updated_at ASC, id ASC
				LIMIT ?
			`, cols), userID, sinceTimeStr, sinceTimeStr, sinceTimeStr, sinceID, limit+1)
		} else {
			cursorParts := strings.SplitN(cursor, "|", 2)
			if len(cursorParts) != 2 {
				return nil, "", fmt.Errorf("invalid cursor format")
			}
			cursorTime, cursorID := cursorParts[0], cursorParts[1]
			rows, err = db.Query(fmt.Sprintf(`
				SELECT %s
				FROM notes
				WHERE user_id = ?
				  AND COALESCE(note_type, 'note') IN ('note', 'canvas')
				  AND (is_deleted = 0 OR updated_at >= ?)
				  AND (updated_at > ? OR (updated_at = ? AND id > ?))
				ORDER BY updated_at ASC, id ASC
				LIMIT ?
			`, cols), userID, sinceTimeStr, cursorTime, cursorTime, cursorID, limit+1)
		}
	} else {
		// Full mode: ORDER BY updated_at DESC, id DESC
		if cursor == "" {
			rows, err = db.Query(fmt.Sprintf(`
				SELECT %s
				FROM notes
				WHERE user_id = ? AND is_deleted = 0 AND COALESCE(note_type, 'note') IN ('note', 'canvas')
				ORDER BY updated_at DESC, id DESC
				LIMIT ?
			`, cols), userID, limit+1)
		} else {
			parts := strings.SplitN(cursor, "|", 2)
			if len(parts) != 2 {
				return nil, "", fmt.Errorf("invalid cursor format")
			}
			cursorTime, cursorID := parts[0], parts[1]
			rows, err = db.Query(fmt.Sprintf(`
				SELECT %s
				FROM notes
				WHERE user_id = ? AND is_deleted = 0 AND COALESCE(note_type, 'note') IN ('note', 'canvas')
				  AND (updated_at < ? OR (updated_at = ? AND id < ?))
				ORDER BY updated_at DESC, id DESC
				LIMIT ?
			`, cols), userID, cursorTime, cursorTime, cursorID, limit+1)
		}
	}

	if err != nil {
		return nil, "", fmt.Errorf("failed to list notes: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		note, scanErr := scanNote(rows, slim, userID)
		if scanErr != nil {
			return nil, "", scanErr
		}
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("error iterating notes: %w", err)
	}

	// Check for next page
	var nextCursor string
	if len(notes) > limit {
		lastNote := notes[limit-1]
		nextCursor = fmt.Sprintf("%s|%s", lastNote.UpdatedAt.Format(time.RFC3339Nano), lastNote.ID)
		notes = notes[:limit]
	}

	return notes, nextCursor, nil
}

// ListNotesByFolder returns notes in a specific folder.
// For the /Journal folder, returns journal notes sorted by date (newest first).
// For other folders, excludes journal notes from the list.
// Also includes shared notes that the user has placed into this folder (with active share check).
func (db *DB) ListNotesByFolder(userID int, folderPath string, fields string) ([]Note, error) {
	slim := fields == "slim"
	cols := selectColumns(slim)

	var query string

	if folderPath == "/Journal" {
		query = fmt.Sprintf(`
			SELECT %s, 0 as is_shared
			FROM notes
			WHERE folder_path = ? AND user_id = ? AND is_deleted = 0
			  AND note_type = 'journal'
			ORDER BY journal_date DESC`, cols)
	} else if folderPath == "/Rezepte" {
		query = fmt.Sprintf(`
			SELECT %s, 0 as is_shared
			FROM notes
			WHERE user_id = ? AND note_type = 'recipe' AND is_deleted = 0
			ORDER BY updated_at DESC`, cols)
	} else {
		query = fmt.Sprintf(`
			SELECT %[1]s, is_shared
			FROM (
				SELECT %[1]s, 0 as is_shared
				FROM notes
				WHERE folder_path = ? AND user_id = ? AND is_deleted = 0
				  AND COALESCE(note_type, 'note') IN ('note', 'canvas')

				UNION ALL

				SELECT %[2]s, 1 as is_shared
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
			ORDER BY display_order ASC, title ASC`,
			cols,
			sharedSelectColumns(slim))
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
		note, scanErr := scanNoteWithShared(rows, slim, userID)
		if scanErr != nil {
			return nil, scanErr
		}
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notes: %w", err)
	}

	return notes, nil
}

// sharedSelectColumns returns columns for the shared notes sub-query in UNION.
// These are prefixed with "n." for the JOIN query.
func sharedSelectColumns(slim bool) string {
	if slim {
		return `n.id, n.title, '' as content, n.folder_path, n.version, 99999 as display_order,
	n.color, n.created_at, n.updated_at,
	NULL as encrypted_content, n.content_encrypted, n.encrypted_title, n.title_encrypted,
	n.wrapped_dek, n.wrapped_dek_recovery, n.encryption_version, n.encryption_metadata,
	n.note_type, n.journal_date, n.ai_enabled, n.is_deleted`
	}
	return `n.id, n.title, n.content, n.folder_path, n.version, 99999 as display_order,
	n.color, n.created_at, n.updated_at,
	n.encrypted_content, n.content_encrypted, n.encrypted_title, n.title_encrypted,
	n.wrapped_dek, n.wrapped_dek_recovery, n.encryption_version, n.encryption_metadata,
	n.note_type, n.journal_date, n.ai_enabled, n.is_deleted`
}
