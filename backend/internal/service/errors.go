package service

import "github.com/xela-io/xelanote/internal/db"

// Re-export database errors so the API layer doesn't need to import db directly.
var (
	ErrNotFound          = db.ErrNotFound
	ErrVersionMismatch   = db.ErrVersionMismatch
	ErrDuplicate         = db.ErrDuplicate
	ErrInvalidQuery      = db.ErrInvalidQuery
	ErrRefreshTokenReuse = db.ErrRefreshTokenReuse
)

// ValidationError represents a user-facing validation error that is safe to
// return to the client (as opposed to internal errors from DB, bcrypt, etc.).
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }
