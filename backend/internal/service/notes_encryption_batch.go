package service

import "fmt"

// DEKUpdate represents a single DEK re-encryption update.
type DEKUpdate struct {
	NoteID     string
	WrappedDEK string
}

// BatchUpdateWrappedDEKs updates wrapped DEKs for multiple notes in a transaction.
// Used after password changes to re-wrap all DEKs with the new KEK.
func (s *NoteService) BatchUpdateWrappedDEKs(userID int, updates []struct {
	NoteID     string `json:"note_id"`
	WrappedDEK string `json:"wrapped_dek"`
}) (int, error) {
	if len(updates) == 0 {
		return 0, nil
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	updatedCount := 0
	for _, update := range updates {
		// Verify note belongs to user
		note, err := s.db.GetNote(userID, update.NoteID)
		if err != nil {
			return 0, fmt.Errorf("failed to get note %s: %w", update.NoteID, err)
		}
		if note == nil {
			return 0, fmt.Errorf("note %s not found or unauthorized", update.NoteID)
		}

		// Only update if note is encrypted
		if !note.ContentEncrypted {
			s.logger.Warn("skipping non-encrypted note in DEK batch update", "note_id", update.NoteID)
			continue
		}

		// Update wrapped_dek
		_, err = tx.Exec(`
			UPDATE notes
			SET wrapped_dek = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ? AND user_id = ?
		`, update.WrappedDEK, update.NoteID, userID)
		if err != nil {
			return 0, fmt.Errorf("failed to update DEK for note %s: %w", update.NoteID, err)
		}

		updatedCount++

		// Invalidate cache for this note
		s.cache.Delete(noteCacheKey(userID, update.NoteID))
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate folder caches (since updated_at changed)
	// We don't know which folders were affected, so we invalidate the whole cache
	s.invalidateQuickSearchCache(userID)

	s.logger.Info("batch DEK re-encryption completed", "user_id", userID, "updated_count", updatedCount)

	return updatedCount, nil
}

// UserHasEncryptedNotes checks if a user has any encrypted notes.
// This is used to prevent accidental salt regeneration which would make existing encrypted notes unreadable.
func (s *NoteService) UserHasEncryptedNotes(userID int) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*)
		FROM notes
		WHERE user_id = ?
		  AND deleted_at IS NULL
		  AND encrypted_content IS NOT NULL
		  AND encrypted_content != ''
	`, userID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
