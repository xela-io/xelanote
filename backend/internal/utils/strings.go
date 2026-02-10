package utils

import "strings"

// NormalizeTitle returns a normalized version of a title for matching.
// This matches the title_norm column format in the database.
// Used for case-insensitive title comparisons and wikilink resolution.
func NormalizeTitle(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}
