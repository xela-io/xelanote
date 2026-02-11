package db

import (
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
