package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/xela-io/xelanote/internal/db"
)

const (
	minReWrappedDEKBytes = 32
	maxReWrappedDEKBytes = 256
)

// ChangeEmail changes a user's email with password verification and session invalidation
func (s *UserService) ChangeEmail(userID int, newEmail, currentPassword, currentRefreshToken string) error {
	// Validate email format using net/mail parser
	newEmail = strings.TrimSpace(newEmail)
	if newEmail == "" || !isValidEmail(newEmail) {
		return ErrInvalidEmail
	}

	// Get current user
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword))
	if err != nil {
		return ErrInvalidPassword
	}

	// Check if email is already in use
	_, err = s.db.GetUserByEmail(newEmail)
	if err == nil {
		return ErrEmailInUse
	}
	if err != db.ErrNotFound {
		return err
	}

	// Update email
	err = s.db.UpdateUserEmail(userID, newEmail)
	if err != nil {
		if err == db.ErrDuplicate {
			return ErrEmailInUse
		}
		return err
	}

	// Invalidate other sessions (keep current session)
	if currentRefreshToken != "" {
		err = s.db.DeleteAllUserRefreshTokensExcept(userID, currentRefreshToken)
		if err != nil {
			// Log but don't fail the operation
			// The email was already changed successfully
		}
	}

	return nil
}

// ChangePassword changes a user's password with verification and session invalidation.
// Deprecated: Use ChangePasswordWithDEKRewrap for users with encrypted notes.
// This function is kept for backwards compatibility with users who have no encrypted content.
func (s *UserService) ChangePassword(userID int, currentPassword, newPassword, currentRefreshToken string) error {
	return s.ChangePasswordWithDEKRewrap(userID, currentPassword, newPassword, nil, nil, currentRefreshToken)
}

// ChangePasswordWithDEKRewrap changes a user's password and re-wraps all encrypted DEKs atomically.
// For backwards compatibility, reWrappedNotes and reWrappedVersions are optional:
// - If both are nil/empty AND user has no encrypted notes: password is changed without DEK re-wrapping
// - If both are nil/empty BUT user has encrypted notes: returns error (re-wrapping required)
// - If provided: validates that all encrypted notes/versions are included and updates them atomically
//
// CRITICAL: Password update and DEK re-wrapping happen in a SINGLE transaction.
// If either fails, the entire operation is rolled back to prevent data corruption.
func (s *UserService) ChangePasswordWithDEKRewrap(
	userID int,
	currentPassword string,
	newPassword string,
	reWrappedNotes map[string]string, // noteID -> new_wrapped_dek
	reWrappedVersions map[string]string, // versionID -> new_wrapped_dek
	currentRefreshToken string,
) error {
	// Validate new password length
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	// Get current user
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword))
	if err != nil {
		return ErrInvalidPassword
	}

	shouldUpdateWrappedDEKs, err := s.validateReWrappedDEKCoverage(userID, reWrappedNotes, reWrappedVersions)
	if err != nil {
		return err
	}

	// Hash new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}

	// Start atomic transaction for password + DEK updates
	tx, err := s.db.BeginTx(context.Background())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update password within transaction
	err = tx.UpdateUserPasswordTx(userID, string(newPasswordHash))
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Update wrapped DEKs within the SAME transaction
	if shouldUpdateWrappedDEKs {
		err = tx.BulkUpdateWrappedDEKsTx(userID, reWrappedNotes, reWrappedVersions)
		if err != nil {
			return fmt.Errorf("failed to update wrapped DEKs: %w", err)
		}
	}

	// Commit the atomic transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit password change: %w", err)
	}

	// Post-commit cleanup (non-critical, can fail without rolling back)

	// Invalidate recovery key (it was derived from the old password)
	err = s.db.InvalidateRecoveryKey(userID)
	if err != nil {
		// Log but don't fail - recovery key invalidation is non-critical
		// User will need to set a new recovery key
	}

	// Invalidate other sessions (keep current session)
	if currentRefreshToken != "" {
		err = s.db.DeleteAllUserRefreshTokensExcept(userID, currentRefreshToken)
		if err != nil {
			// Log but don't fail the operation
			// The password was already changed successfully
		}
	}

	return nil
}

func (s *UserService) validateReWrappedDEKCoverage(
	userID int,
	reWrappedNotes map[string]string,
	reWrappedVersions map[string]string,
) (bool, error) {
	encryptedNotes, err := s.db.GetAllEncryptedNotesForUser(userID)
	if err != nil {
		return false, fmt.Errorf("failed to check encrypted notes: %w", err)
	}

	encryptedVersions, err := s.db.GetAllEncryptedVersionsForUser(userID)
	if err != nil {
		return false, fmt.Errorf("failed to check encrypted versions: %w", err)
	}

	hasEncryptedContent := len(encryptedNotes) > 0 || len(encryptedVersions) > 0

	if len(reWrappedNotes) == 0 && len(reWrappedVersions) == 0 {
		if hasEncryptedContent {
			return false, errors.New("DEK re-wrapping required: user has encrypted notes or versions")
		}
		return false, nil
	}

	if !hasEncryptedContent {
		return false, errors.New("no encrypted content to re-wrap")
	}

	allowedNoteIDs := make(map[string]struct{}, len(encryptedNotes))
	allowedVersionIDs := make(map[string]struct{}, len(encryptedVersions))

	for _, note := range encryptedNotes {
		allowedNoteIDs[note.ID] = struct{}{}
		if _, ok := reWrappedNotes[note.ID]; !ok {
			return false, fmt.Errorf("missing re-wrapped DEK for note %s", note.ID)
		}
		if err := validateReWrappedDEKValue(reWrappedNotes[note.ID]); err != nil {
			return false, fmt.Errorf("invalid re-wrapped DEK for note %s: %w", note.ID, err)
		}
	}

	for _, version := range encryptedVersions {
		versionIDStr := fmt.Sprintf("%d", version.ID)
		allowedVersionIDs[versionIDStr] = struct{}{}
		if _, ok := reWrappedVersions[versionIDStr]; !ok {
			return false, fmt.Errorf("missing re-wrapped DEK for version %d", version.ID)
		}
		if err := validateReWrappedDEKValue(reWrappedVersions[versionIDStr]); err != nil {
			return false, fmt.Errorf("invalid re-wrapped DEK for version %d: %w", version.ID, err)
		}
	}

	for noteID := range reWrappedNotes {
		if _, ok := allowedNoteIDs[noteID]; !ok {
			return false, fmt.Errorf("unexpected re-wrapped DEK for note %s", noteID)
		}
	}

	for versionID := range reWrappedVersions {
		if _, ok := allowedVersionIDs[versionID]; !ok {
			return false, fmt.Errorf("unexpected re-wrapped DEK for version %s", versionID)
		}
	}

	return true, nil
}

func validateReWrappedDEKValue(base64DEK string) error {
	if base64DEK == "" {
		return errors.New("wrapped DEK is required")
	}

	decoded, err := base64.StdEncoding.DecodeString(base64DEK)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding: %w", err)
	}

	if len(decoded) < minReWrappedDEKBytes {
		return fmt.Errorf("wrapped DEK too short (%d bytes)", len(decoded))
	}
	if len(decoded) > maxReWrappedDEKBytes {
		return fmt.Errorf("wrapped DEK too long (%d bytes)", len(decoded))
	}

	return nil
}
