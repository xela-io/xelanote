package api

import "net/http"

// serveCaptchaPage serves the static CAPTCHA HTML page for iframe embedding.
// This endpoint has custom security headers that differ from the global middleware:
// - frame-ancestors is set to allow embedding (instead of 'none')
// - X-Frame-Options is omitted (to allow iframe embedding from desktop apps)
// The page only contains the Turnstile widget and no sensitive data.
func (s *Server) serveCaptchaPage(w http.ResponseWriter, r *http.Request) {
	// Custom CSP for the CAPTCHA page: allows Cloudflare Turnstile and iframe embedding
	csp := "default-src 'none'; " +
		"script-src https://challenges.cloudflare.com 'unsafe-inline'; " +
		"style-src 'unsafe-inline'; " +
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
