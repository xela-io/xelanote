package api

import (
	"bytes"
	"errors"
	"image"
	// Register image decoders for content sniffing
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"

	// WebP decoder (requires golang.org/x/image)
	_ "golang.org/x/image/webp"
)

const (
	maxRecipeSuggestionImageBytes = 5 << 20 // 5MB
	// Hard limits against decompression-bomb style images.
	maxDecodedImagePixels    = 25_000_000
	maxDecodedImageDimension = 10_000
)

func (s *Server) extractIngredientsFromPhoto(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse multipart form (5MB limit)
	r.Body = http.MaxBytesReader(w, r.Body, maxRecipeSuggestionImageBytes)
	if err := r.ParseMultipartForm(maxRecipeSuggestionImageBytes); err != nil {
		respondError(w, http.StatusRequestEntityTooLarge, "file too large (max 5MB)")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		respondError(w, http.StatusBadRequest, "image file is required")
		return
	}
	defer file.Close()

	imageData, err := io.ReadAll(io.LimitReader(file, maxRecipeSuggestionImageBytes+1))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to read image")
		return
	}
	if len(imageData) > maxRecipeSuggestionImageBytes {
		respondError(w, http.StatusRequestEntityTooLarge, "image exceeds 5MB limit")
		return
	}

	decodedMime, err := validateImageUploadData(imageData)
	if err != nil {
		var statusErr interface{ StatusCode() int }
		if errors.As(err, &statusErr) {
			respondError(w, statusErr.StatusCode(), err.Error())
			return
		}
		s.respondInternalErr(w, "failed to process image", err)
		return
	}

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

	imageData, decodedMime, locale, err := parseImageUpload(w, r)
	if err != nil {
		var statusErr interface{ StatusCode() int }
		if errors.As(err, &statusErr) {
			respondError(w, statusErr.StatusCode(), err.Error())
			return
		}
		s.respondInternalErr(w, "failed to process image", err)
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

type requestStatusError struct {
	code int
	msg  string
}

func (e requestStatusError) Error() string   { return e.msg }
func (e requestStatusError) StatusCode() int { return e.code }

func parseImageUpload(w http.ResponseWriter, r *http.Request) ([]byte, string, string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRecipeSuggestionImageBytes)
	if err := r.ParseMultipartForm(maxRecipeSuggestionImageBytes); err != nil {
		return nil, "", "", requestStatusError{code: http.StatusRequestEntityTooLarge, msg: "file too large (max 5MB)"}
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		return nil, "", "", requestStatusError{code: http.StatusBadRequest, msg: "image file is required"}
	}
	defer file.Close()

	imageData, err := io.ReadAll(io.LimitReader(file, maxRecipeSuggestionImageBytes+1))
	if err != nil {
		return nil, "", "", requestStatusError{code: http.StatusInternalServerError, msg: "failed to read image"}
	}
	if len(imageData) > maxRecipeSuggestionImageBytes {
		return nil, "", "", requestStatusError{code: http.StatusRequestEntityTooLarge, msg: "image exceeds 5MB limit"}
	}

	decodedMime, err := validateImageUploadData(imageData)
	if err != nil {
		return nil, "", "", err
	}

	return imageData, decodedMime, r.FormValue("locale"), nil
}

func validateImageUploadData(imageData []byte) (string, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		return "", requestStatusError{code: http.StatusBadRequest, msg: "invalid image file: could not decode"}
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return "", requestStatusError{code: http.StatusBadRequest, msg: "invalid image dimensions"}
	}
	if cfg.Width > maxDecodedImageDimension || cfg.Height > maxDecodedImageDimension ||
		int64(cfg.Width)*int64(cfg.Height) > maxDecodedImagePixels {
		return "", requestStatusError{code: http.StatusRequestEntityTooLarge, msg: "image dimensions exceed allowed limits"}
	}

	// Full decode as content sniffing after strict dimension checks.
	if _, _, err := image.Decode(bytes.NewReader(imageData)); err != nil {
		return "", requestStatusError{code: http.StatusBadRequest, msg: "invalid image file: could not decode"}
	}

	return "image/" + format, nil
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
