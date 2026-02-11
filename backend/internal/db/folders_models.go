package db

import "time"

// Folder represents a folder in the hierarchy.
type Folder struct {
	ID                int       `json:"id"`
	Path              string    `json:"path"`
	ParentID          *int      `json:"parent_id"` // Removed omitempty - need to serialize null for virtual root (Migration 025)
	Name              string    `json:"name"`
	NoteCount         int       `json:"note_count"`
	DisplayOrder      int       `json:"display_order"`
	Color             *string   `json:"color,omitempty"`
	AIEnabledDefault  bool      `json:"ai_enabled_default"` // Default for new notes in this folder
	EncryptionDefault bool      `json:"encryption_default"` // Default encryption for new notes (true=encrypted)
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
