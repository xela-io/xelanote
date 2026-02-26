package service

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// maxCollapseEntries is the maximum number of keys allowed in a collapse_state map.
const maxCollapseEntries = 50

// groupHashPattern matches either:
// - plain base36 hash strings (preview details groups), or
// - namespaced live-preview task keys ("tasks:<base36>")
var groupHashPattern = regexp.MustCompile(`^(?:[0-9a-z]{1,7}|tasks:[0-9a-z]{1,7})$`)

// GetNoteUserState returns the state_data JSON for a note+user pair.
// Access is granted to note owners and users with share permission.
func (s *NoteService) GetNoteUserState(userID int, noteID string) (*string, error) {
	if err := s.checkNoteAccess(userID, noteID); err != nil {
		return nil, err
	}
	return s.db.GetNoteUserState(userID, noteID)
}

// UpdateNoteUserCollapseState validates and persists collapse_state for a note.
// raw is the JSON-encoded collapse_state value (map[string]bool or null).
func (s *NoteService) UpdateNoteUserCollapseState(userID int, noteID string, raw *json.RawMessage) error {
	if err := s.checkNoteAccess(userID, noteID); err != nil {
		return err
	}

	// Build the state_data envelope
	var stateData string

	if raw == nil || string(*raw) == "null" {
		stateData = `{"collapse_state":null}`
	} else {
		// Validate: must be map[string]bool
		var m map[string]bool
		if err := json.Unmarshal(*raw, &m); err != nil {
			return &ValidationError{Message: "collapse_state must be a map of string keys to boolean values"}
		}

		if len(m) > maxCollapseEntries {
			return &ValidationError{Message: fmt.Sprintf("collapse_state exceeds maximum of %d entries", maxCollapseEntries)}
		}

		for key := range m {
			if !groupHashPattern.MatchString(key) {
				return &ValidationError{Message: "collapse_state keys must be base36 strings (1-7 chars) or tasks:<base36>"}
			}
		}

		envelope := map[string]interface{}{"collapse_state": m}
		data, err := json.Marshal(envelope)
		if err != nil {
			return fmt.Errorf("failed to marshal state_data: %w", err)
		}
		stateData = string(data)
	}

	return s.db.UpsertNoteUserState(userID, noteID, stateData)
}

// checkNoteAccess verifies the user owns or has share access to the note.
func (s *NoteService) checkNoteAccess(userID int, noteID string) error {
	note, err := s.db.GetNote(userID, noteID)
	if err != nil {
		return fmt.Errorf("failed to check note access: %w", err)
	}
	if note == nil {
		// Not owner — check share permission
		role, permErr := s.db.GetSharePermission(userID, noteID)
		if permErr != nil {
			return fmt.Errorf("failed to check share permission: %w", permErr)
		}
		if role == "" {
			return ErrNotFound
		}
	}
	return nil
}
