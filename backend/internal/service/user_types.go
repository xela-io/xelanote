package service

import (
	"errors"
	"net/mail"
	"strings"

	"github.com/xela-io/xelanote/internal/db"
)

// Type aliases for user-related DB types.
// Allows the API layer to reference these types without importing db directly.
type WebAuthnCredential = db.WebAuthnCredential
type AIModelPreferences = db.AIModelPreferences
type LockoutRecord = db.LockoutRecord
type UserPreferences = db.UserPreferences

type HomeDashboardLayoutPreferences struct {
	Version           int                            `json:"version"`
	CollapsedSections HomeDashboardCollapsedSections `json:"collapsed_sections"`
	RightSectionOrder []string                       `json:"right_section_order"`
}

// OpenTabsPayload is the server-side schema for open_tabs.
type OpenTabsPayload struct {
	Version      int            `json:"version"`
	Tabs         []OpenTabEntry `json:"tabs"`
	ActiveNoteID *string        `json:"active_note_id"`
}

// OpenTabEntry represents a single tab (only note_id, no title for E2E encryption compat).
type OpenTabEntry struct {
	NoteID string `json:"note_id"`
}

type HomeDashboardCollapsedSections struct {
	Hero     bool `json:"hero"`
	Recent   bool `json:"recent"`
	Activity bool `json:"activity"`
	Created  bool `json:"created"`
	All      bool `json:"all"`
}

// Validation errors
var (
	ErrInvalidTheme                = errors.New("invalid theme")
	ErrInvalidEditorMode           = errors.New("invalid editor mode")
	ErrInvalidAIProvider           = errors.New("invalid AI provider")
	ErrInvalidAIModel              = errors.New("invalid AI model")
	ErrInvalidDietaryPreference    = errors.New("invalid dietary preference")
	ErrInvalidPassword             = errors.New("invalid password")
	ErrEmailInUse                  = errors.New("email already in use")
	ErrPasswordTooShort            = errors.New("password must be at least 8 characters")
	ErrInvalidEmail                = errors.New("invalid email format")
	ErrRecoveryResetNeedsDEKRewrap = errors.New("password recovery blocked: encrypted notes require DEK re-wrapping")
	ErrInvalidHomeDashboardLayout  = errors.New("invalid home dashboard layout")
	ErrInvalidOpenTabs             = errors.New("invalid open tabs")
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

var validDietaryPreferences = map[string]bool{
	"none":        true,
	"vegetarian":  true,
	"vegan":       true,
	"pescatarian": true,
	"flexitarian": true,
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

func isValidDietaryPreference(pref string) bool {
	return validDietaryPreferences[pref]
}

// isValidEmail validates an email address using Go's net/mail package
// with additional checks to reject unusual but technically valid formats
// like user@[192.168.1.1] or single-label domains like user@localhost.
func isValidEmail(email string) bool {
	if len(email) < 5 {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	// Use the parsed address (strips display name if present)
	at := strings.LastIndex(addr.Address, "@")
	if at < 1 {
		return false
	}
	domain := addr.Address[at+1:]
	// Domain must contain a dot (rejects IP-literals and single-label domains)
	return strings.Contains(domain, ".")
}
