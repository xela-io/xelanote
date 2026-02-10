package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/parser"
	"github.com/xela-io/xelanote/internal/service"
)

// Validation constants for notes
const (
	MaxNoteTitleLength   = 500              // Maximum characters for note title
	MaxNoteContentLength = 10 * 1024 * 1024 // 10MB max content (for notes with embedded images)
	MaxFolderPathLength  = 1000             // Maximum characters for folder path
)

// ClientLink represents a link extracted by the client (for E2E encrypted notes)
type ClientLink struct {
	TargetTitle string `json:"target_title"`
}

// ClientDueDate represents a due date extracted by the client (for E2E encrypted notes)
type ClientDueDate struct {
	DueDate     string `json:"due_date"`
	LineText    string `json:"line_text"`
	LineIndex   int    `json:"line_index"`
	IsTaskItem  bool   `json:"is_task_item"`
	IsCompleted bool   `json:"is_completed"`
}

// NoteRequest represents the request body for creating/updating a note.
type NoteRequest struct {
	Title      string `json:"title"`
	Content    string `json:"content"`
	FolderPath string `json:"folder_path,omitempty"`
	// Encryption fields
	EncryptedTitle     *string  `json:"encrypted_title,omitempty"`
	TitleEncrypted     bool     `json:"title_encrypted"`
	EncryptedContent   string   `json:"encrypted_content,omitempty"`   // Base64
	WrappedDEK         string   `json:"wrapped_dek,omitempty"`         // Base64
	EncryptionMetadata string   `json:"encryption_metadata,omitempty"` // JSON
	Keywords           []string `json:"keywords,omitempty"`
	// Client-side extracted links (for E2E encrypted notes where server can't parse content)
	Links []ClientLink `json:"links,omitempty"`
	// Client-side extracted due dates (for E2E encrypted notes where server can't parse content)
	DueDates []ClientDueDate `json:"due_dates,omitempty"`
	// Journal fields
	NoteType    string  `json:"note_type,omitempty"`    // "note" (default) or "journal"
	JournalDate *string `json:"journal_date,omitempty"` // YYYY-MM-DD for journal notes
}

// NoteListResponse represents a paginated list of notes.
type NoteListResponse struct {
	Notes      []db.Note `json:"notes"`
	NextCursor string    `json:"next_cursor,omitempty"`
}

// validateNoteFields checks common field constraints for note create/update/decrypt.
// folderPath is only validated if non-empty (decrypt doesn't send it).
func validateNoteFields(title, content, folderPath string) error {
	if title == "" {
		return fmt.Errorf("title is required")
	}
	if len(title) > MaxNoteTitleLength {
		return fmt.Errorf("title too long")
	}
	if len(content) > MaxNoteContentLength {
		return fmt.Errorf("content too long")
	}
	if len(folderPath) > MaxFolderPathLength {
		return fmt.Errorf("folder path too long")
	}
	return nil
}

// validateClientLinks validates client-provided links and returns an error response if invalid.
// Returns (linkTitles, true) on success, or (nil, false) if a validation error was sent.
func validateClientLinks(w http.ResponseWriter, links []ClientLink) ([]string, bool) {
	if len(links) > service.MaxLinksPerNote {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("too many links (max %d)", service.MaxLinksPerNote))
		return nil, false
	}
	for i, l := range links {
		if len(l.TargetTitle) > service.MaxLinkTitleLength {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("link %d title too long (max %d chars)", i, service.MaxLinkTitleLength))
			return nil, false
		}
	}
	titles := make([]string, len(links))
	for i, l := range links {
		titles[i] = l.TargetTitle
	}
	return titles, true
}

// convertClientDueDates converts client-provided due dates to the parser format.
func convertClientDueDates(dueDates []ClientDueDate) []parser.DueDate {
	result := make([]parser.DueDate, len(dueDates))
	for i, dd := range dueDates {
		result[i] = parser.DueDate{
			Date:        dd.DueDate,
			LineText:    dd.LineText,
			LineIndex:   dd.LineIndex,
			IsTaskItem:  dd.IsTaskItem,
			IsCompleted: dd.IsCompleted,
		}
	}
	return result
}

// NoteTitleInfo represents minimal note info for link suggestions
type NoteTitleInfo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Encrypted bool   `json:"encrypted"` // true if title is encrypted
}

// generateETag generates a hashed ETag for a note to prevent information disclosure
// Uses SHA256 hash of note.ID + note.Version to obscure version numbers
func generateETag(noteID string, version int) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", noteID, version)))
	etag := hex.EncodeToString(hash[:8]) // First 8 bytes = 16 hex chars
	return fmt.Sprintf(`"%s"`, etag)
}

// parseETag parses an ETag value and attempts to extract the version
// Supports both old (integer) and new (hashed) ETags for backward compatibility
func parseETag(etag string, noteID string, currentVersion int) (int, error) {
	// Strip quotes
	etag = strings.Trim(etag, `"`)

	// Try parsing as old-style integer ETag (backward compatibility)
	if version, err := strconv.Atoi(etag); err == nil {
		return version, nil
	}

	// It's a hashed ETag - verify by re-hashing current version
	expectedETag := strings.Trim(generateETag(noteID, currentVersion), `"`)
	if etag == expectedETag {
		return currentVersion, nil
	}

	return 0, fmt.Errorf("invalid ETag")
}

func (s *Server) resolveETagVersion(w http.ResponseWriter, userID int, noteID, ifMatch string) (int, bool) {
	if ifMatch == "" {
		respondError(w, http.StatusBadRequest, "If-Match header required")
		return 0, false
	}

	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil {
		s.respondInternalErr(w, "failed to get note", err)
		return 0, false
	}
	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return 0, false
	}

	version, err := parseETag(ifMatch, noteID, note.Version)
	if err != nil {
		respondError(w, http.StatusPreconditionFailed, "invalid or outdated ETag")
		return 0, false
	}

	return version, true
}

// handleCreateNoteError handles errors from note creation with appropriate HTTP responses.
func handleCreateNoteError(w http.ResponseWriter, err error, title string, isJournal bool) {
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		// SQLite reports: "UNIQUE constraint failed: notes.user_id, notes.journal_date"
		if isJournal && strings.Contains(err.Error(), "journal_date") {
			respondError(w, http.StatusConflict,
				"Journal für dieses Datum existiert bereits")
			return
		}
		// Fallback: title conflict
		respondError(w, http.StatusConflict,
			fmt.Sprintf("Eine Notiz mit dem Titel '%s' existiert bereits in diesem Ordner", title))
		return
	}
	// Feature not enabled
	if strings.Contains(err.Error(), "journal feature not enabled") {
		respondError(w, http.StatusForbidden, "journal feature not enabled")
		return
	}
	if strings.Contains(err.Error(), "recipe feature not enabled") {
		respondError(w, http.StatusForbidden, "recipe feature not enabled")
		return
	}
	respondError(w, http.StatusInternalServerError, "failed to create note")
}
