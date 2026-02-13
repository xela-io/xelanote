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
