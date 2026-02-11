package api

// ShareNoteRequest represents the request to share a note.
type ShareNoteRequest struct {
	Identifier string `json:"identifier"` // Username or email
	Role       string `json:"role"`       // "viewer" or "editor"
}

// UpdateShareRoleRequest represents the request to update a share role.
type UpdateShareRoleRequest struct {
	Role string `json:"role"` // "viewer" or "editor"
}

// SharedNoteUpdateRequest represents the request to update a shared note.
type SharedNoteUpdateRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Version int    `json:"version"`
}

// ShareFolderRequest represents the request to share a folder.
type ShareFolderRequest struct {
	Identifier string `json:"identifier"` // Username or email
	Role       string `json:"role"`       // "viewer" or "editor"
}

// PlacementRequest represents the request to place a shared note in a folder.
type PlacementRequest struct {
	FolderID int `json:"folder_id"`
}
