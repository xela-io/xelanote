package service

import "github.com/xela-io/xelanote/internal/db"

// Type aliases for sharing-related DB types.
// Allows the API layer to reference these types without importing db directly.

type NoteShare = db.NoteShare
type FolderShare = db.FolderShare
type SharedNote = db.SharedNote
type SharedFolder = db.SharedFolder
type UserSearchResult = db.UserSearchResult
