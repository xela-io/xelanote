// Package service contains the business logic for xelanote.
package service

import (
	"errors"
	"fmt"

	"github.com/xela-io/xelanote/internal/db"
)

// GetNoteVersions returns a paginated list of versions for a note.
func (s *NoteService) GetNoteVersions(userID int, noteID string, limit int, cursor string) ([]db.NoteVersion, string, int, error) {
	return s.db.GetNoteVersions(userID, noteID, limit, cursor)
}

// GetNoteVersion retrieves a specific version of a note.
func (s *NoteService) GetNoteVersion(userID int, noteID string, version int) (*db.NoteVersion, error) {
	return s.db.GetNoteVersion(userID, noteID, version)
}

// RestoreVersion restores a note to a previous version.
// This is non-destructive: the current state is saved as a snapshot first.
// Handles 4 scenarios: plaintext→plaintext, encrypted→encrypted, plaintext→encrypted, encrypted→plaintext.
func (s *NoteService) RestoreVersion(userID int, noteID string, targetVersion, currentVersion int) (*db.Note, error) {
	// 1. Get current note and verify optimistic lock
	current, err := s.db.GetNote(userID, noteID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, db.ErrNotFound
	}
	if current.Version != currentVersion {
		return nil, db.ErrVersionMismatch
	}

	// 2. Get the target version to restore
	target, err := s.db.GetNoteVersion(userID, noteID, targetVersion)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, db.ErrNotFound
	}

	// 3. Save current state as a snapshot (non-destructive restore)
	// Choose snapshot method based on CURRENT note's encryption state
	if current.ContentEncrypted {
		if err := s.db.CreateEncryptedNoteVersion(
			userID,
			noteID,
			current.Version,
			current.Title,
			current.EncryptedTitle,
			current.TitleEncrypted,
			current.EncryptedContent,
			current.WrappedDEK,
			current.EncryptionVersion,
		); err != nil {
			s.logger.Error("failed to create encrypted snapshot before restore",
				"err", err, "note_id", noteID, "user_id", userID)
			// Continue anyway - restore is more important
		}
	} else {
		if err := s.db.CreateNoteVersion(userID, noteID, current.Version, current.Title, current.Content); err != nil {
			s.logger.Error("failed to create snapshot before restore",
				"err", err, "note_id", noteID, "user_id", userID)
			// Continue anyway - restore is more important
		}
	}

	// 4. Update the note with the target version's content
	// Choose update method based on TARGET version's encryption state
	if target.ContentEncrypted {
		// Validate encrypted version has required fields
		if len(target.EncryptedContent) == 0 || target.WrappedDEK == "" {
			return nil, errors.New("cannot restore: encrypted version is corrupted (missing encrypted content or DEK)")
		}

		// Reconstruct encryption_metadata from constants + wrapped_dek
		// Note: encryption_metadata is not stored in versions, so we rebuild it
		// using the standard encryption parameters (XChaCha20-Poly1305, Argon2id)
		encryptionMetadata := fmt.Sprintf(
			`{"version":2,"algorithm":"XChaCha20-Poly1305","kdf":"Argon2id","kdf_strength":"interactive","nonce_bytes":24,"wrapped_dek":"%s"}`,
			target.WrappedDEK,
		)

		return s.UpdateEncryptedNote(
			userID,
			noteID,
			target.Title,
			target.EncryptedTitle,
			target.TitleEncrypted,
			target.EncryptedContent,
			target.WrappedDEK,
			encryptionMetadata,
			current.FolderPath,
			nil, // keywords - not stored in versions
			current.EncryptedFolderPath,
			currentVersion,
		)
	}

	// Restore to plaintext version
	return s.UpdateNote(userID, noteID, target.Title, target.Content, current.FolderPath, currentVersion)
}

// PruneAllVersions removes old versions for all notes, keeping the most recent keepCount versions per note.
// Returns the total number of deleted versions.
func (s *NoteService) PruneAllVersions(keepCount int) (int, error) {
	userIDs, err := s.db.GetUsersWithVersions()
	if err != nil {
		return 0, err
	}

	totalPruned := 0
	for _, userID := range userIDs {
		pruned, err := s.db.PruneAllUserVersions(userID, keepCount)
		if err != nil {
			s.logger.Error("prune failed", "user", userID, "err", err)
			continue
		}
		totalPruned += pruned
	}
	return totalPruned, nil
}
