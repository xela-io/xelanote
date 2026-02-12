package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Tag API Request/Response types

type SetNoteTagsRequest struct {
	Tags []string `json:"tags"`
}

// getAllTags returns all tags for the authenticated user.
func (s *Server) getAllTags(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	tags, err := s.noteService.GetAllTags(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to list tags", err)
		return
	}

	// Sicherstellen dass nie nil zurückgegeben wird
	tags = ensureSlice(tags)

	respondJSON(w, http.StatusOK, tags)
}

// getNoteTags returns all tags for a specific note.
func (s *Server) getNoteTags(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note id is required")
		return
	}

	// Verify note ownership
	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}
	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	tags, err := s.noteService.GetNoteTags(noteID)
	if err != nil {
		s.respondInternalErr(w, "failed to get note tags", err)
		return
	}

	tags = ensureSlice(tags)

	respondJSON(w, http.StatusOK, tags)
}

// setNoteTags sets the tags for a note, replacing any existing tags.
func (s *Server) setNoteTags(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note id is required")
		return
	}

	// Verify note ownership
	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}
	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	var req SetNoteTagsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.noteService.SetNoteTags(noteID, userID, req.Tags); err != nil {
		s.respondInternalErr(w, "failed to set note tags", err)
		return
	}

	// Return the updated tags
	tags, err := s.noteService.GetNoteTags(noteID)
	if err != nil {
		s.respondInternalErr(w, "failed to get updated note tags", err)
		return
	}

	tags = ensureSlice(tags)

	respondJSON(w, http.StatusOK, tags)
}

// deleteTag deletes a tag and all its associations.
func (s *Server) deleteTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	tagID, ok := parseIntParam(w, r, "tagId", "invalid tag id")
	if !ok {
		return
	}

	if err := s.noteService.DeleteTag(userID, tagID); err != nil {
		s.respondInternalErr(w, "failed to delete tag", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
