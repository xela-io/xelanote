package db

// boolToInt converts a bool to SQLite integer (0 or 1).
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
