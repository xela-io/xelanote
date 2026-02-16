package api

import (
	"fmt"

	"github.com/xela-io/xelanote/internal/auth"
	"github.com/xela-io/xelanote/internal/service"
)

// signRecipeImageURLs signs all image URLs in a RecipeDetail with fresh signatures.
func (s *Server) signRecipeImageURLs(detail *service.RecipeDetail) {
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
	userID, filename, err := service.ParseUploadURL(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid image URL format: %s", baseURL)
	}

	sig, expires, err := auth.GenerateUploadSignature(userID, filename, s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("generate signature: %w", err)
	}

	return fmt.Sprintf("/api/uploads/%d/%s?signature=%s&expires=%d", userID, filename, sig, expires), nil
}
