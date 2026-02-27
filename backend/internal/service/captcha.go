// Package service provides Cloudflare Turnstile CAPTCHA verification.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// TurnstileService handles Cloudflare Turnstile CAPTCHA verification.
type TurnstileService struct {
	secretKey  string
	siteKey    string
	enabled    bool
	httpClient *http.Client
	log        *slog.Logger
	// allowedHosts contains trusted hostnames extracted from CORS_ALLOWED_ORIGINS.
	// If empty, hostname enforcement is disabled (development/test convenience).
	allowedHosts map[string]struct{}
}

// TurnstileResponse represents the response from Cloudflare's Turnstile verification API.
type TurnstileResponse struct {
	Success     bool     `json:"success"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
	ChallengeTS string   `json:"challenge_ts,omitempty"`
	Hostname    string   `json:"hostname,omitempty"`
}

// NewTurnstileService creates a new TurnstileService.
// If secretKey or siteKey is empty, CAPTCHA verification is disabled.
func NewTurnstileService(secretKey, siteKey string, log *slog.Logger) *TurnstileService {
	enabled := secretKey != "" && siteKey != ""

	if log == nil {
		log = slog.Default()
	}

	if enabled {
		log.Info("Turnstile CAPTCHA enabled")
	} else {
		log.Info("Turnstile CAPTCHA disabled (missing TURNSTILE_SECRET_KEY or TURNSTILE_SITE_KEY)")
	}

	return &TurnstileService{
		secretKey: secretKey,
		siteKey:   siteKey,
		enabled:   enabled,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		log:          log,
		allowedHosts: make(map[string]struct{}),
	}
}

// IsEnabled returns true if CAPTCHA verification is enabled.
func (s *TurnstileService) IsEnabled() bool {
	return s.enabled
}

// GetSiteKey returns the public site key for frontend use.
func (s *TurnstileService) GetSiteKey() string {
	return s.siteKey
}

// SetAllowedOrigins configures trusted hostnames for Turnstile hostname validation.
// Origins should match CORS_ALLOWED_ORIGINS entries (e.g. https://app.example.com).
func (s *TurnstileService) SetAllowedOrigins(origins []string) {
	hosts := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}

		parsed, err := url.Parse(trimmed)
		if err != nil {
			continue
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			continue
		}
		host := normalizeCaptchaHost(parsed.Host)
		if host == "" {
			continue
		}
		hosts[host] = struct{}{}
	}

	s.allowedHosts = hosts
	if s.enabled && len(hosts) == 0 {
		s.log.Warn("turnstile hostname validation disabled (no trusted origins configured)")
	}
}

func hashRemoteIP(remoteIP string) string {
	if remoteIP == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(remoteIP))
	return hex.EncodeToString(sum[:8])
}

// Verify validates a Turnstile CAPTCHA token.
// If CAPTCHA is disabled, it always returns nil (success).
// If CAPTCHA is enabled and the token is empty, it returns an error.
func (s *TurnstileService) Verify(ctx context.Context, token, remoteIP string) error {
	// If CAPTCHA is disabled, always allow
	if !s.enabled {
		return nil
	}

	// Token is required when CAPTCHA is enabled
	if strings.TrimSpace(token) == "" {
		return errors.New("captcha token required")
	}

	// Prepare form data for Cloudflare API
	formData := url.Values{
		"secret":   {s.secretKey},
		"response": {token},
	}

	// Include remote IP if provided
	if remoteIP != "" {
		formData.Set("remoteip", remoteIP)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, turnstileVerifyURL, strings.NewReader(formData.Encode()))
	if err != nil {
		s.log.Error("failed to create turnstile verification request", "error", err)
		return fmt.Errorf("captcha verification failed: internal error")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.log.Error("failed to send turnstile verification request", "error", err)
		return fmt.Errorf("captcha verification failed: unable to connect to verification service")
	}
	defer resp.Body.Close()

	// Parse response
	var result TurnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		s.log.Error("failed to parse turnstile response", "error", err)
		return fmt.Errorf("captcha verification failed: invalid response from verification service")
	}

	// Check verification result
	if !result.Success {
		s.log.Warn("turnstile verification failed",
			"error_codes", result.ErrorCodes,
			"remote_ip_hash", hashRemoteIP(remoteIP),
		)

		// Return user-friendly error message based on error codes
		if len(result.ErrorCodes) > 0 {
			switch result.ErrorCodes[0] {
			case "missing-input-secret":
				return fmt.Errorf("captcha verification failed: server configuration error")
			case "invalid-input-secret":
				return fmt.Errorf("captcha verification failed: server configuration error")
			case "missing-input-response":
				return fmt.Errorf("captcha token required")
			case "invalid-input-response":
				return fmt.Errorf("captcha verification failed: invalid token")
			case "bad-request":
				return fmt.Errorf("captcha verification failed: malformed request")
			case "timeout-or-duplicate":
				return fmt.Errorf("captcha verification failed: token expired or already used")
			default:
				return fmt.Errorf("captcha verification failed")
			}
		}
		return fmt.Errorf("captcha verification failed")
	}

	if len(s.allowedHosts) > 0 {
		normalizedResultHost := normalizeCaptchaHost(result.Hostname)
		if _, ok := s.allowedHosts[normalizedResultHost]; !ok {
			s.log.Warn("turnstile verification failed: hostname not allowed",
				"received_hostname", normalizedResultHost,
				"remote_ip_hash", hashRemoteIP(remoteIP),
			)
			return fmt.Errorf("captcha verification failed")
		}
	}

	s.log.Debug("turnstile verification successful",
		"challenge_ts", result.ChallengeTS,
		"hostname", result.Hostname,
	)

	return nil
}

func normalizeCaptchaHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ""
	}

	// Accept host:port inputs from HTTP Host headers.
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		return parsedHost
	}
	if strings.HasPrefix(host, "[") && strings.Contains(host, "]") {
		return strings.Trim(host, "[]")
	}
	return host
}
