package api

import (
	"log/slog"
	"net/http"
)

// --- LLM API Key Management (BYOK) ---

// handleSetAPIKey returns a handler that stores an API key for the given provider.
func (s *Server) handleSetAPIKey(p apiKeyProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserID(r)
		if !ok {
			respondError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req struct {
			APIKey string `json:"api_key"`
		}
		if err := decodeJSON(w, r, &req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.APIKey == "" {
			respondError(w, http.StatusBadRequest, "api_key is required")
			return
		}

		if err := p.setKey(userID, req.APIKey); err != nil {
			if err == p.validationErr {
				respondError(w, http.StatusBadRequest, p.invalidKeyMsg)
			} else {
				s.respondInternalErr(w, "failed to store "+p.name+" API key", err)
			}
			return
		}

		p.invalidateCache(userID)
		s.logger().Info(p.name+"_api_key_set",
			slog.Int("user_id", userID),
			slog.String("event", "api_key_set"),
			securityIPAttr(r))

		respondJSON(w, http.StatusOK, map[string]string{"message": "API key stored successfully"})
	}
}

// handleDeleteAPIKey returns a handler that removes the API key for the given provider.
func (s *Server) handleDeleteAPIKey(p apiKeyProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserID(r)
		if !ok {
			respondError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		if err := p.deleteKey(userID); err != nil {
			s.respondInternalErr(w, "failed to delete "+p.name+" API key", err)
			return
		}

		p.invalidateCache(userID)
		s.logger().Info(p.name+"_api_key_deleted",
			slog.Int("user_id", userID),
			slog.String("event", "api_key_deleted"),
			securityIPAttr(r))

		respondJSON(w, http.StatusOK, map[string]string{"message": "API key deleted successfully"})
	}
}

// handleGetAPIKeyStatus returns a handler that checks the API key status for the given provider.
func (s *Server) handleGetAPIKeyStatus(p apiKeyProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserID(r)
		if !ok {
			respondError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		status, err := p.getKeyStatus(userID)
		if err != nil {
			s.respondInternalErr(w, "failed to get "+p.name+" API key status", err)
			return
		}

		respondJSON(w, http.StatusOK, status)
	}
}
