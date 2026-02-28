package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/service"
	"github.com/xela-io/xelanote/internal/websocket"
)

func (s *Server) updateNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

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
	titleForValidation := req.Title
	if titleForValidation == "" && req.TitleEncrypted && req.EncryptedTitle != nil && *req.EncryptedTitle != "" {
		// Encrypted titles intentionally keep plaintext title empty.
		titleForValidation = "encrypted-title"
	}

	if err := validateNoteFields(titleForValidation, req.Content, req.FolderPath); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.FolderPath == "" {
		req.FolderPath = "/"
	}

	note, ok := s.executeNoteUpdate(w, userID, id, req, version)
	if !ok {
		return
	}

	if !s.applyNoteUpdateSideEffects(w, userID, id, req) {
		return
	}

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

// executeNoteUpdate handles encrypted vs plaintext update branching.
func (s *Server) executeNoteUpdate(w http.ResponseWriter, userID int, id string, req NoteRequest, version int) (*service.Note, bool) {
	if req.EncryptedContent != "" && req.WrappedDEK != "" {
		if err := ValidateEncryptedNoteRequest(req.EncryptedContent, req.WrappedDEK, req.EncryptionMetadata, req.EncryptedTitle); err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("encryption validation failed: %v", err))
			return nil, false
		}
		encryptedBlob, err := base64.StdEncoding.DecodeString(req.EncryptedContent)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid encrypted content")
			return nil, false
		}
		note, err := s.noteService.UpdateEncryptedNote(
			userID, id, req.Title, req.EncryptedTitle, req.TitleEncrypted,
			encryptedBlob, req.WrappedDEK, req.EncryptionMetadata,
			req.FolderPath, nil, version, // Privacy hardening: drop plaintext keywords for encrypted notes.
		)
		if err != nil {
			respondNoteUpdateError(s, w, err, "failed to update encrypted note")
			return nil, false
		}
		return note, true
	}

	note, err := s.noteService.UpdateNote(userID, id, req.Title, req.Content, req.FolderPath, version)
	if err != nil {
		respondNoteUpdateError(s, w, err, "failed to update note")
		return nil, false
	}
	return note, true
}

func respondNoteUpdateError(s *Server, w http.ResponseWriter, err error, logMsg string) {
	if errors.Is(err, service.ErrNotFound) {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}
	if errors.Is(err, service.ErrVersionMismatch) {
		respondError(w, http.StatusConflict, "version mismatch - note was modified")
		return
	}
	s.respondInternalErr(w, logMsg, err)
}

// applyNoteUpdateSideEffects processes links, due dates, and graph cache after an update.
// Returns false if link validation failed (response already written).
func (s *Server) applyNoteUpdateSideEffects(w http.ResponseWriter, userID int, id string, req NoteRequest) bool {
	// Privacy hardening: encrypted notes must not leak links/due-dates metadata server-side.
	// Always clear previously persisted metadata and ignore client-provided values.
	if req.EncryptedContent != "" && req.WrappedDEK != "" {
		if err := s.noteService.UpdateLinksFromClient(userID, id, []string{}); err != nil {
			s.logger().Error("failed to clear links for encrypted note", "err", err, "note_id", id)
		}
		if err := s.noteService.SetNoteDueDates(id, userID, nil); err != nil {
			s.logger().Error("failed to clear due dates for encrypted note", "err", err, "note_id", id)
		}
		if s.graphService != nil {
			s.graphService.InvalidateGraphCache(userID)
		}
		return true
	}

	if len(req.Links) > 0 {
		linkTitles, ok := validateClientLinks(w, req.Links)
		if !ok {
			return false
		}
		if err := s.noteService.UpdateLinksFromClient(userID, id, linkTitles); err != nil {
			s.logger().Error("failed to update links from client", "err", err, "note_id", id)
		}
	}
	if len(req.DueDates) > 0 {
		if err := s.noteService.SetNoteDueDates(id, userID, convertClientDueDates(req.DueDates)); err != nil {
			s.logger().Error("failed to set due dates from client", "err", err, "note_id", id)
		}
	}
	if s.graphService != nil {
		s.graphService.InvalidateGraphCache(userID)
	}
	return true
}
