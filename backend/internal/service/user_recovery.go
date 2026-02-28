package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/xela-io/xelanote/internal/db"
)

const (
	recoveryResetTokenTTL   = 15 * time.Minute
	recoveryResetTokenBytes = 32
)

// SetRecoveryKey sets a recovery key for password recovery
// The recoveryKeyHash should be a bcrypt hash of the user-provided recovery key
// The salt is used for client-side Argon2id key derivation
func (s *UserService) SetRecoveryKey(userID int, recoveryKeyHash string, salt []byte) error {
	return s.SetRecoveryKeyWithRecoveryWrappedDEKs(userID, recoveryKeyHash, salt, nil, nil)
}

// SetRecoveryKeyWithRecoveryWrappedDEKs stores recovery credentials.
// For encrypted accounts this requires complete recovery wrappers for notes and versions.
func (s *UserService) SetRecoveryKeyWithRecoveryWrappedDEKs(
	userID int,
	recoveryKeyHash string,
	salt []byte,
	recoveryWrappedNotes map[string]string,
	recoveryWrappedVersions map[string]string,
) error {
	if recoveryKeyHash == "" {
		return errors.New("recovery key hash is required")
	}
	if len(salt) == 0 {
		return errors.New("recovery key salt is required")
	}
	encryptedNotes, err := s.db.GetAllEncryptedNotesForUser(userID)
	if err != nil {
		return fmt.Errorf("failed to check encrypted notes: %w", err)
	}
	encryptedVersions, err := s.db.GetAllEncryptedVersionsForUser(userID)
	if err != nil {
		return fmt.Errorf("failed to check encrypted versions: %w", err)
	}

	hasEncryptedContent := len(encryptedNotes) > 0 || len(encryptedVersions) > 0
	if !hasEncryptedContent {
		if len(recoveryWrappedNotes) > 0 || len(recoveryWrappedVersions) > 0 {
			return errors.New("no encrypted content to re-wrap")
		}
		return s.db.SetRecoveryKey(userID, recoveryKeyHash, salt)
	}

	if len(recoveryWrappedNotes) == 0 && len(recoveryWrappedVersions) == 0 {
		return ErrRecoveryWrappedDEKsRequired
	}

	allowedNoteIDs := make(map[string]struct{}, len(encryptedNotes))
	allowedVersionIDs := make(map[string]struct{}, len(encryptedVersions))

	for _, note := range encryptedNotes {
		allowedNoteIDs[note.ID] = struct{}{}
		wrappedDEK, ok := recoveryWrappedNotes[note.ID]
		if !ok {
			return fmt.Errorf("missing recovery-wrapped DEK for note %s", note.ID)
		}
		if err := validateReWrappedDEKValue(wrappedDEK); err != nil {
			return fmt.Errorf("invalid recovery-wrapped DEK for note %s: %w", note.ID, err)
		}
	}

	for _, version := range encryptedVersions {
		versionIDStr := fmt.Sprintf("%d", version.ID)
		allowedVersionIDs[versionIDStr] = struct{}{}
		wrappedDEK, ok := recoveryWrappedVersions[versionIDStr]
		if !ok {
			return fmt.Errorf("missing recovery-wrapped DEK for version %d", version.ID)
		}
		if err := validateReWrappedDEKValue(wrappedDEK); err != nil {
			return fmt.Errorf("invalid recovery-wrapped DEK for version %d: %w", version.ID, err)
		}
	}

	for noteID := range recoveryWrappedNotes {
		if _, ok := allowedNoteIDs[noteID]; !ok {
			return fmt.Errorf("unexpected recovery-wrapped DEK for note %s", noteID)
		}
	}
	for versionID := range recoveryWrappedVersions {
		if _, ok := allowedVersionIDs[versionID]; !ok {
			return fmt.Errorf("unexpected recovery-wrapped DEK for version %s", versionID)
		}
	}

	tx, err := s.db.BeginTx(context.Background())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := tx.SetRecoveryKeyTx(userID, recoveryKeyHash, salt); err != nil {
		return fmt.Errorf("failed to set recovery key: %w", err)
	}
	if err := tx.BulkUpdateRecoveryWrappedDEKsTx(userID, recoveryWrappedNotes, recoveryWrappedVersions); err != nil {
		return fmt.Errorf("failed to store recovery wrapped DEKs: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit recovery key setup: %w", err)
	}

	return nil
}

// GetRecoveryKeySalt retrieves the recovery key salt for a user
// Returns ErrNotFound if no recovery key is set
func (s *UserService) GetRecoveryKeySalt(userID int) ([]byte, error) {
	ok, err := s.isRecoveryReady(userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, db.ErrNotFound
	}

	return s.db.GetRecoveryKeySalt(userID)
}

// RecoverPasswordWithRecoveryKey resets a user's password using a recovery key
// This function:
// 1. Validates the recovery key
// 2. Updates the password
// 3. Invalidates all sessions (including current)
func (s *UserService) RecoverPasswordWithRecoveryKey(userID int, recoveryKey, newPassword string) error {
	// Validate new password length
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	// Get user preferences to retrieve recovery key hash
	prefs, err := s.db.GetUserPreferences(userID)
	if err != nil {
		return err
	}

	// Check if recovery key is set
	if prefs.RecoveryKeyHash == nil || *prefs.RecoveryKeyHash == "" {
		return errors.New("no recovery key set for this user")
	}

	// Verify recovery key
	err = bcrypt.CompareHashAndPassword([]byte(*prefs.RecoveryKeyHash), []byte(recoveryKey))
	if err != nil {
		return errors.New("invalid recovery key")
	}

	// Hard block for encrypted users until recovery-based DEK re-wrap is implemented.
	hasEncryptedContent, err := s.hasEncryptedNotesOrVersions(userID)
	if err != nil {
		return fmt.Errorf("failed to check encrypted content: %w", err)
	}
	if hasEncryptedContent {
		return ErrRecoveryResetNeedsDEKRewrap
	}

	// Hash new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}

	// Update password
	err = s.db.UpdateUserPassword(userID, string(newPasswordHash))
	if err != nil {
		return err
	}

	// Invalidate ALL sessions (user must log in with new password)
	err = s.db.DeleteAllUserRefreshTokens(userID)
	if err != nil {
		// Log but don't fail - password was already changed
	}

	return nil
}

// RecoverPasswordWithRecoveryKeyByEmail resets a user's password using email + recovery key
// This is a public endpoint (no authentication required) for password recovery
func (s *UserService) RecoverPasswordWithRecoveryKeyByEmail(email, recoveryKey, newPassword string) error {
	// Validate inputs
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return errors.New("email is required")
	}
	if recoveryKey == "" {
		return errors.New("recovery key is required")
	}
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	// Get user by email
	user, err := s.db.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// Don't reveal whether user exists (timing attack mitigation)
			// Perform a dummy bcrypt comparison to maintain constant time
			_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(recoveryKey))
			return errors.New("invalid email or recovery key")
		}
		return err
	}

	// Get user preferences to retrieve recovery key hash
	prefs, err := s.db.GetUserPreferences(user.ID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return errors.New("invalid recovery request")
		}
		return err
	}

	// Check if recovery key is set
	if prefs.RecoveryKeyHash == nil || *prefs.RecoveryKeyHash == "" {
		return errors.New("invalid recovery request")
	}

	// Verify recovery key
	err = bcrypt.CompareHashAndPassword([]byte(*prefs.RecoveryKeyHash), []byte(recoveryKey))
	if err != nil {
		return errors.New("invalid email or recovery key")
	}

	// Hard block for encrypted users until recovery-based DEK re-wrap is implemented.
	hasEncryptedContent, err := s.hasEncryptedNotesOrVersions(user.ID)
	if err != nil {
		return fmt.Errorf("failed to check encrypted content: %w", err)
	}
	if hasEncryptedContent {
		return ErrRecoveryResetNeedsDEKRewrap
	}

	// Hash new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}

	// Update password
	err = s.db.UpdateUserPassword(user.ID, string(newPasswordHash))
	if err != nil {
		return err
	}

	// Invalidate ALL sessions (user must log in with new password)
	err = s.db.DeleteAllUserRefreshTokens(user.ID)
	if err != nil {
		// Log but don't fail - password was already changed
	}

	return nil
}

// GetRecoveryKeySaltByEmail retrieves the recovery key salt for a user by email
// This is a public endpoint for the recovery flow
func (s *UserService) GetRecoveryKeySaltByEmail(email string) ([]byte, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, errors.New("email is required")
	}

	// Get user by email
	user, err := s.db.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// Don't reveal whether user exists - use generic error
			return nil, errors.New("recovery key not available")
		}
		return nil, err
	}

	hasEncryptedContent, err := s.hasEncryptedNotesOrVersions(user.ID)
	if err != nil {
		return nil, err
	}
	if hasEncryptedContent {
		ready, readyErr := s.isRecoveryReady(user.ID)
		if readyErr != nil {
			return nil, readyErr
		}
		if !ready {
			return nil, errors.New("recovery key not available")
		}
	}

	salt, err := s.db.GetRecoveryKeySalt(user.ID)
	if err != nil {
		// Don't reveal whether recovery key exists - use same generic error
		return nil, errors.New("recovery key not available")
	}
	return salt, nil
}

// BeginRecoveryResetByEmail verifies recovery credentials and creates a short-lived one-time reset token.
func (s *UserService) BeginRecoveryResetByEmail(email, recoveryKey string) (*RecoveryVerifyResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || recoveryKey == "" {
		return nil, errors.New("invalid email or recovery key")
	}

	user, err := s.db.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(recoveryKey))
			return nil, errors.New("invalid email or recovery key")
		}
		return nil, err
	}

	prefs, err := s.db.GetUserPreferences(user.ID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, errors.New("invalid email or recovery key")
		}
		return nil, err
	}
	if prefs.RecoveryKeyHash == nil || *prefs.RecoveryKeyHash == "" {
		return nil, errors.New("invalid email or recovery key")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*prefs.RecoveryKeyHash), []byte(recoveryKey)); err != nil {
		return nil, errors.New("invalid email or recovery key")
	}

	rawToken, tokenHash, err := generateRecoveryResetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate recovery reset token: %w", err)
	}

	if err := s.db.CreateRecoveryResetToken(user.ID, tokenHash, time.Now().UTC().Add(recoveryResetTokenTTL)); err != nil {
		return nil, fmt.Errorf("failed to store recovery reset token: %w", err)
	}

	return &RecoveryVerifyResult{
		RecoveryResetToken: rawToken,
		EncryptionSalt:     derefString(user.EncryptionSalt),
	}, nil
}

// GetRecoveryWrappedDEKs lists encrypted note/version recovery wrappers for a valid reset token.
func (s *UserService) GetRecoveryWrappedDEKs(recoveryResetToken string) ([]RecoveryWrappedDEKEntry, []RecoveryWrappedDEKEntry, error) {
	if recoveryResetToken == "" {
		return nil, nil, ErrInvalidRecoveryResetToken
	}

	userID, err := s.db.ValidateRecoveryResetToken(hashRecoveryResetToken(recoveryResetToken))
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, nil, ErrInvalidRecoveryResetToken
		}
		return nil, nil, err
	}

	notes, err := s.db.GetAllEncryptedNotesForUser(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch encrypted notes: %w", err)
	}
	versions, err := s.db.GetAllEncryptedVersionsForUser(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch encrypted versions: %w", err)
	}

	noteEntries := make([]RecoveryWrappedDEKEntry, 0, len(notes))
	for _, note := range notes {
		if note.WrappedDEKRecovery == "" {
			return nil, nil, ErrRecoveryRewrapNotConfigured
		}
		noteEntries = append(noteEntries, RecoveryWrappedDEKEntry{
			ID:                 note.ID,
			WrappedDEKRecovery: note.WrappedDEKRecovery,
		})
	}

	versionEntries := make([]RecoveryWrappedDEKEntry, 0, len(versions))
	for _, version := range versions {
		if version.WrappedDEKRecovery == "" {
			return nil, nil, ErrRecoveryRewrapNotConfigured
		}
		versionEntries = append(versionEntries, RecoveryWrappedDEKEntry{
			ID:                 fmt.Sprintf("%d", version.ID),
			WrappedDEKRecovery: version.WrappedDEKRecovery,
		})
	}

	return noteEntries, versionEntries, nil
}

// FinalizeRecoveryResetWithToken updates password and wrapped DEKs atomically using a one-time recovery token.
func (s *UserService) FinalizeRecoveryResetWithToken(
	recoveryResetToken string,
	newPassword string,
	reWrappedNotes map[string]string,
	reWrappedVersions map[string]string,
) error {
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}
	if recoveryResetToken == "" {
		return ErrInvalidRecoveryResetToken
	}

	tokenHash := hashRecoveryResetToken(recoveryResetToken)
	userID, err := s.db.ValidateRecoveryResetToken(tokenHash)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return ErrInvalidRecoveryResetToken
		}
		return err
	}

	shouldUpdateWrappedDEKs, err := s.validateReWrappedDEKCoverage(userID, reWrappedNotes, reWrappedVersions)
	if err != nil {
		return err
	}

	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(context.Background())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	tokenUserID, err := tx.ConsumeRecoveryResetTokenTx(tokenHash)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return ErrInvalidRecoveryResetToken
		}
		return fmt.Errorf("failed to consume recovery reset token: %w", err)
	}
	if tokenUserID != userID {
		return ErrInvalidRecoveryResetToken
	}

	if err := tx.UpdateUserPasswordTx(userID, string(newPasswordHash)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if shouldUpdateWrappedDEKs {
		if err := tx.BulkUpdateWrappedDEKsTx(userID, reWrappedNotes, reWrappedVersions); err != nil {
			return fmt.Errorf("failed to update wrapped DEKs: %w", err)
		}
	}

	if err := tx.InvalidateRecoveryKeyTx(userID); err != nil {
		return fmt.Errorf("failed to invalidate recovery key: %w", err)
	}
	if err := tx.ClearRecoveryWrappedDEKsTx(userID); err != nil {
		return fmt.Errorf("failed to clear recovery wrapped DEKs: %w", err)
	}
	if err := tx.DeleteAllUserRefreshTokensTx(userID); err != nil {
		return fmt.Errorf("failed to invalidate user sessions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit recovery reset: %w", err)
	}

	return nil
}

func (s *UserService) hasEncryptedNotesOrVersions(userID int) (bool, error) {
	encryptedNotes, err := s.db.GetAllEncryptedNotesForUser(userID)
	if err != nil {
		return false, err
	}

	encryptedVersions, err := s.db.GetAllEncryptedVersionsForUser(userID)
	if err != nil {
		return false, err
	}

	return len(encryptedNotes) > 0 || len(encryptedVersions) > 0, nil
}

func (s *UserService) isRecoveryReady(userID int) (bool, error) {
	encryptedNotes, err := s.db.GetAllEncryptedNotesForUser(userID)
	if err != nil {
		return false, fmt.Errorf("failed to check encrypted notes: %w", err)
	}
	encryptedVersions, err := s.db.GetAllEncryptedVersionsForUser(userID)
	if err != nil {
		return false, fmt.Errorf("failed to check encrypted versions: %w", err)
	}

	if len(encryptedNotes) == 0 && len(encryptedVersions) == 0 {
		return true, nil
	}

	for _, note := range encryptedNotes {
		if note.WrappedDEKRecovery == "" {
			return false, nil
		}
	}
	for _, version := range encryptedVersions {
		if version.WrappedDEKRecovery == "" {
			return false, nil
		}
	}

	return true, nil
}

func generateRecoveryResetToken() (string, string, error) {
	raw := make([]byte, recoveryResetTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}

	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, hashRecoveryResetToken(token), nil
}

func hashRecoveryResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
