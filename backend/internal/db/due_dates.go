package db

import (
	"fmt"

	"github.com/xela-io/xelanote/internal/parser"
)

// DueDateWithNote represents a due date with its associated note information.
type DueDateWithNote struct {
	ID          int    `json:"id"`
	NoteID      string `json:"note_id"`
	NoteTitle   string `json:"note_title"`
	DueDate     string `json:"due_date"`
	LineText    string `json:"line_text"`
	LineIndex   int    `json:"line_index"`
	IsTaskItem  bool   `json:"is_task_item"`
	IsCompleted bool   `json:"is_completed"`
}

// SetNoteDueDates replaces all due dates for a note with the given list.
// Uses DELETE + INSERT pattern within a transaction.
func (db *DB) SetNoteDueDates(noteID string, userID int, dueDates []parser.DueDate) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing due dates for this note
	_, err = tx.Exec("DELETE FROM note_due_dates WHERE note_id = ?", noteID)
	if err != nil {
		return err
	}

	// Insert new due dates
	if len(dueDates) > 0 {
		stmt, err := tx.Prepare(`
			INSERT INTO note_due_dates (note_id, user_id, line_text, line_index, due_date, is_task_item, is_completed)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, dd := range dueDates {
			isTask := 0
			if dd.IsTaskItem {
				isTask = 1
			}
			isCompleted := 0
			if dd.IsCompleted {
				isCompleted = 1
			}
			_, err = stmt.Exec(noteID, userID, dd.LineText, dd.LineIndex, dd.Date, isTask, isCompleted)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// BackfillDueDates re-parses all non-deleted plaintext notes and populates note_due_dates.
// Runs once at startup; skips if the table already has data.
func (db *DB) BackfillDueDates() (int, error) {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM note_due_dates").Scan(&count); err != nil {
		return 0, fmt.Errorf("backfill check failed: %w", err)
	}
	if count > 0 {
		return 0, nil
	}

	rows, err := db.Query(`
		SELECT id, user_id, content FROM notes
		WHERE deleted_at IS NULL
		  AND content != ''
		  AND (encrypted_content IS NULL OR encrypted_content = '')
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	total := 0
	for rows.Next() {
		var noteID string
		var userID int
		var content string
		if err := rows.Scan(&noteID, &userID, &content); err != nil {
			return total, err
		}
		dueDates := parser.ParseDueDates(content)
		if len(dueDates) > 0 {
			if err := db.SetNoteDueDates(noteID, userID, dueDates); err != nil {
				return total, fmt.Errorf("backfill note %s: %w", noteID, err)
			}
			total += len(dueDates)
		}
	}
	return total, rows.Err()
}

// GetDueDatesByUser returns all due dates for a user across all non-deleted notes.
func (db *DB) GetDueDatesByUser(userID int, showCompleted bool) ([]DueDateWithNote, error) {
	query := `
		SELECT dd.id, dd.note_id, n.title, dd.due_date, dd.line_text, dd.line_index, dd.is_task_item, dd.is_completed
		FROM note_due_dates dd
		JOIN notes n ON dd.note_id = n.id
		WHERE dd.user_id = ? AND n.deleted_at IS NULL
	`
	if !showCompleted {
		query += ` AND dd.is_completed = 0`
	}
	query += ` ORDER BY dd.due_date ASC`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DueDateWithNote
	for rows.Next() {
		var d DueDateWithNote
		var isTask, isCompleted int
		if err := rows.Scan(&d.ID, &d.NoteID, &d.NoteTitle, &d.DueDate, &d.LineText, &d.LineIndex, &isTask, &isCompleted); err != nil {
			return nil, err
		}
		d.IsTaskItem = isTask != 0
		d.IsCompleted = isCompleted != 0
		results = append(results, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Return empty slice instead of nil
	if results == nil {
		results = []DueDateWithNote{}
	}
	return results, nil
}
