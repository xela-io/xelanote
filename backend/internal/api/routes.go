package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (s *Server) setupRoutes() {
	r := s.router

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Compress(5)) // gzip level 5 - good compression/speed balance
	r.Use(s.corsMiddleware)
	r.Use(securityHeadersMiddleware)

	// API routes
	r.Route("/api", func(r chi.Router) {
		s.registerPublicRoutes(r)
		s.registerProtectedRoutes(r)
	})

	// Health check (public)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// CAPTCHA page for iframe embedding (used by desktop apps)
	// Served with custom security headers to allow iframe embedding
	r.Get("/captcha", s.serveCaptchaPage)
}

func (s *Server) registerPublicRoutes(r chi.Router) {
	// Public config endpoint (for CAPTCHA settings)
	r.Get("/config", s.getConfig)
	r.Get("/changelog", s.getChangelog)

	// Rate-limited auth endpoints
	r.With(rateLimitMiddleware(s.registerLimiter)).Post("/auth/register", s.register)
	r.With(rateLimitMiddleware(s.loginLimiter)).Post("/auth/login", s.login)
	r.With(rateLimitMiddleware(s.refreshLimiter)).Post("/auth/refresh", s.refresh)
	r.Post("/auth/logout", s.logout)

	// FIDO2 public auth endpoints (with pending login token)
	r.With(rateLimitMiddleware(s.fido2BeginLimiter)).Post("/auth/fido2/begin", s.beginFIDO2Auth)
	r.With(rateLimitMiddleware(s.fido2FinishLimiter)).Post("/auth/fido2/finish", s.finishFIDO2Auth)

	// Public recovery endpoints (rate-limited)
	r.With(rateLimitMiddleware(s.recoveryLimiter)).Post("/auth/recovery/salt", s.getRecoveryKeySaltByEmail)
	r.With(rateLimitMiddleware(s.recoveryLimiter)).Post("/auth/recovery/reset-password", s.resetPasswordWithRecoveryKey)

	// Error reporting (public, rate-limited)
	r.With(rateLimitMiddleware(s.errorReportLimiter)).Post("/error-reports", s.submitErrorReport)
}

func (s *Server) registerProtectedRoutes(r chi.Router) {
	// Protected routes (authentication required)
	r.Group(func(r chi.Router) {
		// Apply auth middleware to all routes in this group
		r.Use(s.authMiddleware)
		// Apply CSRF protection to state-changing requests
		r.Use(s.csrfMiddleware)

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
		r.Get("/uploads/{user_id}/{filename}", s.serveUpload) // Authenticated serving with ownership verification

		// Import endpoints (rate-limited to prevent CPU/memory exhaustion)
		r.With(rateLimitMiddleware(s.importLimiter)).Post("/import/markdown", s.importMarkdown)

		s.registerProtectedResourceRoutes(r)
		s.registerProtectedUtilityRoutes(r)
	})
}
