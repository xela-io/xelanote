package service

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/xela-io/xelanote/internal/db"
)

// SetRecoveryKey sets a recovery key for password recovery
// The recoveryKeyHash should be a bcrypt hash of the user-provided recovery key
// The salt is used for client-side Argon2id key derivation
func (s *UserService) SetRecoveryKey(userID int, recoveryKeyHash string, salt []byte) error {
	if recoveryKeyHash == "" {
		return errors.New("recovery key hash is required")
	}
	if len(salt) == 0 {
		return errors.New("recovery key salt is required")
	}

	return s.db.SetRecoveryKey(userID, recoveryKeyHash, salt)
}

// GetRecoveryKeySalt retrieves the recovery key salt for a user
// Returns ErrNotFound if no recovery key is set
func (s *UserService) GetRecoveryKeySalt(userID int) ([]byte, error) {
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
			bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(recoveryKey))
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

	salt, err := s.db.GetRecoveryKeySalt(user.ID)
	if err != nil {
		// Don't reveal whether recovery key exists - use same generic error
		return nil, errors.New("recovery key not available")
	}
	return salt, nil
}
