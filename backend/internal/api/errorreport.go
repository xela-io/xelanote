package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/xela-io/xelanote/internal/service"
)

// MaxErrorReportBodySize limits error report payloads to 16KB.
const MaxErrorReportBodySize = 16 << 10

var fingerprintRegex = regexp.MustCompile(`^[0-9a-f]{16}$`)

type errorReportRequest struct {
	Type             string `json:"type"`
	ErrorType        string `json:"error_type"`
	Message          string `json:"message"`
	Stack            string `json:"stack"`
	Fingerprint      string `json:"fingerprint"`
	URL              string `json:"url"`
	Component        string `json:"component"`
	AppVersion       string `json:"app_version"`
	Description      string `json:"description"`
	StepsToReproduce string `json:"steps_to_reproduce"`
}

func (s *Server) submitErrorReport(w http.ResponseWriter, r *http.Request) {
	if s.errorReportService == nil || !s.errorReportService.IsEnabled() {
		respondError(w, http.StatusServiceUnavailable, "error reporting is not available")
		return
	}

	// Apply body size limit before decoding
	r.Body = http.MaxBytesReader(w, r.Body, MaxErrorReportBodySize)

	var req errorReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == "http: request body too large" {
			respondError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Sanitize: trim whitespace
	req.Message = strings.TrimSpace(req.Message)
	req.Description = strings.TrimSpace(req.Description)
	req.StepsToReproduce = strings.TrimSpace(req.StepsToReproduce)
	req.ErrorType = strings.TrimSpace(req.ErrorType)
	req.URL = strings.TrimSpace(req.URL)
	req.Stack = strings.TrimSpace(req.Stack)

	// Validate type
	if req.Type != "automatic" && req.Type != "manual" {
		respondError(w, http.StatusBadRequest, "type must be 'automatic' or 'manual'")
		return
	}

	// Validate message
	if req.Type == "automatic" && len(req.Message) < 3 {
		respondError(w, http.StatusBadRequest, "message must be at least 3 characters")
		return
	}
	if req.Type == "manual" && len(req.Message) < 10 {
		respondError(w, http.StatusBadRequest, "message must be at least 10 characters")
		return
	}
	if len(req.Message) > 500 {
		respondError(w, http.StatusBadRequest, "message must be at most 500 characters")
		return
	}

	// Validate fingerprint
	if !fingerprintRegex.MatchString(req.Fingerprint) {
		respondError(w, http.StatusBadRequest, "fingerprint must be 16 lowercase hex characters")
		return
	}

	// Validate optional field lengths
	if len(req.Stack) > 4000 {
		respondError(w, http.StatusBadRequest, "stack must be at most 4000 characters")
		return
	}
	if len(req.Description) > 2000 {
		respondError(w, http.StatusBadRequest, "description must be at most 2000 characters")
		return
	}
	if len(req.StepsToReproduce) > 2000 {
		respondError(w, http.StatusBadRequest, "steps_to_reproduce must be at most 2000 characters")
		return
	}
	if len(req.ErrorType) > 50 {
		respondError(w, http.StatusBadRequest, "error_type must be at most 50 characters")
		return
	}
	if len(req.URL) > 500 {
		respondError(w, http.StatusBadRequest, "url must be at most 500 characters")
		return
	}
	if req.URL != "" && !strings.HasPrefix(req.URL, "/") {
		respondError(w, http.StatusBadRequest, "url must be a relative path starting with /")
		return
	}

	// Use UserAgent from HTTP header, not from client payload
	userAgent := r.Header.Get("User-Agent")

	report := service.ErrorReport{
		Type:             req.Type,
		ErrorType:        req.ErrorType,
		Message:          req.Message,
		Stack:            req.Stack,
		Fingerprint:      req.Fingerprint,
		URL:              req.URL,
		Component:        req.Component,
		UserAgent:        userAgent,
		AppVersion:       req.AppVersion,
		Description:      req.Description,
		StepsToReproduce: req.StepsToReproduce,
	}

	result, err := s.errorReportService.SubmitReport(r.Context(), report)
	if err != nil {
		s.logger().Error("error report submission failed", "error", err)
		// Return accepted: false but don't expose internal errors
		respondJSON(w, http.StatusOK, service.ErrorReportResult{Accepted: false})
		return
	}

	respondJSON(w, http.StatusOK, result)
}
