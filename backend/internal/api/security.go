package api

import "net/http"

// securityHeadersMiddleware adds security headers to all responses.
// This includes Content-Security-Policy, X-Frame-Options, X-Content-Type-Options,
// Referrer-Policy, and Permissions-Policy.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CSP: 'unsafe-inline' required for SvelteKit adapter-static inline scripts
		// and Svelte scoped styles. XSS mitigation: DOMPurify sanitizes all user HTML.
		// Accepted risk per SEC-004 security audit. Hash-based CSP would require
		// build-pipeline automation since inline script hashes change every build.
		// 'wasm-unsafe-eval' needed for libsodium WebAssembly (E2E encryption)
		// Cloudflare Turnstile requires: script-src, frame-src, connect-src for challenges.cloudflare.com
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval' https://challenges.cloudflare.com; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: blob:; " +
			"connect-src 'self' ws: wss: https://challenges.cloudflare.com; " +
			"frame-src https://challenges.cloudflare.com; " +
			"worker-src 'self' blob:; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'"

		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		// HSTS: Enforce HTTPS for 1 year, include subdomains, allow preload list
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		next.ServeHTTP(w, r)
	})
}
