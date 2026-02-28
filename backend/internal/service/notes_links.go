// Package service contains the business logic for xelanote.
package service

import (
	"errors"
	"fmt"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/parser"
)

// Validation limits for client-submitted links.
const (
	MaxLinksPerNote    = 500 // Maximum number of links per note
	MaxLinkTitleLength = 200 // Maximum characters per link title
)

// ErrTooManyLinks is returned when a note has more links than allowed.
var ErrTooManyLinks = errors.New("too many links (max 500)")

func (s *NoteService) getOwnedNoteForMetadataUpdate(userID int, noteID string) (*db.Note, error) {
	note, err := s.db.GetNote(userID, noteID)
	if err != nil {
		return nil, err
	}
	if note == nil {
		return nil, db.ErrNotFound
	}
	return note, nil
}

// UpdateLinksFromClient processes client-submitted link titles and updates the links tables.
// This is used for E2E encrypted notes where the server cannot parse the content.
// Links are validated, deduplicated, and resolved against existing notes.
func (s *NoteService) UpdateLinksFromClient(userID int, noteID string, linkTitles []string) error {
	note, err := s.getOwnedNoteForMetadataUpdate(userID, noteID)
	if err != nil {
		return err
	}

	// Defense-in-depth: encrypted notes must never persist plaintext link metadata.
	// Always clear, regardless of any client-provided link titles.
	if note.ContentEncrypted {
		return s.db.SetLinks(noteID, nil, nil)
	}

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
		targetNote, lookupErr := s.db.GetNoteByTitle(userID, title)
		if lookupErr != nil && !errors.Is(lookupErr, db.ErrNotFound) {
			return fmt.Errorf("failed to lookup note %q: %w", title, lookupErr)
		}

		if targetNote != nil {
			resolvedIDs = append(resolvedIDs, targetNote.ID)
		} else {
			unresolvedRefs = append(unresolvedRefs, title)
		}
	}

	return s.db.SetLinks(noteID, resolvedIDs, unresolvedRefs)
}

// SetNoteDueDates sets due dates for a note from client-provided data.
// Used by the API layer for encrypted notes where the server cannot parse content.
func (s *NoteService) SetNoteDueDates(noteID string, userID int, dueDates []parser.DueDate) error {
	note, err := s.getOwnedNoteForMetadataUpdate(userID, noteID)
	if err != nil {
		return err
	}

	// Defense-in-depth: encrypted notes must never persist plaintext due-date metadata.
	// Always clear, regardless of any client-provided due-date payload.
	if note.ContentEncrypted {
		return s.db.SetNoteDueDates(noteID, userID, nil)
	}

	return s.db.SetNoteDueDates(noteID, userID, dueDates)
}

// RecordTaskEvent records a task completion/reopen event.
func (s *NoteService) RecordTaskEvent(event db.TaskEvent) error {
	return s.db.RecordTaskEvent(event)
}

// updateDueDates parses content and updates the note_due_dates table.
func (s *NoteService) updateDueDates(userID int, noteID, content string) {
	dueDates := parser.ParseDueDates(content)
	if err := s.db.SetNoteDueDates(noteID, userID, dueDates); err != nil {
		s.logger.Error("failed to update due dates", "err", err, "note_id", noteID, "user_id", userID)
	}
}

// updateCanvasLinks extracts file node references from canvas JSON content
// and updates the links/unresolved_links tables.
func (s *NoteService) updateCanvasLinks(userID int, noteID, content string) error {
	fileRefs, err := parser.ExtractCanvasFileRefs(content)
	if err != nil {
		// Invalid JSON is not fatal -- canvas might be empty/malformed during editing
		return nil
	}

	var resolvedIDs []string
	var unresolvedRefs []string

	seen := make(map[string]bool)
	for _, title := range fileRefs {
		titleNorm := parser.NormalizeTitle(title)
		if seen[titleNorm] {
			continue
		}
		seen[titleNorm] = true

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
