package db

import (
	"fmt"
	"time"
)

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
