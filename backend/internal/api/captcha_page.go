package api

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
)

// serveCaptchaPage serves the static CAPTCHA HTML page for iframe embedding.
// This endpoint has custom security headers that differ from the global middleware:
// - frame-ancestors is set to allow embedding (instead of 'none')
// - X-Frame-Options is omitted (to allow iframe embedding from desktop apps)
// The page only contains the Turnstile widget and no sensitive data.
func (s *Server) serveCaptchaPage(w http.ResponseWriter, r *http.Request) {
	nonce, err := generateCSPNonce()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Custom CSP for the CAPTCHA page: allows Cloudflare Turnstile and iframe embedding
	csp := "default-src 'none'; " +
		"script-src 'nonce-" + nonce + "' https://challenges.cloudflare.com; " +
		"style-src 'nonce-" + nonce + "'; " +
		"frame-src https://challenges.cloudflare.com; " +
		"connect-src https://challenges.cloudflare.com; " +
		"frame-ancestors *"

	// Override the global security headers for this endpoint
	w.Header().Set("Content-Security-Policy", csp)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	// Explicitly remove X-Frame-Options to allow iframe embedding
	w.Header().Del("X-Frame-Options")

	// Read and serve the embedded HTML file
	data, err := captchaHTML.ReadFile("static/captcha.html")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	html := strings.ReplaceAll(string(data), "__NONCE__", nonce)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

func generateCSPNonce() (string, error) {
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(nonceBytes), nil
}
