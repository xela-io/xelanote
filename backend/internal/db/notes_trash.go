package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

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
	if err := ensureRowsAffectedWithContext(result, "failed to check rows affected"); err != nil {
		return nil, err
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
	return ensureRowsAffectedWithContext(result, "failed to check rows affected")
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
