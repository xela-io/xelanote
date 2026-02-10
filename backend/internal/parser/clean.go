// Package parser provides utilities for cleaning markdown content.
package parser

import (
	"regexp"
)

// Precompiled regex patterns for color tags
var (
	colorOpenTagRegex  = regexp.MustCompile(`\{color:[^}]+\}`)
	colorCloseTagRegex = regexp.MustCompile(`\{/color\}`)
)

// StripColorTags removes all color syntax tags from the given text.
// This is useful for generating clean search snippets.
//
// Examples:
//
//	StripColorTags("{color:primary}hello{/color}") returns "hello"
//	StripColorTags("{color:#ff0000}red text{/color}") returns "red text"
//	StripColorTags("normal text") returns "normal text"
func StripColorTags(text string) string {
	// Remove opening tags: {color:VALUE}
	text = colorOpenTagRegex.ReplaceAllString(text, "")
	// Remove closing tags: {/color}
	text = colorCloseTagRegex.ReplaceAllString(text, "")
	return text
}
