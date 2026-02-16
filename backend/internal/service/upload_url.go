package service

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// ParseUploadURL extracts userID and filename from a URL of the form
// /api/uploads/{userID}/{filename}. Returns an error if the URL does
// not match the expected format.
func ParseUploadURL(rawURL string) (userID int, filename string, err error) {
	if !strings.HasPrefix(rawURL, "/api/uploads/") {
		return 0, "", fmt.Errorf("URL must start with /api/uploads/")
	}

	trimmed := strings.TrimPrefix(rawURL, "/api/uploads/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, "", fmt.Errorf("invalid upload URL format: %s", rawURL)
	}

	userID, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid user ID in upload URL: %s", rawURL)
	}

	filename = filepath.Base(parts[1])
	if filename == "." || filename == ".." || filename == "" {
		return 0, "", fmt.Errorf("invalid filename in upload URL: %s", rawURL)
	}

	return userID, filename, nil
}
