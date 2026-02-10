package utils

import (
	"fmt"
	"path"
	"strings"
)

// ValidateFolderPath prüft ob ein Pfad valide ist.
// CRITICAL: Validierung VOR Normalisierung! Sonst könnte path.Clean ".." entfernen.
func ValidateFolderPath(p string) error {
	if p == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if !strings.HasPrefix(p, "/") {
		return fmt.Errorf("path must start with /")
	}
	// Virtual root: Root folder (/) cannot be created - it only exists conceptually
	if p == "/" {
		return fmt.Errorf("cannot create root folder - root is virtual")
	}
	// FIXED: Check .. BEFORE any normalization
	if strings.Contains(p, "..") {
		return fmt.Errorf("path cannot contain ..")
	}
	// Weitere Validierungen
	if strings.Contains(p, "//") {
		return fmt.Errorf("path cannot contain double slashes")
	}
	if strings.HasSuffix(p, "/") && p != "/" {
		return fmt.Errorf("path cannot end with / (except root)")
	}
	return nil
}

// NormalizeFolderPath normalisiert einen Ordnerpfad.
// CALL ValidateFolderPath FIRST!
func NormalizeFolderPath(p string) string {
	// Remove trailing slash except for root
	p = path.Clean(p)
	if p == "." {
		return "/"
	}
	// Ensure leading slash
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

// GetParentPath gibt den Parent-Pfad zurück.
func GetParentPath(p string) string {
	if p == "/" {
		return ""
	}
	parent := path.Dir(p)
	if parent == "." {
		return "/"
	}
	return parent
}

// GetFolderName extrahiert den Ordnernamen aus dem Pfad.
func GetFolderName(p string) string {
	if p == "/" {
		return "Root"
	}
	return path.Base(p)
}
