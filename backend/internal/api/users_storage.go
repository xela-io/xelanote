package api

import "net/http"

// getStorageQuota returns the current user's storage quota information.
func (s *Server) getStorageQuota(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	quota, err := s.adminService.GetUserStorageQuota(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get storage quota")
		return
	}

	respondJSON(w, http.StatusOK, quota)
}
