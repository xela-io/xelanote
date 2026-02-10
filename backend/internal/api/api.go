// Package api provides HTTP handlers for the xelanote API.
package api

import (
	"embed"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/xela-io/xelanote/internal/jobs"
	"github.com/xela-io/xelanote/internal/service"
	"github.com/xela-io/xelanote/internal/websocket"
)

//go:embed static/captcha.html
var captchaHTML embed.FS

// isTestEnv returns true if running in test environment
func isTestEnv() bool {
	env := os.Getenv("XELANOTE_ENV")
	return env == "test" || env == "testing"
}

// Server holds the HTTP server dependencies.
type Server struct {
	noteService      *service.NoteService
	authService      *service.AuthService
	tfaService       *service.TwoFactorService
	graphService     *service.GraphService
	templateService  *service.TemplateService
	snippetService   *service.SnippetService
	userService      *service.UserService
	adminService     *service.AdminService
	activityService  *service.ActivityService
	settingsService  *service.SettingsService
	turnstileService *service.TurnstileService
	summarizeService *service.SummarizeService
	jobManager       *jobs.JobManager
	wsManager        *websocket.Manager
	log              *slog.Logger
	jwtSecret        []byte
	dataDir          string
	allowedOrigins   []string
	router           chi.Router
	// Rate limiters for auth endpoints
	registerLimiter   *RateLimiter
	loginLimiter      *RateLimiter
	refreshLimiter    *RateLimiter
	tfaVerifyLimiter  *RateLimiter
	backupCodeLimiter *RateLimiter
	recoveryLimiter   *RateLimiter
	// Rate limiters for resource-intensive endpoints
	uploadLimiter         *RateLimiter
	importLimiter         *RateLimiter
	searchLimiter         *RateLimiter
	passwordChangeLimiter *RateLimiter
	emailChangeLimiter    *RateLimiter
	recoveryKeyLimiter    *RateLimiter
	llmLimiter            *RateLimiter // Shared limiter for all LLM endpoints (summarize, tag suggestions, link suggestions, spell-check)
	fido2BeginLimiter     *RateLimiter
	fido2FinishLimiter    *RateLimiter
	// Account lockout for brute-force protection
	accountLockout *AccountLockout
	// Stream content store for SSE summarization
	streamContent *streamContentStore
	// FIDO2 service
	fido2Service *service.FIDO2Service
	// Sharing service
	sharingService    *service.SharingService
	shareLimiter      *RateLimiter
	userSearchLimiter *RateLimiter
	// Error reporting service
	errorReportService *service.ErrorReportService
	errorReportLimiter *RateLimiter
	// Recipe service
	recipeService           *service.RecipeService
	recipeSuggestionService *service.RecipeSuggestionService
}

// NewServer creates a new API server.
func NewServer(noteService *service.NoteService, authService *service.AuthService, tfaService *service.TwoFactorService, fido2Service *service.FIDO2Service, graphService *service.GraphService, templateService *service.TemplateService, snippetService *service.SnippetService, userService *service.UserService, adminService *service.AdminService, activityService *service.ActivityService, settingsService *service.SettingsService, turnstileService *service.TurnstileService, summarizeService *service.SummarizeService, sharingService *service.SharingService, errorReportService *service.ErrorReportService, jobManager *jobs.JobManager, wsManager *websocket.Manager, logger *slog.Logger, jwtSecret []byte, dataDir string, allowedOrigins []string) *Server {
	// Validate CORS configuration in production (SEC-003)
	env := os.Getenv("XELANOTE_ENV")
	if env == "production" && len(allowedOrigins) == 0 {
		logger.Error("FATAL: CORS_ALLOWED_ORIGINS must be set in production mode",
			slog.String("env", env),
			slog.Int("allowed_origins_count", len(allowedOrigins)))
		// Use log.Fatal instead of os.Exit for proper cleanup and log flushing
		log.Fatal("CORS_ALLOWED_ORIGINS must be set in production mode")
	}

	// Use higher rate limits in test environment to allow E2E tests to run
	registerLimit := 5
	loginLimit := 10
	refreshLimit := 30
	tfaLimit := 5
	backupLimit := 3
	recoveryLimit := 3
	// Resource-intensive endpoint limits
	uploadLimit := 20
	importLimit := 10
	passwordChangeLimit := 3
	emailChangeLimit := 3
	recoveryKeyLimit := 3
	fido2Limit := 10
	searchLimit := 120    // 120 requests per minute for search
	llmLimit := 10        // 10 requests per minute for all LLM endpoints (shared)
	shareLimit := 20      // 20 shares per minute
	userSearchLimit := 30 // 30 user searches per minute
	errorReportLimit := 5 // 5 error reports per hour
	if isTestEnv() {
		registerLimit = 1000
		loginLimit = 1000
		refreshLimit = 1000
		tfaLimit = 1000
		backupLimit = 1000
		recoveryLimit = 1000
		uploadLimit = 1000
		importLimit = 1000
		passwordChangeLimit = 1000
		emailChangeLimit = 1000
		recoveryKeyLimit = 1000
		fido2Limit = 1000
		searchLimit = 10000 // Effectively disabled for tests
		llmLimit = 10000    // Effectively disabled for tests
		shareLimit = 10000
		userSearchLimit = 10000
		errorReportLimit = 10000
	}

	s := &Server{
		noteService:      noteService,
		authService:      authService,
		tfaService:       tfaService,
		graphService:     graphService,
		templateService:  templateService,
		snippetService:   snippetService,
		userService:      userService,
		adminService:     adminService,
		activityService:  activityService,
		settingsService:  settingsService,
		turnstileService: turnstileService,
		summarizeService: summarizeService,
		jobManager:       jobManager,
		wsManager:        wsManager,
		log:              logger,
		jwtSecret:        jwtSecret,
		dataDir:          dataDir,
		allowedOrigins:   allowedOrigins,
		router:           chi.NewRouter(),
		// Rate limiters: register=5/hour, login=10/15min, refresh=30/hour, 2fa=5/15min, backup=3/15min, recovery=3/15min
		// In test mode, all limits are increased to 1000/hour
		registerLimiter:   NewRateLimiter(registerLimit, time.Hour, registerLimit),
		loginLimiter:      NewRateLimiter(loginLimit, 15*time.Minute, loginLimit),
		refreshLimiter:    NewRateLimiter(refreshLimit, time.Hour, refreshLimit),
		tfaVerifyLimiter:  NewRateLimiter(tfaLimit, 15*time.Minute, tfaLimit),
		backupCodeLimiter: NewRateLimiter(backupLimit, 15*time.Minute, backupLimit),
		recoveryLimiter:   NewRateLimiter(recoveryLimit, 15*time.Minute, recoveryLimit),
		// Resource-intensive endpoint limiters: uploads=20/hour, import=10/hour, search=120/min, summarize=10/min
		// Sensitive user operations: password/email/recovery-key changes=3/hour
		uploadLimiter:         NewRateLimiter(uploadLimit, time.Hour, uploadLimit),
		importLimiter:         NewRateLimiter(importLimit, time.Hour, importLimit),
		searchLimiter:         NewRateLimiter(searchLimit, time.Minute, 30), // 120/min, burst 30
		passwordChangeLimiter: NewRateLimiter(passwordChangeLimit, time.Hour, passwordChangeLimit),
		emailChangeLimiter:    NewRateLimiter(emailChangeLimit, time.Hour, emailChangeLimit),
		recoveryKeyLimiter:    NewRateLimiter(recoveryKeyLimit, time.Hour, recoveryKeyLimit),
		llmLimiter:            NewRateLimiter(llmLimit, time.Minute, llmLimit), // 10/min shared for all LLM endpoints
		fido2BeginLimiter:     NewRateLimiter(fido2Limit, 15*time.Minute, fido2Limit),
		fido2FinishLimiter:    NewRateLimiter(fido2Limit, 15*time.Minute, fido2Limit),
		fido2Service:          fido2Service,
		sharingService:        sharingService,
		shareLimiter:          NewRateLimiter(shareLimit, time.Minute, shareLimit),
		userSearchLimiter:     NewRateLimiter(userSearchLimit, time.Minute, 10), // 30/min, burst 10
		errorReportService:    errorReportService,
		errorReportLimiter:    NewRateLimiter(errorReportLimit, time.Hour, 3), // 5/hour, burst 3
		streamContent:         newStreamContentStore(),
		// Account lockout: 10 global attempts (5 per-IP), 30s initial lockout (doubles each time), 30min max
		// In test mode, use 1000 attempts to effectively disable lockout
		accountLockout: NewAccountLockout(func() int {
			if isTestEnv() {
				return 1000
			}
			return 10
		}(), 30*time.Second, 30*time.Minute, logger),
	}
	s.setupRoutes()
	return s
}

// SetRecipeService sets the recipe service (called after server creation).
func (s *Server) SetRecipeService(rs *service.RecipeService) {
	s.recipeService = rs
}

// SetRecipeSuggestionService sets the recipe suggestion service (called after server creation).
func (s *Server) SetRecipeSuggestionService(rss *service.RecipeSuggestionService) {
	s.recipeSuggestionService = rss
}

// logger returns the server logger (returns no-op logger if nil for tests)
func (s *Server) logger() *slog.Logger {
	if s.log == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return s.log
}

// Router returns the HTTP router.
func (s *Server) Router() chi.Router {
	return s.router
}

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
		// Public routes (no authentication required)
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

			// Notes endpoints
			r.Route("/notes", func(r chi.Router) {
				r.Get("/", s.listNotes)
				r.Get("/titles", s.listNoteTitles)                     // Lightweight endpoint for link suggestions
				r.Get("/titles/ai-enabled", s.listNoteTitlesAIEnabled) // Only AI-enabled notes for Claude link suggestions
				r.Post("/", s.createNote)
				r.Post("/reorder", s.reorderNotes)
				r.Get("/{id}", s.getNote)
				r.Put("/{id}", s.updateNote)
				r.Delete("/{id}", s.deleteNote)
				r.Post("/{id}/rename", s.renameNote)
				r.Get("/{id}/backlinks", s.getBacklinks)
				r.Put("/{id}/color", s.updateNoteColor)

				// AI-enabled (Claude API opt-in) endpoints
				r.Get("/{id}/ai-enabled", s.getNoteAIEnabled)
				r.Put("/{id}/ai-enabled", s.updateNoteAIEnabled)

				// LLM endpoints (rate-limited with shared limiter)
				r.With(rateLimitMiddleware(s.llmLimiter)).Post("/{id}/summarize", s.summarizeNote)
				r.With(rateLimitMiddleware(s.llmLimiter)).Get("/{id}/summarize/stream", s.summarizeNoteStream)
				r.With(rateLimitMiddleware(s.llmLimiter)).Post("/{id}/summarize/prepare", s.prepareSummarizeStream)
				r.With(rateLimitMiddleware(s.llmLimiter)).Post("/{id}/suggest-tags", s.suggestTags)
				r.With(rateLimitMiddleware(s.llmLimiter)).Post("/{id}/suggest-links", s.suggestLinks)
				r.With(rateLimitMiddleware(s.llmLimiter)).Post("/{id}/format-markdown", s.formatMarkdown)
				r.With(rateLimitMiddleware(s.llmLimiter)).Post("/{id}/ai-transform", s.aiTransform)

				// Encryption toggle
				r.Post("/{id}/decrypt", s.decryptNote)

				// Batch DEK re-encryption (for password changes)
				r.Post("/batch-reencrypt-deks", s.batchReencryptDEKs)

				// Tag endpoints for notes
				r.Get("/{id}/tags", s.getNoteTags)
				r.Put("/{id}/tags", s.setNoteTags)

				// Version history endpoints
				r.Get("/{id}/versions", s.listVersions)
				r.Get("/{id}/versions/compare", s.compareVersions)
				r.Get("/{id}/versions/{version}", s.getVersion)
				r.Post("/{id}/versions/{version}/restore", s.restoreVersion)

				// Sharing endpoints (per-note)
				r.With(rateLimitMiddleware(s.shareLimiter)).Post("/{id}/shares", s.shareNote)
				r.Get("/{id}/shares", s.getNoteShares)
				r.Put("/{id}/shares/{userId}", s.updateShareRole)
				r.Delete("/{id}/shares/{userId}", s.removeShare)

				// Task event logging
				r.Post("/{id}/task-events", s.recordTaskEvent)
			})

			// Shared notes endpoints (notes shared WITH current user)
			r.Get("/shared", s.getSharedNotes)
			r.Get("/shared/folders", s.getSharedFoldersHandler)
			r.Get("/shared/folders/{id}/notes", s.getSharedFolderNotesHandler)
			r.Get("/shared/recipes", s.listSharedRecipes)
			r.Get("/shared/collections", s.listSharedCollections)
			r.Get("/shared/collections/{id}/items", s.listSharedCollectionItems)
			r.Post("/shared/collections/{id}/items", s.addToSharedCollection)
			r.Delete("/shared/collections/{id}/items/{noteId}", s.removeFromSharedCollection)
			r.Get("/shared/{id}", s.getSharedNote)
			r.Put("/shared/{id}", s.updateSharedNote)
			r.Post("/shared/{id}/placement", s.placeSharedNoteHandler)
			r.Delete("/shared/{id}/placement", s.removePlacementHandler)

			// User search for share dialog
			r.With(rateLimitMiddleware(s.userSearchLimiter)).Get("/users/search", s.searchUsers)

			// Folders endpoints
			r.Route("/folders", func(r chi.Router) {
				r.Get("/", s.getAllFolders)
				r.Post("/", s.createFolder)
				r.Post("/reorder", s.reorderFolders)
				r.Put("/{id}/move", s.moveFolder)
				r.Put("/{id}/rename", s.renameFolder)
				r.Put("/{id}/color", s.updateFolderColor)
				r.Delete("/{id}", s.deleteFolder)

				// AI-enabled default (Claude API opt-in) endpoints
				r.Get("/{id}/ai-enabled", s.getFolderAIEnabledDefault)
				r.Put("/{id}/ai-enabled", s.updateFolderAIEnabledDefault)

				// Encryption default endpoints
				r.Get("/{id}/encryption-default", s.getFolderEncryptionDefault)
				r.Put("/{id}/encryption-default", s.updateFolderEncryptionDefault)

				// Folder sharing endpoints
				r.With(rateLimitMiddleware(s.shareLimiter)).Post("/{id}/shares", s.shareFolderHandler)
				r.Get("/{id}/shares", s.getFolderSharesHandler)
				r.Put("/{id}/shares/{userId}", s.updateFolderShareRoleHandler)
				r.Delete("/{id}/shares/{userId}", s.removeFolderShareHandler)
			})

			// Tags endpoints
			r.Route("/tags", func(r chi.Router) {
				r.Get("/", s.getAllTags)
				r.Delete("/{tagId}", s.deleteTag)
			})

			// Templates endpoints
			r.Route("/templates", func(r chi.Router) {
				r.Get("/", s.getAllTemplates)
				r.Get("/{id}", s.getTemplate)
				r.Post("/", s.createTemplate)
				r.Put("/{id}", s.updateTemplate)
				r.Delete("/{id}", s.deleteTemplate)
			})

			// Snippets endpoints
			r.Route("/snippets", func(r chi.Router) {
				r.Get("/", s.getAllSnippets)
				r.Get("/{id}", s.getSnippet)
				r.Post("/", s.createSnippet)
				r.Put("/{id}", s.updateSnippet)
				r.Delete("/{id}", s.deleteSnippet)
			})

			// Features endpoints (user-specific feature toggles)
			r.Route("/features", func(r chi.Router) {
				r.Get("/", s.listFeatures)
				r.Get("/{feature}", s.getFeature)
				r.Put("/{feature}", s.setFeature)
			})

			// Journal endpoints (lookup only, creation via POST /notes)
			r.Route("/journal", func(r chi.Router) {
				r.Get("/", s.getJournalLookup)
				r.Get("/calendar", s.getJournalCalendar)
				r.Get("/calendar/year", s.getJournalCalendarYear)
				r.Get("/entries", s.listJournalEntries)
			})

			// Recipe endpoints
			r.Route("/recipes", func(r chi.Router) {
				r.Get("/", s.listRecipes)
				r.Get("/{id}", s.getRecipeDetail)
				r.Put("/{id}/metadata", s.updateRecipeMetadata)
				r.Put("/{id}/ingredients", s.setRecipeIngredients)
				r.Get("/{id}/scaled", s.getScaledIngredients)

				// Recipe images
				r.Post("/{id}/images", s.addRecipeImage)
				r.Put("/{id}/images/order", s.reorderRecipeImages)
				r.Put("/{id}/images/{imageId}", s.updateRecipeImageCaption)
				r.Delete("/{id}/images/{imageId}", s.deleteRecipeImage)

				// Collections (owner-only)
				r.Get("/collections", s.listRecipeCollections)
				r.Post("/collections", s.createRecipeCollection)
				r.Put("/collections/{id}", s.updateRecipeCollection)
				r.Delete("/collections/{id}", s.deleteRecipeCollection)
				r.Post("/collections/{id}/items", s.addRecipeToCollection)
				r.Delete("/collections/{id}/items/{noteId}", s.removeRecipeFromCollection)
				r.Get("/collections/{id}/items", s.listCollectionItems)

				// Collection sharing (owner-only management)
				r.With(rateLimitMiddleware(s.shareLimiter)).Post("/collections/{id}/shares", s.shareCollection)
				r.Get("/collections/{id}/shares", s.getCollectionShares)
				r.Put("/collections/{id}/shares/{userId}", s.updateCollectionShareRole)
				r.Delete("/collections/{id}/shares/{userId}", s.removeCollectionShare)

				// AI-powered recipe suggestions (rate-limited)
				r.Route("/suggestions", func(r chi.Router) {
					r.With(rateLimitMiddleware(s.llmLimiter)).Post("/similar", s.findSimilarRecipes)
					r.With(rateLimitMiddleware(s.llmLimiter)).Post("/by-ingredients", s.suggestByIngredients)
					r.With(rateLimitMiddleware(s.llmLimiter)).Post("/save-generated", s.saveGeneratedRecipe)
					r.With(rateLimitMiddleware(s.llmLimiter)).Post("/extract-ingredients", s.extractIngredientsFromPhoto)
				})
			})

			// User endpoints (preferences, email, password)
			r.Route("/users", func(r chi.Router) {
				r.Get("/preferences", s.getPreferences)
				r.Put("/preferences", s.updatePreferences)
				r.Put("/preferences/encryption", s.updateEncryptionPreferences)
				r.Put("/preferences/security", s.updateSecurityPreferences)
				// Sensitive operations are rate-limited to prevent abuse
				r.With(rateLimitMiddleware(s.emailChangeLimiter)).Put("/email", s.changeEmail)
				r.With(rateLimitMiddleware(s.passwordChangeLimiter)).Put("/password", s.changePassword)

				// Recovery key endpoints (authenticated, rate-limited)
				r.With(rateLimitMiddleware(s.recoveryKeyLimiter)).Post("/recovery-key", s.setRecoveryKey)
				r.Get("/recovery-key/salt", s.getRecoveryKeySalt)

				// WebAuthn credential endpoints
				r.Post("/webauthn/credentials", s.addWebAuthnCredential)
				r.Delete("/webauthn/credentials", s.deleteWebAuthnCredential)
				r.Patch("/webauthn/credentials/touch", s.touchWebAuthnCredential)

				// Claude API Key endpoints (BYOK - Bring Your Own Key)
				r.Put("/api-key", s.setClaudeAPIKey)
				r.Delete("/api-key", s.deleteClaudeAPIKey)
				r.Get("/api-key/status", s.getClaudeAPIKeyStatus)

				// Gemini API Key endpoints (BYOK - Bring Your Own Key)
				r.Put("/gemini-api-key", s.setGeminiAPIKey)
				r.Delete("/gemini-api-key", s.deleteGeminiAPIKey)
				r.Get("/gemini-api-key/status", s.getGeminiAPIKeyStatus)
			})

			// Search and export endpoints (with rate limiting to prevent DoS)
			r.With(rateLimitMiddleware(s.searchLimiter)).Get("/search", s.search)
			r.With(rateLimitMiddleware(s.searchLimiter)).Get("/quick-search", s.quickSearch)
			r.Get("/folders-legacy", s.getFolders) // Keep old endpoint for compatibility
			r.Get("/export/markdown", s.exportMarkdown)

			// Due dates overview endpoint
			r.Get("/due-dates", s.getDueDates)

			// Trash endpoints
			r.Get("/trash", s.listTrash)
			r.Get("/trash/count", s.getTrashCount)
			r.Delete("/trash", s.emptyTrash)
			r.Post("/notes/{id}/restore", s.restoreNote)
			r.Delete("/notes/{id}/permanent", s.permanentlyDeleteNote)

			// LLM endpoints (not note-specific)
			r.Route("/llm", func(r chi.Router) {
				r.With(rateLimitMiddleware(s.llmLimiter)).Post("/spell-check", s.spellCheck)
			})

			// Jobs endpoints
			r.Get("/jobs/{id}", s.getJobStatus)

			// Graph endpoints
			r.Get("/graph", s.getGlobalGraph)

			// WebSocket endpoint (token from query param, not header)
			r.Get("/ws", s.handleWebSocket)

			// Admin routes (requires admin role)
			r.Route("/admin", func(r chi.Router) {
				r.Use(s.adminMiddleware)

				// Stats
				r.Get("/stats", s.getAdminStats)
				r.Get("/stats/detailed", s.getDetailedStats)

				// Users
				r.Get("/users", s.listAllUsers)
				r.Get("/users/{id}", s.getUserDetails)
				r.Put("/users/{id}/admin", s.toggleUserAdmin)
				r.Delete("/users/{id}", s.deleteUserAdmin)

				// Activity logs
				r.Get("/activity", s.getActivityLogs)

				// Settings
				r.Get("/settings", s.getSettings)
				r.Put("/settings", s.updateSettings)
			})
		})
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

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowAll := len(s.allowedOrigins) == 0

		// SEC-003: Never use wildcard (*) with credentials - echo origin instead
		if allowAll {
			if origin != "" {
				// Development mode: echo the requesting origin
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				// Log warning in permissive mode
				if !isTestEnv() {
					s.logger().Warn("CORS in permissive mode - echoing origin",
						slog.String("origin", origin),
						slog.String("advice", "Set CORS_ALLOWED_ORIGINS in production"))
				}
			}
		} else if origin != "" && originAllowed(origin, s.allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		} else if r.Method == "OPTIONS" && origin != "" {
			respondError(w, http.StatusForbidden, "origin not allowed")
			return
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		// Added Authorization header for JWT authentication, Cookie for cookie-based auth, and X-CSRF-Token for CSRF protection
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, If-Match, Authorization, Cookie, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers", "ETag")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func originAllowed(origin string, allowed []string) bool {
	for _, entry := range allowed {
		if entry == origin {
			return true
		}
	}
	return false
}

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
		json.NewEncoder(w).Encode(data)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// MaxJSONBodySize limits the maximum size of JSON request bodies (1MB)
const MaxJSONBodySize int64 = 1 << 20

// MaxLargeJSONBodySize is the limit for endpoints that handle large payloads (16MB)
const MaxLargeJSONBodySize int64 = 16 << 20

func decodeJSON(w http.ResponseWriter, r *http.Request, v interface{}) error {
	// Limit request body size to prevent memory exhaustion attacks
	r.Body = http.MaxBytesReader(w, r.Body, MaxJSONBodySize)
	return json.NewDecoder(r.Body).Decode(v)
}

func decodeJSONWithLimit(w http.ResponseWriter, r *http.Request, v interface{}, maxBytes int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	return json.NewDecoder(r.Body).Decode(v)
}
