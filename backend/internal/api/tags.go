package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/db"
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
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Sicherstellen dass nie nil zurückgegeben wird
	if tags == nil {
		tags = []db.Tag{}
	}

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
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if tags == nil {
		tags = []db.Tag{}
	}

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
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return the updated tags
	tags, err := s.noteService.GetNoteTags(noteID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if tags == nil {
		tags = []db.Tag{}
	}

	respondJSON(w, http.StatusOK, tags)
}

// deleteTag deletes a tag and all its associations.
func (s *Server) deleteTag(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	tagIDStr := chi.URLParam(r, "tagId")
	tagID, err := strconv.Atoi(tagIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid tag id")
		return
	}

	if err := s.noteService.DeleteTag(userID, tagID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
