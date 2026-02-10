package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// getJobStatus handles GET /api/jobs/{id}
func (s *Server) getJobStatus(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	jobID := chi.URLParam(r, "id")
	if jobID == "" {
		respondError(w, http.StatusBadRequest, "job ID is required")
		return
	}

	// Get job from manager
	job, err := s.jobManager.GetJob(jobID)
	if err != nil {
		respondError(w, http.StatusNotFound, "job not found")
		return
	}

	// Verify ownership
	if job.UserID != userID {
		respondError(w, http.StatusForbidden, "access denied")
		return
	}

	// Return job status
	respondJSON(w, http.StatusOK, job)
}
