// Package service contains the business logic for xelanote.
package service

import "github.com/xela-io/xelanote/internal/db"

// --- Tag Management Functions ---

// GetAllTags returns all tags for a user.
func (s *NoteService) GetAllTags(userID int) ([]db.Tag, error) {
	return s.db.GetAllTags(userID)
}

// GetNoteTags returns all tags for a specific note.
func (s *NoteService) GetNoteTags(noteID string) ([]db.Tag, error) {
	return s.db.GetNoteTags(noteID)
}

// SetNoteTags sets the tags for a note, replacing any existing tags.
func (s *NoteService) SetNoteTags(noteID string, userID int, tagNames []string) error {
	if err := s.db.SetNoteTags(noteID, userID, tagNames); err != nil {
		return err
	}

	// Invalidate cache for the note
	s.invalidateNoteCache(userID, noteID)
	s.invalidateQuickSearchCache(userID)

	return nil
}

// DeleteTag deletes a tag and all its associations.
func (s *NoteService) DeleteTag(userID int, tagID int) error {
	if err := s.db.DeleteTag(userID, tagID); err != nil {
		return err
	}

	// Invalidate search cache since tag associations changed
	s.invalidateQuickSearchCache(userID)

	return nil
}
