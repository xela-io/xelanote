package service

import (
	"fmt"
	"time"

	"github.com/xela-io/xelanote/internal/db"
)

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
