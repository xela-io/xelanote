package api

import (
	"net/http"
)

// ============================================================================
// AI-Enabled Default (Claude API Opt-In) Endpoints
// ============================================================================

// updateFolderAIEnabledDefault toggles the ai_enabled_default flag for a folder.
// New notes created in this folder will inherit this setting.
func (s *Server) updateFolderAIEnabledDefault(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id, ok := parseIntParam(w, r, "id", "invalid folder id")
	if !ok {
		return
	}

	var req UpdateFolderAIEnabledRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update ai_enabled_default flag
	if err := s.noteService.UpdateFolderAIEnabledDefault(userID, id, req.AIEnabled); err != nil {
		s.respondInternalErr(w, "failed to update AI setting", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "ok",
		"ai_enabled": req.AIEnabled,
	})
}

// getFolderAIEnabledDefault returns the ai_enabled_default status for a folder.
func (s *Server) getFolderAIEnabledDefault(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id, ok := parseIntParam(w, r, "id", "invalid folder id")
	if !ok {
		return
	}

	aiEnabled, err := s.noteService.GetFolderAIEnabledDefault(userID, id)
	if err != nil {
		s.respondInternalErr(w, "failed to get AI setting", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ai_enabled": aiEnabled,
	})
}
