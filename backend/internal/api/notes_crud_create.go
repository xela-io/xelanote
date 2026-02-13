package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/xela-io/xelanote/internal/service"
	"github.com/xela-io/xelanote/internal/websocket"
)

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
			respondError(w, http.StatusBadRequest, "journal_date not allowed for recipe notes")
			return
		}
		if s.recipeService == nil {
			respondError(w, http.StatusInternalServerError, "recipe service not available")
			return
		}
	}

	// Journal validation
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

	var note *service.Note
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
		if err := s.noteService.SetNoteDueDates(note.ID, userID, convertClientDueDates(req.DueDates)); err != nil {
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
