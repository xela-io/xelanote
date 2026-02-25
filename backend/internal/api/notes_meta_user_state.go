package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/service"
)

// getNoteUserState returns the per-user state for a note.
func (s *Server) getNoteUserState(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")

	stateData, err := s.noteService.GetNoteUserState(userID, noteID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		s.respondInternalErr(w, "failed to get note user state", err)
		return
	}

	// Parse state_data to extract collapse_state, or return null
	if stateData == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{"collapse_state": nil})
		return
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(*stateData), &envelope); err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{"collapse_state": nil})
		return
	}

	cs, exists := envelope["collapse_state"]
	if !exists {
		respondJSON(w, http.StatusOK, map[string]interface{}{"collapse_state": nil})
		return
	}

	respondJSON(w, http.StatusOK, map[string]json.RawMessage{"collapse_state": cs})
}

// updateNoteUserState updates the per-user collapse state for a note.
func (s *Server) updateNoteUserState(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")

	var req UpdateNoteUserStateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := s.noteService.UpdateNoteUserCollapseState(userID, noteID, &req.CollapseState)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		var ve *service.ValidationError
		if errors.As(err, &ve) {
			respondError(w, http.StatusUnprocessableEntity, ve.Message)
			return
		}
		s.respondInternalErr(w, "failed to update note user state", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
