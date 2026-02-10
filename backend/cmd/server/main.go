package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/xela-io/xelanote/internal/api"
	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/fido2"
	"github.com/xela-io/xelanote/internal/jobs"
	"github.com/xela-io/xelanote/internal/llm"
	"github.com/xela-io/xelanote/internal/service"
	"github.com/xela-io/xelanote/internal/websocket"
)

//go:embed all:static
var staticFS embed.FS

func main() {
	// Command line flags
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "", "Database path (default: ./data/xelanote.db or $XELANOTE_DB)")
	flag.Parse()

	// Load JWT secret from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// Validate JWT_SECRET strength (SEC-001)
	const defaultWeakSecret = "your-secret-key-min-32-characters-long-for-hs256"
	if jwtSecret == defaultWeakSecret {
		log.Fatal("JWT_SECRET cannot be the default example value. Generate a strong secret with: openssl rand -hex 32")
	}
	if len(jwtSecret) < 64 {
		log.Fatalf("JWT_SECRET must be at least 64 characters long (current length: %d). Generate a strong secret with: openssl rand -hex 32", len(jwtSecret))
	}

	// Determine database path
	databasePath := *dbPath
	if databasePath == "" {
		databasePath = os.Getenv("XELANOTE_DB")
	}
	if databasePath == "" {
		databasePath = "./data/xelanote.db"
	}

	dbKey, err := loadDatabaseKey()
	if err != nil {
		log.Fatalf("Failed to load database key: %v", err)
	}

	// Ensure data directory exists
	dataDir := filepath.Dir(databasePath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Ensure uploads directory exists
	uploadDir := filepath.Join(dataDir, "uploads")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
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

	// Create structured logger (SEC-008: needed for security event logging)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create services
	noteService := service.NewNoteService(database)
	tfaService := service.NewTwoFactorService(database, logger)
	authService := service.NewAuthService(database, []byte(jwtSecret), tfaService)
	templateService := service.NewTemplateService(database)
	snippetService := service.NewSnippetService(database)
	userService := service.NewUserService(database)
	adminService := service.NewAdminService(database, dataDir)
	activityService := service.NewActivityService(database)
	settingsService := service.NewSettingsService(database)

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

	// Create graph service (reuse noteService cache)
	graphService := service.NewGraphService(database, noteService.GetCache())

	// Inject graph service into note service for cache invalidation
	noteService.SetGraphService(graphService)

	// Create LLM provider router and summarize service
	providerRouter := llm.NewProviderRouter(database)
	summarizeService := service.NewSummarizeService(database, providerRouter, logger)

	// Create and start job manager
	jobManager := jobs.NewJobManager(4) // 4 workers
	jobManager.RegisterHandler(jobs.JobTypeRenameNote, jobs.HandleRenameNoteJob(noteService))
	jobManager.Start()
	log.Println("Job manager started with 4 workers")

	// Initialize trusted proxies
	trustedProxiesEnv := os.Getenv("TRUSTED_PROXIES")
	api.InitTrustedProxies(trustedProxiesEnv)

	// Start version pruning job (runs daily, keeps 100 versions per note)
	pruneCtx, pruneCancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		// Run once at startup
		pruned, err := noteService.PruneAllVersions(100)
		if err != nil {
			log.Printf("Version pruning failed: %v", err)
		} else if pruned > 0 {
			log.Printf("Pruned %d old versions at startup", pruned)
		}

		for {
			select {
			case <-ticker.C:
				pruned, err := noteService.PruneAllVersions(100)
				if err != nil {
					log.Printf("Version pruning failed: %v", err)
				} else if pruned > 0 {
					log.Printf("Pruned %d old versions", pruned)
				}
			case <-pruneCtx.Done():
				return
			}
		}
	}()

	// NOTE: Auto-summary scheduler disabled (cloud providers only - Claude/Gemini)
	// Users need to configure their own API keys and trigger summaries manually.
	// This avoids unexpected API costs for users.
	log.Println("Auto-summary scheduler disabled (cloud providers require API key)")

	// Create sharing service
	sharingService := service.NewSharingService(database)

	// Create error reporting service (Forgejo integration)
	forgejoURL := os.Getenv("FORGEJO_URL")
	forgejoRepo := os.Getenv("FORGEJO_REPO")
	forgejoAPIToken := os.Getenv("FORGEJO_API_TOKEN")
	var forgejoOwner, forgejoRepoName string
	if parts := strings.SplitN(forgejoRepo, "/", 2); len(parts) == 2 {
		forgejoOwner = parts[0]
		forgejoRepoName = parts[1]
	}
	errorReportService := service.NewErrorReportService(forgejoURL, forgejoOwner, forgejoRepoName, forgejoAPIToken, logger)
	if errorReportService.IsEnabled() {
		if err := errorReportService.EnsureLabels(context.Background()); err != nil {
			logger.Warn("failed to ensure error report labels", "error", err)
		}
	}

	// Create and start WebSocket manager
	wsManager := websocket.NewManager(logger)
	go wsManager.Run()
	log.Println("WebSocket manager started")

	// Check environment mode
	env := os.Getenv("XELANOTE_ENV")
	if env == "" {
		logger.Warn("XELANOTE_ENV not set, running in development mode",
			slog.String("advice", "Set XELANOTE_ENV=production for production deployments"))
	}

	// Create WebAuthn/FIDO2 manager
	rpID := os.Getenv("WEBAUTHN_RP_ID")
	if rpID == "" {
		if env == "production" {
			log.Fatal("WEBAUTHN_RP_ID must be set in production mode")
		}
		rpID = "localhost"
		logger.Warn("WEBAUTHN_RP_ID not set, using localhost (dev mode only)")
	}

	rpOrigins := parseAllowedOrigins(os.Getenv("WEBAUTHN_RP_ORIGINS"))
	if len(rpOrigins) == 0 {
		// Fallback: derive from CORS origins or use localhost default
		rpOrigins = parseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
		if len(rpOrigins) == 0 {
			rpOrigins = []string{"http://localhost:5173", "http://localhost:8080"}
		}
	}

	fido2Manager, err := fido2.NewManager("xelanote", rpID, rpOrigins)
	if err != nil {
		log.Fatalf("Failed to create WebAuthn manager: %v", err)
	}
	fido2Service := service.NewFIDO2Service(database, fido2Manager, tfaService, logger)

	// Create Turnstile CAPTCHA service
	turnstileSecretKey := os.Getenv("TURNSTILE_SECRET_KEY")
	turnstileSiteKey := os.Getenv("TURNSTILE_SITE_KEY")
	turnstileService := service.NewTurnstileService(turnstileSecretKey, turnstileSiteKey, logger)

	// Create API server
	allowedOrigins := parseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
	server := api.NewServer(noteService, authService, tfaService, fido2Service, graphService, templateService, snippetService, userService, adminService, activityService, settingsService, turnstileService, summarizeService, sharingService, errorReportService, jobManager, wsManager, logger, []byte(jwtSecret), dataDir, allowedOrigins)
	recipeService := service.NewRecipeService(database, noteService)
	server.SetRecipeService(recipeService)
	recipeSuggestionService := service.NewRecipeSuggestionService(database, providerRouter, recipeService)
	server.SetRecipeSuggestionService(recipeSuggestionService)
	router := server.Router()

	// Serve static files for SPA (if embedded)
	// Debug: List embedded files
	log.Println("Checking embedded files...")
	entries, _ := fs.ReadDir(staticFS, ".")
	log.Printf("Root entries: %d", len(entries))
	for _, entry := range entries {
		log.Printf("  - %s (isDir: %v)", entry.Name(), entry.IsDir())
	}

	// Check static/ directory contents
	staticEntries, err := fs.ReadDir(staticFS, "static")
	if err != nil {
		log.Printf("Error reading static dir: %v", err)
	} else {
		log.Printf("Static entries: %d", len(staticEntries))
		for i, entry := range staticEntries {
			if i < 10 { // Show first 10
				log.Printf("  static/%s (isDir: %v)", entry.Name(), entry.IsDir())
			}
		}
	}

	// setCacheHeaders sets Cache-Control based on the URL path.
	// Vite-hashed files under /_app/immutable/ can be cached aggressively.
	// All other files must revalidate on each request.
	setCacheHeaders := func(w http.ResponseWriter, path string) {
		if strings.HasPrefix(path, "/_app/immutable/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
	}

	staticSub, err := fs.Sub(staticFS, "static")
	if err == nil {
		// Serve static files
		fileServer := http.FileServer(http.FS(staticSub))
		router.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to serve the file
			path := r.URL.Path
			if path == "/" {
				path = "/index.html"
			}

			// Check if file exists
			if _, err := staticFS.Open("static" + path); err == nil {
				setCacheHeaders(w, path)
				fileServer.ServeHTTP(w, r)
				return
			}

			// Fall back to index.html for SPA routing
			setCacheHeaders(w, "/index.html")
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
		}))
	}

	// Start pprof server (ONLY when explicitly enabled via PPROF_ENABLED=true)
	// Security: Disabled by default, must be explicitly enabled even in dev/staging
	pprofEnabled := os.Getenv("PPROF_ENABLED") == "true"

	if pprofEnabled {
		go func() {
			// Dedicated ServeMux for security (prevents exposure of other handlers)
			pprofMux := http.NewServeMux()
			pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
			pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
			pprofMux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
			pprofMux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
			pprofMux.Handle("/debug/pprof/block", pprof.Handler("block"))
			pprofMux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))

			pprofAddr := "127.0.0.1:6060"
			log.Printf("pprof server available at http://%s/debug/pprof/", pprofAddr)
			log.Printf("  CPU Profile: http://%s/debug/pprof/profile?seconds=30", pprofAddr)
			log.Printf("  Heap Profile: http://%s/debug/pprof/heap", pprofAddr)
			log.Printf("  Goroutines: http://%s/debug/pprof/goroutine", pprofAddr)

			if err := http.ListenAndServe(pprofAddr, pprofMux); err != nil {
				log.Printf("pprof server failed: %v", err)
			}
		}()
	}

	// Setup graceful shutdown
	srv := &http.Server{
		Addr:              *addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Handle shutdown signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		pruneCancel() // Stop the pruning job

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	// Start server
	log.Printf("Starting server on %s", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
	log.Println("Server stopped")
}

func loadDatabaseKey() (string, error) {
	keyFile := strings.TrimSpace(os.Getenv("XELANOTE_DB_KEY_FILE"))
	if keyFile != "" {
		content, err := os.ReadFile(keyFile)
		if err != nil {
			return "", err
		}
		key := strings.TrimSpace(string(content))
		if key == "" {
			return "", fmt.Errorf("database key file is empty")
		}
		return key, nil
	}

	key := strings.TrimSpace(os.Getenv("XELANOTE_DB_KEY"))
	return key, nil
}

func parseAllowedOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin == "" {
			continue
		}
		origins = append(origins, origin)
	}
	if len(origins) == 0 {
		return nil
	}
	return origins
}
