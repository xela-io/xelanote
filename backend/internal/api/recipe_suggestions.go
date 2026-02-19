package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	// Register image decoders for content sniffing
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xela-io/xelanote/internal/htmlutil"
	"github.com/xela-io/xelanote/internal/llm"
	"github.com/xela-io/xelanote/internal/service"
	// WebP decoder (requires golang.org/x/image)
	_ "golang.org/x/image/webp"
)

const (
	maxAutoImportRecipeImages   = 3
	maxRecipeImageCandidates    = 8
	maxAutoImportedImageSize    = 5 << 20 // 5MB
	minMainRecipeImageDimension = 300
)

// --- Request/Response types ---

type similarRecipesRequest struct {
	NoteID       string `json:"note_id"`
	CollectionID *int   `json:"collection_id,omitempty"`
	Locale       string `json:"locale"`
}

type byIngredientsRequest struct {
	Ingredients  []string `json:"ingredients"`
	CollectionID *int     `json:"collection_id,omitempty"`
	Locale       string   `json:"locale"`
}

type importRecipeFromURLRequest struct {
	URL    string `json:"url"`
	Locale string `json:"locale"`
}

// --- Handlers ---

func (s *Server) findSimilarRecipes(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req similarRecipesRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.NoteID == "" {
		respondError(w, http.StatusBadRequest, "note_id is required")
		return
	}

	results, err := s.recipeSuggestionService.FindSimilarRecipes(
		r.Context(), userID, req.NoteID, req.CollectionID, req.Locale,
	)
	if err != nil {
		s.handleSuggestionError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
	})
}

func (s *Server) suggestByIngredients(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req byIngredientsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Ingredients) == 0 {
		respondError(w, http.StatusBadRequest, "at least one ingredient is required")
		return
	}

	result, err := s.recipeSuggestionService.SuggestByIngredients(
		r.Context(), userID, req.Ingredients, req.CollectionID, req.Locale,
	)
	if err != nil {
		s.handleSuggestionError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

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
		respondError(w, http.StatusBadRequest, err.Error())
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

func (s *Server) extractIngredientsFromPhoto(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse multipart form (5MB limit)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		respondError(w, http.StatusRequestEntityTooLarge, "file too large (max 5MB)")
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		respondError(w, http.StatusBadRequest, "image file is required")
		return
	}
	defer file.Close()

	imageData, err := io.ReadAll(io.LimitReader(file, 5<<20+1))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to read image")
		return
	}
	if len(imageData) > 5<<20 {
		respondError(w, http.StatusRequestEntityTooLarge, "image exceeds 5MB limit")
		return
	}

	// Content-sniff: decode the image to verify it's actually an image
	_, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid image file: could not decode")
		return
	}

	// Derive MIME type from decoded format (not from HTTP header)
	decodedMime := "image/" + format

	locale := r.FormValue("locale")

	ingredients, err := s.recipeSuggestionService.ExtractIngredientsFromPhoto(
		r.Context(), userID, imageData, decodedMime, locale,
	)
	if err != nil {
		s.handleSuggestionError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ingredients": ingredients,
	})
}

func (s *Server) importRecipeFromImage(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	imageData, decodedMime, locale, err := parseImageUpload(r)
	if err != nil {
		var statusErr interface{ StatusCode() int }
		if errors.As(err, &statusErr) {
			respondError(w, statusErr.StatusCode(), err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	recipe, err := s.recipeSuggestionService.ExtractRecipeFromImage(
		r.Context(), userID, imageData, decodedMime, locale,
	)
	if err != nil {
		s.handleSuggestionError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, recipe)
}

func (s *Server) importRecipeFromURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req importRecipeFromURLRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.URL) == "" {
		respondError(w, http.StatusBadRequest, "url is required")
		return
	}

	recipe, err := s.recipeSuggestionService.ExtractRecipeFromURL(r.Context(), userID, req.URL, req.Locale)
	if err != nil {
		switch {
		case errors.Is(err, htmlutil.ErrInvalidURL), errors.Is(err, htmlutil.ErrDisallowedAddress):
			respondError(w, http.StatusBadRequest, "invalid or disallowed URL")
		case errors.Is(err, htmlutil.ErrFetchFailed):
			respondError(w, http.StatusBadGateway, "failed to fetch recipe URL")
		default:
			s.handleSuggestionError(w, err)
		}
		return
	}

	respondJSON(w, http.StatusOK, recipe)
}

// handleSuggestionError maps service-layer errors to appropriate HTTP status codes.
func (s *Server) handleSuggestionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, llm.ErrNoProviderAvailable):
		respondError(w, http.StatusFailedDependency, "no AI provider configured - add API key in settings")
	case errors.Is(err, llm.ErrVisionNotAvailable):
		respondError(w, http.StatusFailedDependency, "no vision-capable AI provider available")
	case errors.Is(err, service.ErrNoRecipeFound):
		respondError(w, http.StatusUnprocessableEntity, "no recipe found in input")
	case errors.Is(err, service.ErrRecipeEncrypted):
		respondError(w, http.StatusForbidden, "not available for encrypted recipes")
	default:
		errMsg := err.Error()
		if isUpstreamError(errMsg) {
			respondError(w, http.StatusBadGateway, "AI provider temporarily unavailable")
		} else {
			s.log.Error("recipe suggestion error", "error", errMsg)
			respondError(w, http.StatusInternalServerError, "failed to process request")
		}
	}
}

type requestStatusError struct {
	code int
	msg  string
}

func (e requestStatusError) Error() string   { return e.msg }
func (e requestStatusError) StatusCode() int { return e.code }

func parseImageUpload(r *http.Request) ([]byte, string, string, error) {
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		return nil, "", "", requestStatusError{code: http.StatusRequestEntityTooLarge, msg: "file too large (max 5MB)"}
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		return nil, "", "", requestStatusError{code: http.StatusBadRequest, msg: "image file is required"}
	}
	defer file.Close()

	imageData, err := io.ReadAll(io.LimitReader(file, 5<<20+1))
	if err != nil {
		return nil, "", "", requestStatusError{code: http.StatusInternalServerError, msg: "failed to read image"}
	}
	if len(imageData) > 5<<20 {
		return nil, "", "", requestStatusError{code: http.StatusRequestEntityTooLarge, msg: "image exceeds 5MB limit"}
	}

	_, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, "", "", requestStatusError{code: http.StatusBadRequest, msg: "invalid image file: could not decode"}
	}

	decodedMime := "image/" + format
	return imageData, decodedMime, r.FormValue("locale"), nil
}

// isUpstreamError checks if an error message indicates an upstream LLM failure.
func isUpstreamError(msg string) bool {
	for _, pattern := range []string{"API error", "returned status", "failed to send request"} {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
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

func filterMainImageCandidates(candidates []string) []string {
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		u := strings.ToLower(strings.TrimSpace(candidate))
		if u == "" {
			continue
		}
		if strings.Contains(u, "logo") || strings.Contains(u, "icon") || strings.Contains(u, "avatar") {
			continue
		}
		result = append(result, candidate)
	}
	return result
}

func isLikelyMainRecipeImage(data []byte) bool {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return false
	}
	if cfg.Width < minMainRecipeImageDimension || cfg.Height < minMainRecipeImageDimension {
		return false
	}
	return true
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
	if err := os.MkdirAll(userUploadDir, 0755); err != nil {
		return "", fmt.Errorf("create upload directory: %w", err)
	}

	filePath := filepath.Join(userUploadDir, filename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
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
