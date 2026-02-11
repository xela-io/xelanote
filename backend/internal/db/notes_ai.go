package db

import (
	"database/sql"
	"fmt"
	"time"
)

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
	return ensureRowsAffectedWithContext(result, "failed to get rows affected")
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
