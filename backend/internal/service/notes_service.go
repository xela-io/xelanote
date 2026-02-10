// Package service contains the business logic for xelanote.
package service

import (
	"log/slog"
	"time"

	"github.com/xela-io/xelanote/internal/cache"
	"github.com/xela-io/xelanote/internal/db"
)

// NoteService handles note operations with link management.
type NoteService struct {
	db           *db.DB
	logger       *slog.Logger
	cache        *cache.Cache
	graphService *GraphService
}

// NewNoteService creates a new NoteService.
func NewNoteService(database *db.DB) *NoteService {
	return &NoteService{
		db:     database,
		logger: slog.Default(),
		cache:  cache.NewCache(5 * time.Minute), // 5 minute TTL
	}
}

// SetGraphService sets the graph service for cache invalidation.
// This must be called after both NoteService and GraphService are created.
func (s *NoteService) SetGraphService(gs *GraphService) {
	s.graphService = gs
}

// GetCache returns the cache instance used by this service.
func (s *NoteService) GetCache() *cache.Cache {
	return s.cache
}

// GetDB returns the database instance used by this service.
func (s *NoteService) GetDB() *db.DB {
	return s.db
}
