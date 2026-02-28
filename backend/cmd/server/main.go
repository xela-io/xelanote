package main

import (
	"embed"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/xela-io/xelanote/internal/api"
	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/service"
)

//go:embed all:static
var staticFS embed.FS

func main() {
	// Command line flags
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "", "Database path (default: ./data/xelanote.db or $XELANOTE_DB)")
	flag.Parse()

	jwtSecret := loadJWTSecret()
	databasePath := resolveDatabasePath(*dbPath)
	dbKey, err := loadDatabaseKey()
	if err != nil {
		log.Fatalf("Failed to load database key: %v", err)
	}

	dataDir, err := ensureDataDir(databasePath)
	if err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	if err := ensureUploadsDir(dataDir); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}

	// Open database with configurable journal mode (WAL default, DELETE fallback)
	journalMode := strings.TrimSpace(os.Getenv("XELANOTE_JOURNAL_MODE"))
	if journalMode == "" {
		journalMode = "wal"
	}
	log.Printf("Opening database at %s (journal_mode=%s)", databasePath, journalMode)
	if dbKey != "" {
		log.Println("Database encryption enabled (SQLCipher)")
	}
	database, err := db.Open(databasePath, dbKey, db.OpenOptions{JournalMode: journalMode})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Run migrations
	log.Println("Running database migrations...")
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// In test mode, enable registration so E2E tests can create users
	if os.Getenv("XELANOTE_ENV") == "test" {
		if err := database.SetSetting("registration_enabled", "true"); err != nil {
			log.Printf("WARNING: Failed to enable registration for test mode: %v", err)
		}
	}

	// Run PRAGMA optimize at startup to update query planner statistics
	database.Optimize()

	// Schedule periodic PRAGMA optimize (daily)
	optimizeCancel := database.StartOptimizeScheduler(24 * time.Hour)

	logger := newLogger()
	core := initCoreServices(database, []byte(jwtSecret), dataDir, logger)
	postStartupMaintenance(core.activity, database)

	// Cleanup expired and revoked refresh tokens at startup
	if cleaned, err := database.CleanupExpiredRefreshTokens(); err != nil {
		log.Printf("Refresh token cleanup failed: %v", err)
	} else if cleaned > 0 {
		log.Printf("Cleaned up %d expired/revoked refresh tokens", cleaned)
	}

	// Backfill due dates for existing notes (one-time after migration 042)
	if backfilled, err := database.BackfillDueDates(); err != nil {
		log.Printf("Due dates backfill failed: %v", err)
	} else if backfilled > 0 {
		log.Printf("Backfilled %d due dates from existing notes", backfilled)
	}

	graphService := initGraphService(database, core.note)
	providerRouter := initProviderRouter(database)
	summarizeService := initSummarizeService(database, providerRouter, logger)
	jobManager := startJobManager(core.note)

	// Initialize trusted proxies.
	// Security hardening: in production, explicit trusted proxies are required.
	env := os.Getenv("XELANOTE_ENV")
	trustedProxiesEnv := os.Getenv("TRUSTED_PROXIES")
	if env == "production" && trustedProxiesEnv == "" {
		log.Fatal("TRUSTED_PROXIES must be set when XELANOTE_ENV=production")
	}
	api.InitTrustedProxies(trustedProxiesEnv)

	pruneCancel := startVersionPruner(core.note)

	// NOTE: Auto-summary scheduler disabled (cloud providers only - Claude/Gemini)
	// Users need to configure their own API keys and trigger summaries manually.
	// This avoids unexpected API costs for users.
	log.Println("Auto-summary scheduler disabled (cloud providers require API key)")

	sharingService := service.NewSharingService(database)
	errorReportService := initErrorReportService(logger)
	wsManager := startWebSocketManager(logger)
	env = checkEnvironment(logger)
	fido2Service := initFIDO2Service(database, core.tfa, logger, env)
	turnstileService := initTurnstileService(logger)
	recipeService := service.NewRecipeService(database, core.note)
	recipeSuggestionService := service.NewRecipeSuggestionService(database, providerRouter, recipeService)
	canvasService := service.NewCanvasService(database, core.note)
	shoppingService := service.NewShoppingService(database, providerRouter)
	telemetryService := service.NewTelemetryService(database, logger)

	// Create API server
	allowedOrigins := parseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if env == "" && len(allowedOrigins) == 0 {
		log.Fatal("CORS_ALLOWED_ORIGINS must be set when XELANOTE_ENV is empty (strict default)")
	}
	server := api.NewServer(api.ServerConfig{
		NoteService:      core.note,
		AuthService:      core.auth,
		TwoFactorService: core.tfa,
		FIDO2Service:     fido2Service,
		GraphService:     graphService,
		TemplateService:  core.template,
		SnippetService:   core.snippet,
		UserService:      core.user,
		AdminService:     core.admin,
		ActivityService:  core.activity,
		SettingsService:  core.settings,
		TurnstileService: turnstileService,
		SummarizeService: summarizeService,
		SharingService:   sharingService,
		ErrorReportSvc:   errorReportService,
		RecipeService:    recipeService,
		RecipeSuggestSvc: recipeSuggestionService,
		CanvasService:    canvasService,
		ShoppingSvc:      shoppingService,
		TelemetrySvc:     telemetryService,
		LockoutDB:        database,
		DBPing:           database.Ping,
		JobManager:       jobManager,
		WSManager:        wsManager,
		Logger:           logger,
		JWTSecret:        []byte(jwtSecret),
		DataDir:          dataDir,
		AllowedOrigins:   allowedOrigins,
	})
	router := server.Router()

	setupStaticFiles(router)
	startPprofServerIfEnabled()

	// Setup graceful shutdown
	srv := &http.Server{
		Addr:              *addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	cleanup := func() {
		optimizeCancel()
		database.Optimize() // Final optimize before shutdown
		wsManager.Stop()
		jobManager.Stop()
		core.note.Close()
		core.snippet.Close()
		core.template.Close()
		core.admin.Close()
		recipeSuggestionService.Close()
		errorReportService.Close()
	}
	setupGracefulShutdown(srv, pruneCancel, cleanup)

	// Start server
	log.Printf("Starting server on %s", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
	log.Println("Server stopped")
}
