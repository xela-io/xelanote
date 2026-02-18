package service

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/xela-io/xelanote/internal/db"
)

// Sharing-specific errors
var (
	ErrCannotShareEncrypted       = errors.New("E2E encrypted notes cannot be shared")
	ErrCannotShareWithSelf        = errors.New("cannot share a note with yourself")
	ErrNotNoteOwner               = errors.New("only the note owner can manage shares")
	ErrCannotShareEncryptedFolder = errors.New("folders with encryption enabled cannot be shared")
	ErrFolderHasEncryptedNotes    = errors.New("folder contains encrypted notes and cannot be shared")
	ErrNotFolderOwner             = errors.New("only the folder owner can manage folder shares")
	ErrNoShareAccess              = errors.New("you don't have access to this note")
	ErrCannotPlaceOwnNote         = errors.New("cannot place your own note — use move instead")
	ErrUserNotFound               = errors.New("user not found")
)

// SharingService handles note and folder sharing operations.
type SharingService struct {
	db     *db.DB
	logger *slog.Logger
}

// NewSharingService creates a new SharingService.
func NewSharingService(database *db.DB) *SharingService {
	return &SharingService{
		db:     database,
		logger: slog.Default(),
	}
}

func (s *SharingService) requireNoteOwner(ownerUserID int, noteID string) error {
	actualOwnerID, err := s.db.GetNoteOwnerUserID(noteID)
	if err != nil {
		return err
	}
	if actualOwnerID != ownerUserID {
		return ErrNotNoteOwner
	}
	return nil
}

func (s *SharingService) requireFolderOwner(ownerUserID, folderID int) error {
	actualOwnerID, err := s.db.GetFolderOwnerUserID(folderID)
	if err != nil {
		return err
	}
	if actualOwnerID != ownerUserID {
		return ErrNotFolderOwner
	}
	return nil
}

// ============================================================================
// Note Sharing
// ============================================================================

// ShareNote shares a note with another user.
// Validates ownership, encryption status, and prevents self-sharing.
func (s *SharingService) ShareNote(ownerUserID int, noteID string, targetIdentifier string, role string) (*db.NoteShare, error) {
	// Check ownership
	if err := s.requireNoteOwner(ownerUserID, noteID); err != nil {
		return nil, err
	}

	// Check if note is encrypted
	encrypted, err := s.db.IsNoteEncrypted(noteID)
	if err != nil {
		return nil, err
	}
	if encrypted {
		return nil, ErrCannotShareEncrypted
	}

	// Find target user by username or email
	targetUser, err := s.db.GetUserByUsernameOrEmail(targetIdentifier)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Prevent self-sharing
	if targetUser.ID == ownerUserID {
		return nil, ErrCannotShareWithSelf
	}

	// Create the share
	share, err := s.db.CreateNoteShare(ownerUserID, noteID, targetUser.ID, role)
	if err != nil {
		if errors.Is(err, db.ErrDuplicate) {
			return nil, fmt.Errorf("note is already shared with this user")
		}
		return nil, err
	}

	s.logger.Info("note shared",
		"note_id", noteID,
		"owner_id", ownerUserID,
		"shared_with_id", targetUser.ID,
		"role", role,
	)

	return share, nil
}

// UnshareNote removes a share for a note.
func (s *SharingService) UnshareNote(ownerUserID int, noteID string, targetUserID int) error {
	// Check ownership
	if err := s.requireNoteOwner(ownerUserID, noteID); err != nil {
		return err
	}

	return s.db.DeleteNoteShare(ownerUserID, noteID, targetUserID)
}

// GetNoteShares returns all shares for a note.
func (s *SharingService) GetNoteShares(ownerUserID int, noteID string) ([]db.NoteShare, error) {
	// Check ownership
	if err := s.requireNoteOwner(ownerUserID, noteID); err != nil {
		return nil, err
	}

	return s.db.GetNoteShares(ownerUserID, noteID)
}

// GetSharedNotesForUser returns all notes shared with a user.
func (s *SharingService) GetSharedNotesForUser(userID int) ([]db.SharedNote, error) {
	return s.db.GetSharedNotesForUser(userID)
}

// GetSharedNote returns a single shared note for a user.
func (s *SharingService) GetSharedNote(userID int, noteID string) (*db.SharedNote, error) {
	sn, err := s.db.GetSharedNote(userID, noteID)
	if err != nil {
		return nil, err
	}
	if sn == nil {
		return nil, db.ErrNotFound
	}
	return sn, nil
}

// UpdateShareRole updates the role for a share.
func (s *SharingService) UpdateShareRole(ownerUserID int, noteID string, targetUserID int, role string) error {
	// Check ownership
	if err := s.requireNoteOwner(ownerUserID, noteID); err != nil {
		return err
	}

	return s.db.UpdateNoteShareRole(ownerUserID, noteID, targetUserID, role)
}

// CanAccessSharedNote returns the share role for a user on a note, or empty string if no share.
func (s *SharingService) CanAccessSharedNote(userID int, noteID string) (string, error) {
	return s.db.GetSharePermission(userID, noteID)
}

// UpdateSharedNote updates a shared note (editor role required).
func (s *SharingService) UpdateSharedNote(userID int, noteID string, title, content string, expectedVersion int) (*db.SharedNote, error) {
	// Check share permission
	role, err := s.db.GetSharePermission(userID, noteID)
	if err != nil {
		return nil, err
	}
	if role == "" {
		return nil, db.ErrNotFound
	}
	if role != "editor" {
		return nil, fmt.Errorf("insufficient permissions: viewer cannot edit")
	}

	return s.db.UpdateSharedNote(noteID, title, content, expectedVersion)
}

// SearchUsers searches for users by username or email prefix.
func (s *SharingService) SearchUsers(query string, requestingUserID int) ([]db.UserSearchResult, error) {
	return s.db.SearchUserByUsernameOrEmail(query, requestingUserID)
}

// ============================================================================
// Folder Sharing
// ============================================================================

// ShareFolder shares a folder with another user.
// Validates ownership, encryption status, and prevents self-sharing.
func (s *SharingService) ShareFolder(ownerUserID, folderID int, targetIdentifier, role string) (*db.FolderShare, error) {
	// Check folder ownership
	if err := s.requireFolderOwner(ownerUserID, folderID); err != nil {
		return nil, err
	}

	// Check encryption_default on folder
	folder, err := s.db.GetFolderByID(ownerUserID, folderID)
	if err != nil {
		return nil, err
	}
	if folder.EncryptionDefault {
		return nil, ErrCannotShareEncryptedFolder
	}

	// Check if folder has any encrypted notes
	hasEncrypted, err := s.db.FolderHasEncryptedNotes(folderID)
	if err != nil {
		return nil, err
	}
	if hasEncrypted {
		return nil, ErrFolderHasEncryptedNotes
	}

	// Find target user
	targetUser, err := s.db.GetUserByUsernameOrEmail(targetIdentifier)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Prevent self-sharing
	if targetUser.ID == ownerUserID {
		return nil, ErrCannotShareWithSelf
	}

	// Create the share
	share, err := s.db.CreateFolderShare(ownerUserID, folderID, targetUser.ID, role)
	if err != nil {
		if errors.Is(err, db.ErrDuplicate) {
			return nil, fmt.Errorf("folder is already shared with this user")
		}
		return nil, err
	}

	s.logger.Info("folder shared",
		"folder_id", folderID,
		"owner_id", ownerUserID,
		"shared_with_id", targetUser.ID,
		"role", role,
	)

	return share, nil
}

// UnshareFolder removes a folder share.
func (s *SharingService) UnshareFolder(ownerUserID, folderID, targetUserID int) error {
	// Check ownership
	if err := s.requireFolderOwner(ownerUserID, folderID); err != nil {
		return err
	}

	return s.db.DeleteFolderShare(ownerUserID, folderID, targetUserID)
}

// GetFolderShares returns all shares for a folder.
func (s *SharingService) GetFolderShares(ownerUserID, folderID int) ([]db.FolderShare, error) {
	// Check ownership
	if err := s.requireFolderOwner(ownerUserID, folderID); err != nil {
		return nil, err
	}

	return s.db.GetFolderShares(ownerUserID, folderID)
}

// GetSharedFoldersForUser returns all folders shared with a user.
func (s *SharingService) GetSharedFoldersForUser(userID int) ([]db.SharedFolder, error) {
	return s.db.GetSharedFoldersForUser(userID)
}

// GetSharedFolderNotes returns notes in a shared folder.
func (s *SharingService) GetSharedFolderNotes(userID, folderID int) ([]db.SharedNote, error) {
	return s.db.GetSharedFolderNotes(userID, folderID)
}

// UpdateFolderShareRole updates the role for a folder share.
func (s *SharingService) UpdateFolderShareRole(ownerUserID, folderID, targetUserID int, role string) error {
	// Check ownership
	if err := s.requireFolderOwner(ownerUserID, folderID); err != nil {
		return err
	}

	return s.db.UpdateFolderShareRole(ownerUserID, folderID, targetUserID, role)
}

// ============================================================================
// Placements
// ============================================================================

// PlaceSharedNote places a shared note into the user's own folder.
func (s *SharingService) PlaceSharedNote(userID int, noteID string, folderID int) error {
	// 1. Check that user has share access
	role, err := s.db.GetSharePermission(userID, noteID)
	if err != nil {
		return err
	}
	if role == "" {
		return ErrNoShareAccess
	}

	// 2. Verify note owner != current user
	noteOwnerID, err := s.db.GetNoteOwnerUserID(noteID)
	if err != nil {
		return err
	}
	if noteOwnerID == userID {
		return ErrCannotPlaceOwnNote
	}

	// 3. Verify target folder belongs to user
	folder, err := s.db.GetFolderByID(userID, folderID)
	if err != nil {
		return fmt.Errorf("target folder not found")
	}
	_ = folder

	// 4. Create/update placement (DB layer does additional share check)
	return s.db.CreateOrUpdatePlacement(userID, noteID, folderID)
}

// RemovePlacement removes a shared note placement.
func (s *SharingService) RemovePlacement(userID int, noteID string) error {
	return s.db.DeletePlacement(userID, noteID)
}
