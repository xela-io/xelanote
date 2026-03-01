package db

import "time"

// Note represents a note in the database.
type Note struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	FolderPath   string     `json:"folder_path"`
	Version      int        `json:"version"`
	DisplayOrder int        `json:"display_order"`
	Color        *string    `json:"color,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	// Encryption fields
	EncryptedContent   []byte  `json:"encrypted_content,omitempty"`
	ContentEncrypted   bool    `json:"content_encrypted"`
	EncryptedTitle     *string `json:"encrypted_title,omitempty"`
	TitleEncrypted     bool    `json:"title_encrypted"`
	WrappedDEK         string  `json:"wrapped_dek,omitempty"`
	WrappedDEKRecovery string  `json:"wrapped_dek_recovery,omitempty"`
	EncryptionVersion  int     `json:"encryption_version"`
	EncryptionMetadata  string  `json:"encryption_metadata,omitempty"`
	EncryptedFolderPath *string `json:"encrypted_folder_path,omitempty"`
	// Summary fields (LLM-generated)
	Summary            *string    `json:"summary,omitempty"`
	EncryptedSummary   *string    `json:"encrypted_summary,omitempty"`
	SummaryEncrypted   bool       `json:"summary_encrypted"`
	ContentHash        *string    `json:"content_hash,omitempty"`
	SummaryGeneratedAt *time.Time `json:"summary_generated_at,omitempty"`
	// Journal fields
	NoteType    string  `json:"note_type,omitempty"`    // "note" or "journal"
	JournalDate *string `json:"journal_date,omitempty"` // YYYY-MM-DD for journal notes
	// AI/Claude API fields
	AIEnabled bool `json:"ai_enabled"` // true = Cloud-KI (Claude) erlaubt
	UserID    int  `json:"-"`          // Not exported to JSON, used internally
	// Delta-sync fields
	IsDeleted bool `json:"is_deleted,omitempty"` // true for soft-deleted notes in delta-sync responses
	// Sharing fields (populated by ListNotesByFolder UNION)
	IsShared  bool   `json:"is_shared,omitempty"`  // true if this is a placed shared note
	ShareRole string `json:"share_role,omitempty"` // viewer|editor for shared recipient views
}

// NoteWithBacklinks extends Note with backlink information.
type NoteWithBacklinks struct {
	Note
	BacklinkCount int `json:"backlink_count"`
}

// FolderInfo contains folder path and note count.
type FolderInfo struct {
	Path      string `json:"path"`
	NoteCount int    `json:"note_count"`
}

// NoteNeedingSummary represents a note that needs summary generation.
type NoteNeedingSummary struct {
	ID          string
	UserID      int
	Content     string
	ContentHash *string
	SummaryHash *string // Hash when summary was last generated
}
