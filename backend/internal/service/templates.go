package service

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/xela-io/xelanote/internal/cache"
	"github.com/xela-io/xelanote/internal/db"
)

// TemplateService handles template operations with caching.
type TemplateService struct {
	db     *db.DB
	cache  *cache.Cache
	logger *slog.Logger
}

// NewTemplateService creates a new TemplateService.
func NewTemplateService(database *db.DB) *TemplateService {
	return &TemplateService{
		db:     database,
		cache:  cache.NewCache(5 * time.Minute), // 5 minute TTL
		logger: slog.Default(),
	}
}

func templateCacheKey(userID, templateID int) string {
	return fmt.Sprintf("cache:template:%d:%d", userID, templateID)
}

func templatesCacheKey(userID int) string {
	return fmt.Sprintf("cache:templates:%d", userID)
}

// GetAllTemplates returns all templates for a user with caching.
func (s *TemplateService) GetAllTemplates(userID int) ([]db.Template, error) {
	cacheKey := templatesCacheKey(userID)

	// Try cache first
	if cached, ok := s.cache.Get(cacheKey); ok {
		if templates, ok := cached.([]db.Template); ok {
			return templates, nil
		}
	}

	// Cache miss, query database
	templates, err := s.db.GetAllTemplates(userID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.Set(cacheKey, templates)

	return templates, nil
}

// GetTemplate returns a single template with caching.
func (s *TemplateService) GetTemplate(userID, templateID int) (*db.Template, error) {
	cacheKey := templateCacheKey(userID, templateID)

	// Try cache first
	if cached, ok := s.cache.Get(cacheKey); ok {
		if template, ok := cached.(*db.Template); ok {
			return template, nil
		}
	}

	// Cache miss, query database
	template, err := s.db.GetTemplate(userID, templateID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.Set(cacheKey, template)

	return template, nil
}

// CreateTemplate creates a new template and invalidates cache.
func (s *TemplateService) CreateTemplate(userID int, name, description, title, content string) (*db.Template, error) {
	template, err := s.db.CreateTemplate(userID, name, description, title, content)
	if err != nil {
		return nil, err
	}

	// Invalidate list cache
	s.cache.Delete(templatesCacheKey(userID))

	return template, nil
}

// UpdateTemplate updates a template and invalidates cache.
func (s *TemplateService) UpdateTemplate(userID, templateID int, name, description, title, content string) error {
	err := s.db.UpdateTemplate(userID, templateID, name, description, title, content)
	if err != nil {
		return err
	}

	// Invalidate caches
	s.cache.Delete(templateCacheKey(userID, templateID))
	s.cache.Delete(templatesCacheKey(userID))

	return nil
}

// DeleteTemplate deletes a template and invalidates cache.
func (s *TemplateService) DeleteTemplate(userID, templateID int) error {
	err := s.db.DeleteTemplate(userID, templateID)
	if err != nil {
		return err
	}

	// Invalidate caches
	s.cache.Delete(templateCacheKey(userID, templateID))
	s.cache.Delete(templatesCacheKey(userID))

	return nil
}
