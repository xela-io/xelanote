package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// ComputeContentHash computes a SHA256 hash of the content and returns the first 16 characters.
// Used for change detection to determine if content has been modified since last summary.
func ComputeContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])[:16]
}

// UpdateNoteSummary updates the summary fields for a note.
// For plaintext notes: sets summary, clears encrypted_summary, sets summary_encrypted=false
// For encrypted notes: sets encrypted_summary, clears summary, sets summary_encrypted=true
func (db *DB) UpdateNoteSummary(userID int, noteID, summary string, encrypted bool, generatedAt time.Time) error {
	var result sql.Result
	var err error

	if encrypted {
		// Encrypted note: store in encrypted_summary field
		result, err = db.Exec(`
			UPDATE notes
			SET summary = NULL, encrypted_summary = ?, summary_encrypted = 1, summary_generated_at = ?
			WHERE id = ? AND user_id = ? AND is_deleted = 0
		`, summary, generatedAt.Format(time.RFC3339), noteID, userID)
	} else {
		// Plaintext note: store in summary field
		result, err = db.Exec(`
			UPDATE notes
			SET summary = ?, encrypted_summary = NULL, summary_encrypted = 0, summary_generated_at = ?
			WHERE id = ? AND user_id = ? AND is_deleted = 0
		`, summary, generatedAt.Format(time.RFC3339), noteID, userID)
	}

	if err != nil {
		return fmt.Errorf("failed to update note summary: %w", err)
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

// UpdateNoteContentHash updates the content_hash for a note.
// Called when content is updated to track changes for summary regeneration.
func (db *DB) UpdateNoteContentHash(userID int, noteID, contentHash string) error {
	result, err := db.Exec(`
		UPDATE notes
		SET content_hash = ?
		WHERE id = ? AND user_id = ? AND is_deleted = 0
	`, contentHash, noteID, userID)
	if err != nil {
		return fmt.Errorf("failed to update content hash: %w", err)
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

// GetNotesNeedingSummary returns unencrypted notes that need summary generation.
// A note needs a summary if:
// - It has no summary (summary IS NULL AND summary_generated_at IS NULL)
// - OR its content has changed (content_hash differs from when summary was generated)
// Only returns unencrypted notes (content_encrypted = 0) as encrypted notes require frontend-triggered generation.
func (db *DB) GetNotesNeedingSummary(limit int) ([]NoteNeedingSummary, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.Query(`
		SELECT n.id, n.user_id, n.content, n.content_hash
		FROM notes n
		WHERE n.is_deleted = 0
		  AND n.content_encrypted = 0
		  AND (
		      n.summary_generated_at IS NULL
		      OR n.content_hash IS NULL
		      OR n.summary IS NULL
		  )
		ORDER BY n.updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query notes needing summary: %w", err)
	}
	defer rows.Close()

	var notes []NoteNeedingSummary
	for rows.Next() {
		var note NoteNeedingSummary
		var contentHash sql.NullString

		if err := rows.Scan(&note.ID, &note.UserID, &note.Content, &contentHash); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}

		if contentHash.Valid {
			note.ContentHash = &contentHash.String
		}

		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notes: %w", err)
	}

	return notes, nil
}

// ClearNoteSummary clears the summary for a note (used when content changes significantly).
func (db *DB) ClearNoteSummary(userID int, noteID string) error {
	_, err := db.Exec(`
		UPDATE notes
		SET summary = NULL, encrypted_summary = NULL, summary_generated_at = NULL
		WHERE id = ? AND user_id = ? AND is_deleted = 0
	`, noteID, userID)
	return err
}
