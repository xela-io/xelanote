package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/jobs"
	"github.com/xela-io/xelanote/internal/service"
)

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
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		s.respondInternalErr(w, "failed to rename note", err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}
