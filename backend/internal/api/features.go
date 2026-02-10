package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// listFeatures returns all feature configurations for the authenticated user.
func (s *Server) listFeatures(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	features, err := s.noteService.GetDB().ListUserFeatures(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list features")
		return
	}

	respondJSON(w, http.StatusOK, features)
}

// getFeature returns a specific feature configuration for the authenticated user.
func (s *Server) getFeature(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	feature := chi.URLParam(r, "feature")

	f, err := s.noteService.GetDB().GetUserFeature(userID, feature)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get feature")
		return
	}

	respondJSON(w, http.StatusOK, f)
}

// setFeature enables or disables a feature for the authenticated user.
func (s *Server) setFeature(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	feature := chi.URLParam(r, "feature")

	var req struct {
		Enabled  bool            `json:"enabled"`
		Settings json.RawMessage `json:"settings,omitempty"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.noteService.GetDB().SetUserFeature(userID, feature, req.Enabled, req.Settings); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to set feature")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}
