package db

import "time"

// parseTime tries RFC3339 first, then SQLite datetime format.
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, _ = time.Parse("2006-01-02 15:04:05", s)
	}
	return t
}
