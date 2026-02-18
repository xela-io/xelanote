package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/service"
)

// shareNote handles POST /api/notes/{id}/shares
func (s *Server) shareNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	var req ShareNoteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if msg := validateShareCreateInput(req.Identifier, req.Role); msg != "" {
		respondError(w, http.StatusBadRequest, msg)
		return
	}

	share, err := s.sharingService.ShareNote(userID, noteID, req.Identifier, req.Role)
	if err != nil {
		if status, msg, handled := mapShareCreateError(shareResourceNote, err); handled {
			respondError(w, status, msg)
		} else {
			s.logger().Error("unexpected sharing error", "error", err)
			respondError(w, http.StatusInternalServerError, "an unexpected error occurred")
		}
		return
	}

	respondJSON(w, http.StatusCreated, share)
}

// getNoteShares handles GET /api/notes/{id}/shares
func (s *Server) getNoteShares(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	shares, err := s.sharingService.GetNoteShares(userID, noteID)
	if err != nil {
		if status, msg, handled := mapShareAccessError(shareResourceNote, err); handled {
			respondError(w, status, msg)
			return
		}
		s.respondInternalErr(w, "failed to get note shares", err)
		return
	}

	// Ensure JSON array, not null
	if shares == nil {
		shares = []service.NoteShare{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"shares": shares})
}

// updateShareRole handles PUT /api/notes/{id}/shares/{userId}
func (s *Server) updateShareRole(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	targetUserID, ok := parseIntParam(w, r, "userId", "invalid user ID")
	if !ok {
		return
	}

	var req UpdateShareRoleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if msg := validateShareRoleInput(req.Role); msg != "" {
		respondError(w, http.StatusBadRequest, msg)
		return
	}

	err := s.sharingService.UpdateShareRole(userID, noteID, targetUserID, req.Role)
	if err != nil {
		if status, msg, handled := mapShareMutateError(shareResourceNote, "update", err); handled {
			respondError(w, status, msg)
			return
		}
		s.respondInternalErr(w, "failed to update share role", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// removeShare handles DELETE /api/notes/{id}/shares/{userId}
func (s *Server) removeShare(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	targetUserID, ok := parseIntParam(w, r, "userId", "invalid user ID")
	if !ok {
		return
	}

	err := s.sharingService.UnshareNote(userID, noteID, targetUserID)
	if err != nil {
		if status, msg, handled := mapShareMutateError(shareResourceNote, "remove", err); handled {
			respondError(w, status, msg)
			return
		}
		s.respondInternalErr(w, "failed to remove share", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getSharedNotes handles GET /api/shared
func (s *Server) getSharedNotes(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	notes, err := s.sharingService.GetSharedNotesForUser(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to get shared notes", err)
		return
	}

	// Ensure JSON array, not null
	if notes == nil {
		notes = []service.SharedNote{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"notes": notes})
}

// getSharedNote handles GET /api/shared/{id}
func (s *Server) getSharedNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	note, err := s.sharingService.GetSharedNote(userID, noteID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "shared note not found")
			return
		}
		s.respondInternalErr(w, "failed to get shared note", err)
		return
	}

	respondJSON(w, http.StatusOK, note)
}

// updateSharedNote handles PUT /api/shared/{id}
func (s *Server) updateSharedNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	var req SharedNoteUpdateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Version <= 0 {
		respondError(w, http.StatusBadRequest, "version required")
		return
	}

	note, err := s.sharingService.UpdateSharedNote(userID, noteID, req.Title, req.Content, req.Version)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "shared note not found")
			return
		}
		if errors.Is(err, service.ErrVersionMismatch) {
			respondError(w, http.StatusConflict, "version conflict - note was modified")
			return
		}
		respondError(w, http.StatusForbidden, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, note)
}
