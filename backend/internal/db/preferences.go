package db

import (
	"database/sql"
	"time"
)

// UserPreferences represents user settings stored in the database
type UserPreferences struct {
	ID              int     `json:"id"`
	UserID          int     `json:"user_id"`
	Theme           string  `json:"theme"`
	EditorMode      string  `json:"editor_mode"`
	KeywordsEnabled bool    `json:"keywords_enabled"`
	EncryptTitles   bool    `json:"encrypt_titles"`
	SecurityLevel   string  `json:"security_level"`    // NEW: paranoid | balanced | convenient
	AutoLockTimeout int     `json:"auto_lock_timeout"` // NEW: minutes (0 = never)
	RecoveryKeyHash *string `json:"-"`                 // Not exposed in JSON for security
	RecoveryKeySalt []byte  `json:"-"`                 // Not exposed in JSON for security
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// GetUserPreferences retrieves preferences for a user
// Returns nil if no preferences exist (user should call GetOrCreateUserPreferences)
func (db *DB) GetUserPreferences(userID int) (*UserPreferences, error) {
	var prefs UserPreferences
	var keywordsEnabled, encryptTitles int
	err := db.QueryRow(`
		SELECT id, user_id, theme, editor_mode,
		       COALESCE(keywords_enabled, 0),
		       COALESCE(encrypt_titles, 0),
		       COALESCE(security_level, 'balanced'),
		       COALESCE(auto_lock_timeout, 15),
		       recovery_key_hash, recovery_key_salt,
		       created_at, updated_at
		FROM user_preferences
		WHERE user_id = ?
	`, userID).Scan(
		&prefs.ID, &prefs.UserID, &prefs.Theme, &prefs.EditorMode,
		&keywordsEnabled, &encryptTitles,
		&prefs.SecurityLevel, &prefs.AutoLockTimeout,
		&prefs.RecoveryKeyHash, &prefs.RecoveryKeySalt,
		&prefs.CreatedAt, &prefs.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	prefs.KeywordsEnabled = keywordsEnabled == 1
	prefs.EncryptTitles = encryptTitles == 1

	return &prefs, nil
}

// GetOrCreateUserPreferences retrieves preferences for a user, creating defaults if not exist
// Returns preferences and a boolean indicating if the row was newly created
func (db *DB) GetOrCreateUserPreferences(userID int) (*UserPreferences, bool, error) {
	prefs, err := db.GetUserPreferences(userID)
	if err == nil {
		return prefs, false, nil
	}
	if err != ErrNotFound {
		return nil, false, err
	}

	// Create default preferences
	now := time.Now().Format(time.RFC3339)
	result, err := db.Exec(`
		INSERT INTO user_preferences (user_id, theme, editor_mode, keywords_enabled, encrypt_titles, created_at, updated_at)
		VALUES (?, 'default-dark', 'split', 0, 0, ?, ?)
	`, userID, now, now)
	if err != nil {
		return nil, false, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, false, err
	}
	return &UserPreferences{
		ID:              int(id),
		UserID:          userID,
		Theme:           "default-dark",
		EditorMode:      "split",
		KeywordsEnabled: false,
		EncryptTitles:   false,
		SecurityLevel:   "balanced",
		AutoLockTimeout: 15,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, true, nil
}

// UpsertUserPreferences updates or creates user preferences
func (db *DB) UpsertUserPreferences(userID int, theme, editorMode string) (*UserPreferences, error) {
	now := time.Now().Format(time.RFC3339)

	// Try to update first
	result, err := db.Exec(`
		UPDATE user_preferences
		SET theme = ?, editor_mode = ?, updated_at = ?
		WHERE user_id = ?
	`, theme, editorMode, now, userID)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		// Insert new row
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, keywords_enabled, encrypt_titles, created_at, updated_at)
			VALUES (?, ?, ?, 0, 0, ?, ?)
		`, userID, theme, editorMode, now, now)
		if err != nil {
			return nil, err
		}
	}

	// Fetch and return the updated preferences
	return db.GetUserPreferences(userID)
}

// UpdateEncryptionPreferences updates encryption-related preferences
func (db *DB) UpdateEncryptionPreferences(userID int, keywordsEnabled, encryptTitles bool) error {
	now := time.Now().Format(time.RFC3339)

	keywordsInt := 0
	if keywordsEnabled {
		keywordsInt = 1
	}

	encryptTitlesInt := 0
	if encryptTitles {
		encryptTitlesInt = 1
	}

	result, err := db.Exec(`
		UPDATE user_preferences
		SET keywords_enabled = ?, encrypt_titles = ?, updated_at = ?
		WHERE user_id = ?
	`, keywordsInt, encryptTitlesInt, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		// User preferences don't exist, create them
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, keywords_enabled, encrypt_titles, created_at, updated_at)
			VALUES (?, 'default-dark', 'split', ?, ?, ?, ?)
		`, userID, keywordsInt, encryptTitlesInt, now, now)
		return err
	}

	return nil
}

// SetRecoveryKey stores the hashed recovery key and salt for a user
func (db *DB) SetRecoveryKey(userID int, recoveryKeyHash string, recoveryKeySalt []byte) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE user_preferences
		SET recovery_key_hash = ?, recovery_key_salt = ?, updated_at = ?
		WHERE user_id = ?
	`, recoveryKeyHash, recoveryKeySalt, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		// User preferences don't exist, create them with recovery key
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, recovery_key_hash, recovery_key_salt, created_at, updated_at)
			VALUES (?, 'default-dark', 'split', ?, ?, ?, ?)
		`, userID, recoveryKeyHash, recoveryKeySalt, now, now)
		return err
	}

	return nil
}

// GetRecoveryKeySalt retrieves the recovery key salt for a user
func (db *DB) GetRecoveryKeySalt(userID int) ([]byte, error) {
	var salt []byte
	err := db.QueryRow(`
		SELECT recovery_key_salt
		FROM user_preferences
		WHERE user_id = ? AND recovery_key_salt IS NOT NULL
	`, userID).Scan(&salt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return salt, nil
}

// UpdateSecurityPreferences updates security-related preferences (security_level and auto_lock_timeout)
func (db *DB) UpdateSecurityPreferences(userID int, securityLevel string, autoLockTimeout int) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE user_preferences
		SET security_level = ?, auto_lock_timeout = ?, updated_at = ?
		WHERE user_id = ?
	`, securityLevel, autoLockTimeout, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		// User preferences don't exist, create them
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, security_level, auto_lock_timeout, created_at, updated_at)
			VALUES (?, 'default-dark', 'split', ?, ?, ?, ?)
		`, userID, securityLevel, autoLockTimeout, now, now)
		return err
	}

	return nil
}

// WebAuthnCredential represents a WebAuthn credential (imported from service package)
type WebAuthnCredential struct {
	ID           int64
	UserID       int64
	CredentialID string
	DeviceName   string
	CreatedAt    string
	LastUsedAt   *string
}

// AddWebAuthnCredential adds a new WebAuthn credential for a user
func (db *DB) AddWebAuthnCredential(userID int64, credentialID, deviceName string) (*WebAuthnCredential, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	result, err := db.Exec(`
		INSERT INTO webauthn_credentials (user_id, credential_id, device_name, created_at)
		VALUES (?, ?, ?, ?)
	`, userID, credentialID, deviceName, now)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &WebAuthnCredential{
		ID:           id,
		UserID:       userID,
		CredentialID: credentialID,
		DeviceName:   deviceName,
		CreatedAt:    now,
		LastUsedAt:   nil,
	}, nil
}

// GetWebAuthnCredentials retrieves all WebAuthn credentials for a user
func (db *DB) GetWebAuthnCredentials(userID int64) ([]WebAuthnCredential, error) {
	rows, err := db.Query(`
		SELECT id, user_id, credential_id, device_name, created_at, last_used_at
		FROM webauthn_credentials
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credentials []WebAuthnCredential
	for rows.Next() {
		var c WebAuthnCredential
		var lastUsed sql.NullString

		err := rows.Scan(&c.ID, &c.UserID, &c.CredentialID, &c.DeviceName, &c.CreatedAt, &lastUsed)
		if err != nil {
			return nil, err
		}

		if lastUsed.Valid {
			c.LastUsedAt = &lastUsed.String
		}

		credentials = append(credentials, c)
	}

	return credentials, nil
}

// DeleteWebAuthnCredential deletes a WebAuthn credential
func (db *DB) DeleteWebAuthnCredential(userID int64, credentialID string) error {
	_, err := db.Exec(`
		DELETE FROM webauthn_credentials
		WHERE user_id = ? AND credential_id = ?
	`, userID, credentialID)
	return err
}

// TouchWebAuthnCredential updates the last_used_at timestamp for a credential
func (db *DB) TouchWebAuthnCredential(userID int64, credentialID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		UPDATE webauthn_credentials
		SET last_used_at = ?
		WHERE user_id = ? AND credential_id = ?
	`, now, userID, credentialID)
	return err
}

// InvalidateRecoveryKey clears the recovery key hash and salt for a user.
// This should be called when the user changes their password, as the old recovery key
// is no longer valid (it was derived from the old password).
func (db *DB) InvalidateRecoveryKey(userID int) error {
	now := time.Now().Format(time.RFC3339)

	_, err := db.Exec(`
		UPDATE user_preferences
		SET recovery_key_hash = NULL, recovery_key_salt = NULL, updated_at = ?
		WHERE user_id = ?
	`, now, userID)
	return err
}

// --- Claude API Key Management (BYOK) ---

// SetClaudeAPIKey stores the encrypted Claude API key for a user.
// The key should be encrypted with AES-256-GCM before calling this function.
func (db *DB) SetClaudeAPIKey(userID int, encryptedKey string) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE user_preferences
		SET encrypted_claude_api_key = ?, claude_api_key_updated_at = ?, updated_at = ?
		WHERE user_id = ?
	`, encryptedKey, now, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		// User preferences don't exist, create them with the API key
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, encrypted_claude_api_key, claude_api_key_updated_at, created_at, updated_at)
			VALUES (?, 'default-dark', 'split', ?, ?, ?, ?)
		`, userID, encryptedKey, now, now, now)
		return err
	}

	return nil
}

// GetClaudeAPIKey retrieves the encrypted Claude API key for a user.
// Returns ErrNotFound if no key is stored.
func (db *DB) GetClaudeAPIKey(userID int) (string, error) {
	var encryptedKey sql.NullString
	err := db.QueryRow(`
		SELECT encrypted_claude_api_key
		FROM user_preferences
		WHERE user_id = ?
	`, userID).Scan(&encryptedKey)

	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if !encryptedKey.Valid || encryptedKey.String == "" {
		return "", ErrNotFound
	}

	return encryptedKey.String, nil
}

// DeleteClaudeAPIKey removes the Claude API key for a user.
func (db *DB) DeleteClaudeAPIKey(userID int) error {
	now := time.Now().Format(time.RFC3339)

	_, err := db.Exec(`
		UPDATE user_preferences
		SET encrypted_claude_api_key = NULL, claude_api_key_updated_at = NULL, updated_at = ?
		WHERE user_id = ?
	`, now, userID)
	return err
}

// HasClaudeAPIKey checks if a user has a Claude API key stored.
func (db *DB) HasClaudeAPIKey(userID int) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM user_preferences
		WHERE user_id = ? AND encrypted_claude_api_key IS NOT NULL AND encrypted_claude_api_key != ''
	`, userID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetClaudeAPIKeyUpdatedAt retrieves the timestamp when the Claude API key was last updated.
// Returns nil if no key is stored.
func (db *DB) GetClaudeAPIKeyUpdatedAt(userID int) (*string, error) {
	var updatedAt sql.NullString
	err := db.QueryRow(`
		SELECT claude_api_key_updated_at
		FROM user_preferences
		WHERE user_id = ?
	`, userID).Scan(&updatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !updatedAt.Valid {
		return nil, nil
	}

	return &updatedAt.String, nil
}

// --- Gemini API Key Management (BYOK) ---

// SetGeminiAPIKey stores the encrypted Gemini API key for a user.
// The key should be encrypted with AES-256-GCM before calling this function.
func (db *DB) SetGeminiAPIKey(userID int, encryptedKey string) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE user_preferences
		SET encrypted_gemini_api_key = ?, gemini_api_key_updated_at = ?, updated_at = ?
		WHERE user_id = ?
	`, encryptedKey, now, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		// User preferences don't exist, create them with the API key
		_, err = db.Exec(`
			INSERT INTO user_preferences (user_id, theme, editor_mode, encrypted_gemini_api_key, gemini_api_key_updated_at, created_at, updated_at)
			VALUES (?, 'default-dark', 'split', ?, ?, ?, ?)
		`, userID, encryptedKey, now, now, now)
		return err
	}

	return nil
}

// GetGeminiAPIKey retrieves the encrypted Gemini API key for a user.
// Returns ErrNotFound if no key is stored.
func (db *DB) GetGeminiAPIKey(userID int) (string, error) {
	var encryptedKey sql.NullString
	err := db.QueryRow(`
		SELECT encrypted_gemini_api_key
		FROM user_preferences
		WHERE user_id = ?
	`, userID).Scan(&encryptedKey)

	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if !encryptedKey.Valid || encryptedKey.String == "" {
		return "", ErrNotFound
	}

	return encryptedKey.String, nil
}

// DeleteGeminiAPIKey removes the Gemini API key for a user.
func (db *DB) DeleteGeminiAPIKey(userID int) error {
	now := time.Now().Format(time.RFC3339)

	_, err := db.Exec(`
		UPDATE user_preferences
		SET encrypted_gemini_api_key = NULL, gemini_api_key_updated_at = NULL, updated_at = ?
		WHERE user_id = ?
	`, now, userID)
	return err
}

// HasGeminiAPIKey checks if a user has a Gemini API key stored.
func (db *DB) HasGeminiAPIKey(userID int) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM user_preferences
		WHERE user_id = ? AND encrypted_gemini_api_key IS NOT NULL AND encrypted_gemini_api_key != ''
	`, userID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetGeminiAPIKeyUpdatedAt retrieves the timestamp when the Gemini API key was last updated.
// Returns nil if no key is stored.
func (db *DB) GetGeminiAPIKeyUpdatedAt(userID int) (*string, error) {
	var updatedAt sql.NullString
	err := db.QueryRow(`
		SELECT gemini_api_key_updated_at
		FROM user_preferences
		WHERE user_id = ?
	`, userID).Scan(&updatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !updatedAt.Valid {
		return nil, nil
	}

	return &updatedAt.String, nil
}
