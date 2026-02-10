package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xela-io/xelanote/internal/db"
)

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		respondError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = min(parsed, 200) // Cap at 200 for search results
		}
	}

	results, err := s.noteService.Search(r.Context(), userID, query, limit)
	if err != nil {
		// Return 400 for invalid queries (validation errors), 500 for server errors
		if errors.Is(err, db.ErrInvalidQuery) {
			respondError(w, http.StatusBadRequest, err.Error())
		} else {
			s.respondInternalErr(w, "failed to search notes", err)
		}
		return
	}

	results = ensureSlice(results)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"query":   query,
	})
}

func (s *Server) quickSearch(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Parse query parameters
	query := r.URL.Query().Get("q")
	foldersParam := r.URL.Query().Get("folders")
	tagsParam := r.URL.Query().Get("tags")
	createdAfterParam := r.URL.Query().Get("created_after")
	createdBeforeParam := r.URL.Query().Get("created_before")
	updatedAfterParam := r.URL.Query().Get("updated_after")
	updatedBeforeParam := r.URL.Query().Get("updated_before")

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = min(parsed, 100) // Cap at 100 for quick search
		}
	}

	// Check if any filters are present
	hasFilters := foldersParam != "" || tagsParam != "" ||
		createdAfterParam != "" || createdBeforeParam != "" ||
		updatedAfterParam != "" || updatedBeforeParam != ""

	var notes []db.Note
	var err error

	if hasFilters {
		// Use FilteredSearch when filters are present
		filters := db.SearchFilters{
			Query: query,
		}

		// Parse folders (comma-separated)
		if foldersParam != "" {
			filters.Folders = strings.Split(foldersParam, ",")
			for i := range filters.Folders {
				filters.Folders[i] = strings.TrimSpace(filters.Folders[i])
			}
		}

		// Parse tags (comma-separated)
		if tagsParam != "" {
			filters.Tags = strings.Split(tagsParam, ",")
			for i := range filters.Tags {
				filters.Tags[i] = strings.TrimSpace(filters.Tags[i])
			}
		}

		// Parse date filters
		if createdAfterParam != "" {
			if t, parseErr := time.Parse(time.RFC3339, createdAfterParam); parseErr == nil {
				filters.CreatedAfter = t
			}
		}
		if createdBeforeParam != "" {
			if t, parseErr := time.Parse(time.RFC3339, createdBeforeParam); parseErr == nil {
				filters.CreatedBefore = t
			}
		}
		if updatedAfterParam != "" {
			if t, parseErr := time.Parse(time.RFC3339, updatedAfterParam); parseErr == nil {
				filters.UpdatedAfter = t
			}
		}
		if updatedBeforeParam != "" {
			if t, parseErr := time.Parse(time.RFC3339, updatedBeforeParam); parseErr == nil {
				filters.UpdatedBefore = t
			}
		}

		notes, err = s.noteService.FilteredSearch(r.Context(), userID, filters, limit)
	} else {
		// Use legacy QuickSearch when no filters (backward-compatible)
		// For empty queries, return recent notes (useful for wikilink autocomplete)
		notes, err = s.noteService.QuickSearch(r.Context(), userID, query, limit)
	}

	if err != nil {
		// Return 400 for invalid queries (validation errors), 500 for server errors
		if errors.Is(err, db.ErrInvalidQuery) {
			respondError(w, http.StatusBadRequest, err.Error())
		} else {
			s.respondInternalErr(w, "failed to quick search notes", err)
		}
		return
	}

	notes = ensureSlice(notes)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"notes": notes,
	})
}

func (s *Server) getFolders(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	folders, err := s.noteService.GetFoldersWithCounts(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to get folders", err)
		return
	}

	if folders == nil {
		folders = []db.FolderInfo{{Path: "/", NoteCount: 0}}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"folders": folders,
	})
}
