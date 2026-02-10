package service

import (
	"errors"
	"strconv"

	"github.com/xela-io/xelanote/internal/db"
)

// Valid setting keys
var validSettings = map[string]bool{
	"registration_enabled":    true,
	"max_notes_per_user":      true,
	"max_storage_mb_per_user": true,
	"maintenance_mode":        true,
	"activity_retention_days": true,
}

// SettingsService handles system settings operations
type SettingsService struct {
	db *db.DB
}

// NewSettingsService creates a new SettingsService
func NewSettingsService(database *db.DB) *SettingsService {
	return &SettingsService{
		db: database,
	}
}

// GetSettings returns all system settings
func (s *SettingsService) GetSettings() (map[string]string, error) {
	return s.db.GetAllSettings()
}

// GetSetting returns a single setting value
func (s *SettingsService) GetSetting(key string) (string, error) {
	return s.db.GetSetting(key)
}

// UpdateSettings updates multiple settings with validation
// Only the provided keys are updated (partial update)
func (s *SettingsService) UpdateSettings(updates map[string]string) ([]string, error) {
	changedKeys := []string{}

	for key, value := range updates {
		// Check if key is valid
		if !validSettings[key] {
			return nil, errors.New("invalid setting key: " + key)
		}

		// Validate value based on key
		if err := s.validateSetting(key, value); err != nil {
			return nil, err
		}

		changedKeys = append(changedKeys, key)
	}

	// Apply updates
	if err := s.db.SetSettings(updates); err != nil {
		return nil, err
	}

	return changedKeys, nil
}

// validateSetting validates a setting value
func (s *SettingsService) validateSetting(key, value string) error {
	switch key {
	case "registration_enabled", "maintenance_mode":
		if value != "true" && value != "false" {
			return errors.New(key + " must be 'true' or 'false'")
		}
	case "max_notes_per_user", "max_storage_mb_per_user", "activity_retention_days":
		num, err := strconv.Atoi(value)
		if err != nil {
			return errors.New(key + " must be a valid integer")
		}
		if num < 0 {
			return errors.New(key + " cannot be negative")
		}
	}
	return nil
}

// IsRegistrationEnabled checks if registration is enabled
func (s *SettingsService) IsRegistrationEnabled() (bool, error) {
	return s.db.IsRegistrationEnabled()
}

// IsMaintenanceMode checks if maintenance mode is enabled
func (s *SettingsService) IsMaintenanceMode() (bool, error) {
	return s.db.IsMaintenanceMode()
}

// GetMaxNotesPerUser returns the max notes limit (0 = unlimited)
func (s *SettingsService) GetMaxNotesPerUser() (int, error) {
	return s.db.GetMaxNotesPerUser()
}

// GetMaxStorageMBPerUser returns the max storage limit in MB (0 = unlimited)
func (s *SettingsService) GetMaxStorageMBPerUser() (int, error) {
	return s.db.GetMaxStorageMBPerUser()
}

// GetActivityRetentionDays returns the activity log retention period (0 = unlimited)
func (s *SettingsService) GetActivityRetentionDays() (int, error) {
	return s.db.GetActivityRetentionDays()
}

// CheckNoteLimitForUser checks if a user has reached their note limit
// Returns (canCreate, currentCount, limit, error)
func (s *SettingsService) CheckNoteLimitForUser(userID int, noteCount int) (bool, int, int, error) {
	limit, err := s.GetMaxNotesPerUser()
	if err != nil {
		return false, 0, 0, err
	}

	// 0 means unlimited
	if limit == 0 {
		return true, noteCount, 0, nil
	}

	return noteCount < limit, noteCount, limit, nil
}
