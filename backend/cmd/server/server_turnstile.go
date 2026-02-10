package main

import (
	"log/slog"
	"os"

	"github.com/xela-io/xelanote/internal/service"
)

func initTurnstileService(logger *slog.Logger) *service.TurnstileService {
	turnstileSecretKey := os.Getenv("TURNSTILE_SECRET_KEY")
	turnstileSiteKey := os.Getenv("TURNSTILE_SITE_KEY")
	return service.NewTurnstileService(turnstileSecretKey, turnstileSiteKey, logger)
}
