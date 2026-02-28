package service

import (
	"errors"
	"log/slog"

	"github.com/xela-io/xelanote/internal/db"
)

var ErrCanvasFeatureNotEnabled = errors.New("canvas feature not enabled")

// CanvasService handles canvas-specific business logic.
type CanvasService struct {
	db     *db.DB
	logger *slog.Logger
	notes  *NoteService
}

// NewCanvasService creates a new CanvasService.
func NewCanvasService(database *db.DB, noteService *NoteService) *CanvasService {
	return &CanvasService{
		db:     database,
		logger: slog.Default(),
		notes:  noteService,
	}
}

// checkFeatureEnabled verifies the canvas feature is enabled for the user.
func (s *CanvasService) checkFeatureEnabled(userID int) error {
	feature, err := s.db.GetUserFeature(userID, "canvas")
	if err != nil {
		return err
	}
	if !feature.Enabled {
		return ErrCanvasFeatureNotEnabled
	}
	return nil
}

// CreateCanvasNote creates a new plaintext canvas note.
func (s *CanvasService) CreateCanvasNote(userID int, title, content, folderPath string) (*db.Note, error) {
	if err := s.notes.checkNoteLimit(userID); err != nil {
		return nil, err
	}

	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}

	// Validate canvas content if non-empty
	if content != "" {
		if err := ValidateCanvasContent(content); err != nil {
			return nil, err
		}
	}

	note, err := s.db.CreateCanvasNote(userID, title, content, folderPath)
	if err != nil {
		return nil, err
	}

	s.notes.invalidateFolderCache(userID)
	s.notes.invalidateQuickSearchCache(userID)
	s.notes.invalidateNotesByFolderCache(userID, folderPath)
	if s.notes.graphService != nil {
		s.notes.graphService.InvalidateGraphCache(userID)
	}

	return note, nil
}

// CreateEncryptedCanvasNote creates a new encrypted canvas note.
func (s *CanvasService) CreateEncryptedCanvasNote(
	userID int,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	_ []string,
	folderPath string,
) (*db.Note, error) {
	return s.CreateEncryptedCanvasNoteWithID(
		userID,
		"",
		title,
		encryptedTitle,
		titleEncrypted,
		encryptedContent,
		wrappedDEK,
		encryptionMetadata,
		nil,
		folderPath,
	)
}

// CreateEncryptedCanvasNoteWithID creates a new encrypted canvas note with
// optional client-provided note ID.
func (s *CanvasService) CreateEncryptedCanvasNoteWithID(
	userID int,
	noteID string,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	_ []string,
	folderPath string,
) (*db.Note, error) {
	if err := s.notes.checkNoteLimit(userID); err != nil {
		return nil, err
	}

	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}

	note, err := s.db.CreateEncryptedCanvasNoteWithID(
		userID, noteID, title, encryptedTitle, titleEncrypted,
		encryptedContent, wrappedDEK, encryptionMetadata,
		nil, folderPath,
	)
	if err != nil {
		return nil, err
	}

	s.notes.invalidateRecoveryKeyBestEffort(userID)
	s.notes.invalidateFolderCache(userID)
	s.notes.invalidateQuickSearchCache(userID)
	s.notes.invalidateNotesByFolderCache(userID, folderPath)
	if s.notes.graphService != nil {
		s.notes.graphService.InvalidateGraphCache(userID)
	}

	return note, nil
}

// ListCanvasNotes returns all canvas notes for a user.
func (s *CanvasService) ListCanvasNotes(userID int) ([]db.Note, error) {
	return s.db.ListCanvasNotes(userID)
}

// ExportCanvas returns the canvas content as JSON Canvas format.
func (s *CanvasService) ExportCanvas(userID int, noteID string) (*db.Note, error) {
	note, err := s.db.GetNote(userID, noteID)
	if err != nil {
		return nil, err
	}
	if note == nil {
		return nil, db.ErrNotFound
	}
	if note.NoteType != db.NoteTypeCanvas {
		return nil, errors.New("note is not a canvas")
	}
	return note, nil
}
