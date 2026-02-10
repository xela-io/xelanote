package api

import (
	"archive/zip"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"
)

func (s *Server) exportMarkdown(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Get all notes for this user
	var allNotes []struct {
		Title      string
		Content    string
		FolderPath string
	}

	cursor := ""
	for {
		notes, nextCursor, err := s.noteService.ListNotes(userID, 100, cursor)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		for _, n := range notes {
			allNotes = append(allNotes, struct {
				Title      string
				Content    string
				FolderPath string
			}{
				Title:      n.Title,
				Content:    n.Content,
				FolderPath: n.FolderPath,
			})
		}

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	// Set headers for ZIP download
	timestamp := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("xelanote-export-%s.zip", timestamp)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	// Create ZIP writer
	zw := zip.NewWriter(w)
	defer zw.Close()

	// Track filenames to avoid duplicates
	usedNames := make(map[string]int)

	for _, note := range allNotes {
		// Build file path - sanitize folder path to prevent path traversal
		folder := sanitizeFolderPath(note.FolderPath)
		if folder != "" {
			folder = folder + "/"
		}

		// Sanitize title for filename
		safeName := sanitizeFilename(note.Title)
		basePath := folder + safeName

		// Handle duplicate names
		finalPath := basePath + ".md"
		if count, exists := usedNames[strings.ToLower(finalPath)]; exists {
			usedNames[strings.ToLower(finalPath)] = count + 1
			finalPath = fmt.Sprintf("%s_%d.md", basePath, count+1)
		}
		usedNames[strings.ToLower(finalPath)] = 1

		// Create file in ZIP
		fw, err := zw.Create(finalPath)
		if err != nil {
			s.logger().Error("failed to create zip entry", "path", finalPath, "error", err)
			continue
		}

		// Write content (add YAML frontmatter for Obsidian compatibility)
		content := fmt.Sprintf("---\ntitle: %q\n---\n\n%s", note.Title, note.Content)
		if _, err := fw.Write([]byte(content)); err != nil {
			s.logger().Error("failed to write zip content", "path", finalPath, "error", err)
		}
	}
}

// sanitizeFolderPath sanitizes a folder path to prevent path traversal attacks.
// It removes leading slashes, resolves ".." components, and ensures the path stays within the root.
func sanitizeFolderPath(folderPath string) string {
	// Remove leading slash
	cleaned := strings.TrimPrefix(folderPath, "/")

	// Clean the path (resolves . and ..)
	cleaned = path.Clean(cleaned)

	// If path.Clean returns "." it means the path was empty or root
	if cleaned == "." {
		return ""
	}

	// Check for path traversal attempts
	// After path.Clean, any remaining ".." means an attempt to escape the root
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, "/..") {
		// Return empty string for malicious paths
		return ""
	}

	// Remove any remaining absolute path indicators
	cleaned = strings.TrimPrefix(cleaned, "/")

	return cleaned
}

// sanitizeFilename removes or replaces characters that are invalid in filenames.
func sanitizeFilename(name string) string {
	// Characters invalid on Windows/Mac/Linux filesystems
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}

	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Trim spaces and dots from ends
	result = strings.TrimSpace(result)
	result = strings.Trim(result, ".")

	// Limit length
	if len(result) > 200 {
		result = result[:200]
	}

	// Fallback for empty names
	if result == "" {
		result = "untitled"
	}

	return path.Clean(result)
}
