package main

import (
	"log/slog"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/llm"
	"github.com/xela-io/xelanote/internal/service"
)

func initProviderRouter(database *db.DB) *llm.ProviderRouter {
	return llm.NewProviderRouter(database)
}

func initSummarizeService(database *db.DB, providerRouter *llm.ProviderRouter, logger *slog.Logger) *service.SummarizeService {
	return service.NewSummarizeService(database, providerRouter, logger)
}
