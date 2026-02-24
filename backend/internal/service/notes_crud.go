// Package service contains the business logic for xelanote.
package service

import (
	"fmt"
	"time"

	"github.com/xela-io/xelanote/internal/db"
)

// CreateNote creates a new note and processes its links.
func (s *NoteService) CreateNote(userID int, title, content, folderPath string) (*db.Note, error) {
	if err := s.checkNoteLimit(userID); err != nil {
		return nil, err
	}

	folderPath = normalizeFolderPath(folderPath)
	note, err := s.db.CreateNote(userID, title, content, folderPath)
	if err != nil {
		return nil, err
	}

	// Process links in the content
	if err := s.updateLinks(userID, note.ID, content); err != nil {
		// Log but don't fail - note is already created
		s.logger.Error("failed to update links after create", "err", err, "note_id", note.ID, "user_id", userID)
	}
	s.updateDueDates(userID, note.ID, content)

	// Check if this new note resolves any unresolved links
	if err := s.resolveUnresolvedLinks(userID, note); err != nil {
		// Log but don't fail
		s.logger.Error("failed to resolve unresolved links after create", "err", err, "note_id", note.ID, "user_id", userID)
	}

	s.cache.Set(noteCacheKey(userID, note.ID), note)
	s.invalidateNotesByFolderCache(userID, folderPath)
	s.invalidateQuickSearchCache(userID)
	if s.graphService != nil {
		s.graphService.InvalidateGraphCache(userID)
	}

	return note, nil
}

// UpdateNote updates a note and reprocesses its links.
// If folderPath is empty, the folder_path is not changed.
// If folderPath is provided, it must start with "/".
// Creates a version snapshot if content or title changed and enough time has passed.
func (s *NoteService) UpdateNote(userID int, id, title, content, folderPath string, version int) (*db.Note, error) {
	// Validate folderPath if provided
	if folderPath != "" && folderPath[0] != '/' {
		return nil, fmt.Errorf("folder path must start with /")
	}

	existingNote, err := s.db.GetNote(userID, id)
	if err != nil {
		return nil, err
	}
	if existingNote == nil {
		return nil, db.ErrNotFound
	}

	// Create a version snapshot if content or title changed
	if existingNote.Content != content || existingNote.Title != title {
		s.createSnapshotIfNeeded(userID, id, existingNote)
	}

	// Update wikilinks in all notes that link to this note (before writing the new title)
	if existingNote.Title != title && existingNote.NoteType != db.NoteTypeJournal {
		s.updateTitleBacklinks(userID, id, existingNote.Title, title)
	}

	// Normalize folder path (remove trailing slash)
	if folderPath != "" {
		folderPath = normalizeFolderPath(folderPath)
	}

	note, err := s.db.UpdateNote(userID, id, title, content, folderPath, version)
	if err != nil {
		return nil, err
	}

	// Canvas notes: extract links from JSON file nodes, skip markdown due-date parsing
	if existingNote.NoteType == db.NoteTypeCanvas {
		if err := s.updateCanvasLinks(userID, id, content); err != nil {
			s.logger.Error("failed to update canvas links", "err", err, "note_id", id, "user_id", userID)
		}
	} else {
		// Markdown notes: wikilink + due-date parsing
		if err := s.updateLinks(userID, id, content); err != nil {
			s.logger.Error("failed to update links after update", "err", err, "note_id", id, "user_id", userID)
		}
		s.updateDueDates(userID, id, content)
	}

	s.invalidateNoteCaches(userID, id, note, existingNote)
	return note, nil
}

// createSnapshotIfNeeded creates a version snapshot if enough time has passed since the last one.
func (s *NoteService) createSnapshotIfNeeded(userID int, id string, existingNote *db.Note) {
	lastSnapshot, err := s.db.GetLatestVersionSnapshot(userID, id)
	if err != nil {
		s.logger.Warn("failed to load latest snapshot", "err", err, "note_id", id, "user_id", userID)
	}
	shouldSnapshot := err != nil || lastSnapshot == nil ||
		time.Since(lastSnapshot.SnapshotAt) > snapshotThreshold

	if shouldSnapshot {
		if err := s.db.CreateNoteVersion(userID, id, existingNote.Version, existingNote.Title, existingNote.Content); err != nil {
			s.logger.Error("failed to create version snapshot", "err", err, "note_id", id, "user_id", userID)
		}
	}
}

// updateTitleBacklinks updates wikilinks in all notes that reference oldTitle to use newTitle.
func (s *NoteService) updateTitleBacklinks(userID int, noteID, oldTitle, newTitle string) {
	linkingNoteIDs, linkErr := s.db.GetNotesLinkingTo(userID, noteID, oldTitle)
	if linkErr != nil {
		s.logger.Error("failed to get notes linking to during title update", "err", linkErr, "note_id", noteID, "user_id", userID)
		return
	}
	if len(linkingNoteIDs) == 0 {
		return
	}

	linkingNotes, batchErr := s.db.GetNotesByIDs(userID, linkingNoteIDs)
	if batchErr != nil {
		s.logger.Error("failed to batch-load linking notes during title update", "err", batchErr, "note_id", noteID, "user_id", userID)
		return
	}

	for _, linkingID := range linkingNoteIDs {
		sourceNote, ok := linkingNotes[linkingID]
		if !ok || sourceNote == nil {
			continue
		}
		newContent := replaceWikilinkTitle(sourceNote.Content, oldTitle, newTitle)
		if newContent != sourceNote.Content {
			updatedSourceNote, updErr := s.db.UpdateNote(userID, sourceNote.ID, sourceNote.Title, newContent, "", sourceNote.Version)
			if updErr != nil {
				s.logger.Error("failed to update backlink content during title update", "err", updErr, "note_id", sourceNote.ID, "user_id", userID)
				continue
			}
			if err := s.updateLinks(userID, sourceNote.ID, newContent); err != nil {
				s.logger.Error("failed to update links after backlink content update", "err", err, "note_id", sourceNote.ID)
			}
			if updatedSourceNote != nil {
				s.cache.Set(noteCacheKey(userID, updatedSourceNote.ID), updatedSourceNote)
				s.invalidateNotesByFolderCache(userID, updatedSourceNote.FolderPath)
			}
		}
	}
}

// invalidateNoteCaches clears all relevant caches after a note update.
func (s *NoteService) invalidateNoteCaches(userID int, noteID string, note *db.Note, existingNote *db.Note) {
	s.cache.Set(noteCacheKey(userID, noteID), note)
	if existingNote != nil {
		s.invalidateNotesByFolderCache(userID, existingNote.FolderPath)
	}
	s.invalidateNotesByFolderCache(userID, note.FolderPath)
	s.invalidateQuickSearchCache(userID)
	if s.graphService != nil {
		s.graphService.InvalidateGraphCache(userID)
	}
}

// GetNote retrieves a note by ID.
func (s *NoteService) GetNote(userID int, id string) (*db.Note, error) {
	key := noteCacheKey(userID, id)
	if cached, ok := s.cache.Get(key); ok {
		return cached.(*db.Note), nil
	}

	note, err := s.db.GetNote(userID, id)
	if err != nil || note == nil {
		return note, err
	}

	s.cache.Set(key, note)
	return note, nil
}

// UpdateNoteColor updates the color of a note.
func (s *NoteService) UpdateNoteColor(userID int, noteID string, color *string) error {
	if err := s.db.UpdateNoteColor(userID, noteID, color); err != nil {
		return err
	}

	s.invalidateNoteCache(userID, noteID)
	s.invalidateQuickSearchCache(userID)
	return nil
}
