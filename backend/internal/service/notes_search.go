// Package service contains the business logic for xelanote.
package service

import (
	"context"

	"github.com/xela-io/xelanote/internal/db"
)

// ListNotes returns a paginated list of notes.
func (s *NoteService) ListNotes(userID int, limit int, cursor string) ([]db.Note, string, error) {
	return s.db.ListNotes(userID, limit, cursor)
}

// GetBacklinks returns all notes linking to the given note.
func (s *NoteService) GetBacklinks(userID int, noteID string) ([]db.Backlink, error) {
	return s.db.GetBacklinks(userID, noteID)
}

// Search performs a full-text search.
func (s *NoteService) Search(ctx context.Context, userID int, query string, limit int) ([]db.SearchResult, error) {
	return s.db.Search(ctx, userID, query, limit)
}

// QuickSearch performs a fast title search for the quick switcher.
func (s *NoteService) QuickSearch(ctx context.Context, userID int, query string, limit int) ([]db.Note, error) {
	key := quickSearchCacheKey(userID, query, limit)
	if cached, ok := s.cache.Get(key); ok {
		return cached.([]db.Note), nil
	}

	results, err := s.db.QuickSearch(ctx, userID, query, limit)
	if err != nil {
		return nil, err
	}

	s.cache.Set(key, results)
	return results, nil
}

// FilteredSearch performs a search with optional filters (tags, folders, dates).
// Note: Filtered search results are not cached due to the many possible filter combinations.
func (s *NoteService) FilteredSearch(ctx context.Context, userID int, filters db.SearchFilters, limit int) ([]db.Note, error) {
	return s.db.FilteredSearch(ctx, userID, filters, limit)
}

// GetNoteByTitle finds note by title (case-insensitive).
func (s *NoteService) GetNoteByTitle(userID int, title string) (*db.Note, error) {
	return s.db.GetNoteByTitle(userID, title)
}

// GetNoteByTitleInFolder retrieves a note by title within a specific folder.
func (s *NoteService) GetNoteByTitleInFolder(userID int, title, folderPath string) (*db.Note, error) {
	return s.db.GetNoteByTitleInFolder(userID, title, folderPath)
}

// GetDueDatesByUser returns all due dates for a user across all non-deleted notes.
func (s *NoteService) GetDueDatesByUser(userID int, showCompleted bool) ([]db.DueDateWithNote, error) {
	return s.db.GetDueDatesByUser(userID, showCompleted)
}
