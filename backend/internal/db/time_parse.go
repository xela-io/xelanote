package db

import (
	"fmt"
	"time"
)

func parseRFC3339Timestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		// Fallback: SQLite's datetime('now') produces "2006-01-02 15:04:05"
		parsed, err2 := time.Parse("2006-01-02 15:04:05", value)
		if err2 != nil {
			return time.Time{}, fmt.Errorf("invalid RFC3339 timestamp %q: %w", value, err)
		}
		return parsed.UTC(), nil
	}
	return parsed, nil
}
