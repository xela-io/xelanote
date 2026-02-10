package db

import (
	"fmt"
	"time"
)

// TaskEvent represents a task completion or reopening event.
type TaskEvent struct {
	ID                 int       `json:"id"`
	NoteID             string    `json:"note_id"`
	ActorUserID        int       `json:"actor_user_id"`
	TaskText           *string   `json:"task_text,omitempty"`
	TaskIndex          int       `json:"task_index"`
	EncryptedTaskText  *string   `json:"encrypted_task_text,omitempty"`
	WrappedDEK         *string   `json:"wrapped_dek,omitempty"`
	EncryptionMetadata *string   `json:"encryption_metadata,omitempty"`
	TextEncrypted      bool      `json:"text_encrypted"`
	EventType          string    `json:"event_type"`
	CreatedAt          time.Time `json:"created_at"`
}

// RecordTaskEvent inserts a new task event into the database.
func (db *DB) RecordTaskEvent(event TaskEvent) error {
	_, err := db.Exec(`
		INSERT INTO task_events (note_id, actor_user_id, task_text, task_index, encrypted_task_text, wrapped_dek, encryption_metadata, text_encrypted, event_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.NoteID, event.ActorUserID, event.TaskText, event.TaskIndex,
		event.EncryptedTaskText, event.WrappedDEK, event.EncryptionMetadata,
		event.TextEncrypted, event.EventType,
	)
	if err != nil {
		return fmt.Errorf("failed to record task event: %w", err)
	}
	return nil
}

// GetTaskEventsByUser returns task events for a user within a time range.
func (db *DB) GetTaskEventsByUser(userID int, from, to time.Time) ([]TaskEvent, error) {
	rows, err := db.Query(`
		SELECT id, note_id, actor_user_id, task_text, task_index, encrypted_task_text, wrapped_dek, encryption_metadata, text_encrypted, event_type, created_at
		FROM task_events
		WHERE actor_user_id = ? AND created_at >= ? AND created_at <= ?
		ORDER BY created_at DESC`,
		userID, from.Format(time.DateTime), to.Format(time.DateTime),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query task events by user: %w", err)
	}
	defer rows.Close()

	return scanTaskEvents(rows)
}

// GetTaskEventsByNote returns task events for a specific note owned by the user.
func (db *DB) GetTaskEventsByNote(noteID string, userID int) ([]TaskEvent, error) {
	rows, err := db.Query(`
		SELECT id, note_id, actor_user_id, task_text, task_index, encrypted_task_text, wrapped_dek, encryption_metadata, text_encrypted, event_type, created_at
		FROM task_events
		WHERE note_id = ? AND actor_user_id = ?
		ORDER BY created_at DESC`,
		noteID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query task events by note: %w", err)
	}
	defer rows.Close()

	return scanTaskEvents(rows)
}

func scanTaskEvents(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
}) ([]TaskEvent, error) {
	var events []TaskEvent
	for rows.Next() {
		var e TaskEvent
		var createdAt string
		err := rows.Scan(
			&e.ID, &e.NoteID, &e.ActorUserID, &e.TaskText, &e.TaskIndex,
			&e.EncryptedTaskText, &e.WrappedDEK, &e.EncryptionMetadata,
			&e.TextEncrypted, &e.EventType, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task event: %w", err)
		}
		e.CreatedAt, _ = time.Parse(time.DateTime, createdAt)
		events = append(events, e)
	}
	return events, nil
}
