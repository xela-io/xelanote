package service

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/xela-io/xelanote/internal/db"
)

var noteIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// GetOrCreatePreferences retrieves user preferences, creating defaults if needed
// Returns preferences and a boolean indicating if the row was newly created
func (s *UserService) GetOrCreatePreferences(userID int) (*db.UserPreferences, bool, error) {
	return s.db.GetOrCreateUserPreferences(userID)
}

// UpdateHomeDashboardLayoutPreference validates and stores the home dashboard layout JSON.
// raw == nil or "null" clears the stored layout.
func (s *UserService) UpdateHomeDashboardLayoutPreference(userID int, raw json.RawMessage) (*db.UserPreferences, error) {
	if len(raw) == 0 || string(raw) == "null" {
		if err := s.db.UpdateHomeDashboardLayout(userID, nil); err != nil {
			return nil, err
		}
		return s.db.GetUserPreferences(userID)
	}

	var layout HomeDashboardLayoutPreferences
	if err := json.Unmarshal(raw, &layout); err != nil {
		return nil, ErrInvalidHomeDashboardLayout
	}

	if layout.Version != 1 {
		return nil, ErrInvalidHomeDashboardLayout
	}
	if len(layout.RightSectionOrder) != 4 {
		return nil, ErrInvalidHomeDashboardLayout
	}
	allowed := map[string]bool{
		"recent":   true,
		"activity": true,
		"created":  true,
		"all":      true,
	}
	seen := map[string]bool{}
	for _, id := range layout.RightSectionOrder {
		if !allowed[id] || seen[id] {
			return nil, ErrInvalidHomeDashboardLayout
		}
		seen[id] = true
	}

	normalized, err := json.Marshal(layout)
	if err != nil {
		return nil, err
	}
	jsonStr := string(normalized)
	if err := s.db.UpdateHomeDashboardLayout(userID, &jsonStr); err != nil {
		return nil, err
	}
	return s.db.GetUserPreferences(userID)
}

// ValidateOpenTabs validates the open tabs JSON payload.
func ValidateOpenTabs(raw json.RawMessage) error {
	var payload OpenTabsPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ErrInvalidOpenTabs
	}

	if payload.Version != 1 {
		return ErrInvalidOpenTabs
	}

	if len(payload.Tabs) > 50 {
		return ErrInvalidOpenTabs
	}

	seen := map[string]bool{}
	for _, tab := range payload.Tabs {
		if !noteIDPattern.MatchString(tab.NoteID) {
			return ErrInvalidOpenTabs
		}
		if seen[tab.NoteID] {
			return ErrInvalidOpenTabs
		}
		seen[tab.NoteID] = true
	}

	if payload.ActiveNoteID != nil {
		if !seen[*payload.ActiveNoteID] {
			return ErrInvalidOpenTabs
		}
	}

	return nil
}

// UpdateOpenTabsPreference validates and stores the open tabs JSON.
// raw == nil or "null" clears the stored tabs.
func (s *UserService) UpdateOpenTabsPreference(userID int, raw json.RawMessage) (*db.UserPreferences, error) {
	if len(raw) == 0 || string(raw) == "null" {
		if err := s.db.UpdateOpenTabs(userID, nil); err != nil {
			return nil, err
		}
		return s.db.GetUserPreferences(userID)
	}

	if err := ValidateOpenTabs(raw); err != nil {
		return nil, err
	}

	// Re-marshal to normalize
	var payload OpenTabsPayload
	_ = json.Unmarshal(raw, &payload)
	normalized, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	jsonStr := string(normalized)
	if err := s.db.UpdateOpenTabs(userID, &jsonStr); err != nil {
		return nil, err
	}
	return s.db.GetUserPreferences(userID)
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

// GetActiveAIProvider returns the selected AI provider for a user.
func (s *UserService) GetActiveAIProvider(userID int) (string, error) {
	return s.db.GetActiveAIProvider(userID)
}

// SetActiveAIProvider sets the selected AI provider for a user.
func (s *UserService) SetActiveAIProvider(userID int, provider string) error {
	if !isValidAIProvider(provider) {
		return ErrInvalidAIProvider
	}
	return s.db.SetActiveAIProvider(userID, provider)
}

// GetDietaryPreference returns the dietary preference for a user.
func (s *UserService) GetDietaryPreference(userID int) (string, error) {
	return s.db.GetDietaryPreference(userID)
}

// SetDietaryPreference sets the dietary preference for a user.
func (s *UserService) SetDietaryPreference(userID int, pref string) error {
	pref = strings.TrimSpace(strings.ToLower(pref))
	if !isValidDietaryPreference(pref) {
		return ErrInvalidDietaryPreference
	}
	return s.db.SetDietaryPreference(userID, pref)
}

// GetAIModelPreferences returns saved model overrides for all providers.
func (s *UserService) GetAIModelPreferences(userID int) (*db.AIModelPreferences, error) {
	return s.db.GetAIModelPreferences(userID)
}

// SetAIModelPreferences saves model overrides for all providers.
// Empty values are valid and reset to provider defaults.
func (s *UserService) SetAIModelPreferences(userID int, models *db.AIModelPreferences) error {
	models.ClaudeModel = strings.TrimSpace(models.ClaudeModel)
	models.GeminiModel = strings.TrimSpace(models.GeminiModel)
	models.ChatGPTModel = strings.TrimSpace(models.ChatGPTModel)

	if !isValidModelString(models.ClaudeModel) || !isValidModelString(models.GeminiModel) || !isValidModelString(models.ChatGPTModel) {
		return ErrInvalidAIModel
	}
	return s.db.SetAIModelPreferences(userID, models)
}

func isValidModelString(model string) bool {
	if model == "" {
		return true
	}
	if len(model) > 100 {
		return false
	}
	for _, r := range model {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		switch r {
		case '-', '_', '.', ':':
			continue
		default:
			return false
		}
	}
	return true
}
