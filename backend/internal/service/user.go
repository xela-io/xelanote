package service

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/xela-io/xelanote/internal/db"
)

// Validation errors
var (
	ErrInvalidTheme      = errors.New("invalid theme")
	ErrInvalidEditorMode = errors.New("invalid editor mode")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrEmailInUse        = errors.New("email already in use")
	ErrPasswordTooShort  = errors.New("password must be at least 8 characters")
	ErrInvalidEmail      = errors.New("invalid email format")
)

// Valid theme IDs (must match frontend themes/index.ts)
var validThemes = map[string]bool{
	"default-light":    true,
	"default-dark":     true,
	"nord-light":       true,
	"nord-dark":        true,
	"solarized-light":  true,
	"solarized-dark":   true,
	"dracula":          true,
	"catppuccin-latte": true,
	"catppuccin-mocha": true,
	"dark-pastels":     true,
	"gruvbox-light":    true,
	"gruvbox-dark":     true,
	"tokyo-night":      true,
	"one-dark":         true,
	"one-light":        true,
	"monokai":          true,
	"ayu-light":        true,
	"ayu-mirage":       true,
	"rose-pine-moon":   true,
	"rose-pine-dawn":   true,
	"kanagawa":         true,
	"everforest-dark":  true,
	"everforest-light": true,
}

// Valid editor modes
var validEditorModes = map[string]bool{
	"edit":    true,
	"preview": true,
	"split":   true,
}

// UserService handles user-related business logic
type UserService struct {
	db *db.DB
}

// NewUserService creates a new UserService
func NewUserService(database *db.DB) *UserService {
	return &UserService{
		db: database,
	}
}

// isValidTheme checks if a theme ID is valid
func isValidTheme(theme string) bool {
	return validThemes[theme]
}

// isValidEditorMode checks if an editor mode is valid
func isValidEditorMode(mode string) bool {
	return validEditorModes[mode]
}

// isValidEmail validates an email address using Go's net/mail package.
// This is more robust than a simple @ check and handles edge cases.
func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// GetOrCreatePreferences retrieves user preferences, creating defaults if needed
// Returns preferences and a boolean indicating if the row was newly created
func (s *UserService) GetOrCreatePreferences(userID int) (*db.UserPreferences, bool, error) {
	return s.db.GetOrCreateUserPreferences(userID)
}

// UpdatePreferences updates user preferences with validation
func (s *UserService) UpdatePreferences(userID int, theme, editorMode string) (*db.UserPreferences, error) {
	// Validate theme
	if !isValidTheme(theme) {
		return nil, ErrInvalidTheme
	}

	// Validate editor mode
	if !isValidEditorMode(editorMode) {
		return nil, ErrInvalidEditorMode
	}

	return s.db.UpsertUserPreferences(userID, theme, editorMode)
}

// UpdateEncryptionPreferences updates encryption-related user preferences
func (s *UserService) UpdateEncryptionPreferences(userID int, keywordsEnabled, encryptTitles bool) error {
	return s.db.UpdateEncryptionPreferences(userID, keywordsEnabled, encryptTitles)
}

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

	// Check if user has encrypted notes/versions
	encryptedNotes, err := s.db.GetAllEncryptedNotesForUser(userID)
	if err != nil {
		return fmt.Errorf("failed to check encrypted notes: %w", err)
	}

	encryptedVersions, err := s.db.GetAllEncryptedVersionsForUser(userID)
	if err != nil {
		return fmt.Errorf("failed to check encrypted versions: %w", err)
	}

	hasEncryptedContent := len(encryptedNotes) > 0 || len(encryptedVersions) > 0

	// Backwards compatibility: if no re-wrapped DEKs provided
	if len(reWrappedNotes) == 0 && len(reWrappedVersions) == 0 {
		if hasEncryptedContent {
			return errors.New("DEK re-wrapping required: user has encrypted notes or versions")
		}
		// User has no encrypted content, proceed with simple password change
	} else {
		// Validate that all encrypted notes are included in reWrappedNotes
		for _, note := range encryptedNotes {
			if _, ok := reWrappedNotes[note.ID]; !ok {
				return fmt.Errorf("missing re-wrapped DEK for note %s", note.ID)
			}
		}

		// Validate that all encrypted versions are included in reWrappedVersions
		for _, version := range encryptedVersions {
			versionIDStr := fmt.Sprintf("%d", version.ID)
			if _, ok := reWrappedVersions[versionIDStr]; !ok {
				return fmt.Errorf("missing re-wrapped DEK for version %d", version.ID)
			}
		}
	}

	// Hash new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}

	// Start atomic transaction for password + DEK updates
	tx, err := s.db.BeginTx()
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
	if len(reWrappedNotes) > 0 || len(reWrappedVersions) > 0 {
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
		if err == db.ErrNotFound {
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
		if err == db.ErrNotFound {
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
		if err == db.ErrNotFound {
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

// UpdateSecurityPreferences updates security-related user preferences
// (security_level and auto_lock_timeout)
func (s *UserService) UpdateSecurityPreferences(userID int, securityLevel *string, autoLockTimeout *int) (*db.UserPreferences, error) {
	// Get existing preferences
	prefs, _, err := s.GetOrCreatePreferences(userID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if securityLevel != nil {
		prefs.SecurityLevel = *securityLevel
	}
	if autoLockTimeout != nil {
		prefs.AutoLockTimeout = *autoLockTimeout
	}

	// Save to database
	err = s.db.UpdateSecurityPreferences(userID, prefs.SecurityLevel, prefs.AutoLockTimeout)
	if err != nil {
		return nil, err
	}

	// Return updated preferences
	return s.db.GetUserPreferences(userID)
}

// AddWebAuthnCredential adds a new WebAuthn credential for a user
func (s *UserService) AddWebAuthnCredential(userID int64, credentialID, deviceName string) (*db.WebAuthnCredential, error) {
	return s.db.AddWebAuthnCredential(userID, credentialID, deviceName)
}

// GetWebAuthnCredentials retrieves all WebAuthn credentials for a user
func (s *UserService) GetWebAuthnCredentials(userID int64) ([]db.WebAuthnCredential, error) {
	return s.db.GetWebAuthnCredentials(userID)
}

// DeleteWebAuthnCredential deletes a WebAuthn credential
func (s *UserService) DeleteWebAuthnCredential(userID int64, credentialID string) error {
	return s.db.DeleteWebAuthnCredential(userID, credentialID)
}

// TouchWebAuthnCredential updates the last_used_at timestamp for a credential
func (s *UserService) TouchWebAuthnCredential(userID int64, credentialID string) error {
	return s.db.TouchWebAuthnCredential(userID, credentialID)
}

// --- Claude API Key Management (BYOK) ---

// Validation errors for Claude API Key
var (
	ErrInvalidClaudeAPIKey = errors.New("invalid Claude API key format")
	ErrNoClaudeAPIKey      = errors.New("no Claude API key configured")
)

// SetClaudeAPIKey validates, encrypts, and stores a Claude API key for a user.
// The key is encrypted with AES-256-GCM before storage.
func (s *UserService) SetClaudeAPIKey(userID int, apiKey string) error {
	// Import crypto package inline to avoid circular dependency
	// crypto.ValidateClaudeAPIKey and crypto.EncryptAPIKey will be used
	return setClaudeAPIKeyImpl(s.db, userID, apiKey)
}

// GetClaudeAPIKey retrieves and decrypts the Claude API key for a user.
// Returns ErrNoClaudeAPIKey if no key is stored.
func (s *UserService) GetClaudeAPIKey(userID int) (string, error) {
	return getClaudeAPIKeyImpl(s.db, userID)
}

// DeleteClaudeAPIKey removes the Claude API key for a user.
func (s *UserService) DeleteClaudeAPIKey(userID int) error {
	return s.db.DeleteClaudeAPIKey(userID)
}

// HasClaudeAPIKey checks if a user has a Claude API key stored.
func (s *UserService) HasClaudeAPIKey(userID int) (bool, error) {
	return s.db.HasClaudeAPIKey(userID)
}

// GetClaudeAPIKeyStatus returns status information about the stored API key.
// Does NOT return the actual key, only metadata.
func (s *UserService) GetClaudeAPIKeyStatus(userID int) (*ClaudeAPIKeyStatus, error) {
	hasKey, err := s.db.HasClaudeAPIKey(userID)
	if err != nil {
		return nil, err
	}

	if !hasKey {
		return &ClaudeAPIKeyStatus{
			HasKey:    false,
			UpdatedAt: nil,
			MaskedKey: nil,
		}, nil
	}

	updatedAt, err := s.db.GetClaudeAPIKeyUpdatedAt(userID)
	if err != nil {
		return nil, err
	}

	// Get masked key (decrypt and mask)
	decryptedKey, err := getClaudeAPIKeyImpl(s.db, userID)
	if err != nil {
		return nil, err
	}

	masked := maskClaudeAPIKey(decryptedKey)

	return &ClaudeAPIKeyStatus{
		HasKey:    true,
		UpdatedAt: updatedAt,
		MaskedKey: &masked,
	}, nil
}

// ClaudeAPIKeyStatus represents the status of a user's Claude API key.
type ClaudeAPIKeyStatus struct {
	HasKey    bool    `json:"has_key"`
	UpdatedAt *string `json:"updated_at,omitempty"`
	MaskedKey *string `json:"masked_key,omitempty"` // e.g., "sk-ant-api0...xxxx"
}

// maskClaudeAPIKey returns a masked version of the API key for display.
// Shows first 10 and last 4 characters.
func maskClaudeAPIKey(apiKey string) string {
	if len(apiKey) <= 14 {
		return "****"
	}
	return apiKey[:10] + "..." + apiKey[len(apiKey)-4:]
}

// --- Gemini API Key Management (BYOK) ---

// Validation errors for Gemini API Key
var (
	ErrInvalidGeminiAPIKey = errors.New("invalid Gemini API key format")
	ErrNoGeminiAPIKey      = errors.New("no Gemini API key configured")
)

// SetGeminiAPIKey validates, encrypts, and stores a Gemini API key for a user.
// The key is encrypted with AES-256-GCM before storage.
func (s *UserService) SetGeminiAPIKey(userID int, apiKey string) error {
	return setGeminiAPIKeyImpl(s.db, userID, apiKey)
}

// GetGeminiAPIKey retrieves and decrypts the Gemini API key for a user.
// Returns ErrNoGeminiAPIKey if no key is stored.
func (s *UserService) GetGeminiAPIKey(userID int) (string, error) {
	return getGeminiAPIKeyImpl(s.db, userID)
}

// DeleteGeminiAPIKey removes the Gemini API key for a user.
func (s *UserService) DeleteGeminiAPIKey(userID int) error {
	return s.db.DeleteGeminiAPIKey(userID)
}

// HasGeminiAPIKey checks if a user has a Gemini API key stored.
func (s *UserService) HasGeminiAPIKey(userID int) (bool, error) {
	return s.db.HasGeminiAPIKey(userID)
}

// GetGeminiAPIKeyStatus returns status information about the stored API key.
// Does NOT return the actual key, only metadata.
func (s *UserService) GetGeminiAPIKeyStatus(userID int) (*GeminiAPIKeyStatus, error) {
	hasKey, err := s.db.HasGeminiAPIKey(userID)
	if err != nil {
		return nil, err
	}

	if !hasKey {
		return &GeminiAPIKeyStatus{
			HasKey:    false,
			UpdatedAt: nil,
			MaskedKey: nil,
		}, nil
	}

	updatedAt, err := s.db.GetGeminiAPIKeyUpdatedAt(userID)
	if err != nil {
		return nil, err
	}

	// Get masked key (decrypt and mask)
	decryptedKey, err := getGeminiAPIKeyImpl(s.db, userID)
	if err != nil {
		return nil, err
	}

	masked := maskGeminiAPIKey(decryptedKey)

	return &GeminiAPIKeyStatus{
		HasKey:    true,
		UpdatedAt: updatedAt,
		MaskedKey: &masked,
	}, nil
}

// GeminiAPIKeyStatus represents the status of a user's Gemini API key.
type GeminiAPIKeyStatus struct {
	HasKey    bool    `json:"has_key"`
	UpdatedAt *string `json:"updated_at,omitempty"`
	MaskedKey *string `json:"masked_key,omitempty"` // e.g., "AIzaSy...xxxx"
}

// maskGeminiAPIKey returns a masked version of the API key for display.
// Shows first 7 and last 4 characters.
func maskGeminiAPIKey(apiKey string) string {
	if len(apiKey) <= 11 {
		return "****"
	}
	return apiKey[:7] + "..." + apiKey[len(apiKey)-4:]
}
