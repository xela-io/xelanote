package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/xela-io/xelanote/internal/service"
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

		s.registerNotesRoutes(r)
		s.registerSharedRoutes(r)
		s.registerFolderRoutes(r)
		s.registerTagsRoutes(r)
		s.registerTemplatesRoutes(r)
		s.registerSnippetsRoutes(r)
		s.registerFeaturesRoutes(r)
		s.registerJournalRoutes(r)
		s.registerRecipeRoutes(r)
		s.registerUserRoutes(r)
		s.registerSearchExportRoutes(r)
		s.registerTrashRoutes(r)
		s.registerLLMRoutes(r)
		s.registerJobRoutes(r)
		s.registerGraphRoutes(r)
		s.registerWebsocketRoutes(r)
		s.registerAdminRoutes(r)
	})
}

func (s *Server) registerNotesRoutes(r chi.Router) {
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
}

func (s *Server) registerSharedRoutes(r chi.Router) {
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
}

func (s *Server) registerFolderRoutes(r chi.Router) {
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
}

func (s *Server) registerTagsRoutes(r chi.Router) {
	r.Route("/tags", func(r chi.Router) {
		r.Get("/", s.getAllTags)
		r.Delete("/{tagId}", s.deleteTag)
	})
}

func (s *Server) registerTemplatesRoutes(r chi.Router) {
	r.Route("/templates", func(r chi.Router) {
		r.Get("/", s.getAllTemplates)
		r.Get("/{id}", s.getTemplate)
		r.Post("/", s.createTemplate)
		r.Put("/{id}", s.updateTemplate)
		r.Delete("/{id}", s.deleteTemplate)
	})
}

func (s *Server) registerSnippetsRoutes(r chi.Router) {
	r.Route("/snippets", func(r chi.Router) {
		r.Get("/", s.getAllSnippets)
		r.Get("/{id}", s.getSnippet)
		r.Post("/", s.createSnippet)
		r.Put("/{id}", s.updateSnippet)
		r.Delete("/{id}", s.deleteSnippet)
	})
}

func (s *Server) registerFeaturesRoutes(r chi.Router) {
	r.Route("/features", func(r chi.Router) {
		r.Get("/", s.listFeatures)
		r.Get("/{feature}", s.getFeature)
		r.Put("/{feature}", s.setFeature)
	})
}

func (s *Server) registerJournalRoutes(r chi.Router) {
	// Journal endpoints (lookup only, creation via POST /notes)
	r.Route("/journal", func(r chi.Router) {
		r.Get("/", s.getJournalLookup)
		r.Get("/calendar", s.getJournalCalendar)
		r.Get("/calendar/year", s.getJournalCalendarYear)
		r.Get("/entries", s.listJournalEntries)
	})
}

func (s *Server) registerRecipeRoutes(r chi.Router) {
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
}

func (s *Server) registerUserRoutes(r chi.Router) {
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

		// LLM API Key endpoints (BYOK - Bring Your Own Key)
		claudeKey := apiKeyProvider{
			name:            "claude",
			setKey:          s.userService.SetClaudeAPIKey,
			deleteKey:       s.userService.DeleteClaudeAPIKey,
			getKeyStatus:    func(uid int) (any, error) { return s.userService.GetClaudeAPIKeyStatus(uid) },
			invalidateCache: s.summarizeService.InvalidateClaudeClient,
			validationErr:   service.ErrInvalidClaudeAPIKey,
			invalidKeyMsg:   "invalid Claude API key format (must start with sk-ant-)",
		}
		r.Put("/api-key", s.handleSetAPIKey(claudeKey))
		r.Delete("/api-key", s.handleDeleteAPIKey(claudeKey))
		r.Get("/api-key/status", s.handleGetAPIKeyStatus(claudeKey))

		geminiKey := apiKeyProvider{
			name:            "gemini",
			setKey:          s.userService.SetGeminiAPIKey,
			deleteKey:       s.userService.DeleteGeminiAPIKey,
			getKeyStatus:    func(uid int) (any, error) { return s.userService.GetGeminiAPIKeyStatus(uid) },
			invalidateCache: s.summarizeService.InvalidateGeminiClient,
			validationErr:   service.ErrInvalidGeminiAPIKey,
			invalidKeyMsg:   "invalid Gemini API key format (must start with AIza)",
		}
		r.Put("/gemini-api-key", s.handleSetAPIKey(geminiKey))
		r.Delete("/gemini-api-key", s.handleDeleteAPIKey(geminiKey))
		r.Get("/gemini-api-key/status", s.handleGetAPIKeyStatus(geminiKey))
	})
}

func (s *Server) registerSearchExportRoutes(r chi.Router) {
	// Search and export endpoints (with rate limiting to prevent DoS)
	r.With(rateLimitMiddleware(s.searchLimiter)).Get("/search", s.search)
	r.With(rateLimitMiddleware(s.searchLimiter)).Get("/quick-search", s.quickSearch)
	r.Get("/folders-legacy", s.getFolders) // Keep old endpoint for compatibility
	r.Get("/export/markdown", s.exportMarkdown)

	// Due dates overview endpoint
	r.Get("/due-dates", s.getDueDates)
}

func (s *Server) registerTrashRoutes(r chi.Router) {
	// Trash endpoints
	r.Get("/trash", s.listTrash)
	r.Get("/trash/count", s.getTrashCount)
	r.Delete("/trash", s.emptyTrash)
	r.Post("/notes/{id}/restore", s.restoreNote)
	r.Delete("/notes/{id}/permanent", s.permanentlyDeleteNote)
}

func (s *Server) registerLLMRoutes(r chi.Router) {
	// LLM endpoints (not note-specific)
	r.Route("/llm", func(r chi.Router) {
		r.With(rateLimitMiddleware(s.llmLimiter)).Post("/spell-check", s.spellCheck)
	})
}

func (s *Server) registerJobRoutes(r chi.Router) {
	// Jobs endpoints
	r.Get("/jobs/{id}", s.getJobStatus)
}

func (s *Server) registerGraphRoutes(r chi.Router) {
	// Graph endpoints
	r.Get("/graph", s.getGlobalGraph)
}

func (s *Server) registerWebsocketRoutes(r chi.Router) {
	// WebSocket endpoint (token from query param, not header)
	r.Get("/ws", s.handleWebSocket)
}

func (s *Server) registerAdminRoutes(r chi.Router) {
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
}
