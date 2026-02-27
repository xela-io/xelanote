package api

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/auth"
)

// getUserUploadMu returns a per-user mutex for serializing upload quota checks.
func (s *Server) getUserUploadMu(userID int) *sync.Mutex {
	mu, _ := s.uploadMu.LoadOrStore(userID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

const (
	MaxUploadSize = 10 << 20 // 10MB
	UploadDir     = "uploads"
)

var allowedTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// uploadImage handles image file uploads
func (s *Server) uploadImage(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from JWT context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Parse multipart form (max 10MB)
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize)
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		respondError(w, http.StatusBadRequest, "file too large (max 10MB)")
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "no file provided")
		return
	}
	defer file.Close()

	// Validate content type
	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && err != io.EOF {
		respondError(w, http.StatusBadRequest, "failed to read file header")
		return
	}
	contentType := http.DetectContentType(head[:n])
	ext, ok := allowedTypes[contentType]
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid file type (only images allowed)")
		return
	}

	// Generate unique filename
	uuid, err := generateUUID()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate file id")
		return
	}
	filename := uuid + ext

	// Create user upload directory
	userUploadDir := filepath.Join(s.dataDir, UploadDir, fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(userUploadDir, 0750); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create upload directory")
		return
	}

	// Acquire per-user mutex to prevent TOCTOU race condition on quota checks.
	// This serializes concurrent uploads for the same user so that the quota
	// check and file write are atomic with respect to each other.
	uploadMu := s.getUserUploadMu(userID)
	uploadMu.Lock()

	maxStorageMB, err := s.settingsService.GetMaxStorageMBPerUser()
	if err != nil {
		uploadMu.Unlock()
		respondError(w, http.StatusInternalServerError, "failed to check storage limit")
		return
	}

	// Pre-write quota check using header size
	if maxStorageMB > 0 && header.Size > 0 {
		currentUsageMB := s.adminService.GetUserStorageMB(userID)
		fileSizeMB := float64(header.Size) / (1024 * 1024)

		if currentUsageMB+fileSizeMB > float64(maxStorageMB) {
			uploadMu.Unlock()
			respondError(w, http.StatusForbidden, "storage limit would be exceeded")
			return
		}
	}

	// Save file (still under lock)
	filePath := filepath.Join(userUploadDir, filename)
	dst, err := os.Create(filePath) //nolint:gosec // path constructed from validated user ID + generated UUID filename
	if err != nil {
		uploadMu.Unlock()
		respondError(w, http.StatusInternalServerError, "failed to save file")
		return
	}

	reader := io.MultiReader(bytes.NewReader(head[:n]), file)
	if _, err := io.Copy(dst, reader); err != nil {
		dst.Close()
		uploadMu.Unlock()
		respondError(w, http.StatusInternalServerError, "failed to write file")
		return
	}

	// Close file before quota check (so file size is accurate)
	dst.Close()

	// Post-write quota check as safety net for chunked uploads where header.Size may be 0
	if maxStorageMB > 0 {
		usedMB := s.adminService.GetUserStorageMB(userID)
		if usedMB > float64(maxStorageMB) {
			os.Remove(filePath)
			uploadMu.Unlock()
			respondError(w, http.StatusForbidden, "storage limit exceeded")
			return
		}
	}

	// Release lock after file is written and quota is verified
	uploadMu.Unlock()

	// Generate thumbnail for image types (non-fatal)
	thumbFilename := generateThumbnail(filePath, contentType)

	// SEC-L04: Generate signed URL for secure access (7 days expiry)
	signature, expires, err := auth.GenerateUploadSignature(userID, filename, s.jwtSecret)
	if err != nil {
		s.log.Warn("failed to generate upload signature",
			"user_id", userID,
			"filename", filename,
			"error", err.Error())
		signature = "" // Non-fatal: Cookie fallback will be used
	}

	// Return URL with signature if available, otherwise plain URL (cookie fallback)
	url := fmt.Sprintf("/api/uploads/%d/%s", userID, filename)
	if signature != "" {
		url = fmt.Sprintf("%s?signature=%s&expires=%d", url, signature, expires)
	}

	resp := map[string]string{
		"url":      url,
		"filename": header.Filename,
	}

	if thumbFilename != "" {
		thumbURL := thumbnailURL(userID, thumbFilename)
		thumbSig, thumbExp, thumbErr := auth.GenerateUploadSignature(userID, thumbFilename, s.jwtSecret)
		if thumbErr == nil && thumbSig != "" {
			thumbURL = fmt.Sprintf("%s?signature=%s&expires=%d", thumbURL, thumbSig, thumbExp)
		}
		resp["thumbnail_url"] = thumbURL
	}

	respondJSON(w, http.StatusOK, resp)
}

// serveUpload serves uploaded files with user isolation (SEC-002)
// SEC-L04: Supports signed URLs (primary) with cookie fallback (backward compat)
func (s *Server) serveUpload(w http.ResponseWriter, r *http.Request) {
	userIDParam := chi.URLParam(r, "user_id")
	filename := chi.URLParam(r, "filename")

	// SEC-002: Parse file owner ID from URL parameter
	fileOwnerID, err := strconv.Atoi(userIDParam)
	if err != nil || fileOwnerID <= 0 {
		respondError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	// SEC-L04: Try signed URL authentication first
	authenticated := false
	signature := r.URL.Query().Get("signature")
	expiresStr := r.URL.Query().Get("expires")

	if signature != "" && expiresStr != "" {
		expires, parseErr := strconv.ParseInt(expiresStr, 10, 64)
		if parseErr == nil {
			validationErr := auth.ValidateUploadSignature(fileOwnerID, filename, signature, expires, s.jwtSecret)
			if validationErr == nil {
				// Signature valid - proceed to file serving
				authenticated = true
				s.log.Debug("upload served via signed URL",
					"user_id", fileOwnerID,
					"filename", filename)
			} else {
				s.log.Warn("invalid upload signature",
					"user_id", fileOwnerID,
					"filename", filename,
					"error", validationErr.Error())
			}
		}
	}

	// Fallback: JWT cookie authentication (backward compatibility)
	if !authenticated {
		authUserID, ok := getUserID(r)
		if !ok {
			respondError(w, http.StatusUnauthorized, "user not authenticated")
			return
		}

		// SEC-002: Verify the requested file belongs to the authenticated user
		if fileOwnerID != authUserID {
			respondError(w, http.StatusForbidden, "access denied")
			return
		}
	}

	// Validate filename (prevent path traversal)
	cleanedFilename := filepath.Base(filename)
	if cleanedFilename != filename || strings.TrimSpace(cleanedFilename) == "." {
		respondError(w, http.StatusBadRequest, "invalid filename")
		return
	}

	userUploadDir := filepath.Join(s.dataDir, UploadDir, fmt.Sprintf("%d", fileOwnerID))
	filePath := filepath.Join(userUploadDir, cleanedFilename)

	// Security check: ensure the final path is within the user's upload directory
	// Use separator suffix to prevent uploads/1 matching uploads/10
	if !strings.HasPrefix(filePath, userUploadDir+string(filepath.Separator)) {
		respondError(w, http.StatusForbidden, "invalid path")
		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	// Determine content type from extension
	ext := filepath.Ext(filename)
	contentType := "application/octet-stream"
	for mime, e := range allowedTypes {
		if e == ext {
			contentType = mime
			break
		}
	}

	// Set caching headers (1 year)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	// Serve file
	http.ServeFile(w, r, filePath)
}

// DEPRECATED (SEC-002): serveUploadPublic has been removed
// All upload serving now requires authentication via serveUpload()
// This function is kept commented for reference only - DO NOT USE
//
// Rationale: Public upload serving was a security risk even with UUID filenames.
// Authentication + ownership verification is now required for all file access.
// If public file sharing is needed in the future, implement a separate
// sharing mechanism with explicit user consent and time-limited tokens.

// generateUUID creates a random UUID-like string
func generateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
