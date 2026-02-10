package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/jobs"
)

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
