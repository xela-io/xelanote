package service

import "github.com/xela-io/xelanote/internal/db"

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

// IsAllowedNoteType validates whether a note type is supported.
// Empty means default note type.
func IsAllowedNoteType(noteType string) bool {
	if noteType == "" || noteType == "note" {
		return true
	}
	return db.AllowedNoteTypes[noteType]
}
