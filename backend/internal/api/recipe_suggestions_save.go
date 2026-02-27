package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xela-io/xelanote/internal/htmlutil"
	"github.com/xela-io/xelanote/internal/service"
)

func (s *Server) saveGeneratedRecipe(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req service.SaveGeneratedRecipeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	note, err := s.recipeSuggestionService.SaveGeneratedRecipe(userID, req)
	if err != nil {
		s.respondInternalErr(w, "failed to save recipe", err)
		return
	}

	importedImages := 0
	if req.SourceURL != nil {
		importedImages = s.importMainRecipeImagesFromURL(r.Context(), userID, note.ID, *req.SourceURL)
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"note_id":         note.ID,
		"title":           note.Title,
		"imported_images": importedImages,
	})
}

func (s *Server) importMainRecipeImagesFromURL(ctx context.Context, userID int, noteID string, sourceURL string) int {
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		return 0
	}

	rawHTML, finalURL, err := htmlutil.FetchHTML(ctx, sourceURL)
	if err != nil {
		s.log.Warn("auto-import recipe images: fetch html failed", "error", err, "source_url", sourceURL)
		return 0
	}

	candidates := htmlutil.ExtractImageCandidates(rawHTML, finalURL)
	filtered := filterMainImageCandidates(candidates)
	if len(filtered) == 0 {
		return 0
	}
	if len(filtered) > maxRecipeImageCandidates {
		filtered = filtered[:maxRecipeImageCandidates]
	}

	selected := s.recipeSuggestionService.SelectMainRecipeImages(
		ctx, userID, finalURL, filtered, maxAutoImportRecipeImages,
	)
	if len(selected) == 0 {
		return 0
	}

	imported := 0
	seen := make(map[string]bool)
	for _, imageURL := range selected {
		if imported >= maxAutoImportRecipeImages {
			break
		}
		if seen[imageURL] {
			continue
		}
		seen[imageURL] = true

		data, mimeType, err := htmlutil.FetchImage(ctx, imageURL, maxAutoImportedImageSize)
		if err != nil {
			s.log.Warn("auto-import recipe images: fetch image failed", "error", err, "image_url", imageURL)
			continue
		}

		if !isLikelyMainRecipeImage(data) {
			continue
		}

		uploadURL, err := s.saveImportedRecipeImage(userID, data, mimeType)
		if err != nil {
			s.log.Warn("auto-import recipe images: save image failed", "error", err, "image_url", imageURL)
			continue
		}

		if _, err := s.recipeService.AddRecipeImage(userID, noteID, uploadURL, nil); err != nil {
			s.log.Warn("auto-import recipe images: attach image failed", "error", err, "image_url", imageURL)
			continue
		}

		imported++
	}

	return imported
}

func (s *Server) saveImportedRecipeImage(userID int, data []byte, mimeType string) (string, error) {
	ext, ok := allowedTypes[mimeType]
	if !ok {
		return "", fmt.Errorf("unsupported image type: %s", mimeType)
	}

	maxStorageMB, err := s.settingsService.GetMaxStorageMBPerUser()
	if err != nil {
		return "", fmt.Errorf("failed to check storage limit: %w", err)
	}

	if maxStorageMB > 0 {
		currentUsageMB := s.adminService.GetUserStorageMB(userID)
		fileSizeMB := float64(len(data)) / (1024 * 1024)
		if currentUsageMB+fileSizeMB > float64(maxStorageMB) {
			return "", fmt.Errorf("storage limit would be exceeded")
		}
	}

	uuid, err := generateUUID()
	if err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}
	filename := uuid + ext

	userUploadDir := filepath.Join(s.dataDir, UploadDir, strconv.Itoa(userID))
	if err := os.MkdirAll(userUploadDir, 0750); err != nil {
		return "", fmt.Errorf("create upload directory: %w", err)
	}

	filePath := filepath.Join(userUploadDir, filename)
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", fmt.Errorf("write image: %w", err)
	}

	if maxStorageMB > 0 {
		usedMB := s.adminService.GetUserStorageMB(userID)
		if usedMB > float64(maxStorageMB) {
			_ = os.Remove(filePath)
			return "", fmt.Errorf("storage limit exceeded")
		}
	}

	return fmt.Sprintf("/api/uploads/%d/%s", userID, filename), nil
}
