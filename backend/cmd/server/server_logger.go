package main

import (
	"log/slog"
	"os"
)

func newLogger() *slog.Logger {
	// SEC-008: structured logger needed for security event logging.
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
