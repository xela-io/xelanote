package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// TEMPORARY: Landing page only mode — set to true to disable all API endpoints
// except /health and /api/config. Set to false to re-enable the full API.
const landingPageOnly = true

// maintenanceMiddleware returns 503 for all requests when landing page only mode is active.
func maintenanceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if landingPageOnly {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"Service temporarily unavailable"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) setupRoutes() {
	r := s.router

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(s.panicRecoveryMiddleware)
	r.Use(middleware.RequestID)
	r.Use(requestIDLoggerMiddleware)
	r.Use(middleware.Compress(5)) // gzip level 5 - good compression/speed balance
	r.Use(s.corsMiddleware)
	r.Use(securityHeadersMiddleware)

	// API routes
	r.Route("/api", func(r chi.Router) {
		// These stay available even in landing page only mode
		r.Get("/config", s.getConfig)
		r.Get("/changelog", s.getChangelog)

		// Everything else is blocked in landing page only mode
		r.Group(func(r chi.Router) {
			r.Use(maintenanceMiddleware)
			s.registerPublicRoutes(r)
			s.registerProtectedRoutes(r)
		})
	})

	// Health check (public) — verifies DB connectivity and disk space
	r.Get("/health", s.handleHealth)

	// CAPTCHA page for iframe embedding (used by desktop apps)
	// Served with custom security headers to allow iframe embedding
	r.Get("/captcha", s.serveCaptchaPage)
}

func (s *Server) registerPublicRoutes(r chi.Router) {
	// NOTE: /config and /changelog are registered in setupRoutes() above the
	// maintenance middleware so they remain available in landing page only mode.

	// Rate-limited auth endpoints
	r.With(rateLimitMiddleware(s.registerLimiter)).Post("/auth/register", s.register)
	r.With(rateLimitMiddleware(s.loginLimiter)).Post("/auth/login", s.login)

	// SEC-006: Cookie-authenticated endpoints with CSRF protection
	// Bearer-only requests (desktop/CLI) skip CSRF automatically (csrfMiddleware logic)
	r.Group(func(r chi.Router) {
		r.Use(s.csrfMiddleware)
		r.With(rateLimitMiddleware(s.refreshLimiter)).Post("/auth/refresh", s.refresh)
		r.Post("/auth/logout", s.logout)
	})

	// FIDO2 public auth endpoints (with pending login token)
	r.With(rateLimitMiddleware(s.fido2BeginLimiter)).Post("/auth/fido2/begin", s.beginFIDO2Auth)
	r.With(rateLimitMiddleware(s.fido2FinishLimiter)).Post("/auth/fido2/finish", s.finishFIDO2Auth)

	// Public recovery endpoints (rate-limited)
	r.With(rateLimitMiddleware(s.recoveryLimiter)).Post("/auth/recovery/salt", s.getRecoveryKeySaltByEmail)
	r.With(rateLimitMiddleware(s.recoveryLimiter)).Post("/auth/recovery/verify", s.verifyRecoveryKey)
	r.With(rateLimitMiddleware(s.recoveryLimiter)).Get("/auth/recovery/encrypted-deks", s.getRecoveryWrappedDEKs)
	r.With(rateLimitMiddleware(s.recoveryLimiter)).Post("/auth/recovery/reset-password", s.resetPasswordWithRecoveryKey)
	r.With(rateLimitMiddleware(s.recoveryLimiter)).Post("/auth/recovery/reset-password-v2", s.resetPasswordWithRecoveryToken)

	// Error reporting (public, rate-limited)
	r.With(rateLimitMiddleware(s.errorReportLimiter)).Post("/error-reports", s.submitErrorReport)
}

func (s *Server) registerProtectedRoutes(r chi.Router) {
	// Standard API routes: auth + CSRF + request timeout.
	// middleware.Timeout cancels the request context after 60s and returns 504.
	// Handlers should check ctx.Done() to respect the deadline (e.g. LLM calls).
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Use(s.csrfMiddleware)
		r.Use(middleware.Timeout(60 * time.Second))

		// Auth endpoints
		r.Get("/auth/me", s.me)

		// 2FA endpoints (protected, requires authentication)
		// All mutating 2FA operations are rate-limited to prevent abuse
		r.Route("/2fa", func(r chi.Router) {
			r.With(rateLimitMiddleware(s.tfaVerifyLimiter)).Post("/setup", s.setupTwoFactor)
			r.With(rateLimitMiddleware(s.tfaVerifyLimiter)).Post("/verify", s.verifyTwoFactor)
			r.With(rateLimitMiddleware(s.tfaVerifyLimiter)).Delete("/", s.disableTwoFactor)
			r.Get("/status", s.getTwoFactorStatus)
			r.With(rateLimitMiddleware(s.backupCodeLimiter)).Post("/backup-codes/regenerate", s.regenerateBackupCodes)

			// FIDO2 credential management (protected)
			r.Route("/fido2", func(r chi.Router) {
				r.Post("/register/begin", s.beginFIDO2Registration)
				r.Post("/register/finish", s.finishFIDO2Registration)
				r.Get("/credentials", s.listFIDO2Credentials)
				r.Delete("/credentials/{id}", s.deleteFIDO2Credential)
			})
		})

		// Upload endpoints (SEC-002: Both upload and serving require authentication)
		r.With(rateLimitMiddleware(s.uploadLimiter)).Post("/uploads", s.uploadImage)
		r.With(rateLimitMiddleware(s.uploadLimiter)).Post("/uploads/encrypted", s.uploadEncryptedBlob)
		r.Get("/uploads/{user_id}/{filename}", s.serveUpload) // Authenticated serving with ownership verification

		// Import endpoints (rate-limited to prevent CPU/memory exhaustion)
		r.With(rateLimitMiddleware(s.importLimiter)).Post("/import/markdown", s.importMarkdown)

		s.registerProtectedResourceRoutes(r)
		s.registerProtectedUtilityRoutes(r)
	})

	// Long-lived connections: auth required but NO timeout middleware.
	// middleware.Timeout buffers responses, which breaks SSE streaming and WebSocket upgrades.
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.With(rateLimitMiddleware(s.llmLimiter)).Get("/notes/{id}/summarize/stream", s.summarizeNoteStream)
		r.Get("/ws", s.handleWebSocket)
	})
}
