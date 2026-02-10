package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/xela-io/xelanote/internal/db"
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

type Frontmatter struct {
	Title string   `yaml:"title"`
	Tags  []string `yaml:"tags"`
}

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

	result := ImportResponse{}
	createdFolders := make(map[string]bool)

	for _, file := range req.Files {
		// Parse frontmatter and content
		title, content := parseMarkdown(file.Content)
		if title == "" {
			// Fallback: use filename without extension
			title = strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
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
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			result.Failed++
			if len(result.Errors) < 10 {
				result.Errors = append(result.Errors,
					"DB-Fehler für "+title+": "+err.Error())
			}
			continue
		}
		if existing != nil {
			result.Skipped++
			result.Errors = append(result.Errors,
				fmt.Sprintf("Übersprungen (existiert bereits in %s): %s", folderPath, title))
			continue
		}

		// Create folder hierarchy if needed
		if folderPath != "/" {
			if err := s.ensureFolderPath(userID, folderPath, createdFolders); err != nil {
				result.Failed++
				result.Errors = append(result.Errors,
					"Ordner-Fehler für "+title+": "+err.Error())
				continue
			}
		}

		// Create note
		note, err := s.noteService.CreateNote(userID, title, content, folderPath)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors,
				"Fehler beim Importieren von "+title+": "+err.Error())
			continue
		}

		// Broadcast creation to WebSocket clients
		payload, _ := json.Marshal(note)
		s.wsManager.BroadcastToUser(userID, websocket.Message{
			Type:    "note.created",
			Payload: payload,
		})

		result.Imported++
	}

	result.FoldersCreated = len(createdFolders)
	respondJSON(w, http.StatusOK, result)
}

// parseMarkdown extracts title from YAML frontmatter
func parseMarkdown(content string) (title, body string) {
	// Check for YAML frontmatter
	if !strings.HasPrefix(content, "---\n") {
		return "", content
	}

	// Find closing ---
	end := strings.Index(content[4:], "\n---\n")
	if end == -1 {
		return "", content
	}

	yamlContent := content[4 : end+4]
	body = strings.TrimSpace(content[end+9:])

	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return "", content // Ignore parse errors
	}

	return fm.Title, body
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
			if !errors.Is(err, db.ErrNotFound) {
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
				if parentErr != nil && !errors.Is(parentErr, db.ErrNotFound) {
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
