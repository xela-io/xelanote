// Package service contains the business logic for xelanote.
package service

import (
	"errors"

	"github.com/xela-io/xelanote/internal/db"
)

func (s *NoteService) invalidateRecoveryKeyBestEffort(userID int) {
	if err := s.db.InvalidateRecoveryKey(userID); err != nil {
		s.logger.Warn("failed to invalidate recovery key after encryption", "user_id", userID, "error", err)
	}
}

// CreateEncryptedNote creates a new encrypted note.
func (s *NoteService) CreateEncryptedNote(
	userID int,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	_ []string,
	_ string,
	encryptedFolderPath *string,
) (*db.Note, error) {
	return s.CreateEncryptedNoteWithID(
		userID,
		"",
		title,
		encryptedTitle,
		titleEncrypted,
		encryptedContent,
		wrappedDEK,
		encryptionMetadata,
		nil,
		"",
		encryptedFolderPath,
	)
}

// CreateEncryptedNoteWithID creates a new encrypted note with optional client-provided note ID.
func (s *NoteService) CreateEncryptedNoteWithID(
	userID int,
	noteID string,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	_ []string,
	_ string,
	encryptedFolderPath *string,
) (*db.Note, error) {
	// Validate
	if len(encryptedContent) == 0 || wrappedDEK == "" {
		return nil, errors.New("encrypted content and wrapped DEK required")
	}

	serverFolderPath := normalizedEncryptedFolderPath()

	// Create note in DB
	note, err := s.db.CreateEncryptedNoteWithID(
		userID,
		noteID,
		title,
		encryptedTitle,
		titleEncrypted,
		encryptedContent,
		wrappedDEK,
		encryptionMetadata,
		serverFolderPath,
		encryptedFolderPath,
	)
	if err != nil {
		return nil, err
	}

	// Invalidate caches
	s.invalidateRecoveryKeyBestEffort(userID)
	s.invalidateQuickSearchCache(userID)
	s.invalidateNotesByFolderCache(userID, serverFolderPath)

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
	_ []string,
	_ string,
	encryptedFolderPath *string,
	journalDate string,
) (*db.Note, error) {
	return s.CreateEncryptedJournalNoteWithID(
		userID,
		"",
		title,
		encryptedTitle,
		titleEncrypted,
		encryptedContent,
		wrappedDEK,
		encryptionMetadata,
		nil,
		"",
		encryptedFolderPath,
		journalDate,
	)
}

// CreateEncryptedJournalNoteWithID creates a new encrypted journal note for a specific date
// with optional client-provided note ID.
func (s *NoteService) CreateEncryptedJournalNoteWithID(
	userID int,
	noteID string,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	_ []string,
	_ string,
	encryptedFolderPath *string,
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

	serverFolderPath := normalizedEncryptedFolderPath()

	// Create encrypted journal note (also creates /Journal folder if needed)
	note, err := s.db.CreateEncryptedJournalNoteWithID(
		userID, noteID, title, encryptedTitle, titleEncrypted,
		encryptedContent, wrappedDEK, encryptionMetadata,
		serverFolderPath, encryptedFolderPath, journalDate,
	)
	if err != nil {
		return nil, err
	}

	// Invalidate caches (including folder cache since Journal folder may have been created)
	s.invalidateRecoveryKeyBestEffort(userID)
	s.invalidateFolderCache(userID)
	s.invalidateQuickSearchCache(userID)
	s.invalidateNotesByFolderCache(userID, serverFolderPath)
	if s.graphService != nil {
		s.graphService.InvalidateGraphCache(userID)
	}

	return note, nil
}
