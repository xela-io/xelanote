package api

import (
	"net/http"
)

// getDueDates returns all due dates for the authenticated user.
func (s *Server) getDueDates(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	showCompleted := r.URL.Query().Get("show_completed") == "true"

	dueDates, err := s.noteService.GetDueDatesByUser(userID, showCompleted)
	if err != nil {
		s.logger().Error("failed to get due dates", "err", err, "user_id", userID)
		respondError(w, http.StatusInternalServerError, "failed to get due dates")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"due_dates": dueDates,
	})
}
