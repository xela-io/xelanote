package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// JSON response helpers

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	// Prevent caching of authenticated API responses by default.
	// Individual handlers can override by setting the header before calling respondJSON.
	if w.Header().Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("failed to encode JSON response", slog.Any("error", err))
		}
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// respondInternalErr logs the error and responds with a generic 500 message.
// Use this instead of respondError(w, 500, err.Error()) to avoid leaking internal details.
func (s *Server) respondInternalErr(w http.ResponseWriter, msg string, err error) {
	s.logger().Error(msg, slog.Any("error", err))
	respondError(w, http.StatusInternalServerError, "internal server error")
}
