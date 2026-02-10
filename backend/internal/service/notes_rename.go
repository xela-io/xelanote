// Package service contains the business logic for xelanote.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xela-io/xelanote/internal/db"
)

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
