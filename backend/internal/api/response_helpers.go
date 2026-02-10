package api

// ensureSlice returns an empty slice if items is nil, otherwise returns items unchanged.
// This ensures JSON serialization produces [] instead of null.
func ensureSlice[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}
