package db

import (
	"fmt"
	"regexp"
)

// hexColorRegex validates hex color format (#RRGGBB)
var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// validateHexColor validates that a color string is a valid hex color
func validateHexColor(color string) error {
	if color != "" && !hexColorRegex.MatchString(color) {
		return fmt.Errorf("invalid color format, expected #RRGGBB")
	}
	return nil
}
