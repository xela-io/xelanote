package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/xela-io/xelanote/internal/service"
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

// respondInternalErr logs the error with the request ID, enqueues an error
// report to Forgejo, and responds with a generic 500.
// Use this instead of respondError(w, 500, err.Error()) to avoid leaking internal details.
func (s *Server) respondInternalErr(w http.ResponseWriter, msg string, err error) {
	attrs := []any{slog.Any("error", err)}
	if reqID := w.Header().Get("X-Request-Id"); reqID != "" {
		attrs = append(attrs, slog.String("request_id", reqID))
	}
	s.logger().Error(msg, attrs...)

	// Enqueue error report for automatic Forgejo issue creation
	if s.errorReportService != nil {
		s.errorReportService.EnqueueReport(service.ErrorReport{
			Type:        "automatic",
			ErrorType:   "BackendError",
			Message:     msg,
			Stack:       err.Error(),
			Fingerprint: service.ComputeFingerprint("BackendError", msg),
			Component:   "backend",
		})
	}

	respondError(w, http.StatusInternalServerError, "internal server error")
}
