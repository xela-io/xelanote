package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/xela-io/xelanote/internal/parser"
)

const (
	MaxSearchTerms      = 20  // Max terms per query
	MaxSearchTermLength = 100 // Max chars per term
	MaxTotalQueryLength = 500 // Max total query length
	// SearchTimeout defines the maximum time for FTS5 queries.
	// TODO: Make configurable via XELANOTE_SEARCH_TIMEOUT environment variable if needed.
	SearchTimeout = 5 * time.Second
)

// escapeSnippetHTML escapes HTML but preserves FTS5 mark tags.
// This prevents XSS attacks while keeping search highlighting functional.
func escapeSnippetHTML(s string) string {
	// Use null bytes as temporary placeholders (safe since they can't appear in valid UTF-8 text)
	s = strings.ReplaceAll(s, "<mark>", "\x00MARK_OPEN\x00")
	s = strings.ReplaceAll(s, "</mark>", "\x00MARK_CLOSE\x00")
	s = html.EscapeString(s)
	s = strings.ReplaceAll(s, "\x00MARK_OPEN\x00", "<mark>")
	s = strings.ReplaceAll(s, "\x00MARK_CLOSE\x00", "</mark>")
	return s
}

func validateSearchQuery(query string) error {
	// Check total length
	if len(query) > MaxTotalQueryLength {
		return fmt.Errorf("search query too long: %d characters (max %d)",
			len(query), MaxTotalQueryLength)
	}

	// Check term count and lengths
	terms := strings.Fields(query)
	if len(terms) > MaxSearchTerms {
		return fmt.Errorf("too many search terms: %d (max %d)",
			len(terms), MaxSearchTerms)
	}

	for _, term := range terms {
		if len(term) > MaxSearchTermLength {
			truncated := term
			if len(term) > 20 {
				truncated = term[:20]
			}
			return fmt.Errorf("search term too long: '%s...' (max %d characters)",
				truncated, MaxSearchTermLength)
		}
	}

	return nil
}

// SearchResult represents a search result with snippet.
type SearchResult struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Snippet         string   `json:"snippet"`
	Rank            float64  `json:"rank"`
	Encrypted       bool     `json:"encrypted,omitempty"`
	TitleEncrypted  bool     `json:"title_encrypted,omitempty"`
	EncryptedTitle  *string  `json:"encrypted_title,omitempty"`
	MatchedKeywords []string `json:"matched_keywords,omitempty"`
}

// buildFTSQuery builds an FTS5 MATCH query from search terms.
// Each term is quoted and gets a prefix-match wildcard appended.
func buildFTSQuery(terms []string) string {
	var ftsQuery strings.Builder
	for i, term := range terms {
		if i > 0 {
			ftsQuery.WriteString(" ")
		}
		term = strings.ReplaceAll(term, "\"", "\"\"")
		ftsQuery.WriteString("\"")
		ftsQuery.WriteString(term)
		ftsQuery.WriteString("\"*")
	}
	return ftsQuery.String()
}

// Search performs a full-text search using FTS5, combining plaintext and keyword results.
func (db *DB) Search(ctx context.Context, userID int, query string, limit int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)

	// Validate query complexity
	if err := validateSearchQuery(query); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidQuery, err)
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	terms := strings.Fields(query)
	if len(terms) == 0 {
		return []SearchResult{}, nil
	}

	ftsQ := buildFTSQuery(terms)

	// Context with timeout to prevent ReDoS
	ctx, cancel := context.WithTimeout(ctx, SearchTimeout)
	defer cancel()

	// Search plaintext notes via notes_fts
	plaintext, err := db.searchPlaintext(ctx, userID, ftsQ, limit)
	if err != nil {
		return nil, err
	}

	// Search keyword index (graceful degradation on failure)
	keywords, err := db.searchKeywords(ctx, userID, ftsQ, limit)
	if err != nil {
		slog.Warn("keyword search failed, returning plaintext results only", "error", err)
		return plaintext, nil
	}

	return mergeSearchResults(plaintext, keywords, limit), nil
}

// searchPlaintext searches the notes_fts index for plaintext content matches.
func (db *DB) searchPlaintext(ctx context.Context, userID int, ftsQuery string, limit int) ([]SearchResult, error) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT n.id, n.title, snippet(notes_fts, 1, '<mark>', '</mark>', '...', 32) as snippet, bm25(notes_fts) as rank
		FROM notes_fts
		JOIN notes n ON notes_fts.rowid = n.note_rowid
		WHERE notes_fts MATCH ? AND n.user_id = ? AND n.is_deleted = 0
		  AND COALESCE(n.note_type, 'note') = 'note'
		ORDER BY rank
		LIMIT ?
	`, ftsQuery, userID, limit)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("search query timeout (5s exceeded) - please use fewer or shorter terms")
		}
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Title, &r.Snippet, &r.Rank); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		r.Snippet = parser.StripColorTags(r.Snippet)
		r.Snippet = escapeSnippetHTML(r.Snippet)
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate search results: %w", err)
	}

	return results, nil
}

// searchKeywords searches the note_keywords_fts index for keyword matches on encrypted notes.
func (db *DB) searchKeywords(ctx context.Context, userID int, ftsQuery string, limit int) ([]SearchResult, error) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT n.id, n.title, COALESCE(n.title_encrypted, 0), n.encrypted_title,
		       GROUP_CONCAT(DISTINCT nk.keyword) as matched_keywords,
		       COUNT(DISTINCT nk.keyword) as match_count
		FROM note_keywords_fts nkf
		JOIN note_keywords nk ON nkf.rowid = nk.id
		JOIN notes n ON nk.note_id = n.id
		WHERE note_keywords_fts MATCH ? AND n.user_id = ? AND n.is_deleted = 0
		  AND COALESCE(n.note_type, 'note') = 'note'
		GROUP BY n.id
		ORDER BY match_count DESC
		LIMIT ?
	`, ftsQuery, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search keywords: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var titleEncrypted int
		var encryptedTitle sql.NullString
		var matchedKeywords string
		var matchCount int
		if err := rows.Scan(&r.ID, &r.Title, &titleEncrypted, &encryptedTitle, &matchedKeywords, &matchCount); err != nil {
			return nil, fmt.Errorf("failed to scan keyword result: %w", err)
		}
		r.Encrypted = true
		r.TitleEncrypted = titleEncrypted == 1
		if encryptedTitle.Valid {
			r.EncryptedTitle = &encryptedTitle.String
		}
		// Parse matched keywords, sort for stable output, limit to 10
		if matchedKeywords != "" {
			kws := strings.Split(matchedKeywords, ",")
			sort.Strings(kws)
			if len(kws) > 10 {
				kws = kws[:10]
			}
			r.MatchedKeywords = kws
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate keyword search results: %w", err)
	}

	return results, nil
}

// mergeSearchResults combines plaintext and keyword results.
// Plaintext results come first (richer snippets), keyword results are appended.
// Duplicates are merged: plaintext snippet is kept, keywords are added.
func mergeSearchResults(plaintext, keywords []SearchResult, limit int) []SearchResult {
	seen := make(map[string]int, len(plaintext))
	merged := make([]SearchResult, 0, len(plaintext)+len(keywords))

	for _, r := range plaintext {
		seen[r.ID] = len(merged)
		merged = append(merged, r)
	}

	for _, r := range keywords {
		if idx, exists := seen[r.ID]; exists {
			// Duplicate: keep plaintext snippet, add matched keywords
			merged[idx].MatchedKeywords = r.MatchedKeywords
		} else {
			merged = append(merged, r)
		}
	}

	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged
}

// QuickSearch performs a fast title-only search for the quick switcher.
func (db *DB) QuickSearch(ctx context.Context, userID int, query string, limit int) ([]Note, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// For empty query, return recent notes (excluding journals)
	if query == "" {
		rows, err := db.QueryContext(ctx, `
			SELECT id, title, content, folder_path, version, created_at, updated_at
			FROM notes
			WHERE user_id = ? AND is_deleted = 0
			  AND COALESCE(note_type, 'note') IN ('note', 'canvas')
			ORDER BY updated_at DESC
			LIMIT ?
		`, userID, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to quick search: %w", err)
		}
		defer rows.Close()

		var notes []Note
		for rows.Next() {
			var note Note
			var createdAt, updatedAt string
			if err := rows.Scan(&note.ID, &note.Title, &note.Content, &note.FolderPath, &note.Version, &createdAt, &updatedAt); err != nil {
				return nil, fmt.Errorf("failed to scan note: %w", err)
			}
			note.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
			note.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
			notes = append(notes, note)
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("failed to iterate quick search results: %w", err)
		}

		return notes, nil
	}

	// Use LIKE for prefix matching on title (excluding journals)
	pattern := "%" + strings.ToLower(query) + "%"

	rows, err := db.QueryContext(ctx, `
		SELECT id, title, content, folder_path, version, created_at, updated_at
		FROM notes
		WHERE title_norm LIKE ? AND user_id = ? AND is_deleted = 0
		  AND COALESCE(note_type, 'note') IN ('note', 'canvas')
		ORDER BY
			CASE WHEN title_norm LIKE ? THEN 0 ELSE 1 END,
			updated_at DESC
		LIMIT ?
	`, pattern, userID, strings.ToLower(query)+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to quick search: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		var createdAt, updatedAt string
		if err := rows.Scan(&note.ID, &note.Title, &note.Content, &note.FolderPath, &note.Version, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		note.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		note.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate quick search results: %w", err)
	}

	return notes, nil
}

// SearchFilters holds optional filters for filtered search.
type SearchFilters struct {
	Query         string    // Search query for title
	Folders       []string  // Filter by folder paths (OR logic)
	Tags          []string  // Filter by tag names (AND logic)
	CreatedAfter  time.Time // Filter by creation date (after)
	CreatedBefore time.Time // Filter by creation date (before)
	UpdatedAfter  time.Time // Filter by update date (after)
	UpdatedBefore time.Time // Filter by update date (before)
}

// FilteredSearch performs a search with optional filters.
// This extends QuickSearch with folder, tag, and date filters.
func (db *DB) FilteredSearch(ctx context.Context, userID int, filters SearchFilters, limit int) ([]Note, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// Build dynamic query with filters
	var queryBuilder strings.Builder
	var args []interface{}

	// Base SELECT with optional JOIN for tags (excluding journals)
	if len(filters.Tags) > 0 {
		queryBuilder.WriteString(`
			SELECT DISTINCT n.id, n.title, n.content, n.folder_path, n.version, n.created_at, n.updated_at
			FROM notes n
			INNER JOIN note_tags nt ON nt.note_id = n.id
			INNER JOIN tags t ON t.id = nt.tag_id
			WHERE n.user_id = ? AND n.is_deleted = 0
			  AND COALESCE(n.note_type, 'note') = 'note'
		`)
		args = append(args, userID)
	} else {
		queryBuilder.WriteString(`
			SELECT id, title, content, folder_path, version, created_at, updated_at
			FROM notes n
			WHERE n.user_id = ? AND n.is_deleted = 0
			  AND COALESCE(n.note_type, 'note') = 'note'
		`)
		args = append(args, userID)
	}

	// Filter by query (title search)
	if filters.Query != "" {
		pattern := "%" + strings.ToLower(filters.Query) + "%"
		queryBuilder.WriteString(" AND title_norm LIKE ?")
		args = append(args, pattern)
	}

	// Filter by folders (OR logic: note can be in ANY of the selected folders)
	if len(filters.Folders) > 0 {
		queryBuilder.WriteString(" AND (")
		for i, folder := range filters.Folders {
			if i > 0 {
				queryBuilder.WriteString(" OR ")
			}
			queryBuilder.WriteString("n.folder_path = ?")
			args = append(args, folder)
		}
		queryBuilder.WriteString(")")
	}

	// Filter by created date
	if !filters.CreatedAfter.IsZero() {
		queryBuilder.WriteString(" AND n.created_at >= ?")
		args = append(args, filters.CreatedAfter.Format(time.RFC3339))
	}
	if !filters.CreatedBefore.IsZero() {
		queryBuilder.WriteString(" AND n.created_at <= ?")
		args = append(args, filters.CreatedBefore.Format(time.RFC3339))
	}

	// Filter by updated date
	if !filters.UpdatedAfter.IsZero() {
		queryBuilder.WriteString(" AND n.updated_at >= ?")
		args = append(args, filters.UpdatedAfter.Format(time.RFC3339))
	}
	if !filters.UpdatedBefore.IsZero() {
		queryBuilder.WriteString(" AND n.updated_at <= ?")
		args = append(args, filters.UpdatedBefore.Format(time.RFC3339))
	}

	// Filter by tags (AND logic: note must have ALL selected tags)
	if len(filters.Tags) > 0 {
		// Normalize tag names for case-insensitive matching
		normalizedTags := make([]string, len(filters.Tags))
		for i, tag := range filters.Tags {
			normalizedTags[i] = strings.ToLower(strings.TrimSpace(tag))
		}

		queryBuilder.WriteString(" AND t.user_id = ? AND t.name_norm IN (")
		args = append(args, userID)
		for i, tagNorm := range normalizedTags {
			if i > 0 {
				queryBuilder.WriteString(", ")
			}
			queryBuilder.WriteString("?")
			args = append(args, tagNorm)
		}
		queryBuilder.WriteString(")")

		// GROUP BY and HAVING to ensure ALL tags are present (AND logic)
		queryBuilder.WriteString(" GROUP BY n.id HAVING COUNT(DISTINCT t.id) = ?")
		args = append(args, len(filters.Tags))
	}

	// Order by relevance (prefix match first) and updated date
	if filters.Query != "" {
		prefixPattern := strings.ToLower(filters.Query) + "%"
		queryBuilder.WriteString(`
			ORDER BY
				CASE WHEN title_norm LIKE ? THEN 0 ELSE 1 END,
				updated_at DESC
		`)
		args = append(args, prefixPattern)
	} else {
		queryBuilder.WriteString(" ORDER BY updated_at DESC")
	}

	// Limit
	queryBuilder.WriteString(" LIMIT ?")
	args = append(args, limit)

	// Execute query
	rows, err := db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to filtered search: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		var createdAt, updatedAt string
		if err := rows.Scan(&note.ID, &note.Title, &note.Content, &note.FolderPath, &note.Version, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		note.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		note.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate filtered search results: %w", err)
	}

	return notes, nil
}
