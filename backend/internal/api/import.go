package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/xela-io/xelanote/internal/service"
	"github.com/xela-io/xelanote/internal/utils"
	"github.com/xela-io/xelanote/internal/websocket"
	"gopkg.in/yaml.v3"
)

type ImportRequest struct {
	Files             []ImportFile `json:"files"`
	PreserveStructure bool         `json:"preserve_structure"`
}

type ImportFile struct {
	Path     string `json:"path"` // Original path for folder extraction
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

type ImportResponse struct {
	Imported       int      `json:"imported"`
	Skipped        int      `json:"skipped"`
	Failed         int      `json:"failed"`
	FoldersCreated int      `json:"folders_created"`
	Errors         []string `json:"errors,omitempty"`
}

type FrontmatterTags []string

func (t *FrontmatterTags) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		for _, node := range value.Content {
			*t = append(*t, node.Value)
		}
	case yaml.ScalarNode:
		if value.Value != "" {
			*t = append(*t, value.Value)
		}
	}
	return nil
}

type Frontmatter struct {
	Title string          `yaml:"title"`
	Tags  FrontmatterTags `yaml:"tags"`
}

const maxImportErrors = 50
const maxImportFiles = 1000

// importMarkdown handles bulk markdown file import
func (s *Server) importMarkdown(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req ImportRequest
	if err := decodeJSONWithLimit(w, r, &req, MaxLargeJSONBodySize); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Files) == 0 {
		respondError(w, http.StatusBadRequest, "no files provided")
		return
	}
	if len(req.Files) > maxImportFiles {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("too many files (max %d)", maxImportFiles))
		return
	}

	result := ImportResponse{}
	createdFolders := make(map[string]bool)

	for _, file := range req.Files {
		if file.Content == "" {
			result.Skipped++
			addImportError(&result, "Skipped (empty content): "+file.Filename)
			continue
		}
		// Parse frontmatter and content
		title, tags, content := parseMarkdown(file.Content)
		if title == "" {
			// Fallback: use filename without extension
			title = strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
		}
		if err := validateNoteFields(title, content, ""); err != nil {
			result.Failed++
			addImportError(&result, "Invalid note for "+file.Filename+": "+err.Error())
			continue
		}

		// Extract folder path from file path
		folderPath := "/"
		if req.PreserveStructure && file.Path != "" {
			dir := filepath.Dir(file.Path)
			if dir != "." && dir != "/" {
				folderPath = "/" + strings.Trim(filepath.ToSlash(dir), "/")
			}
		}

		// Check for duplicate title in the same folder (folder-scoped uniqueness)
		existing, err := s.noteService.GetNoteByTitleInFolder(userID, title, folderPath)
		if err != nil && !errors.Is(err, service.ErrNotFound) {
			result.Failed++
			addImportError(&result, "Database error for "+title+": "+err.Error())
			continue
		}
		if existing != nil {
			result.Skipped++
			addImportError(&result, fmt.Sprintf("Skipped (already exists in %s): %s", folderPath, title))
			continue
		}

		// Create folder hierarchy if needed
		if folderPath != "/" {
			if err := s.ensureFolderPath(userID, folderPath, createdFolders); err != nil {
				result.Failed++
				result.Errors = append(result.Errors,
					"Folder error for "+title+": "+err.Error())
				continue
			}
		}
		if err := validateNoteFields(title, content, folderPath); err != nil {
			result.Failed++
			addImportError(&result, "Invalid folder path for "+title+": "+err.Error())
			continue
		}

		// Create note
		note, err := s.noteService.CreateNote(userID, title, content, folderPath)
		if err != nil {
			result.Failed++
			addImportError(&result, "Failed to import "+title+": "+err.Error())
			continue
		}

		if len(tags) > 0 {
			if err := s.noteService.SetNoteTags(note.ID, userID, tags); err != nil {
				addImportError(&result, "Tag error for "+title+": "+err.Error())
			}
		}

		// Broadcast creation to WebSocket clients
		payload, err := json.Marshal(note)
		if err != nil {
			s.logger().Error("failed to encode note.created payload", "err", err, "note_id", note.ID)
		} else {
			s.wsManager.BroadcastToUser(userID, websocket.Message{
				Type:    "note.created",
				Payload: payload,
			})
		}

		result.Imported++
	}

	result.FoldersCreated = len(createdFolders)
	respondJSON(w, http.StatusOK, result)
}

func addImportError(result *ImportResponse, message string) {
	if len(result.Errors) >= maxImportErrors {
		return
	}
	result.Errors = append(result.Errors, message)
}

// parseMarkdown extracts title and tags from YAML frontmatter.
func parseMarkdown(content string) (title string, tags []string, body string) {
	// Check for YAML frontmatter
	if !strings.HasPrefix(content, "---\n") {
		return "", nil, content
	}

	// Find closing ---
	end := strings.Index(content[4:], "\n---\n")
	if end == -1 {
		return "", nil, content
	}

	yamlContent := content[4 : end+4]
	body = strings.TrimSpace(content[end+9:])

	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return "", nil, content // Ignore parse errors
	}

	if len(fm.Tags) > 0 {
		for _, tag := range fm.Tags {
			trimmed := strings.TrimSpace(tag)
			if trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
	}
	return fm.Title, tags, body
}

// ensureFolderPath creates folder hierarchy
func (s *Server) ensureFolderPath(userID int, path string, cache map[string]bool) error {
	// Validate path before any processing
	if err := utils.ValidateFolderPath(path); err != nil {
		return err
	}

	// Check cache first
	if cache[path] {
		return nil
	}

	// Split path into segments
	segments := strings.Split(strings.Trim(path, "/"), "/")

	currentPath := ""
	for _, segment := range segments {
		if segment == "" {
			continue
		}

		parentPath := currentPath
		currentPath = filepath.Join(currentPath, segment)
		if currentPath != "/" {
			currentPath = "/" + strings.TrimPrefix(currentPath, "/")
		}

		// Check if folder exists
		_, err := s.noteService.GetFolderByPath(userID, currentPath)
		if err != nil {
			// Only proceed if folder not found; propagate other errors
			if !errors.Is(err, service.ErrNotFound) {
				return err
			}

			// Folder doesn't exist, create it
			var parentID *int
			if parentPath == "" || parentPath == "/" {
				// Top-level folder - virtual root (parent_id = NULL)
				parentID = nil
			} else {
				// Nested folder - get parent folder ID
				parent, parentErr := s.noteService.GetFolderByPath(userID, parentPath)
				if parentErr != nil && !errors.Is(parentErr, service.ErrNotFound) {
					return parentErr
				}
				if parent != nil {
					parentID = &parent.ID
				}
			}

			_, err = s.noteService.CreateFolder(userID, currentPath, parentID)
			if err != nil {
				return err
			}

			cache[currentPath] = true
		} else {
			cache[currentPath] = true
		}
	}

	return nil
}
