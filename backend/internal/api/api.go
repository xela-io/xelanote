// Package api provides HTTP handlers for the xelanote API.
package api

import (
	"embed"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
)

//go:embed static/captcha.html
var captchaHTML embed.FS

// NewServer creates a new API server.
func NewServer(cfg ServerConfig) *Server {
	// Validate CORS configuration in non-development environments (SEC-003)
	env := os.Getenv("XELANOTE_ENV")
	if env != "" && env != "development" && env != "test" && env != "testing" && len(cfg.AllowedOrigins) == 0 {
		cfg.Logger.Error("FATAL: CORS_ALLOWED_ORIGINS must be set in non-development environments",
			slog.String("env", env),
			slog.Int("allowed_origins_count", len(cfg.AllowedOrigins)))
		log.Fatalf("CORS_ALLOWED_ORIGINS must be set when XELANOTE_ENV=%s", env)
	}

	limits := buildRateLimitConfig()

	s := &Server{
		noteService:             cfg.NoteService,
		authService:             cfg.AuthService,
		tfaService:              cfg.TFAService,
		graphService:            cfg.GraphService,
		templateService:         cfg.TemplateService,
		snippetService:          cfg.SnippetService,
		userService:             cfg.UserService,
		adminService:            cfg.AdminService,
		activityService:         cfg.ActivityService,
		settingsService:         cfg.SettingsService,
		turnstileService:        cfg.TurnstileService,
		summarizeService:        cfg.SummarizeService,
		recipeService:           cfg.RecipeService,
		recipeSuggestionService: cfg.RecipeSuggestSvc,
		jobManager:              cfg.JobManager,
		wsManager:               cfg.WSManager,
		log:                     cfg.Logger,
		jwtSecret:               cfg.JWTSecret,
		dataDir:                 cfg.DataDir,
		allowedOrigins:          cfg.AllowedOrigins,
		router:                  chi.NewRouter(),
		// Rate limiters: register=5/hour, login=10/15min, refresh=30/hour, 2fa=5/15min, backup=3/15min, recovery=3/15min
		// In test mode, all limits are increased to 1000/hour
		registerLimiter:   NewRateLimiter(limits.registerLimit, time.Hour, limits.registerLimit),
		loginLimiter:      NewRateLimiter(limits.loginLimit, 15*time.Minute, limits.loginLimit),
		refreshLimiter:    NewRateLimiter(limits.refreshLimit, time.Hour, limits.refreshLimit),
		tfaVerifyLimiter:  NewRateLimiter(limits.tfaLimit, 15*time.Minute, limits.tfaLimit),
		backupCodeLimiter: NewRateLimiter(limits.backupLimit, 15*time.Minute, limits.backupLimit),
		recoveryLimiter:   NewRateLimiter(limits.recoveryLimit, 15*time.Minute, limits.recoveryLimit),
		// Resource-intensive endpoint limiters: uploads=20/hour, import=10/hour, search=120/min, summarize=10/min
		// Sensitive user operations: password/email/recovery-key changes=3/hour
		uploadLimiter:         NewRateLimiter(limits.uploadLimit, time.Hour, limits.uploadLimit),
		importLimiter:         NewRateLimiter(limits.importLimit, time.Hour, limits.importLimit),
		searchLimiter:         NewRateLimiter(limits.searchLimit, time.Minute, 30), // 120/min, burst 30
		passwordChangeLimiter: NewRateLimiter(limits.passwordChangeLimit, time.Hour, limits.passwordChangeLimit),
		emailChangeLimiter:    NewRateLimiter(limits.emailChangeLimit, time.Hour, limits.emailChangeLimit),
		recoveryKeyLimiter:    NewRateLimiter(limits.recoveryKeyLimit, time.Hour, limits.recoveryKeyLimit),
		llmLimiter:            NewRateLimiter(limits.llmLimit, time.Minute, limits.llmLimit), // 10/min shared for all LLM endpoints
		fido2BeginLimiter:     NewRateLimiter(limits.fido2Limit, 15*time.Minute, limits.fido2Limit),
		fido2FinishLimiter:    NewRateLimiter(limits.fido2Limit, 15*time.Minute, limits.fido2Limit),
		fido2Service:          cfg.FIDO2Service,
		sharingService:        cfg.SharingService,
		shareLimiter:          NewRateLimiter(limits.shareLimit, time.Minute, limits.shareLimit),
		userSearchLimiter:     NewRateLimiter(limits.userSearchLimit, time.Minute, 10), // 30/min, burst 10
		errorReportService:    cfg.ErrorReportSvc,
		errorReportLimiter:    NewRateLimiter(limits.errorReportLimit, time.Hour, 3), // 5/hour, burst 3
		telemetryService:      cfg.TelemetrySvc,
		perfMetricsLimiter:    NewRateLimiter(limits.perfMetricsLimit, time.Hour, 10),                  // 30/hour, burst 10
		analyticsLimiter:      NewRateLimiter(limits.analyticsLimit, time.Hour, limits.analyticsLimit), // 20/hour
		streamContent:         newStreamContentStore(),
		// Account lockout: 10 global attempts (5 per-IP), 30s initial lockout (doubles each time), 30min max
		// In test mode, use 1000 attempts to effectively disable lockout
		accountLockout: NewAccountLockout(limits.lockoutAttempts, 30*time.Second, 30*time.Minute, cfg.Logger),
	}
	s.setupRoutes()
	return s
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
