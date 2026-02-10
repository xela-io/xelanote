// Package service contains the business logic for xelanote.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/xela-io/xelanote/internal/cache"
	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/parser"
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

// DeleteNote soft-deletes a note.
func (s *NoteService) DeleteNote(userID int, id string) error {
	note, err := s.db.GetNote(userID, id)
	if err != nil {
		return err
	}

	if err := s.db.DeleteNote(userID, id); err != nil {
		return err
	}

	s.invalidateNoteCache(userID, id)
	if note != nil {
		s.invalidateNotesByFolderCache(userID, note.FolderPath)
	}
	s.invalidateQuickSearchCache(userID)
	if s.graphService != nil {
		s.graphService.InvalidateGraphCache(userID)
	}
	return nil
}

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

// Validation limits for client-submitted links
const (
	MaxLinksPerNote    = 500 // Maximum number of links per note
	MaxLinkTitleLength = 200 // Maximum characters per link title
)

// ErrTooManyLinks is returned when a note has more links than allowed
var ErrTooManyLinks = errors.New("too many links (max 500)")

// UpdateLinksFromClient processes client-submitted link titles and updates the links tables.
// This is used for E2E encrypted notes where the server cannot parse the content.
// Links are validated, deduplicated, and resolved against existing notes.
func (s *NoteService) UpdateLinksFromClient(userID int, noteID string, linkTitles []string) error {
	// Validate total count
	if len(linkTitles) > MaxLinksPerNote {
		return ErrTooManyLinks
	}

	var resolvedIDs []string
	var unresolvedRefs []string

	seen := make(map[string]bool)
	for _, title := range linkTitles {
		// Skip empty titles
		if title == "" {
			continue
		}

		// Skip titles that are too long (silently skip instead of error)
		if len(title) > MaxLinkTitleLength {
			s.logger.Warn("skipping link with title too long",
				"note_id", noteID,
				"title_length", len(title),
				"max_length", MaxLinkTitleLength)
			continue
		}

		titleNorm := parser.NormalizeTitle(title)
		if titleNorm == "" || seen[titleNorm] {
			continue
		}
		seen[titleNorm] = true

		// Try to find the target note (within user's notes only)
		targetNote, err := s.db.GetNoteByTitle(userID, title)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return fmt.Errorf("failed to lookup note %q: %w", title, err)
		}

		if targetNote != nil {
			resolvedIDs = append(resolvedIDs, targetNote.ID)
		} else {
			unresolvedRefs = append(unresolvedRefs, title)
		}
	}

	return s.db.SetLinks(noteID, resolvedIDs, unresolvedRefs)
}

// GetDueDatesByUser returns all due dates for a user across all non-deleted notes.
func (s *NoteService) GetDueDatesByUser(userID int, showCompleted bool) ([]db.DueDateWithNote, error) {
	return s.db.GetDueDatesByUser(userID, showCompleted)
}

// updateDueDates parses content and updates the note_due_dates table.
func (s *NoteService) updateDueDates(userID int, noteID, content string) {
	dueDates := parser.ParseDueDates(content)
	if err := s.db.SetNoteDueDates(noteID, userID, dueDates); err != nil {
		s.logger.Error("failed to update due dates", "err", err, "note_id", noteID, "user_id", userID)
	}
}

// updateLinks parses content and updates the links/unresolved_links tables.
func (s *NoteService) updateLinks(userID int, noteID, content string) error {
	result := parser.Parse(content)

	var resolvedIDs []string
	var unresolvedRefs []string

	seen := make(map[string]bool)
	for _, link := range result.Links {
		titleNorm := parser.NormalizeTitle(link.TargetTitle)
		if seen[titleNorm] {
			continue
		}
		seen[titleNorm] = true

		// Try to find the target note (within user's notes only)
		targetNote, err := s.db.GetNoteByTitle(userID, link.TargetTitle)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return fmt.Errorf("failed to lookup note %q: %w", link.TargetTitle, err)
		}

		if targetNote != nil {
			resolvedIDs = append(resolvedIDs, targetNote.ID)
		} else {
			unresolvedRefs = append(unresolvedRefs, link.TargetTitle)
		}
	}

	return s.db.SetLinks(noteID, resolvedIDs, unresolvedRefs)
}

// resolveUnresolvedLinks checks if a new note resolves any pending unresolved links.
// For encrypted notes (where content is not parseable), this directly converts unresolved
// links to resolved links without re-parsing the source note's content.
func (s *NoteService) resolveUnresolvedLinks(userID int, note *db.Note) error {
	// Find notes that have unresolved links to this note's title
	backlinks, err := s.db.GetUnresolvedBacklinks(userID, note.Title)
	if err != nil {
		return err
	}

	// For each note with an unresolved link, resolve it directly
	for _, bl := range backlinks {
		sourceNote, err := s.db.GetNote(userID, bl.ID)
		if err != nil || sourceNote == nil {
			continue
		}

		// If the source note is encrypted (content is empty), we can't re-parse it.
		// Instead, directly convert the unresolved link to a resolved link.
		if sourceNote.ContentEncrypted || sourceNote.Content == "" {
			if err := s.db.ResolveUnresolvedLink(bl.ID, note.Title, note.ID); err != nil {
				s.logger.Error("failed to resolve unresolved link directly",
					"err", err, "source_id", bl.ID, "target_id", note.ID)
			}
		} else {
			// For plaintext notes, re-parse content to update all links
			if err := s.updateLinks(userID, sourceNote.ID, sourceNote.Content); err != nil {
				s.logger.Error("failed to update links for unresolved backlink",
					"err", err, "note_id", sourceNote.ID, "user_id", userID)
			}
		}
	}

	return nil
}

// RenameNote renames a note and updates all links.
func (s *NoteService) RenameNote(ctx context.Context, userID int, noteID, newTitle string) (map[string]interface{}, error) {
	// Get the note first to check ownership and get old title
	note, err := s.db.GetNote(userID, noteID)
	if err != nil {
		return nil, err
	}
	if note == nil {
		return nil, db.ErrNotFound
	}

	// Journal notes cannot be renamed (their identity is tied to the date)
	if note.NoteType == db.NoteTypeJournal {
		return nil, errors.New("cannot rename journal entries")
	}

	oldTitle := note.Title

	// Update the note title
	_, err = s.db.UpdateNoteTitle(userID, noteID, newTitle, note.Version)
	if err != nil {
		return nil, err
	}

	// Find all notes that link to this note (by ID or by old title)
	linkingNoteIDs, err := s.db.GetNotesLinkingTo(userID, noteID, oldTitle)
	if err != nil {
		return nil, fmt.Errorf("failed to get notes linking to: %w", err)
	}

	// OPTIMIZATION: Fetch all linking notes in a single batch query (fixes N+1 problem)
	linkingNotes, err := s.db.GetNotesByIDs(userID, linkingNoteIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get linking notes in batch: %w", err)
	}

	updatedNotes := []string{}
	updatedFolderPaths := map[string]struct{}{}
	for _, linkingID := range linkingNoteIDs {
		sourceNote, ok := linkingNotes[linkingID]
		if !ok || sourceNote == nil {
			continue
		}

		// Replace old title with new title in wikilinks
		newContent := replaceWikilinkTitle(sourceNote.Content, oldTitle, newTitle)
		if newContent != sourceNote.Content {
			updatedSourceNote, err := s.db.UpdateNote(userID, sourceNote.ID, sourceNote.Title, newContent, "", sourceNote.Version)
			if err != nil {
				// Log but continue
				s.logger.Error("failed to update backlink content after rename", "err", err, "note_id", sourceNote.ID, "user_id", userID)
				continue
			}

			// Reprocess links for this note
			s.updateLinks(userID, sourceNote.ID, newContent)
			updatedNotes = append(updatedNotes, sourceNote.Title)

			if updatedSourceNote != nil {
				s.cache.Set(noteCacheKey(userID, updatedSourceNote.ID), updatedSourceNote)
				updatedFolderPaths[updatedSourceNote.FolderPath] = struct{}{}
			}
		}
	}

	// Reload the renamed note to get updated data (including new version)
	updatedNote, err := s.db.GetNote(userID, noteID)
	if err != nil || updatedNote == nil {
		return nil, db.ErrNotFound
	}

	s.cache.Set(noteCacheKey(userID, noteID), updatedNote)
	s.invalidateNotesByFolderCache(userID, updatedNote.FolderPath)
	for path := range updatedFolderPaths {
		s.invalidateNotesByFolderCache(userID, path)
	}
	s.invalidateQuickSearchCache(userID)
	if s.graphService != nil {
		s.graphService.InvalidateGraphCache(userID)
	}

	return map[string]interface{}{
		"note":               updatedNote,
		"updated_note_count": len(updatedNotes),
	}, nil
}

// replaceWikilinkTitle replaces all occurrences of [[oldTitle]] with [[newTitle]].
func replaceWikilinkTitle(content, oldTitle, newTitle string) string {
	// Simple replacement - could be improved with proper parsing
	oldLink := "[[" + oldTitle + "]]"
	newLink := "[[" + newTitle + "]]"
	return strings.ReplaceAll(content, oldLink, newLink)
}

// GetNoteByTitle finds note by title (case-insensitive)
func (s *NoteService) GetNoteByTitle(userID int, title string) (*db.Note, error) {
	return s.db.GetNoteByTitle(userID, title)
}

// GetNoteByTitleInFolder retrieves a note by title within a specific folder.
func (s *NoteService) GetNoteByTitleInFolder(userID int, title, folderPath string) (*db.Note, error) {
	return s.db.GetNoteByTitleInFolder(userID, title, folderPath)
}

// --- Trash Management Functions ---

// ListDeletedNotes returns a paginated list of soft-deleted notes.
func (s *NoteService) ListDeletedNotes(userID int, limit int, cursor string) ([]db.Note, string, error) {
	return s.db.ListDeletedNotes(userID, limit, cursor)
}

// RestoreNote restores a soft-deleted note and reprocesses its links.
func (s *NoteService) RestoreNote(userID int, id string) (*db.Note, error) {
	// Attempt to restore the note
	note, err := s.db.RestoreNote(userID, id)
	if err != nil {
		// Check if error is due to UNIQUE constraint conflict
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			// Note: We can't determine the exact title/folder without querying the deleted note
			// So we provide a generic but helpful message
			return nil, fmt.Errorf("kann nicht wiederherstellen: Eine Notiz mit diesem Titel existiert bereits im Zielordner")
		}
		return nil, err
	}

	// Reprocess links in the restored note
	if err := s.updateLinks(userID, note.ID, note.Content); err != nil {
		// Log but don't fail - note is already restored
	}
	s.updateDueDates(userID, note.ID, note.Content)

	// Check if this note resolves any unresolved links
	if err := s.resolveUnresolvedLinks(userID, note); err != nil {
		// Log but don't fail
	}

	s.cache.Set(noteCacheKey(userID, note.ID), note)
	s.invalidateNotesByFolderCache(userID, note.FolderPath)
	s.invalidateQuickSearchCache(userID)

	return note, nil
}

// PermanentlyDeleteNote performs a hard delete on a note.
func (s *NoteService) PermanentlyDeleteNote(userID int, id string) error {
	if err := s.db.PermanentlyDeleteNote(userID, id); err != nil {
		return err
	}

	s.invalidateNoteCache(userID, id)
	s.invalidateQuickSearchCache(userID)
	return nil
}

// GetDeletedNotesCount returns the count of soft-deleted notes.
func (s *NoteService) GetDeletedNotesCount(userID int) (int, error) {
	return s.db.GetDeletedNotesCount(userID)
}

// EmptyTrash permanently deletes all soft-deleted notes.
func (s *NoteService) EmptyTrash(userID int) (int, error) {
	return s.db.EmptyTrash(userID)
}

func (s *NoteService) invalidateNoteCache(userID int, noteID string) {
	s.cache.Delete(noteCacheKey(userID, noteID))
}

// --- Version History Functions ---

// GetNoteVersions returns a paginated list of versions for a note.
func (s *NoteService) GetNoteVersions(userID int, noteID string, limit int, cursor string) ([]db.NoteVersion, string, int, error) {
	return s.db.GetNoteVersions(userID, noteID, limit, cursor)
}

// GetNoteVersion retrieves a specific version of a note.
func (s *NoteService) GetNoteVersion(userID int, noteID string, version int) (*db.NoteVersion, error) {
	return s.db.GetNoteVersion(userID, noteID, version)
}

// RestoreVersion restores a note to a previous version.
// This is non-destructive: the current state is saved as a snapshot first.
// Handles 4 scenarios: plaintext→plaintext, encrypted→encrypted, plaintext→encrypted, encrypted→plaintext.
func (s *NoteService) RestoreVersion(userID int, noteID string, targetVersion, currentVersion int) (*db.Note, error) {
	// 1. Get current note and verify optimistic lock
	current, err := s.db.GetNote(userID, noteID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, db.ErrNotFound
	}
	if current.Version != currentVersion {
		return nil, db.ErrVersionMismatch
	}

	// 2. Get the target version to restore
	target, err := s.db.GetNoteVersion(userID, noteID, targetVersion)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, db.ErrNotFound
	}

	// 3. Save current state as a snapshot (non-destructive restore)
	// Choose snapshot method based on CURRENT note's encryption state
	if current.ContentEncrypted {
		if err := s.db.CreateEncryptedNoteVersion(
			userID,
			noteID,
			current.Version,
			current.Title,
			current.EncryptedTitle,
			current.TitleEncrypted,
			current.EncryptedContent,
			current.WrappedDEK,
			current.EncryptionVersion,
		); err != nil {
			s.logger.Error("failed to create encrypted snapshot before restore",
				"err", err, "note_id", noteID, "user_id", userID)
			// Continue anyway - restore is more important
		}
	} else {
		if err := s.db.CreateNoteVersion(userID, noteID, current.Version, current.Title, current.Content); err != nil {
			s.logger.Error("failed to create snapshot before restore",
				"err", err, "note_id", noteID, "user_id", userID)
			// Continue anyway - restore is more important
		}
	}

	// 4. Update the note with the target version's content
	// Choose update method based on TARGET version's encryption state
	if target.ContentEncrypted {
		// Validate encrypted version has required fields
		if len(target.EncryptedContent) == 0 || target.WrappedDEK == "" {
			return nil, errors.New("cannot restore: encrypted version is corrupted (missing encrypted content or DEK)")
		}

		// Reconstruct encryption_metadata from constants + wrapped_dek
		// Note: encryption_metadata is not stored in versions, so we rebuild it
		// using the standard encryption parameters (XChaCha20-Poly1305, Argon2id)
		encryptionMetadata := fmt.Sprintf(
			`{"version":2,"algorithm":"XChaCha20-Poly1305","kdf":"Argon2id","kdf_strength":"interactive","nonce_bytes":24,"wrapped_dek":"%s"}`,
			target.WrappedDEK,
		)

		return s.UpdateEncryptedNote(
			userID,
			noteID,
			target.Title,
			target.EncryptedTitle,
			target.TitleEncrypted,
			target.EncryptedContent,
			target.WrappedDEK,
			encryptionMetadata,
			current.FolderPath,
			nil, // keywords - not stored in versions
			currentVersion,
		)
	}

	// Restore to plaintext version
	return s.UpdateNote(userID, noteID, target.Title, target.Content, current.FolderPath, currentVersion)
}

// PruneAllVersions removes old versions for all notes, keeping the most recent keepCount versions per note.
// Returns the total number of deleted versions.
func (s *NoteService) PruneAllVersions(keepCount int) (int, error) {
	userIDs, err := s.db.GetUsersWithVersions()
	if err != nil {
		return 0, err
	}

	totalPruned := 0
	for _, userID := range userIDs {
		pruned, err := s.db.PruneAllUserVersions(userID, keepCount)
		if err != nil {
			s.logger.Error("prune failed", "user", userID, "err", err)
			continue
		}
		totalPruned += pruned
	}
	return totalPruned, nil
}

func (s *NoteService) invalidateNotesByFolderCache(userID int, path string) {
	if path == "" {
		return
	}
	s.cache.Delete(notesByFolderCacheKey(userID, normalizeFolderPath(path)))
}

func (s *NoteService) invalidateQuickSearchCache(userID int) {
	s.cache.DeleteByPrefix(fmt.Sprintf("cache:notes:quick:%d:", userID))
}

// --- Tag Management Functions ---

// GetAllTags returns all tags for a user.
func (s *NoteService) GetAllTags(userID int) ([]db.Tag, error) {
	return s.db.GetAllTags(userID)
}

// GetNoteTags returns all tags for a specific note.
func (s *NoteService) GetNoteTags(noteID string) ([]db.Tag, error) {
	return s.db.GetNoteTags(noteID)
}

// SetNoteTags sets the tags for a note, replacing any existing tags.
func (s *NoteService) SetNoteTags(noteID string, userID int, tagNames []string) error {
	if err := s.db.SetNoteTags(noteID, userID, tagNames); err != nil {
		return err
	}

	// Invalidate cache for the note
	s.invalidateNoteCache(userID, noteID)
	s.invalidateQuickSearchCache(userID)

	return nil
}

// DeleteTag deletes a tag and all its associations.
func (s *NoteService) DeleteTag(userID int, tagID int) error {
	if err := s.db.DeleteTag(userID, tagID); err != nil {
		return err
	}

	// Invalidate search cache since tag associations changed
	s.invalidateQuickSearchCache(userID)

	return nil
}

// CreateEncryptedNote creates a new encrypted note with optional keywords
func (s *NoteService) CreateEncryptedNote(
	userID int,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	keywords []string,
	folderPath string,
) (*db.Note, error) {
	// Validate
	if len(encryptedContent) == 0 || wrappedDEK == "" {
		return nil, errors.New("encrypted content and wrapped DEK required")
	}

	// Create note in DB
	note, err := s.db.CreateEncryptedNote(
		userID,
		title,
		encryptedTitle,
		titleEncrypted,
		encryptedContent,
		wrappedDEK,
		encryptionMetadata,
		folderPath,
	)
	if err != nil {
		return nil, err
	}

	// Insert keywords ONLY if user has keywords enabled
	if len(keywords) > 0 {
		prefs, err := s.db.GetUserPreferences(userID)
		if err == nil && prefs.KeywordsEnabled {
			for _, kw := range keywords {
				if err := s.db.InsertNoteKeyword(note.ID, kw); err != nil {
					s.logger.Warn("failed to insert keyword", "error", err)
				}
			}
		} else if len(keywords) > 0 {
			s.logger.Warn("keywords sent but user has keywords disabled", "userID", userID)
		}
	}

	// Invalidate caches
	s.invalidateQuickSearchCache(userID)

	return note, nil
}

// CreateJournalNote creates a new plaintext journal note for a specific date.
func (s *NoteService) CreateJournalNote(userID int, title, content, folderPath, journalDate string) (*db.Note, error) {
	// Check note limit
	maxNotes, err := s.db.GetMaxNotesPerUser()
	if err != nil {
		return nil, err
	}
	if maxNotes > 0 {
		currentCount, err := s.db.GetNoteCountForUser(userID)
		if err != nil {
			return nil, err
		}
		if currentCount >= maxNotes {
			return nil, ErrNoteLimitExceeded
		}
	}

	// Validate journal date
	if err := ValidateJournalDate(journalDate); err != nil {
		return nil, err
	}

	// Check feature enabled
	feature, err := s.db.GetUserFeature(userID, "journal")
	if err != nil {
		return nil, err
	}
	if !feature.Enabled {
		return nil, errors.New("journal feature not enabled")
	}

	// Create journal note (also creates /Journal folder if needed)
	note, err := s.db.CreateJournalNote(userID, title, content, folderPath, journalDate)
	if err != nil {
		return nil, err
	}

	// Parse and save links
	s.updateLinks(userID, note.ID, content)
	s.updateDueDates(userID, note.ID, content)

	// Invalidate caches (including folder cache since Journal folder may have been created)
	s.invalidateFolderCache(userID)
	s.invalidateQuickSearchCache(userID)
	s.invalidateNotesByFolderCache(userID, folderPath)
	if s.graphService != nil {
		s.graphService.InvalidateGraphCache(userID)
	}

	return note, nil
}

// CreateEncryptedJournalNote creates a new encrypted journal note for a specific date.
func (s *NoteService) CreateEncryptedJournalNote(
	userID int,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	keywords []string,
	folderPath string,
	journalDate string,
) (*db.Note, error) {
	// Check note limit
	maxNotes, err := s.db.GetMaxNotesPerUser()
	if err != nil {
		return nil, err
	}
	if maxNotes > 0 {
		currentCount, err := s.db.GetNoteCountForUser(userID)
		if err != nil {
			return nil, err
		}
		if currentCount >= maxNotes {
			return nil, ErrNoteLimitExceeded
		}
	}

	// Validate journal date
	if err := ValidateJournalDate(journalDate); err != nil {
		return nil, err
	}

	// Check feature enabled
	feature, err := s.db.GetUserFeature(userID, "journal")
	if err != nil {
		return nil, err
	}
	if !feature.Enabled {
		return nil, errors.New("journal feature not enabled")
	}

	// Validate encryption fields
	if len(encryptedContent) == 0 || wrappedDEK == "" {
		return nil, errors.New("encrypted content and wrapped DEK required")
	}

	// Create encrypted journal note (also creates /Journal folder if needed)
	note, err := s.db.CreateEncryptedJournalNote(
		userID, title, encryptedTitle, titleEncrypted,
		encryptedContent, wrappedDEK, encryptionMetadata,
		folderPath, journalDate,
	)
	if err != nil {
		return nil, err
	}

	// Insert keywords if user has keywords enabled
	if len(keywords) > 0 {
		prefs, err := s.db.GetUserPreferences(userID)
		if err == nil && prefs.KeywordsEnabled {
			for _, kw := range keywords {
				if err := s.db.InsertNoteKeyword(note.ID, kw); err != nil {
					s.logger.Warn("failed to insert keyword", "error", err)
				}
			}
		}
	}

	// Invalidate caches (including folder cache since Journal folder may have been created)
	s.invalidateFolderCache(userID)
	s.invalidateQuickSearchCache(userID)
	s.invalidateNotesByFolderCache(userID, folderPath)
	if s.graphService != nil {
		s.graphService.InvalidateGraphCache(userID)
	}

	return note, nil
}

// UpdateEncryptedNote updates an existing note with encrypted content
func (s *NoteService) UpdateEncryptedNote(
	userID int,
	noteID string,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	folderPath string,
	keywords []string,
	expectedVersion int,
) (*db.Note, error) {
	// Validate
	if len(encryptedContent) == 0 || wrappedDEK == "" {
		return nil, errors.New("encrypted content and wrapped DEK required")
	}

	// Get existing note for version snapshotting
	existingNote, err := s.db.GetNote(userID, noteID)
	if err != nil {
		return nil, err
	}
	if existingNote == nil {
		return nil, db.ErrNotFound
	}

	// Check if we should create a snapshot
	// For encrypted notes, we check if encrypted content or wrapped DEK changed
	// (we can't compare plaintext content since it's encrypted)
	shouldSnapshot := false
	if existingNote.ContentEncrypted {
		// Always snapshot encrypted notes when updated (conservative approach)
		// Could be optimized by comparing encrypted_content bytes, but that's fragile
		lastSnapshot, err := s.db.GetLatestVersionSnapshot(userID, noteID)
		if err != nil {
			s.logger.Warn("failed to load latest snapshot", "err", err, "note_id", noteID, "user_id", userID)
		}
		shouldSnapshot = err != nil || lastSnapshot == nil ||
			time.Since(lastSnapshot.SnapshotAt) > snapshotThreshold
	}

	if shouldSnapshot {
		// Create snapshot of the state BEFORE the update
		if err := s.db.CreateEncryptedNoteVersion(
			userID,
			noteID,
			existingNote.Version,
			existingNote.Title,
			existingNote.EncryptedTitle,
			existingNote.TitleEncrypted,
			existingNote.EncryptedContent,
			existingNote.WrappedDEK,
			existingNote.EncryptionVersion,
		); err != nil {
			// Log but don't fail - note update is more important
			s.logger.Error("failed to create encrypted version snapshot", "err", err, "note_id", noteID, "user_id", userID)
		}
	}

	// Update note in DB
	note, err := s.db.UpdateEncryptedNote(
		userID,
		noteID,
		title,
		encryptedTitle,
		titleEncrypted,
		encryptedContent,
		wrappedDEK,
		encryptionMetadata,
		folderPath,
		expectedVersion,
	)
	if err != nil {
		return nil, err
	}

	// Update keywords if provided
	if len(keywords) > 0 {
		// Delete old keywords
		if err := s.db.DeleteNoteKeywords(noteID); err != nil {
			s.logger.Warn("failed to delete old keywords", "error", err)
		}

		// Insert new keywords if user has enabled keyword extraction
		prefs, err := s.db.GetUserPreferences(userID)
		if err == nil && prefs.KeywordsEnabled {
			for _, kw := range keywords {
				if err := s.db.InsertNoteKeyword(noteID, kw); err != nil {
					s.logger.Warn("failed to insert keyword", "error", err)
				}
			}
		}
	}

	// Business rule: encrypting a note removes all shares
	if err := s.db.DeleteAllSharesForNote(noteID); err != nil {
		s.logger.Error("failed to remove shares after encryption", "err", err, "note_id", noteID, "user_id", userID)
	} else {
		s.logger.Info("note encrypted, shares removed", "note_id", noteID, "user_id", userID)
	}

	// Business rule: encrypting a recipe note deletes metadata + ingredients
	// (data is serialized into encrypted payload by the frontend)
	if existingNote.NoteType == db.NoteTypeRecipe {
		if err := s.db.DeleteRecipeData(noteID); err != nil {
			s.logger.Error("failed to delete recipe data after encryption", "err", err, "note_id", noteID, "user_id", userID)
		} else {
			s.logger.Info("recipe encrypted, metadata+ingredients removed", "note_id", noteID, "user_id", userID)
		}
	}

	// Invalidate caches
	s.invalidateNoteCache(userID, noteID)
	s.invalidateQuickSearchCache(userID)

	return note, nil
}

// DecryptNote decrypts a note by clearing all encryption fields and setting plaintext content.
// The client sends the already-decrypted title and content.
// Creates a version snapshot before decryption for undo capability.
func (s *NoteService) DecryptNote(userID int, id, title, content string, expectedVersion int) (*db.Note, error) {
	// Get existing note
	existingNote, err := s.db.GetNote(userID, id)
	if err != nil {
		return nil, err
	}
	if existingNote == nil {
		return nil, db.ErrNotFound
	}

	// Validate that note is currently encrypted
	if !existingNote.ContentEncrypted {
		return nil, fmt.Errorf("note is not encrypted")
	}

	// Create version snapshot before decryption
	lastSnapshot, err := s.db.GetLatestVersionSnapshot(userID, id)
	if err != nil {
		s.logger.Warn("failed to load latest snapshot", "err", err, "note_id", id, "user_id", userID)
	}
	shouldSnapshot := err != nil || lastSnapshot == nil ||
		time.Since(lastSnapshot.SnapshotAt) > snapshotThreshold

	if shouldSnapshot {
		if existingNote.ContentEncrypted {
			if err := s.db.CreateEncryptedNoteVersion(
				userID, id, existingNote.Version,
				existingNote.Title,
				existingNote.EncryptedTitle,
				existingNote.TitleEncrypted,
				existingNote.EncryptedContent,
				existingNote.WrappedDEK,
				existingNote.EncryptionVersion,
			); err != nil {
				s.logger.Error("failed to create snapshot before decrypt", "err", err, "note_id", id, "user_id", userID)
			}
		}
	}

	// Decrypt note in DB
	note, err := s.db.DecryptNote(userID, id, title, content, expectedVersion)
	if err != nil {
		return nil, err
	}

	// Reprocess links on the plaintext content
	if err := s.updateLinks(userID, id, content); err != nil {
		s.logger.Error("failed to update links after decrypt", "err", err, "note_id", id, "user_id", userID)
	}
	s.updateDueDates(userID, id, content)

	s.logger.Info("note decrypted", "note_id", id, "user_id", userID)

	// Invalidate caches
	s.invalidateNoteCache(userID, id)
	s.invalidateNotesByFolderCache(userID, note.FolderPath)
	s.invalidateQuickSearchCache(userID)
	if s.graphService != nil {
		s.graphService.InvalidateGraphCache(userID)
	}

	return note, nil
}

// DecryptRecipeNote decrypts a recipe note and optionally restores recipe data.
// If recipeMetadata is provided, it's restored. If recipeIngredients is provided, they're restored.
// If neither is provided, the note type remains 'recipe' but with default metadata (I5 fallback).
func (s *NoteService) DecryptRecipeNote(userID int, id, title, content string, expectedVersion int,
	recipeMetadata *db.RecipeMetadata, recipeIngredients []db.RecipeIngredient) (*db.Note, error) {

	// First decrypt the note normally
	note, err := s.DecryptNote(userID, id, title, content, expectedVersion)
	if err != nil {
		return nil, err
	}

	// Restore recipe data if provided
	if recipeMetadata != nil {
		if err := s.db.SetRecipeMetadata(id, userID, recipeMetadata, ""); err != nil {
			s.logger.Error("failed to restore recipe metadata after decrypt",
				"err", err, "note_id", id, "user_id", userID)
		} else {
			s.logger.Info("recipe metadata restored after decrypt", "note_id", id)
		}

		if len(recipeIngredients) > 0 {
			// Get the newly created metadata's updated_at for optimistic lock
			meta, err := s.db.GetRecipeMetadata(id)
			if err == nil && meta != nil {
				if err := s.db.SetRecipeIngredients(id, userID, recipeIngredients, meta.UpdatedAt); err != nil {
					s.logger.Error("failed to restore recipe ingredients after decrypt",
						"err", err, "note_id", id, "user_id", userID)
				} else {
					s.logger.Info("recipe ingredients restored after decrypt",
						"note_id", id, "count", len(recipeIngredients))
				}
			}
		}
	}

	return note, nil
}

// DEKUpdate represents a single DEK re-encryption update
type DEKUpdate struct {
	NoteID     string
	WrappedDEK string
}

// BatchUpdateWrappedDEKs updates wrapped DEKs for multiple notes in a transaction
// Used after password changes to re-wrap all DEKs with the new KEK
func (s *NoteService) BatchUpdateWrappedDEKs(userID int, updates []struct {
	NoteID     string `json:"note_id"`
	WrappedDEK string `json:"wrapped_dek"`
}) (int, error) {
	if len(updates) == 0 {
		return 0, nil
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	updatedCount := 0
	for _, update := range updates {
		// Verify note belongs to user
		note, err := s.db.GetNote(userID, update.NoteID)
		if err != nil {
			return 0, fmt.Errorf("failed to get note %s: %w", update.NoteID, err)
		}
		if note == nil {
			return 0, fmt.Errorf("note %s not found or unauthorized", update.NoteID)
		}

		// Only update if note is encrypted
		if !note.ContentEncrypted {
			s.logger.Warn("skipping non-encrypted note in DEK batch update", "note_id", update.NoteID)
			continue
		}

		// Update wrapped_dek
		_, err = tx.Exec(`
			UPDATE notes
			SET wrapped_dek = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ? AND user_id = ?
		`, update.WrappedDEK, update.NoteID, userID)
		if err != nil {
			return 0, fmt.Errorf("failed to update DEK for note %s: %w", update.NoteID, err)
		}

		updatedCount++

		// Invalidate cache for this note
		s.cache.Delete(noteCacheKey(userID, update.NoteID))
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate folder caches (since updated_at changed)
	// We don't know which folders were affected, so we invalidate the whole cache
	s.invalidateQuickSearchCache(userID)

	s.logger.Info("batch DEK re-encryption completed", "user_id", userID, "updated_count", updatedCount)

	return updatedCount, nil
}

// UserHasEncryptedNotes checks if a user has any encrypted notes
// This is used to prevent accidental salt regeneration which would make existing encrypted notes unreadable
func (s *NoteService) UserHasEncryptedNotes(userID int) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*)
		FROM notes
		WHERE user_id = ?
		  AND deleted_at IS NULL
		  AND encrypted_content IS NOT NULL
		  AND encrypted_content != ''
	`, userID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// --- AI-Enabled Functions (Claude API Opt-In) ---

// UpdateNoteAIEnabled sets the ai_enabled flag for a note.
// When ai_enabled=true, Claude API features are allowed for this note.
func (s *NoteService) UpdateNoteAIEnabled(userID int, noteID string, aiEnabled bool) error {
	if err := s.db.UpdateNoteAIEnabled(userID, noteID, aiEnabled); err != nil {
		return err
	}

	// Invalidate caches
	s.invalidateNoteCache(userID, noteID)
	s.invalidateQuickSearchCache(userID)

	return nil
}

// GetNoteAIEnabled returns whether Claude API is enabled for a note.
func (s *NoteService) GetNoteAIEnabled(userID int, noteID string) (bool, error) {
	return s.db.GetNoteAIEnabled(userID, noteID)
}

// GetNoteTitlesAIEnabled returns titles of notes with ai_enabled=true for a user.
// Used for Claude API link suggestions.
func (s *NoteService) GetNoteTitlesAIEnabled(userID int) ([]string, error) {
	return s.db.GetNoteTitlesAIEnabled(userID)
}

// --- Encryption Default Functions ---
