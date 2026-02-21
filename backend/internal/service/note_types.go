package service

import (
	"time"

	"github.com/xela-io/xelanote/internal/db"
)

// Type aliases for note-related DB types.
// Allows the API layer to reference these types without importing db directly.

type Note = db.Note
type NoteVersion = db.NoteVersion
type NoteWithBacklinks = db.NoteWithBacklinks
type Backlink = db.Backlink
type GraphData = db.GraphData
type SearchFilters = db.SearchFilters
type FolderInfo = db.FolderInfo
type TaskEvent = db.TaskEvent
type Feature = db.Feature
type ListNotesOptions = db.ListNotesOptions

// IsAllowedNoteType validates whether a note type is supported.
// Empty means default note type.
func IsAllowedNoteType(noteType string) bool {
	if noteType == "" || noteType == "note" {
		return true
	}
	return db.AllowedNoteTypes[noteType]
}

// ParseSyncToken parses a sync token of the format "RFC3339Nano|UUID" into a timestamp and ID.
func ParseSyncToken(token string) (time.Time, string, error) {
	return db.ParseSyncToken(token)
}

// HighWatermark computes the sync_token from a result set.
// It returns the newest (updated_at, id) tuple regardless of sort direction.
func HighWatermark(notes []Note, ascending bool) string {
	return db.HighWatermark(notes, ascending)
}
