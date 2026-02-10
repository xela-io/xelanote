package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/fido2"
	"github.com/xela-io/xelanote/internal/service"
)

func initFIDO2Service(database *db.DB, tfaService *service.TwoFactorService, logger *slog.Logger, env string) *service.FIDO2Service {
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
	return service.NewFIDO2Service(database, fido2Manager, tfaService, logger)
}
