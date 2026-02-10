package service

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/xela-io/xelanote/internal/cache"
	"github.com/xela-io/xelanote/internal/db"
)

// SnippetService handles snippet operations with caching.
type SnippetService struct {
	db     *db.DB
	cache  *cache.Cache
	logger *slog.Logger
}

// NewSnippetService creates a new SnippetService.
func NewSnippetService(database *db.DB) *SnippetService {
	return &SnippetService{
		db:     database,
		cache:  cache.NewCache(5 * time.Minute), // 5 minute TTL
		logger: slog.Default(),
	}
}

func snippetCacheKey(userID, snippetID int) string {
	return fmt.Sprintf("cache:snippet:%d:%d", userID, snippetID)
}

func snippetsCacheKey(userID int) string {
	return fmt.Sprintf("cache:snippets:%d", userID)
}

// GetAllSnippets returns all snippets for a user with caching.
func (s *SnippetService) GetAllSnippets(userID int) ([]db.Snippet, error) {
	cacheKey := snippetsCacheKey(userID)

	// Try cache first
	if cached, ok := s.cache.Get(cacheKey); ok {
		if snippets, ok := cached.([]db.Snippet); ok {
			return snippets, nil
		}
	}

	// Cache miss, query database
	snippets, err := s.db.GetAllSnippets(userID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.Set(cacheKey, snippets)

	return snippets, nil
}

// GetSnippet returns a single snippet with caching.
func (s *SnippetService) GetSnippet(userID, snippetID int) (*db.Snippet, error) {
	cacheKey := snippetCacheKey(userID, snippetID)

	// Try cache first
	if cached, ok := s.cache.Get(cacheKey); ok {
		if snippet, ok := cached.(*db.Snippet); ok {
			return snippet, nil
		}
	}

	// Cache miss, query database
	snippet, err := s.db.GetSnippet(userID, snippetID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.Set(cacheKey, snippet)

	return snippet, nil
}

// CreateSnippet creates a new snippet and invalidates cache.
func (s *SnippetService) CreateSnippet(userID int, name, description, content, shortcut string) (*db.Snippet, error) {
	snippet, err := s.db.CreateSnippet(userID, name, description, content, shortcut)
	if err != nil {
		return nil, err
	}

	// Invalidate list cache
	s.cache.Delete(snippetsCacheKey(userID))

	return snippet, nil
}

// UpdateSnippet updates a snippet and invalidates cache.
func (s *SnippetService) UpdateSnippet(userID, snippetID int, name, description, content, shortcut string) error {
	err := s.db.UpdateSnippet(userID, snippetID, name, description, content, shortcut)
	if err != nil {
		return err
	}

	// Invalidate caches
	s.cache.Delete(snippetCacheKey(userID, snippetID))
	s.cache.Delete(snippetsCacheKey(userID))

	return nil
}

// DeleteSnippet deletes a snippet and invalidates cache.
func (s *SnippetService) DeleteSnippet(userID, snippetID int) error {
	err := s.db.DeleteSnippet(userID, snippetID)
	if err != nil {
		return err
	}

	// Invalidate caches
	s.cache.Delete(snippetCacheKey(userID, snippetID))
	s.cache.Delete(snippetsCacheKey(userID))

	return nil
}
