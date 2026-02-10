// Package service contains the business logic for xelanote.
package service

import (
	"fmt"
	"strings"

	"github.com/xela-io/xelanote/internal/db"
)

// DeleteNote soft-deletes a note.
func (s *NoteService) DeleteNote(userID int, id string) error {
	note, err := s.db.GetNote(userID, id)
	if err != nil {
		return err
	}

	if err := s.db.DeleteNote(userID, id); err != nil {
		return err
	}

	s.invalidateNoteCache(userID, id)
	if note != nil {
		s.invalidateNotesByFolderCache(userID, note.FolderPath)
	}
	s.invalidateQuickSearchCache(userID)
	if s.graphService != nil {
		s.graphService.InvalidateGraphCache(userID)
	}
	return nil
}

// ListDeletedNotes returns a paginated list of soft-deleted notes.
func (s *NoteService) ListDeletedNotes(userID int, limit int, cursor string) ([]db.Note, string, error) {
	return s.db.ListDeletedNotes(userID, limit, cursor)
}

// RestoreNote restores a soft-deleted note and reprocesses its links.
func (s *NoteService) RestoreNote(userID int, id string) (*db.Note, error) {
	// Attempt to restore the note
	note, err := s.db.RestoreNote(userID, id)
	if err != nil {
		// Check if error is due to UNIQUE constraint conflict
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			// Note: We can't determine the exact title/folder without querying the deleted note
			// So we provide a generic but helpful message
			return nil, fmt.Errorf("kann nicht wiederherstellen: Eine Notiz mit diesem Titel existiert bereits im Zielordner")
		}
		return nil, err
	}

	// Reprocess links in the restored note
	if err := s.updateLinks(userID, note.ID, note.Content); err != nil {
		// Log but don't fail - note is already restored
	}
	s.updateDueDates(userID, note.ID, note.Content)

	// Check if this note resolves any unresolved links
	if err := s.resolveUnresolvedLinks(userID, note); err != nil {
		// Log but don't fail
	}

	s.cache.Set(noteCacheKey(userID, note.ID), note)
	s.invalidateNotesByFolderCache(userID, note.FolderPath)
	s.invalidateQuickSearchCache(userID)

	return note, nil
}

// PermanentlyDeleteNote performs a hard delete on a note.
func (s *NoteService) PermanentlyDeleteNote(userID int, id string) error {
	if err := s.db.PermanentlyDeleteNote(userID, id); err != nil {
		return err
	}

	s.invalidateNoteCache(userID, id)
	s.invalidateQuickSearchCache(userID)
	return nil
}

// GetDeletedNotesCount returns the count of soft-deleted notes.
func (s *NoteService) GetDeletedNotesCount(userID int) (int, error) {
	return s.db.GetDeletedNotesCount(userID)
}

// EmptyTrash permanently deletes all soft-deleted notes.
func (s *NoteService) EmptyTrash(userID int) (int, error) {
	return s.db.EmptyTrash(userID)
}
