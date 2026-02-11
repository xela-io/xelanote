package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func loadJWTSecret() string {
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

	return jwtSecret
}

func resolveDatabasePath(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if databasePath := os.Getenv("XELANOTE_DB"); databasePath != "" {
		return databasePath
	}
	return "./data/xelanote.db"
}

func ensureDataDir(databasePath string) (string, error) {
	dataDir := filepath.Dir(databasePath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", err
	}
	return dataDir, nil
}

func ensureUploadsDir(dataDir string) error {
	uploadDir := filepath.Join(dataDir, "uploads")
	return os.MkdirAll(uploadDir, 0755)
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

func checkEnvironment(logger *slog.Logger) string {
	env := os.Getenv("XELANOTE_ENV")
	if env == "" {
		logger.Warn("XELANOTE_ENV not set",
			slog.String("advice", "Set XELANOTE_ENV explicitly (development/test/production). Empty env is treated as strict mode for security checks."))
	}
	return env
}
