package db

import (
	"database/sql"
	"time"
)

// GetNoteUserState returns the state_data JSON string for a given user+note,
// or nil if no row exists.
func (db *DB) GetNoteUserState(userID int, noteID string) (*string, error) {
	var stateData string
	err := db.QueryRow(
		`SELECT state_data FROM note_user_state WHERE user_id = ? AND note_id = ?`,
		userID, noteID,
	).Scan(&stateData)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &stateData, nil
}

// UpsertNoteUserState inserts or updates the state_data for a user+note pair.
func (db *DB) UpsertNoteUserState(userID int, noteID string, stateData string) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE note_user_state
		SET state_data = ?, updated_at = ?
		WHERE user_id = ? AND note_id = ?
	`, stateData, now, userID, noteID)
	if err != nil {
		return err
	}

	rowsAffected, err := rowsAffectedCount(result, "")
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		_, err = db.Exec(`
			INSERT INTO note_user_state (note_id, user_id, state_data, updated_at)
			VALUES (?, ?, ?, ?)
		`, noteID, userID, stateData, now)
		return err
	}

	return nil
}

// DeleteNoteUserState removes the state row for a user+note pair.
func (db *DB) DeleteNoteUserState(userID int, noteID string) error {
	_, err := db.Exec(
		`DELETE FROM note_user_state WHERE user_id = ? AND note_id = ?`,
		userID, noteID,
	)
	return err
}
