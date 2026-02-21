// Package service contains the business logic for xelanote.
package service

import (
	"errors"

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
			if err := s.db.InsertNoteKeywords(note.ID, keywords); err != nil {
				s.logger.Warn("failed to insert keywords", "error", err)
			}
		} else {
			s.logger.Warn("keywords sent but user has keywords disabled", "userID", userID)
		}
	}

	// Invalidate caches
	s.invalidateQuickSearchCache(userID)

	return note, nil
}

// CreateJournalNote creates a new plaintext journal note for a specific date.
func (s *NoteService) CreateJournalNote(userID int, title, content, folderPath, journalDate string) (*db.Note, error) {
	if err := s.checkNoteLimit(userID); err != nil {
		return nil, err
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
	if err := s.updateLinks(userID, note.ID, content); err != nil {
		s.logger.Warn("failed to update links for journal note", "error", err, "note_id", note.ID)
	}
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
	if err := s.checkNoteLimit(userID); err != nil {
		return nil, err
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
			if err := s.db.InsertNoteKeywords(note.ID, keywords); err != nil {
				s.logger.Warn("failed to insert keywords", "error", err)
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
