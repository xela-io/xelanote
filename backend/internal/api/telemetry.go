package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/xela-io/xelanote/internal/service"
)

// MaxTelemetryBodySize limits telemetry payloads to 1KB per data governance rules.
const MaxTelemetryBodySize int64 = 1024

type perfMetricRequest struct {
	MetricName   string  `json:"metric_name"`
	Value        float64 `json:"value"`
	Rating       string  `json:"rating"`
	SanitizedURL string  `json:"sanitized_url"`
}

func (s *Server) submitPerfMetric(w http.ResponseWriter, r *http.Request) {
	if s.telemetryService == nil {
		respondError(w, http.StatusServiceUnavailable, "telemetry not available")
		return
	}

	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxTelemetryBodySize)

	var req perfMetricRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == "http: request body too large" {
			respondError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.MetricName = strings.TrimSpace(req.MetricName)
	req.Rating = strings.TrimSpace(req.Rating)
	req.SanitizedURL = strings.TrimSpace(req.SanitizedURL)

	if !service.ValidPerfMetricNames[req.MetricName] {
		respondError(w, http.StatusBadRequest, "invalid metric_name")
		return
	}
	if !service.ValidPerfMetricRatings[req.Rating] {
		respondError(w, http.StatusBadRequest, "invalid rating")
		return
	}
	if len(req.SanitizedURL) > 200 {
		respondError(w, http.StatusBadRequest, "sanitized_url too long")
		return
	}

	if err := s.telemetryService.RecordPerfMetric(userID, req.MetricName, req.Value, req.Rating, req.SanitizedURL); err != nil {
		s.respondInternalErr(w, "failed to record perf metric", err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]string{"status": "recorded"})
}

type analyticsEventRequest struct {
	EventName string          `json:"event_name"`
	Data      json.RawMessage `json:"data"`
}

func (s *Server) submitAnalyticsEvent(w http.ResponseWriter, r *http.Request) {
	if s.telemetryService == nil {
		respondError(w, http.StatusServiceUnavailable, "telemetry not available")
		return
	}

	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxTelemetryBodySize)

	var req analyticsEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == "http: request body too large" {
			respondError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.EventName = strings.TrimSpace(req.EventName)

	if !service.ValidAnalyticsEventNames[req.EventName] {
		respondError(w, http.StatusBadRequest, "invalid event_name")
		return
	}

	dataJSON := "{}"
	if len(req.Data) > 0 {
		dataJSON = string(req.Data)
	}

	if err := s.telemetryService.RecordAnalyticsEvent(userID, req.EventName, dataJSON); err != nil {
		s.respondInternalErr(w, "failed to record analytics event", err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]string{"status": "recorded"})
}
