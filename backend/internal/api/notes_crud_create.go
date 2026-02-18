package api

import (
	"errors"
	"net/http"
)

func (s *Server) createNote(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	req, ok := decodeAndValidateCreateNoteRequest(w, r)
	if !ok {
		return
	}

	if !s.validateCreateNoteTypeConstraints(w, req) {
		return
	}

	note, err := s.createNoteFromRequest(userID, req)
	if err != nil {
		if errors.Is(err, errCreateValidationEncryptedPayload) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, errCreateValidationEncryptedContent) {
			respondError(w, http.StatusBadRequest, "invalid encrypted content")
			return
		}
		handleCreateNoteError(w, err, req.Title, req.NoteType == "journal")
		return
	}

	if !s.applyCreateNotePostProcessing(w, userID, req, note) {
		return
	}

	s.broadcastNoteCreated(userID, note)

	w.Header().Set("ETag", generateETag(note.ID, note.Version))
	respondJSON(w, http.StatusCreated, note)
}
