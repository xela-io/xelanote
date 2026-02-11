package db

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
)

// boolToInt converts a bool to SQLite integer (0 or 1).
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// validateLastInsertID ensures SQLite insert IDs are positive and fit into int.
func validateLastInsertID(id int64, field string) (int, error) {
	if id <= 0 {
		return 0, fmt.Errorf("invalid %s: %d", field, id)
	}
	if id > math.MaxInt {
		return 0, fmt.Errorf("%s overflows int: %d", field, id)
	}
	return int(id), nil
}

// ensureRowsAffected checks that a SQL write operation affected at least one row.
func ensureRowsAffected(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// ensureRowsAffectedWithContext adds operation context while preserving ErrNotFound semantics.
func ensureRowsAffectedWithContext(result sql.Result, context string) error {
	err := ensureRowsAffected(result)
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	return fmt.Errorf("%s: %w", context, err)
}

// rowsAffectedCount returns rows affected with optional context decoration.
func rowsAffectedCount(result sql.Result, context string) (int64, error) {
	rows, err := result.RowsAffected()
	if err != nil {
		if context == "" {
			return 0, err
		}
		return 0, fmt.Errorf("%s: %w", context, err)
	}
	return rows, nil
}
