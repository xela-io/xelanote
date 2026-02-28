package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/service"
)

func (s *Server) registerUserRoutes(r chi.Router) {
	// User endpoints (preferences, email, password)
	r.Route("/users", func(r chi.Router) {
		r.Get("/preferences", s.getPreferences)
		r.Put("/preferences", s.updatePreferences)
		r.Patch("/preferences", s.patchPreferences)
		r.Put("/preferences/encryption", s.updateEncryptionPreferences)
		r.Put("/preferences/security", s.updateSecurityPreferences)
		r.Get("/ai-provider", s.getAIProviderPreference)
		r.Put("/ai-provider", s.setAIProviderPreference)
		r.Get("/ai-models", s.getAIModels)
		r.Put("/ai-models", s.setAIModels)
		r.Get("/ai-models/available", s.getAvailableAIModels)
		r.Get("/dietary-preference", s.getDietaryPreference)
		r.Put("/dietary-preference", s.setDietaryPreference)
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
			name:      "claude",
			setKey:    s.userService.SetClaudeAPIKey,
			deleteKey: s.userService.DeleteClaudeAPIKey,
			getKeyStatus: func(uid int) (*apiKeyStatusResponse, error) {
				status, err := s.userService.GetClaudeAPIKeyStatus(uid)
				if err != nil {
					return nil, err
				}
				return mapClaudeAPIKeyStatus(status), nil
			},
			invalidateCache: s.summarizeService.InvalidateClaudeClient,
			validationErr:   service.ErrInvalidClaudeAPIKey,
			invalidKeyMsg:   "invalid Claude API key format (must start with sk-ant-)",
		}
		r.Put("/api-key", s.handleSetAPIKey(claudeKey))
		r.Delete("/api-key", s.handleDeleteAPIKey(claudeKey))
		r.Get("/api-key/status", s.handleGetAPIKeyStatus(claudeKey))

		geminiKey := apiKeyProvider{
			name:      "gemini",
			setKey:    s.userService.SetGeminiAPIKey,
			deleteKey: s.userService.DeleteGeminiAPIKey,
			getKeyStatus: func(uid int) (*apiKeyStatusResponse, error) {
				status, err := s.userService.GetGeminiAPIKeyStatus(uid)
				if err != nil {
					return nil, err
				}
				return mapGeminiAPIKeyStatus(status), nil
			},
			invalidateCache: s.summarizeService.InvalidateGeminiClient,
			validationErr:   service.ErrInvalidGeminiAPIKey,
			invalidKeyMsg:   "invalid Gemini API key format (must start with AIza)",
		}
		r.Put("/gemini-api-key", s.handleSetAPIKey(geminiKey))
		r.Delete("/gemini-api-key", s.handleDeleteAPIKey(geminiKey))
		r.Get("/gemini-api-key/status", s.handleGetAPIKeyStatus(geminiKey))

		openAIKey := apiKeyProvider{
			name:      "openai",
			setKey:    s.userService.SetOpenAIAPIKey,
			deleteKey: s.userService.DeleteOpenAIAPIKey,
			getKeyStatus: func(uid int) (*apiKeyStatusResponse, error) {
				status, err := s.userService.GetOpenAIAPIKeyStatus(uid)
				if err != nil {
					return nil, err
				}
				return mapOpenAIAPIKeyStatus(status), nil
			},
			invalidateCache: s.summarizeService.InvalidateChatGPTClient,
			validationErr:   service.ErrInvalidOpenAIAPIKey,
			invalidKeyMsg:   "invalid OpenAI API key format (must start with sk-)",
		}
		r.Put("/openai-api-key", s.handleSetAPIKey(openAIKey))
		r.Delete("/openai-api-key", s.handleDeleteAPIKey(openAIKey))
		r.Get("/openai-api-key/status", s.handleGetAPIKeyStatus(openAIKey))
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

// registerTranscribeRoutes is registered outside the 30s timeout group
// because audio upload + Whisper processing can take up to 120s.
func (s *Server) registerTranscribeRoutes(r chi.Router) {
	r.With(rateLimitMiddleware(s.llmLimiter)).Post("/llm/transcribe", s.transcribeAudio)
}

func (s *Server) registerJobRoutes(r chi.Router) {
	// Jobs endpoints
	r.Get("/jobs/{id}", s.getJobStatus)
}

func (s *Server) registerGraphRoutes(r chi.Router) {
	// Graph endpoints
	r.Get("/graph", s.getGlobalGraph)
}

func (s *Server) registerWebsocketRoutes(_ chi.Router) {
	// WebSocket is registered in the streaming group of registerProtectedRoutes()
	// (without middleware.Timeout, which would break the WS upgrade).
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
