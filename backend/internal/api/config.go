package api

import (
	"fmt"
	"net/http"
)

// Version is set at build time via ldflags:
// -X github.com/xela-io/xelanote/internal/api.Version=...
var Version = "dev"

// ConfigResponse represents the public configuration settings.
type ConfigResponse struct {
	CaptchaEnabled        bool   `json:"captcha_enabled"`
	CaptchaSiteKey        string `json:"captcha_site_key,omitempty"`
	CaptchaIframeURL      string `json:"captcha_iframe_url,omitempty"`
	Version               string `json:"version"`
	ErrorReportingEnabled bool   `json:"error_reporting_enabled"`
}

// getConfig returns the public configuration settings.
// This endpoint is public (no authentication required).
func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	errorReportingEnabled := false
	if s.errorReportService != nil {
		errorReportingEnabled = s.errorReportService.IsEnabled()
	}

	config := ConfigResponse{
		CaptchaEnabled:        s.turnstileService.IsEnabled(),
		Version:               Version,
		ErrorReportingEnabled: errorReportingEnabled,
	}

	// Only include site key and iframe URL if CAPTCHA is enabled
	if config.CaptchaEnabled {
		siteKey := s.turnstileService.GetSiteKey()
		config.CaptchaSiteKey = siteKey
		config.CaptchaIframeURL = fmt.Sprintf("/captcha?sitekey=%s", siteKey)
	}

	respondJSON(w, http.StatusOK, config)
}
