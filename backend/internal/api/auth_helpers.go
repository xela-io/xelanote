package api

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"regexp"

	"github.com/xela-io/xelanote/internal/service"
)

// Username validation for NEW registrations only
// Existing users with non-conforming names are grandfathered
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)

func validateUsername(username string) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("username must be 3-32 characters (alphanumeric, underscore, hyphen only)")
	}
	return nil
}

// isDesktopClient checks if the request is from a desktop client (Electron/Tauri)
// Desktop clients are identified by the X-Client-Type header AND must come from localhost.
// SEC-001: Used to decide whether to include tokens in the response body (for OS keyring storage).
// Even if spoofed, it only exposes tokens that are equivalent to the already-set cookies.
func isDesktopClient(r *http.Request) bool {
	if r.Header.Get("X-Client-Type") != "desktop" {
		return false
	}

	// Only trust desktop header from localhost connections (Electron/Tauri apps)
	// Use RemoteAddr directly - do NOT trust X-Forwarded-For for this security check
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIP = r.RemoteAddr
	}

	return remoteIP == "127.0.0.1" || remoteIP == "::1"
}

// getOrGenerateUserSalt fetches the user's encryption salt or generates one if it doesn't exist
// This handles migration for existing users who don't have a salt yet
func (s *Server) getOrGenerateUserSalt(userID int) ([]byte, error) {
	salt, err := s.authService.GetUserEncryptionSalt(userID)

	// If salt doesn't exist, check if user has encrypted notes
	if err == service.ErrNotFound {
		// CRITICAL: Check if user has encrypted notes before generating new salt
		// If they do, generating a new salt would make all encrypted notes permanently unreadable
		// Skip check if noteService is nil (happens in tests)
		if s.noteService != nil {
			hasEncryptedNotes, checkErr := s.noteService.UserHasEncryptedNotes(userID)
			if checkErr != nil {
				s.logger().Error("Failed to check for encrypted notes",
					slog.Int("user_id", userID),
					slog.String("error", checkErr.Error()))
				return nil, fmt.Errorf("failed to verify encryption status")
			}

			if hasEncryptedNotes {
				// CRITICAL ERROR: User has encrypted notes but salt is missing
				// This should NEVER happen - it means data corruption or migration error
				s.logger().Error("CRITICAL: User has encrypted notes but encryption salt is missing - REFUSING to generate new salt to prevent data loss",
					slog.Int("user_id", userID))
				return nil, fmt.Errorf("encryption salt missing but encrypted notes exist - contact administrator for data recovery")
			}
		}

		// Safe to generate new salt (no encrypted notes exist yet)
		s.logger().Info("Generating new encryption salt for user",
			slog.Int("user_id", userID))
		salt = make([]byte, 16) // 128-bit salt
		if _, err := rand.Read(salt); err != nil {
			return nil, err
		}
		if err := s.authService.SetUserEncryptionSalt(userID, salt); err != nil {
			return nil, err
		}
		return salt, nil
	}

	return salt, err
}
