package api

import (
	"io"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/jobs"
	"github.com/xela-io/xelanote/internal/service"
	"github.com/xela-io/xelanote/internal/websocket"
)

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

// SetRecipeService sets the recipe service (called after server creation).
func (s *Server) SetRecipeService(rs *service.RecipeService) {
	s.recipeService = rs
}

// SetRecipeSuggestionService sets the recipe suggestion service (called after server creation).
func (s *Server) SetRecipeSuggestionService(rss *service.RecipeSuggestionService) {
	s.recipeSuggestionService = rss
}

// logger returns the server logger (returns no-op logger if nil for tests).
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
