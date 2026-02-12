package db

import "errors"

var (
	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = errors.New("not found")

	// ErrVersionMismatch is returned when optimistic locking fails.
	ErrVersionMismatch = errors.New("version mismatch")

	// ErrDuplicate is returned when a unique constraint is violated.
	ErrDuplicate = errors.New("duplicate entry")

	// ErrInvalidQuery is returned when a query is malformed or exceeds limits.
	ErrInvalidQuery = errors.New("invalid query")

	// ErrRefreshTokenReuse is returned when a rotated/revoked refresh token is reused.
	ErrRefreshTokenReuse = errors.New("refresh token reuse detected")
)
