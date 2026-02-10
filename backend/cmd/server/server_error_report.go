package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/xela-io/xelanote/internal/service"
)

func initErrorReportService(logger *slog.Logger) *service.ErrorReportService {
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
	return errorReportService
}
