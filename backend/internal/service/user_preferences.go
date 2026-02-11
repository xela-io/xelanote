package service

import "github.com/xela-io/xelanote/internal/db"

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
