package db

import (
	"database/sql"
	"fmt"
)

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
