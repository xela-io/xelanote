package db

import "time"

// NoteShare represents a sharing record for a note.
type NoteShare struct {
	ID                 int       `json:"id"`
	NoteID             string    `json:"note_id"`
	OwnerUserID        int       `json:"owner_user_id"`
	OwnerUsername      string    `json:"owner_username"`
	SharedWithUserID   int       `json:"shared_with_user_id"`
	SharedWithUsername string    `json:"shared_with_username"`
	Role               string    `json:"role"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// SharedNote represents a note shared with the current user.
// Flat structure (no embedding of Note) to avoid JSON field conflicts.
type SharedNote struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	FolderPath string    `json:"folder_path"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	// note_type: omitempty for backwards compatibility with old clients
	NoteType string `json:"note_type,omitempty"`
	// Sharing-specific fields
	SharedBy  string `json:"shared_by"`
	ShareRole string `json:"share_role"`
	ShareID   int    `json:"share_id"`
}

// FolderShare represents a sharing record for a folder.
type FolderShare struct {
	ID                 int       `json:"id"`
	FolderID           int       `json:"folder_id"`
	FolderPath         string    `json:"folder_path"`
	FolderName         string    `json:"folder_name"`
	OwnerUserID        int       `json:"owner_user_id"`
	OwnerUsername      string    `json:"owner_username"`
	SharedWithUserID   int       `json:"shared_with_user_id"`
	SharedWithUsername string    `json:"shared_with_username"`
	Role               string    `json:"role"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// SharedFolder represents a folder shared with the current user.
type SharedFolder struct {
	ID        int       `json:"id"`
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	NoteCount int       `json:"note_count"`
	SharedBy  string    `json:"shared_by"`
	ShareRole string    `json:"share_role"`
	ShareID   int       `json:"share_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserSearchResult represents a user found via search (for the share dialog).
type UserSearchResult struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}
