// Package service contains the business logic for xelanote.
package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/xela-io/xelanote/internal/db"
)

// CreateEncryptedNote creates a new encrypted note with optional keywords.
func (s *NoteService) CreateEncryptedNote(
	userID int,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	keywords []string,
	folderPath string,
) (*db.Note, error) {
	// Validate
	if len(encryptedContent) == 0 || wrappedDEK == "" {
		return nil, errors.New("encrypted content and wrapped DEK required")
	}

	// Create note in DB
	note, err := s.db.CreateEncryptedNote(
		userID,
		title,
		encryptedTitle,
		titleEncrypted,
		encryptedContent,
		wrappedDEK,
		encryptionMetadata,
		folderPath,
	)
	if err != nil {
		return nil, err
	}

	// Insert keywords ONLY if user has keywords enabled
	if len(keywords) > 0 {
		prefs, err := s.db.GetUserPreferences(userID)
		if err == nil && prefs.KeywordsEnabled {
			for _, kw := range keywords {
				if err := s.db.InsertNoteKeyword(note.ID, kw); err != nil {
					s.logger.Warn("failed to insert keyword", "error", err)
				}
			}
		} else if len(keywords) > 0 {
			s.logger.Warn("keywords sent but user has keywords disabled", "userID", userID)
		}
	}

	// Invalidate caches
	s.invalidateQuickSearchCache(userID)

	return note, nil
}

// CreateJournalNote creates a new plaintext journal note for a specific date.
func (s *NoteService) CreateJournalNote(userID int, title, content, folderPath, journalDate string) (*db.Note, error) {
	// Check note limit
	maxNotes, err := s.db.GetMaxNotesPerUser()
	if err != nil {
		return nil, err
	}
	if maxNotes > 0 {
		currentCount, err := s.db.GetNoteCountForUser(userID)
		if err != nil {
			return nil, err
		}
		if currentCount >= maxNotes {
			return nil, ErrNoteLimitExceeded
		}
	}

	// Validate journal date
	if err := ValidateJournalDate(journalDate); err != nil {
		return nil, err
	}

	// Check feature enabled
	feature, err := s.db.GetUserFeature(userID, "journal")
	if err != nil {
		return nil, err
	}
	if !feature.Enabled {
		return nil, errors.New("journal feature not enabled")
	}

	// Create journal note (also creates /Journal folder if needed)
	note, err := s.db.CreateJournalNote(userID, title, content, folderPath, journalDate)
	if err != nil {
		return nil, err
	}

	// Parse and save links
	s.updateLinks(userID, note.ID, content)
	s.updateDueDates(userID, note.ID, content)

	// Invalidate caches (including folder cache since Journal folder may have been created)
	s.invalidateFolderCache(userID)
	s.invalidateQuickSearchCache(userID)
	s.invalidateNotesByFolderCache(userID, folderPath)
	if s.graphService != nil {
		s.graphService.InvalidateGraphCache(userID)
	}

	return note, nil
}

// CreateEncryptedJournalNote creates a new encrypted journal note for a specific date.
func (s *NoteService) CreateEncryptedJournalNote(
	userID int,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	keywords []string,
	folderPath string,
	journalDate string,
) (*db.Note, error) {
	// Check note limit
	maxNotes, err := s.db.GetMaxNotesPerUser()
	if err != nil {
		return nil, err
	}
	if maxNotes > 0 {
		currentCount, err := s.db.GetNoteCountForUser(userID)
		if err != nil {
			return nil, err
		}
		if currentCount >= maxNotes {
			return nil, ErrNoteLimitExceeded
		}
	}

	// Validate journal date
	if err := ValidateJournalDate(journalDate); err != nil {
		return nil, err
	}

	// Check feature enabled
	feature, err := s.db.GetUserFeature(userID, "journal")
	if err != nil {
		return nil, err
	}
	if !feature.Enabled {
		return nil, errors.New("journal feature not enabled")
	}

	// Validate encryption fields
	if len(encryptedContent) == 0 || wrappedDEK == "" {
		return nil, errors.New("encrypted content and wrapped DEK required")
	}

	// Create encrypted journal note (also creates /Journal folder if needed)
	note, err := s.db.CreateEncryptedJournalNote(
		userID, title, encryptedTitle, titleEncrypted,
		encryptedContent, wrappedDEK, encryptionMetadata,
		folderPath, journalDate,
	)
	if err != nil {
		return nil, err
	}

	// Insert keywords if user has keywords enabled
	if len(keywords) > 0 {
		prefs, err := s.db.GetUserPreferences(userID)
		if err == nil && prefs.KeywordsEnabled {
			for _, kw := range keywords {
				if err := s.db.InsertNoteKeyword(note.ID, kw); err != nil {
					s.logger.Warn("failed to insert keyword", "error", err)
				}
			}
		}
	}

	// Invalidate caches (including folder cache since Journal folder may have been created)
	s.invalidateFolderCache(userID)
	s.invalidateQuickSearchCache(userID)
	s.invalidateNotesByFolderCache(userID, folderPath)
	if s.graphService != nil {
		s.graphService.InvalidateGraphCache(userID)
	}

	return note, nil
}

// UpdateEncryptedNote updates an existing note with encrypted content.
func (s *NoteService) UpdateEncryptedNote(
	userID int,
	noteID string,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	folderPath string,
	keywords []string,
	expectedVersion int,
) (*db.Note, error) {
	// Validate
	if len(encryptedContent) == 0 || wrappedDEK == "" {
		return nil, errors.New("encrypted content and wrapped DEK required")
	}

	// Get existing note for version snapshotting
	existingNote, err := s.db.GetNote(userID, noteID)
	if err != nil {
		return nil, err
	}
	if existingNote == nil {
		return nil, db.ErrNotFound
	}

	// Check if we should create a snapshot
	// For encrypted notes, we check if encrypted content or wrapped DEK changed
	// (we can't compare plaintext content since it's encrypted)
	shouldSnapshot := false
	if existingNote.ContentEncrypted {
		// Always snapshot encrypted notes when updated (conservative approach)
		// Could be optimized by comparing encrypted_content bytes, but that's fragile
		lastSnapshot, err := s.db.GetLatestVersionSnapshot(userID, noteID)
		if err != nil {
			s.logger.Warn("failed to load latest snapshot", "err", err, "note_id", noteID, "user_id", userID)
		}
		shouldSnapshot = err != nil || lastSnapshot == nil ||
			time.Since(lastSnapshot.SnapshotAt) > snapshotThreshold
	}

	if shouldSnapshot {
		// Create snapshot of the state BEFORE the update
		if err := s.db.CreateEncryptedNoteVersion(
			userID,
			noteID,
			existingNote.Version,
			existingNote.Title,
			existingNote.EncryptedTitle,
			existingNote.TitleEncrypted,
			existingNote.EncryptedContent,
			existingNote.WrappedDEK,
			existingNote.EncryptionVersion,
		); err != nil {
			// Log but don't fail - note update is more important
			s.logger.Error("failed to create encrypted version snapshot", "err", err, "note_id", noteID, "user_id", userID)
		}
	}

	// Update note in DB
	note, err := s.db.UpdateEncryptedNote(
		userID,
		noteID,
		title,
		encryptedTitle,
		titleEncrypted,
		encryptedContent,
		wrappedDEK,
		encryptionMetadata,
		folderPath,
		expectedVersion,
	)
	if err != nil {
		return nil, err
	}

	// Update keywords if provided
	if len(keywords) > 0 {
		// Delete old keywords
		if err := s.db.DeleteNoteKeywords(noteID); err != nil {
			s.logger.Warn("failed to delete old keywords", "error", err)
		}

		// Insert new keywords if user has enabled keyword extraction
		prefs, err := s.db.GetUserPreferences(userID)
		if err == nil && prefs.KeywordsEnabled {
			for _, kw := range keywords {
				if err := s.db.InsertNoteKeyword(noteID, kw); err != nil {
					s.logger.Warn("failed to insert keyword", "error", err)
				}
			}
		}
	}

	// Business rule: encrypting a note removes all shares
	if err := s.db.DeleteAllSharesForNote(noteID); err != nil {
		s.logger.Error("failed to remove shares after encryption", "err", err, "note_id", noteID, "user_id", userID)
	} else {
		s.logger.Info("note encrypted, shares removed", "note_id", noteID, "user_id", userID)
	}

	// Business rule: encrypting a recipe note deletes metadata + ingredients
	// (data is serialized into encrypted payload by the frontend)
	if existingNote.NoteType == db.NoteTypeRecipe {
		if err := s.db.DeleteRecipeData(noteID); err != nil {
			s.logger.Error("failed to delete recipe data after encryption", "err", err, "note_id", noteID, "user_id", userID)
		} else {
			s.logger.Info("recipe encrypted, metadata+ingredients removed", "note_id", noteID, "user_id", userID)
		}
	}

	// Invalidate caches
	s.invalidateNoteCache(userID, noteID)
	s.invalidateQuickSearchCache(userID)

	return note, nil
}

// DecryptNote decrypts a note by clearing all encryption fields and setting plaintext content.
// The client sends the already-decrypted title and content.
// Creates a version snapshot before decryption for undo capability.
func (s *NoteService) DecryptNote(userID int, id, title, content string, expectedVersion int) (*db.Note, error) {
	// Get existing note
	existingNote, err := s.db.GetNote(userID, id)
	if err != nil {
		return nil, err
	}
	if existingNote == nil {
		return nil, db.ErrNotFound
	}

	// Validate that note is currently encrypted
	if !existingNote.ContentEncrypted {
		return nil, fmt.Errorf("note is not encrypted")
	}

	// Create version snapshot before decryption
	lastSnapshot, err := s.db.GetLatestVersionSnapshot(userID, id)
	if err != nil {
		s.logger.Warn("failed to load latest snapshot", "err", err, "note_id", id, "user_id", userID)
	}
	shouldSnapshot := err != nil || lastSnapshot == nil ||
		time.Since(lastSnapshot.SnapshotAt) > snapshotThreshold

	if shouldSnapshot {
		if existingNote.ContentEncrypted {
			if err := s.db.CreateEncryptedNoteVersion(
				userID, id, existingNote.Version,
				existingNote.Title,
				existingNote.EncryptedTitle,
				existingNote.TitleEncrypted,
				existingNote.EncryptedContent,
				existingNote.WrappedDEK,
				existingNote.EncryptionVersion,
			); err != nil {
				s.logger.Error("failed to create snapshot before decrypt", "err", err, "note_id", id, "user_id", userID)
			}
		}
	}

	// Decrypt note in DB
	note, err := s.db.DecryptNote(userID, id, title, content, expectedVersion)
	if err != nil {
		return nil, err
	}

	// Reprocess links on the plaintext content
	if err := s.updateLinks(userID, id, content); err != nil {
		s.logger.Error("failed to update links after decrypt", "err", err, "note_id", id, "user_id", userID)
	}
	s.updateDueDates(userID, id, content)

	s.logger.Info("note decrypted", "note_id", id, "user_id", userID)

	// Invalidate caches
	s.invalidateNoteCache(userID, id)
	s.invalidateNotesByFolderCache(userID, note.FolderPath)
	s.invalidateQuickSearchCache(userID)
	if s.graphService != nil {
		s.graphService.InvalidateGraphCache(userID)
	}

	return note, nil
}

// DecryptRecipeNote decrypts a recipe note and optionally restores recipe data.
// If recipeMetadata is provided, it's restored. If recipeIngredients is provided, they're restored.
// If neither is provided, the note type remains 'recipe' but with default metadata (I5 fallback).
func (s *NoteService) DecryptRecipeNote(userID int, id, title, content string, expectedVersion int,
	recipeMetadata *db.RecipeMetadata, recipeIngredients []db.RecipeIngredient) (*db.Note, error) {

	// First decrypt the note normally
	note, err := s.DecryptNote(userID, id, title, content, expectedVersion)
	if err != nil {
		return nil, err
	}

	// Restore recipe data if provided
	if recipeMetadata != nil {
		if err := s.db.SetRecipeMetadata(id, userID, recipeMetadata, ""); err != nil {
			s.logger.Error("failed to restore recipe metadata after decrypt",
				"err", err, "note_id", id, "user_id", userID)
		} else {
			s.logger.Info("recipe metadata restored after decrypt", "note_id", id)
		}

		if len(recipeIngredients) > 0 {
			// Get the newly created metadata's updated_at for optimistic lock
			meta, err := s.db.GetRecipeMetadata(id)
			if err == nil && meta != nil {
				if err := s.db.SetRecipeIngredients(id, userID, recipeIngredients, meta.UpdatedAt); err != nil {
					s.logger.Error("failed to restore recipe ingredients after decrypt",
						"err", err, "note_id", id, "user_id", userID)
				} else {
					s.logger.Info("recipe ingredients restored after decrypt",
						"note_id", id, "count", len(recipeIngredients))
				}
			}
		}
	}

	return note, nil
}

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
