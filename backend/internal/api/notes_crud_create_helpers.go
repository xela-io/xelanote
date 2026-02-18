package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/xela-io/xelanote/internal/service"
	"github.com/xela-io/xelanote/internal/websocket"
)

var (
	errCreateValidationEncryptedPayload = errors.New("create note: invalid encrypted payload")
	errCreateValidationEncryptedContent = errors.New("create note: invalid encrypted content")
)

func decodeAndValidateCreateNoteRequest(w http.ResponseWriter, r *http.Request) (*NoteRequest, bool) {
	var req NoteRequest
	if err := decodeJSONWithLimit(w, r, &req, MaxLargeJSONBodySize); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return nil, false
	}

	if err := validateNoteFields(req.Title, req.Content, req.FolderPath); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return nil, false
	}

	if req.FolderPath == "" {
		req.FolderPath = "/"
	}

	return &req, true
}

func (s *Server) validateCreateNoteTypeConstraints(w http.ResponseWriter, req *NoteRequest) bool {
	if !service.IsAllowedNoteType(req.NoteType) {
		respondError(w, http.StatusBadRequest, "invalid note_type")
		return false
	}

	switch req.NoteType {
	case "canvas":
		if req.JournalDate != nil && *req.JournalDate != "" {
			respondError(w, http.StatusBadRequest, "journal_date not allowed for canvas notes")
			return false
		}
		if s.canvasService == nil {
			respondError(w, http.StatusInternalServerError, "canvas service not available")
			return false
		}
	case "recipe":
		if req.JournalDate != nil && *req.JournalDate != "" {
			respondError(w, http.StatusBadRequest, "journal_date not allowed for recipe notes")
			return false
		}
		if s.recipeService == nil {
			respondError(w, http.StatusInternalServerError, "recipe service not available")
			return false
		}
	case "journal":
		if req.JournalDate == nil || *req.JournalDate == "" {
			respondError(w, http.StatusBadRequest, "journal_date is required when note_type is 'journal'")
			return false
		}
		if err := service.ValidateJournalDate(*req.JournalDate); err != nil {
			respondError(w, http.StatusBadRequest, "invalid journal_date format, expected YYYY-MM-DD")
			return false
		}
	default:
		if req.JournalDate != nil && *req.JournalDate != "" {
			respondError(w, http.StatusBadRequest, "journal_date can only be set when note_type is 'journal'")
			return false
		}
	}

	return true
}

func (s *Server) createNoteFromRequest(userID int, req *NoteRequest) (*service.Note, error) {
	if req.EncryptedContent != "" && req.WrappedDEK != "" {
		return s.createEncryptedNoteFromRequest(userID, req)
	}
	return s.createPlaintextNoteFromRequest(userID, req)
}

func (s *Server) createEncryptedNoteFromRequest(userID int, req *NoteRequest) (*service.Note, error) {
	if err := ValidateEncryptedNoteRequest(req.EncryptedContent, req.WrappedDEK, req.EncryptionMetadata, req.EncryptedTitle); err != nil {
		return nil, fmt.Errorf("%w: %v", errCreateValidationEncryptedPayload, err)
	}

	encryptedBlob, err := base64.StdEncoding.DecodeString(req.EncryptedContent)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errCreateValidationEncryptedContent, err)
	}

	switch req.NoteType {
	case "journal":
		return s.noteService.CreateEncryptedJournalNote(
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
	case "recipe":
		return s.recipeService.CreateEncryptedRecipeNote(
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
	case "canvas":
		return s.canvasService.CreateEncryptedCanvasNote(
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
	default:
		return s.noteService.CreateEncryptedNote(
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
}

func (s *Server) createPlaintextNoteFromRequest(userID int, req *NoteRequest) (*service.Note, error) {
	switch req.NoteType {
	case "journal":
		return s.noteService.CreateJournalNote(userID, req.Title, req.Content, req.FolderPath, *req.JournalDate)
	case "recipe":
		return s.recipeService.CreateRecipeNote(userID, req.Title, req.Content, req.FolderPath)
	case "canvas":
		return s.canvasService.CreateCanvasNote(userID, req.Title, req.Content, req.FolderPath)
	default:
		return s.noteService.CreateNote(userID, req.Title, req.Content, req.FolderPath)
	}
}

func (s *Server) applyCreateNotePostProcessing(w http.ResponseWriter, userID int, req *NoteRequest, note *service.Note) bool {
	if len(req.Links) > 0 {
		linkTitles, ok := validateClientLinks(w, req.Links)
		if !ok {
			return false
		}
		if err := s.noteService.UpdateLinksFromClient(userID, note.ID, linkTitles); err != nil {
			s.logger().Error("failed to update links from client", "err", err, "note_id", note.ID)
		}
	}

	if len(req.DueDates) > 0 {
		if err := s.noteService.SetNoteDueDates(note.ID, userID, convertClientDueDates(req.DueDates)); err != nil {
			s.logger().Error("failed to set due dates from client", "err", err, "note_id", note.ID)
		}
	}

	return true
}

func (s *Server) broadcastNoteCreated(userID int, note *service.Note) {
	payload, err := json.Marshal(note)
	if err != nil {
		s.logger().Error("failed to encode note.created payload", "err", err, "note_id", note.ID)
		return
	}

	s.wsManager.BroadcastToUser(userID, websocket.Message{
		Type:    "note.created",
		Payload: payload,
	})
}
