package service

import (
	"encoding/json"

	"github.com/xela-io/xelanote/internal/db"
)

// GetUserFeature retrieves a specific feature configuration for a user.
func (s *NoteService) GetUserFeature(userID int, feature string) (*db.Feature, error) {
	return s.db.GetUserFeature(userID, feature)
}

// ListUserFeatures returns all feature configurations for a user.
func (s *NoteService) ListUserFeatures(userID int) ([]db.Feature, error) {
	return s.db.ListUserFeatures(userID)
}

// SetUserFeature enables or disables a feature for a user.
func (s *NoteService) SetUserFeature(userID int, feature string, enabled bool, settings json.RawMessage) error {
	return s.db.SetUserFeature(userID, feature, enabled, settings)
}
