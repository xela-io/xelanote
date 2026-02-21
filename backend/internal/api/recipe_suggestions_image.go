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
