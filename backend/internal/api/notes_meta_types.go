package api

import "encoding/json"

// RenameRequest represents the request body for renaming a note.
type RenameRequest struct {
	NewTitle string `json:"newTitle"`
}

// UpdateNoteColorRequest represents the request body for updating a note's color.
type UpdateNoteColorRequest struct {
	Color *string `json:"color"`
}

// UpdateNoteUserStateRequest represents the request body for updating per-user note state.
type UpdateNoteUserStateRequest struct {
	CollapseState json.RawMessage `json:"collapse_state"`
}
