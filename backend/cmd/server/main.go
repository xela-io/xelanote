package main

import (
	"embed"
	"flag"
	"log"
	"net/http"
	"os"
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

	// Open database
	log.Printf("Opening database at %s", databasePath)
	if dbKey != "" {
		log.Println("Database encryption enabled (SQLCipher)")
	}
	database, err := db.Open(databasePath, dbKey)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Run migrations
	log.Println("Running database migrations...")
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	logger := newLogger()
	core := initCoreServices(database, []byte(jwtSecret), dataDir, logger)
	postStartupMaintenance(core.activity, database)

	// Cleanup old activity logs at startup
	if cleaned, err := activityService.CleanupOldActivity(); err != nil {
		log.Printf("Activity cleanup failed: %v", err)
	} else if cleaned > 0 {
		log.Printf("Cleaned up %d old activity logs", cleaned)
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

	// Initialize trusted proxies
	trustedProxiesEnv := os.Getenv("TRUSTED_PROXIES")
	api.InitTrustedProxies(trustedProxiesEnv)

	pruneCancel := startVersionPruner(core.note)

	// NOTE: Auto-summary scheduler disabled (cloud providers only - Claude/Gemini)
	// Users need to configure their own API keys and trigger summaries manually.
	// This avoids unexpected API costs for users.
	log.Println("Auto-summary scheduler disabled (cloud providers require API key)")

	sharingService := service.NewSharingService(database)
	errorReportService := initErrorReportService(logger)
	wsManager := startWebSocketManager(logger)
	env := checkEnvironment(logger)
	fido2Service := initFIDO2Service(database, core.tfa, logger, env)
	turnstileService := initTurnstileService(logger)

	// Create API server
	allowedOrigins := parseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
	server := api.NewServer(api.ServerConfig{
		NoteService:      core.note,
		AuthService:      core.auth,
		TFAService:       core.tfa,
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
		JobManager:       jobManager,
		WSManager:        wsManager,
		Logger:           logger,
		JWTSecret:        []byte(jwtSecret),
		DataDir:          dataDir,
		AllowedOrigins:   allowedOrigins,
	})
	recipeService := service.NewRecipeService(database, core.note)
	server.SetRecipeService(recipeService)
	recipeSuggestionService := service.NewRecipeSuggestionService(database, providerRouter, recipeService)
	server.SetRecipeSuggestionService(recipeSuggestionService)
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

	setupGracefulShutdown(srv, pruneCancel)

	// Start server
	log.Printf("Starting server on %s", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
	log.Println("Server stopped")
}
