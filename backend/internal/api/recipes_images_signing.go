package api

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xela-io/xelanote/internal/auth"
	"github.com/xela-io/xelanote/internal/db"
)

// signRecipeImageURLs signs all image URLs in a RecipeDetail with fresh signatures.
func (s *Server) signRecipeImageURLs(detail *db.RecipeDetail) {
	for i, img := range detail.Images {
		signed, err := s.signImageURL(img.ImageURL)
		if err != nil {
			s.logger().Warn("failed to sign recipe image URL", "error", err, "url", img.ImageURL)
			continue
		}
		detail.Images[i].ImageURL = signed
	}
}

// signImageURL takes a base path like /api/uploads/{userID}/{filename}
// and returns a signed URL with signature and expiry query params.
func (s *Server) signImageURL(baseURL string) (string, error) {
	// Parse: /api/uploads/{userID}/{filename}
	trimmed := strings.TrimPrefix(baseURL, "/api/uploads/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid image URL format: %s", baseURL)
	}

	userIDStr := parts[0]
	filename := filepath.Base(parts[1])
	if filename == "." || filename == ".." || filename == "" {
		return "", fmt.Errorf("invalid filename in image URL: %s", baseURL)
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return "", fmt.Errorf("invalid user ID in image URL: %s", baseURL)
	}

	sig, expires, err := auth.GenerateUploadSignature(userID, filename, s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("generate signature: %w", err)
	}

	return fmt.Sprintf("/api/uploads/%d/%s?signature=%s&expires=%d", userID, filename, sig, expires), nil
}
