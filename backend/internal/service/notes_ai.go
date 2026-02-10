// Package service contains the business logic for xelanote.
package service

// UpdateNoteAIEnabled sets the ai_enabled flag for a note.
// When ai_enabled=true, Claude API features are allowed for this note.
func (s *NoteService) UpdateNoteAIEnabled(userID int, noteID string, aiEnabled bool) error {
	if err := s.db.UpdateNoteAIEnabled(userID, noteID, aiEnabled); err != nil {
		return err
	}

	// Invalidate caches
	s.invalidateNoteCache(userID, noteID)
	s.invalidateQuickSearchCache(userID)

	return nil
}

// GetNoteAIEnabled returns whether Claude API is enabled for a note.
func (s *NoteService) GetNoteAIEnabled(userID int, noteID string) (bool, error) {
	return s.db.GetNoteAIEnabled(userID, noteID)
}

// GetNoteTitlesAIEnabled returns titles of notes with ai_enabled=true for a user.
// Used for Claude API link suggestions.
func (s *NoteService) GetNoteTitlesAIEnabled(userID int) ([]string, error) {
	return s.db.GetNoteTitlesAIEnabled(userID)
}
