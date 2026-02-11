package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/websocket"
)

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

	if req.FolderPath == "" {
		req.FolderPath = "/"
	}

	var note *db.Note
	var err error

	// Check if this is an encrypted note update
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
