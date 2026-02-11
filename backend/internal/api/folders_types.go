package api

// Folder API Request/Response types

type CreateFolderRequest struct {
	Path string `json:"path"`
}

type MoveFolderRequest struct {
	NewParentPath string `json:"new_parent_path"`
}

type RenameFolderRequest struct {
	NewName string `json:"new_name"`
}

type ReorderFoldersRequest struct {
	ParentID *int  `json:"parent_id"`
	Items    []int `json:"items"`
}

type UpdateColorRequest struct {
	Color *string `json:"color"`
}

// UpdateFolderAIEnabledRequest represents the request body for toggling ai_enabled_default.
type UpdateFolderAIEnabledRequest struct {
	AIEnabled bool `json:"ai_enabled"`
}

// UpdateFolderEncryptionDefaultRequest represents the request body for toggling encryption_default.
type UpdateFolderEncryptionDefaultRequest struct {
	Encrypted bool `json:"encrypted"`
}
