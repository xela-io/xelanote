package service

import (
	"fmt"
	"log/slog"

	"github.com/xela-io/xelanote/internal/cache"
	"github.com/xela-io/xelanote/internal/db"
)

// GraphService handles graph operations with caching.
type GraphService struct {
	db     *db.DB
	logger *slog.Logger
	cache  *cache.Cache
}

// NewGraphService creates a new GraphService with its own cache (2 min TTL).
func NewGraphService(database *db.DB, cacheInstance *cache.Cache) *GraphService {
	// Note: Using shared cache from NoteService
	// The shared cache has 5 min TTL, but that's fine for graph data
	return &GraphService{
		db:     database,
		logger: slog.Default(),
		cache:  cacheInstance,
	}
}

func globalGraphCacheKey(userID int) string {
	return fmt.Sprintf("cache:graph:global:%d", userID)
}

// GetGlobalGraph returns the global graph for a user with caching (2 min TTL).
func (s *GraphService) GetGlobalGraph(userID int) (*db.GraphData, error) {
	cacheKey := globalGraphCacheKey(userID)

	// Check cache first
	if cached, ok := s.cache.Get(cacheKey); ok {
		if graphData, ok := cached.(*db.GraphData); ok {
			s.logger.Debug("global graph cache hit", "user_id", userID)
			return graphData, nil
		}
	}

	// Cache miss - query database
	s.logger.Debug("global graph cache miss", "user_id", userID)
	graphData, err := s.db.GetGlobalGraph(userID, db.MaxGraphNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get global graph: %w", err)
	}

	// Cache the result (uses cache's TTL)
	s.cache.Set(cacheKey, graphData)

	return graphData, nil
}

// GetFilteredGraph returns a filtered graph for a user (no caching due to dynamic filters).
func (s *GraphService) GetFilteredGraph(userID int, folderPath string) (*db.GraphData, error) {
	return s.db.GetFilteredGraph(userID, folderPath, db.MaxGraphNodes)
}

// InvalidateGraphCache invalidates the global graph cache for a user.
// Should be called when notes are created, updated, deleted, or renamed.
func (s *GraphService) InvalidateGraphCache(userID int) {
	cacheKey := globalGraphCacheKey(userID)
	s.cache.Delete(cacheKey)
	s.logger.Debug("invalidated graph cache", "user_id", userID)
}
