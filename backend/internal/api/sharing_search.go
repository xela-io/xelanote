package api

import (
	"net/http"

	"github.com/xela-io/xelanote/internal/service"
)

// searchUsers handles GET /api/users/search
func (s *Server) searchUsers(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	query := r.URL.Query().Get("q")
	if len(query) < 3 {
		respondError(w, http.StatusBadRequest, "query must be at least 3 characters")
		return
	}

	users, err := s.sharingService.SearchUsers(query, userID)
	if err != nil {
		s.respondInternalErr(w, "failed to search users", err)
		return
	}

	// Ensure JSON array, not null
	if users == nil {
		users = []service.UserSearchResult{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"users": users})
}
