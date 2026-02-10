package api

import (
	"encoding/base64"
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
	"github.com/xela-io/xelanote/internal/service"
	"github.com/xela-io/xelanote/internal/websocket"
)

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
