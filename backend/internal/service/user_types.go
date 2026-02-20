package service

import (
	"errors"
	"net/mail"

	"github.com/xela-io/xelanote/internal/db"
)

// Type alias for WebAuthn credential.
type WebAuthnCredential = db.WebAuthnCredential

// Validation errors
var (
	ErrInvalidTheme      = errors.New("invalid theme")
	ErrInvalidEditorMode = errors.New("invalid editor mode")
	ErrInvalidAIProvider = errors.New("invalid AI provider")
	ErrInvalidAIModel    = errors.New("invalid AI model")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrEmailInUse        = errors.New("email already in use")
	ErrPasswordTooShort  = errors.New("password must be at least 8 characters")
	ErrInvalidEmail      = errors.New("invalid email format")
)

// Valid theme IDs (must match frontend themes/index.ts)
var validThemes = map[string]bool{
	"default-light":    true,
	"default-dark":     true,
	"nord-light":       true,
	"nord-dark":        true,
	"solarized-light":  true,
	"solarized-dark":   true,
	"dracula":          true,
	"catppuccin-latte": true,
	"catppuccin-mocha": true,
	"dark-pastels":     true,
	"gruvbox-light":    true,
	"gruvbox-dark":     true,
	"tokyo-night":      true,
	"one-dark":         true,
	"one-light":        true,
	"monokai":          true,
	"ayu-light":        true,
	"ayu-mirage":       true,
	"rose-pine-moon":   true,
	"rose-pine-dawn":   true,
	"kanagawa":         true,
	"everforest-dark":  true,
	"everforest-light": true,
}

// Valid editor modes
var validEditorModes = map[string]bool{
	"edit":    true,
	"preview": true,
	"split":   true,
}

var validAIProviders = map[string]bool{
	"auto":    true,
	"claude":  true,
	"gemini":  true,
	"chatgpt": true,
}

// UserService handles user-related business logic
type UserService struct {
	db *db.DB
}

// NewUserService creates a new UserService
func NewUserService(database *db.DB) *UserService {
	return &UserService{
		db: database,
	}
}

// isValidTheme checks if a theme ID is valid
func isValidTheme(theme string) bool {
	return validThemes[theme]
}

// isValidEditorMode checks if an editor mode is valid
func isValidEditorMode(mode string) bool {
	return validEditorModes[mode]
}

func isValidAIProvider(provider string) bool {
	return validAIProviders[provider]
}

// isValidEmail validates an email address using Go's net/mail package.
// This is more robust than a simple @ check and handles edge cases.
func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
