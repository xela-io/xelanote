// Package service contains the business logic for xelanote.
package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// normalizeFolderPath ensures consistent folder path format.
// Removes trailing slashes (except for root "/").
func normalizeFolderPath(path string) string {
	if path == "" || path == "/" {
		return "/"
	}
	return strings.TrimSuffix(path, "/")
}

// ErrNoteLimitExceeded is returned when user has reached their note limit
var ErrNoteLimitExceeded = errors.New("note limit exceeded")

// checkNoteLimit returns ErrNoteLimitExceeded when the user has reached the
// configured maximum number of notes. A limit of 0 means unlimited.
func (s *NoteService) checkNoteLimit(userID int) error {
	maxNotes, err := s.db.GetMaxNotesPerUser()
	if err != nil {
		return err
	}
	if maxNotes == 0 {
		return nil
	}
	currentCount, err := s.db.GetNoteCountForUser(userID)
	if err != nil {
		return err
	}
	if currentCount >= maxNotes {
		return ErrNoteLimitExceeded
	}
	return nil
}

// journalDateRegex validates YYYY-MM-DD format
var journalDateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// ValidateJournalDate validates the journal date format (YYYY-MM-DD) and checks if the date is valid.
func ValidateJournalDate(date string) error {
	if !journalDateRegex.MatchString(date) {
		return errors.New("invalid date format, expected YYYY-MM-DD")
	}
	// Parse to verify the date is valid (e.g., 2024-02-30 is invalid)
	_, err := time.Parse("2006-01-02", date)
	return err
}

func noteCacheKey(userID int, noteID string) string {
	return fmt.Sprintf("cache:note:%d:%s", userID, noteID)
}

func notesByFolderCacheKey(userID int, path string) string {
	return fmt.Sprintf("cache:notes:folder:%d:%s", userID, path)
}

func quickSearchCacheKey(userID int, query string, limit int) string {
	normalized := strings.ToLower(strings.TrimSpace(query))
	return fmt.Sprintf("cache:notes:quick:%d:%d:%s", userID, limit, normalized)
}

// snapshotThreshold is the minimum time between snapshots (5 minutes).
const snapshotThreshold = 5 * time.Minute
