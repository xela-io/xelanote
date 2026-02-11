package service

import (
	"errors"
	"time"

	"github.com/xela-io/xelanote/internal/db"
)

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
