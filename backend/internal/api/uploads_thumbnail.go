package api

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"log/slog"
	"os"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	thumbnailMaxDim  = 200
	thumbnailQuality = 80
)

// thumbnailContentTypes lists MIME types eligible for thumbnail generation.
var thumbnailContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// generateThumbnail creates a JPEG thumbnail (max 200x200, preserving aspect ratio)
// for the given image file. Returns the thumbnail filename on success, or empty string
// if thumbnail generation is not applicable or fails (non-fatal).
func generateThumbnail(filePath, contentType string) string {
	if !thumbnailContentTypes[contentType] {
		return ""
	}

	f, err := os.Open(filePath) //nolint:gosec // path is validated by caller
	if err != nil {
		slog.Warn("thumbnail: failed to open source", "path", filePath, "error", err)
		return ""
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		slog.Warn("thumbnail: failed to decode image", "path", filePath, "error", err)
		return ""
	}

	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	// Skip if already smaller than thumbnail size
	if srcW <= thumbnailMaxDim && srcH <= thumbnailMaxDim {
		return ""
	}

	// Calculate new dimensions preserving aspect ratio
	newW, newH := fitDimensions(srcW, srcH, thumbnailMaxDim)

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	// Encode as JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: thumbnailQuality}); err != nil {
		slog.Warn("thumbnail: failed to encode JPEG", "path", filePath, "error", err)
		return ""
	}

	// Save thumbnail alongside original: {uuid}-thumb.jpg
	thumbPath := thumbnailPath(filePath)
	if err := os.WriteFile(thumbPath, buf.Bytes(), 0600); err != nil {
		slog.Warn("thumbnail: failed to write file", "path", thumbPath, "error", err)
		return ""
	}

	return thumbnailFilename(filePath)
}

// fitDimensions calculates new width and height that fit within maxDim,
// preserving the original aspect ratio.
func fitDimensions(w, h, maxDim int) (int, int) {
	if w <= maxDim && h <= maxDim {
		return w, h
	}

	ratio := float64(w) / float64(h)
	if w > h {
		newW := maxDim
		newH := int(float64(newW) / ratio)
		if newH < 1 {
			newH = 1
		}
		return newW, newH
	}
	newH := maxDim
	newW := int(float64(newH) * ratio)
	if newW < 1 {
		newW = 1
	}
	return newW, newH
}

// thumbnailPath returns the filesystem path for a thumbnail given the original path.
// Example: /data/uploads/1/abc123.png → /data/uploads/1/abc123-thumb.jpg
func thumbnailPath(originalPath string) string {
	ext := extOf(originalPath)
	base := strings.TrimSuffix(originalPath, ext)
	return base + "-thumb.jpg"
}

// thumbnailFilename returns just the filename portion for the thumbnail.
// Example: abc123.png → abc123-thumb.jpg
func thumbnailFilename(originalPath string) string {
	name := fileBaseName(originalPath)
	ext := extOf(name)
	base := strings.TrimSuffix(name, ext)
	return base + "-thumb.jpg"
}

func extOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i:]
		}
		if path[i] == '/' || path[i] == '\\' {
			break
		}
	}
	return ""
}

func fileBaseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

// thumbnailURL builds the signed thumbnail URL given user ID and thumbnail filename.
func thumbnailURL(userID int, thumbFilename string) string {
	return fmt.Sprintf("/api/uploads/%d/%s", userID, thumbFilename)
}
