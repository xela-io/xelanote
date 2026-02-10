package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/jobs"
	"github.com/xela-io/xelanote/internal/llm"
	"github.com/xela-io/xelanote/internal/parser"
	"github.com/xela-io/xelanote/internal/service"
	"github.com/xela-io/xelanote/internal/websocket"
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

// ensureNotes returns an empty slice if notes is nil, otherwise returns notes unchanged.
// This ensures JSON serialization produces [] instead of null.
func ensureNotes(notes []db.Note) []db.Note {
	if notes == nil {
		return []db.Note{}
	}
	return notes
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

func (s *Server) listNotes(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (set by authMiddleware)
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = min(parsed, 500) // Cap at 500 to prevent memory exhaustion
		}
	}
	cursor := r.URL.Query().Get("cursor")
	folderPath := r.URL.Query().Get("folder")

	var notes []db.Note
	var nextCursor string
	var err error

	if folderPath != "" {
		notes, err = s.noteService.GetNotesByFolder(userID, folderPath)
	} else {
		notes, nextCursor, err = s.noteService.ListNotes(userID, limit, cursor)
	}

	if err != nil {
		s.respondInternalErr(w, "failed to list notes", err)
		return
	}

	respondJSON(w, http.StatusOK, NoteListResponse{
		Notes:      ensureNotes(notes),
		NextCursor: nextCursor,
	})
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

	return 0, fmt.Errorf("ETag mismatch")
}

// resolveETagVersion parses the If-Match ETag and returns the note version.
// For old-style integer ETags it returns the parsed int directly.
// For hashed ETags it fetches the note to verify the hash and returns the current version.
func (s *Server) resolveETagVersion(w http.ResponseWriter, userID int, noteID, ifMatch string) (int, bool) {
	version, err := parseETag(ifMatch, noteID, 0)
	if err == nil {
		// Old-style integer ETag parsed successfully
		return version, true
	}

	// Hashed ETag - fetch note to verify
	existingNote, fetchErr := s.noteService.GetNote(userID, noteID)
	if fetchErr != nil {
		if errors.Is(fetchErr, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found")
			return 0, false
		}
		respondError(w, http.StatusInternalServerError, "failed to fetch note")
		return 0, false
	}

	version, err = parseETag(ifMatch, existingNote.ID, existingNote.Version)
	if err != nil {
		respondError(w, http.StatusPreconditionFailed, "version mismatch - note was modified")
		return 0, false
	}
	return version, true
}

func (s *Server) createNote(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req NoteRequest
	if err := decodeJSONWithLimit(w, r, &req, MaxLargeJSONBodySize); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateNoteFields(req.Title, req.Content, req.FolderPath); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.FolderPath == "" {
		req.FolderPath = "/"
	}

	// Recipe validation
	if req.NoteType == "recipe" {
		if req.JournalDate != nil && *req.JournalDate != "" {
			respondError(w, http.StatusBadRequest,
				"journal_date cannot be set for recipe notes")
			return
		}
		if req.FolderPath == "" {
			req.FolderPath = "/Rezepte"
		}
	}

	// Journal validation:
	// | note_type      | journal_date   | Result              |
	// |----------------|----------------|---------------------|
	// | "journal"      | nil or ""      | 400 Error           |
	// | "journal"      | "2024-01-15"   | OK (validate format)|
	// | "" or "note"   | nil or ""      | OK (normal note)    |
	// | "" or "note"   | "2024-01-15"   | 400 Error           |
	if req.NoteType == "journal" {
		if req.JournalDate == nil || *req.JournalDate == "" {
			respondError(w, http.StatusBadRequest,
				"journal_date is required when note_type is 'journal'")
			return
		}
		if err := service.ValidateJournalDate(*req.JournalDate); err != nil {
			respondError(w, http.StatusBadRequest,
				"invalid journal_date format, expected YYYY-MM-DD")
			return
		}
	} else if req.JournalDate != nil && *req.JournalDate != "" {
		respondError(w, http.StatusBadRequest,
			"journal_date can only be set when note_type is 'journal'")
		return
	}

	var note *db.Note
	var err error

	// Check if this is an encrypted note
	if req.EncryptedContent != "" && req.WrappedDEK != "" {
		// Validate all encryption fields
		if err := ValidateEncryptedNoteRequest(req.EncryptedContent, req.WrappedDEK, req.EncryptionMetadata, req.EncryptedTitle); err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("encryption validation failed: %v", err))
			return
		}

		// Decode Base64 encrypted content (already validated)
		encryptedBlob, err := base64.StdEncoding.DecodeString(req.EncryptedContent)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid encrypted content")
			return
		}

		// Create encrypted note (journal, recipe, or regular)
		if req.NoteType == "journal" {
			note, err = s.noteService.CreateEncryptedJournalNote(
				userID,
				req.Title,
				req.EncryptedTitle,
				req.TitleEncrypted,
				encryptedBlob,
				req.WrappedDEK,
				req.EncryptionMetadata,
				req.Keywords,
				req.FolderPath,
				*req.JournalDate,
			)
		} else if req.NoteType == "recipe" {
			if s.recipeService == nil {
				respondError(w, http.StatusInternalServerError, "recipe service not available")
				return
			}
			note, err = s.recipeService.CreateEncryptedRecipeNote(
				userID,
				req.Title,
				req.EncryptedTitle,
				req.TitleEncrypted,
				encryptedBlob,
				req.WrappedDEK,
				req.EncryptionMetadata,
				req.Keywords,
				req.FolderPath,
			)
		} else {
			note, err = s.noteService.CreateEncryptedNote(
				userID,
				req.Title,
				req.EncryptedTitle,
				req.TitleEncrypted,
				encryptedBlob,
				req.WrappedDEK,
				req.EncryptionMetadata,
				req.Keywords,
				req.FolderPath,
			)
		}
		if err != nil {
			handleCreateNoteError(w, err, req.Title, req.NoteType == "journal")
			return
		}
	} else {
		// Create plaintext note (legacy support)
		if req.NoteType == "journal" {
			note, err = s.noteService.CreateJournalNote(userID, req.Title, req.Content, req.FolderPath, *req.JournalDate)
		} else if req.NoteType == "recipe" {
			if s.recipeService == nil {
				respondError(w, http.StatusInternalServerError, "recipe service not available")
				return
			}
			note, err = s.recipeService.CreateRecipeNote(userID, req.Title, req.Content, req.FolderPath)
		} else {
			note, err = s.noteService.CreateNote(userID, req.Title, req.Content, req.FolderPath)
		}
		if err != nil {
			handleCreateNoteError(w, err, req.Title, req.NoteType == "journal")
			return
		}
	}

	// Process client-provided links for encrypted notes
	if len(req.Links) > 0 {
		linkTitles, ok := validateClientLinks(w, req.Links)
		if !ok {
			return
		}
		if err := s.noteService.UpdateLinksFromClient(userID, note.ID, linkTitles); err != nil {
			s.logger().Error("failed to update links from client", "err", err, "note_id", note.ID)
		}
	}

	// Process client-provided due dates for encrypted notes
	if len(req.DueDates) > 0 {
		if err := s.noteService.GetDB().SetNoteDueDates(note.ID, userID, convertClientDueDates(req.DueDates)); err != nil {
			s.logger().Error("failed to set due dates from client", "err", err, "note_id", note.ID)
		}
	}

	// Broadcast creation to WebSocket clients
	payload, err := json.Marshal(note)
	if err != nil {
		s.logger().Error("failed to encode note.created payload", "err", err, "note_id", note.ID)
	} else {
		s.wsManager.BroadcastToUser(userID, websocket.Message{
			Type:    "note.created",
			Payload: payload,
		})
	}

	w.Header().Set("ETag", generateETag(note.ID, note.Version))
	respondJSON(w, http.StatusCreated, note)
}

func (s *Server) getNote(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	note, err := s.noteService.GetNote(userID, id)
	if err != nil {
		s.respondInternalErr(w, "failed to get note", err)
		return
	}

	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	w.Header().Set("ETag", generateETag(note.ID, note.Version))
	respondJSON(w, http.StatusOK, note)
}

func (s *Server) updateNote(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	// Check If-Match header for optimistic locking
	ifMatch := r.Header.Get("If-Match")
	if ifMatch == "" {
		respondError(w, http.StatusBadRequest, "If-Match header required")
		return
	}

	version, ok2 := s.resolveETagVersion(w, userID, id, ifMatch)
	if !ok2 {
		return
	}

	var req NoteRequest
	if err := decodeJSONWithLimit(w, r, &req, MaxLargeJSONBodySize); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateNoteFields(req.Title, req.Content, req.FolderPath); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// note_type and journal_date are immutable after creation
	if req.NoteType != "" {
		respondError(w, http.StatusBadRequest, "note_type cannot be modified after creation")
		return
	}
	if req.JournalDate != nil {
		respondError(w, http.StatusBadRequest, "journal_date cannot be modified after creation")
		return
	}

	var note *db.Note
	var err error

	// Check if this is an encrypted note
	if req.EncryptedContent != "" && req.WrappedDEK != "" {
		// Validate all encryption fields
		if err := ValidateEncryptedNoteRequest(req.EncryptedContent, req.WrappedDEK, req.EncryptionMetadata, req.EncryptedTitle); err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("encryption validation failed: %v", err))
			return
		}

		// Decode Base64 encrypted content (already validated)
		encryptedBlob, err := base64.StdEncoding.DecodeString(req.EncryptedContent)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid encrypted content")
			return
		}

		// Update encrypted note
		note, err = s.noteService.UpdateEncryptedNote(
			userID,
			id,
			req.Title,
			req.EncryptedTitle,
			req.TitleEncrypted,
			encryptedBlob,
			req.WrappedDEK,
			req.EncryptionMetadata,
			req.FolderPath,
			req.Keywords,
			version,
		)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				respondError(w, http.StatusNotFound, "note not found")
				return
			}
			if errors.Is(err, db.ErrVersionMismatch) {
				respondError(w, http.StatusConflict, "version mismatch - note was modified")
				return
			}
			s.respondInternalErr(w, "failed to update encrypted note", err)
			return
		}
	} else {
		// Update plaintext note (legacy support)
		note, err = s.noteService.UpdateNote(userID, id, req.Title, req.Content, req.FolderPath, version)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				respondError(w, http.StatusNotFound, "note not found")
				return
			}
			if errors.Is(err, db.ErrVersionMismatch) {
				respondError(w, http.StatusConflict, "version mismatch - note was modified")
				return
			}
			s.respondInternalErr(w, "failed to update note", err)
			return
		}
	}

	// Process client-provided links for encrypted notes
	if len(req.Links) > 0 {
		linkTitles, ok := validateClientLinks(w, req.Links)
		if !ok {
			return
		}
		if err := s.noteService.UpdateLinksFromClient(userID, id, linkTitles); err != nil {
			s.logger().Error("failed to update links from client", "err", err, "note_id", id)
		}
	}

	// Process client-provided due dates for encrypted notes
	if len(req.DueDates) > 0 {
		if err := s.noteService.GetDB().SetNoteDueDates(id, userID, convertClientDueDates(req.DueDates)); err != nil {
			s.logger().Error("failed to set due dates from client", "err", err, "note_id", id)
		}
	}

	// Invalidate graph cache after link updates
	if s.graphService != nil {
		s.graphService.InvalidateGraphCache(userID)
	}

	// Broadcast update to WebSocket clients
	payload, err := json.Marshal(note)
	if err != nil {
		s.respondInternalErr(w, "failed to encode note update", err)
		return
	}
	s.wsManager.BroadcastToUser(userID, websocket.Message{
		Type:    "note.updated",
		Payload: payload,
	})

	w.Header().Set("ETag", generateETag(note.ID, note.Version))
	respondJSON(w, http.StatusOK, note)
}

func (s *Server) deleteNote(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	err := s.noteService.DeleteNote(userID, id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		s.respondInternalErr(w, "failed to delete note", err)
		return
	}

	// Broadcast deletion to WebSocket clients
	payload, err := json.Marshal(map[string]string{"id": id})
	if err != nil {
		s.logger().Error("failed to encode note.deleted payload", "err", err, "note_id", id)
	} else {
		s.wsManager.BroadcastToUser(userID, websocket.Message{
			Type:    "note.deleted",
			Payload: payload,
		})
	}

	w.WriteHeader(http.StatusNoContent)
}

// DecryptNoteRequest represents the request body for decrypting a note.
type DecryptNoteRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	// Recipe fields (optional, only for recipe notes)
	RecipeMetadata    *db.RecipeMetadata    `json:"recipe_metadata,omitempty"`
	RecipeIngredients []db.RecipeIngredient `json:"recipe_ingredients,omitempty"`
}

func (s *Server) decryptNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	// Check If-Match header for optimistic locking
	ifMatch := r.Header.Get("If-Match")
	if ifMatch == "" {
		respondError(w, http.StatusBadRequest, "If-Match header required")
		return
	}

	version, ok2 := s.resolveETagVersion(w, userID, id, ifMatch)
	if !ok2 {
		return
	}

	var req DecryptNoteRequest
	if err := decodeJSONWithLimit(w, r, &req, MaxLargeJSONBodySize); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateNoteFields(req.Title, req.Content, ""); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	var note *db.Note
	var err error
	if req.RecipeMetadata != nil || len(req.RecipeIngredients) > 0 {
		note, err = s.noteService.DecryptRecipeNote(userID, id, req.Title, req.Content, version,
			req.RecipeMetadata, req.RecipeIngredients)
	} else {
		note, err = s.noteService.DecryptNote(userID, id, req.Title, req.Content, version)
	}
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		if errors.Is(err, db.ErrVersionMismatch) {
			respondError(w, http.StatusConflict, "version mismatch - note was modified")
			return
		}
		if err.Error() == "note is not encrypted" {
			respondError(w, http.StatusBadRequest, "note is not encrypted")
			return
		}
		s.respondInternalErr(w, "failed to decrypt note", err)
		return
	}

	// Broadcast update to WebSocket clients
	payload, err := json.Marshal(note)
	if err != nil {
		s.logger().Error("failed to encode note.updated payload", "err", err, "note_id", note.ID)
	} else {
		s.wsManager.BroadcastToUser(userID, websocket.Message{
			Type:    "note.updated",
			Payload: payload,
		})
	}

	w.Header().Set("ETag", generateETag(note.ID, note.Version))
	respondJSON(w, http.StatusOK, note)
}

// RenameRequest represents the request body for renaming a note.
type RenameRequest struct {
	NewTitle string `json:"newTitle"`
}

func (s *Server) renameNote(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	var req RenameRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.NewTitle == "" {
		respondError(w, http.StatusBadRequest, "newTitle is required")
		return
	}

	// Check for async mode
	asyncMode := r.URL.Query().Get("async") == "true"

	if asyncMode {
		// Async mode - submit job and return immediately
		jobID := fmt.Sprintf("job_%d_%d", userID, time.Now().UnixNano())
		job := &jobs.Job{
			ID:     jobID,
			Type:   jobs.JobTypeRenameNote,
			UserID: userID,
			Metadata: map[string]interface{}{
				"noteID":   id,
				"newTitle": req.NewTitle,
			},
		}

		if err := s.jobManager.Submit(job); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to submit job")
			return
		}

		respondJSON(w, http.StatusAccepted, map[string]interface{}{
			"job_id": jobID,
			"status": "pending",
		})
		return
	}

	// Sync mode - execute immediately (existing behavior)
	result, err := s.noteService.RenameNote(r.Context(), userID, id, req.NewTitle)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		s.respondInternalErr(w, "failed to rename note", err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func (s *Server) getBacklinks(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	backlinks, err := s.noteService.GetBacklinks(userID, id)
	if err != nil {
		s.respondInternalErr(w, "failed to get backlinks", err)
		return
	}

	if backlinks == nil {
		backlinks = []db.Backlink{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"backlinks": backlinks,
	})
}

// --- Trash Management Endpoints ---

// listTrash returns a paginated list of soft-deleted notes.
func (s *Server) listTrash(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = min(parsed, 500) // Cap at 500 to prevent memory exhaustion
		}
	}
	cursor := r.URL.Query().Get("cursor")

	notes, nextCursor, err := s.noteService.ListDeletedNotes(userID, limit, cursor)
	if err != nil {
		s.respondInternalErr(w, "failed to list trash", err)
		return
	}

	respondJSON(w, http.StatusOK, NoteListResponse{
		Notes:      ensureNotes(notes),
		NextCursor: nextCursor,
	})
}

// getTrashCount returns the count of soft-deleted notes.
func (s *Server) getTrashCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	count, err := s.noteService.GetDeletedNotesCount(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to get trash count", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"count": count,
	})
}

// restoreNote restores a soft-deleted note.
func (s *Server) restoreNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	note, err := s.noteService.RestoreNote(userID, id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		s.respondInternalErr(w, "failed to restore note", err)
		return
	}

	w.Header().Set("ETag", generateETag(note.ID, note.Version))
	respondJSON(w, http.StatusOK, note)
}

// permanentlyDeleteNote performs a hard delete on a note.
func (s *Server) permanentlyDeleteNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	err := s.noteService.PermanentlyDeleteNote(userID, id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found or not deleted")
			return
		}
		s.respondInternalErr(w, "failed to permanently delete note", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// emptyTrash permanently deletes all soft-deleted notes.
func (s *Server) emptyTrash(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	count, err := s.noteService.EmptyTrash(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to empty trash", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"deleted_count": count,
	})
}

// BatchReencryptDEKsRequest represents a batch update of wrapped DEKs
type BatchReencryptDEKsRequest struct {
	Updates []struct {
		NoteID     string `json:"note_id"`
		WrappedDEK string `json:"wrapped_dek"` // Base64-encoded
	} `json:"updates"`
}

// UpdateNoteColorRequest represents the request body for updating a note's color.
type UpdateNoteColorRequest struct {
	Color *string `json:"color"`
}

// updateNoteColor updates the color of a note.
func (s *Server) updateNoteColor(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	var req UpdateNoteColorRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := s.noteService.UpdateNoteColor(userID, id, req.Color)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// batchReencryptDEKs updates wrapped DEKs for multiple notes in a single transaction
// Used after password changes to re-wrap all DEKs with the new KEK
func (s *Server) batchReencryptDEKs(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req BatchReencryptDEKsRequest
	if err := decodeJSONWithLimit(w, r, &req, MaxLargeJSONBodySize); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Updates) == 0 {
		respondError(w, http.StatusBadRequest, "no updates provided")
		return
	}

	// Validate all wrapped DEKs
	for i, update := range req.Updates {
		if update.NoteID == "" {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("note_id required for update %d", i))
			return
		}
		if err := ValidateWrappedDEK(update.WrappedDEK); err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid wrapped_dek for update %d: %v", i, err))
			return
		}
	}

	// Perform batch update
	updatedCount, err := s.noteService.BatchUpdateWrappedDEKs(userID, req.Updates)
	if err != nil {
		s.logger().Error("failed to batch reencrypt DEKs", "error", err, "user_id", userID)
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "one or more notes not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to update DEKs")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"updated_count": updatedCount,
		"message":       "DEKs successfully re-encrypted",
	})
}

// --- Summary Endpoints ---

// SummarizeNoteRequest represents the request body for summarizing a note.
type SummarizeNoteRequest struct {
	// PlaintextContent is required for encrypted notes (decrypted content from frontend)
	PlaintextContent string `json:"plaintext_content,omitempty"`
	// PlaintextContentHash is the SHA256 hash of plaintext (for E2E notes, computed by frontend)
	PlaintextContentHash string `json:"plaintext_content_hash,omitempty"`
	// EncryptedSummary is the encrypted summary (for E2E notes, encrypted by frontend)
	EncryptedSummary string `json:"encrypted_summary,omitempty"`
}

// SummarizeNoteResponse represents the response from summarizing a note.
type SummarizeNoteResponse struct {
	Summary string `json:"summary"`
}

// summarizeNote generates or retrieves a summary for a note.
// For plaintext notes: generates summary server-side
// For encrypted notes: requires frontend to send decrypted content, generates summary, returns for frontend encryption
func (s *Server) summarizeNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID is required")
		return
	}

	var req SummarizeNoteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		// Allow empty body for plaintext notes
		req = SummarizeNoteRequest{}
	}

	// Check if this is storing a pre-encrypted summary (E2E flow)
	if req.EncryptedSummary != "" && req.PlaintextContentHash != "" {
		// Store pre-encrypted summary
		if err := s.summarizeService.SummarizeNoteEncrypted(
			r.Context(),
			userID,
			noteID,
			req.EncryptedSummary,
			req.PlaintextContentHash,
		); err != nil {
			s.logger().Error("failed to store encrypted summary",
				"error", err,
				"note_id", noteID,
				"user_id", userID,
			)
			respondError(w, http.StatusInternalServerError, "failed to store summary")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
		return
	}

	// Generate summary (for plaintext or intermediate E2E step)
	summary, err := s.summarizeService.SummarizeNote(
		r.Context(),
		userID,
		noteID,
		req.PlaintextContent,
	)
	if err != nil {
		s.logger().Error("failed to summarize note",
			"error", err,
			"note_id", noteID,
			"user_id", userID,
		)

		// Check for specific error types
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		if strings.Contains(err.Error(), "plaintext content required") {
			respondError(w, http.StatusBadRequest, "plaintext_content is required for encrypted notes")
			return
		}

		respondError(w, http.StatusInternalServerError, "failed to generate summary")
		return
	}

	respondJSON(w, http.StatusOK, SummarizeNoteResponse{
		Summary: summary,
	})
}

// prepareSummarizeStream stores plaintext content and returns a one-time token
// for the SSE stream endpoint, avoiding plaintext in URL query parameters.
func (s *Server) prepareSummarizeStream(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID is required")
		return
	}

	var req struct {
		PlaintextContent string `json:"plaintext_content"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.PlaintextContent) > streamStoreMaxContentSize {
		respondError(w, http.StatusRequestEntityTooLarge, "content too large")
		return
	}

	token, err := generateStreamToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate stream token")
		return
	}

	if err := s.streamContent.store(token, userID, noteID, req.PlaintextContent); err != nil {
		respondError(w, http.StatusTooManyRequests, "too many pending stream requests, try again later")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"stream_token": token})
}

// summarizeNoteStream generates a summary with Server-Sent Events for progress.
func (s *Server) summarizeNoteStream(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID is required")
		return
	}

	// Get plaintext_content via stream token (preferred) or query parameter (deprecated)
	var plaintextContent string
	streamToken := r.URL.Query().Get("token")
	if streamToken != "" {
		entry, err := s.streamContent.get(streamToken)
		if err != nil || entry.userID != userID || entry.noteID != noteID {
			respondError(w, http.StatusBadRequest, "invalid or expired stream token")
			return
		}
		plaintextContent = entry.plaintextContent
	} else if pc := r.URL.Query().Get("plaintext_content"); pc != "" {
		s.logger().Warn("deprecated: plaintext_content in query param, use /prepare endpoint",
			"note_id", noteID, "user_id", userID)
		plaintextContent = pc
	}

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Extend write deadline for SSE streaming (5 minutes max)
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Now().Add(5 * time.Minute)); err != nil {
		s.logger().Warn("failed to extend SSE write deadline", "error", err)
	}

	// Send initial event
	fmt.Fprintf(w, "event: start\ndata: {\"status\":\"generating\"}\n\n")
	flusher.Flush()

	// Generate summary with streaming callback
	cached, summary, err := s.summarizeService.SummarizeNoteStream(
		r.Context(),
		userID,
		noteID,
		plaintextContent,
		func(token string) {
			data, err := json.Marshal(token)
			if err != nil {
				s.logger().Error("failed to marshal SSE token", "error", err)
				fmt.Fprintf(w, "event: error\ndata: {\"error\":\"internal encoding error\"}\n\n")
				flusher.Flush()
				return
			}
			fmt.Fprintf(w, "event: token\ndata: %s\n\n", data)
			flusher.Flush()
		},
	)

	if err != nil {
		s.logger().Error("failed to summarize note (stream)",
			"error", err,
			"note_id", noteID,
			"user_id", userID,
		)
		errPayload, marshalErr := json.Marshal(map[string]string{"error": err.Error()})
		if marshalErr != nil {
			s.logger().Error("failed to marshal SSE error", "error", marshalErr)
			errPayload = []byte(`{"error":"internal error"}`)
		}
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", errPayload)
		flusher.Flush()
		return
	}

	// Send completion event
	if cached {
		// For cached summaries, send the full text at once
		cachedPayload, marshalErr := json.Marshal(map[string]string{"summary": summary})
		if marshalErr != nil {
			s.logger().Error("failed to marshal SSE cached summary", "error", marshalErr)
			fmt.Fprintf(w, "event: error\ndata: {\"error\":\"internal encoding error\"}\n\n")
		} else {
			fmt.Fprintf(w, "event: cached\ndata: %s\n\n", cachedPayload)
		}
	} else {
		fmt.Fprintf(w, "event: done\ndata: {\"status\":\"complete\"}\n\n")
	}
	flusher.Flush()
}

// ============================================================================
// LLM Feature: Tag Suggestions
// ============================================================================

// SuggestTagsRequest represents the request body for tag suggestions.
type SuggestTagsRequest struct {
	PlaintextContent string `json:"plaintext_content,omitempty"`
}

// SuggestTagsResponse represents the response from tag suggestions.
type SuggestTagsResponse struct {
	Suggestions []service.TagSuggestion `json:"suggestions"`
}

// suggestTags generates tag suggestions for a note using LLM.
func (s *Server) suggestTags(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID is required")
		return
	}

	var req SuggestTagsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		req = SuggestTagsRequest{}
	}

	// Get note to check if encrypted
	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil {
		s.respondInternalErr(w, "failed to get note for tag suggestions", err)
		return
	}
	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	// Determine content source
	var content string
	if note.ContentEncrypted {
		if req.PlaintextContent == "" {
			respondError(w, http.StatusBadRequest, "plaintext_content is required for encrypted notes")
			return
		}
		content = req.PlaintextContent
	} else {
		content = note.Content
	}

	// Validate content size
	if len(content) > service.MaxPlaintextContent {
		respondError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("content too large (max %d bytes)", service.MaxPlaintextContent))
		return
	}

	// Generate suggestions (include title for better context)
	suggestions, err := s.summarizeService.SuggestTagsForNote(r.Context(), userID, noteID, note.Title, content)
	if err != nil {
		s.logger().Error("tag suggestion failed",
			"error", err,
			"note_id", noteID,
			"user_id", userID,
		)

		switch {
		case errors.Is(err, llm.ErrNoteNotAIEnabled):
			respondError(w, http.StatusForbidden, "AI features not enabled for this note")
		case errors.Is(err, llm.ErrNoProviderAvailable):
			respondError(w, http.StatusPreconditionFailed, "AI provider required - add API key in settings")
		default:
			respondError(w, http.StatusInternalServerError, "failed to generate tag suggestions")
		}
		return
	}

	respondJSON(w, http.StatusOK, SuggestTagsResponse{
		Suggestions: suggestions,
	})
}

// ============================================================================
// LLM Feature: Link Suggestions
// ============================================================================

// SuggestLinksRequest represents the request body for link suggestions.
type SuggestLinksRequest struct {
	PlaintextContent string   `json:"plaintext_content,omitempty"`
	NoteTitles       []string `json:"note_titles"`
	ExistingLinks    []string `json:"existing_links"`
}

// SuggestLinksResponse represents the response from link suggestions.
type SuggestLinksResponse struct {
	Suggestions []service.LinkSuggestion `json:"suggestions"`
}

// suggestLinks generates wikilink suggestions for a note using LLM.
func (s *Server) suggestLinks(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID is required")
		return
	}

	var req SuggestLinksRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate title count
	if len(req.NoteTitles) > service.MaxNoteTitles {
		respondError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("too many note titles (max %d)", service.MaxNoteTitles))
		return
	}

	// Get note to check if encrypted
	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil {
		s.respondInternalErr(w, "failed to get note for link suggestions", err)
		return
	}
	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	// Determine content source
	var content string
	if note.ContentEncrypted {
		if req.PlaintextContent == "" {
			respondError(w, http.StatusBadRequest, "plaintext_content is required for encrypted notes")
			return
		}
		content = req.PlaintextContent
	} else {
		content = note.Content
	}

	// Validate content size
	if len(content) > service.MaxPlaintextContent {
		respondError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("content too large (max %d bytes)", service.MaxPlaintextContent))
		return
	}

	// Generate suggestions (using SuggestLinksForNote with userID and noteID)
	suggestions, err := s.summarizeService.SuggestLinksForNote(r.Context(), userID, noteID, content, req.NoteTitles, req.ExistingLinks)
	if err != nil {
		s.logger().Error("link suggestion failed",
			"error", err,
			"note_id", noteID,
			"user_id", userID,
		)

		switch {
		case errors.Is(err, llm.ErrNoteNotAIEnabled):
			respondError(w, http.StatusForbidden, "AI features not enabled for this note")
		case errors.Is(err, llm.ErrNoProviderAvailable):
			respondError(w, http.StatusPreconditionFailed, "AI provider required - add API key in settings")
		default:
			respondError(w, http.StatusInternalServerError, "failed to generate link suggestions")
		}
		return
	}

	respondJSON(w, http.StatusOK, SuggestLinksResponse{
		Suggestions: suggestions,
	})
}

// listNoteTitles returns a lightweight list of note titles for link suggestions.
// Only returns unencrypted titles to avoid sending encrypted data to LLM.
// Limited to MaxNoteTitlesForSuggestions to prevent memory exhaustion.
const MaxNoteTitlesForSuggestions = 1000

func (s *Server) listNoteTitles(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Fetch notes with pagination, limited to prevent memory exhaustion
	var allNotes []db.Note
	cursor := ""
	for {
		notes, nextCursor, err := s.noteService.ListNotes(userID, 500, cursor)
		if err != nil {
			s.respondInternalErr(w, "failed to list note titles", err)
			return
		}
		allNotes = append(allNotes, notes...)
		// Early exit if we have enough notes
		if len(allNotes) >= MaxNoteTitlesForSuggestions || nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	// Build title list - only include unencrypted titles
	titles := make([]NoteTitleInfo, 0, len(allNotes))
	for _, note := range allNotes {
		// Skip notes with encrypted titles (privacy-first)
		if note.TitleEncrypted {
			continue
		}
		titles = append(titles, NoteTitleInfo{
			ID:        note.ID,
			Title:     note.Title,
			Encrypted: note.ContentEncrypted,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"titles": titles,
	})
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

// ============================================================================
// AI-Enabled (Claude API Opt-In) Endpoints
// ============================================================================

// UpdateAIEnabledRequest represents the request body for toggling ai_enabled.
type UpdateAIEnabledRequest struct {
	AIEnabled bool `json:"ai_enabled"`
}

// updateNoteAIEnabled toggles the ai_enabled flag for a note.
// When ai_enabled=true, Cloud-KI features (Claude API) are allowed for this note.
func (s *Server) updateNoteAIEnabled(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID is required")
		return
	}

	var req UpdateAIEnabledRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Check if note exists and belongs to user
	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil {
		s.respondInternalErr(w, "failed to get note", err)
		return
	}
	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	// Update ai_enabled flag
	if err := s.noteService.UpdateNoteAIEnabled(userID, noteID, req.AIEnabled); err != nil {
		s.respondInternalErr(w, "failed to update AI enabled flag", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "ok",
		"ai_enabled": req.AIEnabled,
	})
}

// getNoteAIEnabled returns the ai_enabled status for a note.
func (s *Server) getNoteAIEnabled(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID is required")
		return
	}

	aiEnabled, err := s.noteService.GetNoteAIEnabled(userID, noteID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		s.respondInternalErr(w, "failed to get AI enabled status", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ai_enabled": aiEnabled,
	})
}

// ============================================================================
// LLM Feature: Format Markdown
// ============================================================================

// FormatMarkdownRequest represents the request body for formatting markdown.
type FormatMarkdownRequest struct {
	PlaintextContent string `json:"plaintext_content,omitempty"` // For E2E-encrypted notes
	SelectionOnly    string `json:"selection_only,omitempty"`    // When only part is formatted
}

// FormatMarkdownResponse represents the response from formatting markdown.
type FormatMarkdownResponse struct {
	FormattedContent string `json:"formatted_content"`
}

// formatMarkdown formats markdown content using an LLM provider.
// POST /api/notes/{id}/format-markdown
func (s *Server) formatMarkdown(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID is required")
		return
	}

	var req FormatMarkdownRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get note to determine content source
	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil {
		s.respondInternalErr(w, "failed to get note for formatting", err)
		return
	}
	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	// Determine content to format
	var content string
	if req.SelectionOnly != "" {
		// Formatting a selection
		content = req.SelectionOnly
	} else if req.PlaintextContent != "" {
		// Encrypted note: use provided plaintext
		content = req.PlaintextContent
	} else if note.ContentEncrypted {
		respondError(w, http.StatusBadRequest, "plaintext_content or selection_only is required for encrypted notes")
		return
	} else {
		// Plaintext note: use content from database
		content = note.Content
	}

	// Format the content
	formatted, err := s.summarizeService.FormatMarkdown(r.Context(), userID, noteID, content)
	if err != nil {
		s.logger().Error("format markdown failed",
			"error", err,
			"note_id", noteID,
			"user_id", userID,
		)

		// Map errors to appropriate HTTP status codes
		switch {
		case errors.Is(err, llm.ErrNoProviderAvailable):
			respondError(w, http.StatusPreconditionFailed, "AI provider required - add API key in settings")
		case errors.Is(err, llm.ErrNoteNotAIEnabled):
			respondError(w, http.StatusForbidden, "AI features are disabled for this note")
		case errors.Is(err, service.ErrContentTooLarge):
			respondError(w, http.StatusRequestEntityTooLarge, "Content too large (max 50KB)")
		case errors.Is(err, service.ErrContentTooShort):
			respondError(w, http.StatusBadRequest, "Content too short (min 10 characters)")
		case errors.Is(err, service.ErrContentEmpty):
			respondError(w, http.StatusBadRequest, "No content to format")
		case errors.Is(err, r.Context().Err()):
			respondError(w, http.StatusGatewayTimeout, "Request timed out - try shorter selection")
		default:
			respondError(w, http.StatusInternalServerError, "Formatting failed")
		}
		return
	}

	respondJSON(w, http.StatusOK, FormatMarkdownResponse{
		FormattedContent: formatted,
	})
}

// ============================================================================
// LLM Feature: AI Transform
// ============================================================================

// AITransformRequest represents the request body for AI text transformation.
type AITransformRequest struct {
	Action           string `json:"action"`                      // format, summarize, expand, translate_de, translate_en, formal, informal, custom
	Content          string `json:"content,omitempty"`           // Plain text content
	PlaintextContent string `json:"plaintext_content,omitempty"` // E2E decrypted content (takes precedence)
	CustomPrompt     string `json:"custom_prompt,omitempty"`     // Only for action="custom"
}

// AITransformResponse represents the response from AI text transformation.
type AITransformResponse struct {
	TransformedContent string `json:"transformed_content"`
}

// aiTransform performs AI-based text transformation.
// POST /api/notes/{id}/ai-transform
func (s *Server) aiTransform(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID is required")
		return
	}

	var req AITransformRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate action
	action := service.AITransformAction(req.Action)
	validActions := map[service.AITransformAction]bool{
		service.ActionFormat:      true,
		service.ActionSummarize:   true,
		service.ActionExpand:      true,
		service.ActionTranslateDE: true,
		service.ActionTranslateEN: true,
		service.ActionFormal:      true,
		service.ActionInformal:    true,
		service.ActionCustom:      true,
	}

	if !validActions[action] {
		respondError(w, http.StatusBadRequest, "unknown action")
		return
	}

	// Validate custom prompt for custom action
	if action == service.ActionCustom && strings.TrimSpace(req.CustomPrompt) == "" {
		respondError(w, http.StatusBadRequest, "custom_prompt is required for custom action")
		return
	}

	// Determine content source: PlaintextContent takes precedence
	content := req.PlaintextContent
	if content == "" {
		content = req.Content
	}

	if strings.TrimSpace(content) == "" {
		respondError(w, http.StatusBadRequest, "content is required")
		return
	}

	// Perform transformation
	result, err := s.summarizeService.AITransform(
		r.Context(),
		userID,
		noteID,
		action,
		content,
		req.CustomPrompt,
	)
	if err != nil {
		s.logger().Error("AI transform failed",
			"error", err,
			"note_id", noteID,
			"user_id", userID,
			"action", req.Action,
		)

		// Map errors to appropriate HTTP status codes
		switch {
		case errors.Is(err, llm.ErrNoProviderAvailable):
			respondError(w, http.StatusPreconditionFailed, "AI provider required - add API key in settings")
		case errors.Is(err, llm.ErrNoteNotAIEnabled):
			respondError(w, http.StatusForbidden, "AI features are disabled for this note")
		case errors.Is(err, service.ErrContentTooLarge):
			respondError(w, http.StatusRequestEntityTooLarge, "Content too large (max 50KB)")
		case errors.Is(err, service.ErrContentTooShort):
			respondError(w, http.StatusBadRequest, "Content too short (min 10 characters)")
		case errors.Is(err, service.ErrContentEmpty):
			respondError(w, http.StatusBadRequest, "No content to transform")
		case errors.Is(err, service.ErrResponseTooLarge):
			respondError(w, http.StatusRequestEntityTooLarge, "Response too large")
		case errors.Is(err, service.ErrUnknownAction):
			respondError(w, http.StatusBadRequest, "Unknown action")
		case errors.Is(err, service.ErrCustomPromptRequired):
			respondError(w, http.StatusBadRequest, "custom_prompt is required for custom action")
		case errors.Is(err, r.Context().Err()):
			respondError(w, http.StatusGatewayTimeout, "Request timed out - try shorter content")
		default:
			respondError(w, http.StatusInternalServerError, "Transformation failed")
		}
		return
	}

	respondJSON(w, http.StatusOK, AITransformResponse{
		TransformedContent: result,
	})
}

// listNoteTitlesAIEnabled returns titles of notes with ai_enabled=true.
// Used for Claude API link suggestions (only AI-enabled notes are included).
func (s *Server) listNoteTitlesAIEnabled(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	titles, err := s.noteService.GetNoteTitlesAIEnabled(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to list AI-enabled note titles", err)
		return
	}

	if titles == nil {
		titles = []string{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"titles": titles,
	})
}

// reorderNotes updates the display order of notes within a folder.
func (s *Server) reorderNotes(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req struct {
		FolderPath string   `json:"folder_path"`
		Items      []string `json:"items"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Items) == 0 {
		respondError(w, http.StatusBadRequest, "items cannot be empty")
		return
	}

	if len(req.FolderPath) > MaxFolderPathLength {
		respondError(w, http.StatusBadRequest, "folder path too long")
		return
	}

	err := s.noteService.ReorderNotes(userID, req.FolderPath, req.Items)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
