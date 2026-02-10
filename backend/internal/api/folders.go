package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/utils"
)

// Folder API Request/Response types

type CreateFolderRequest struct {
	Path string `json:"path"`
}

type MoveFolderRequest struct {
	NewParentPath string `json:"new_parent_path"`
}

type RenameFolderRequest struct {
	NewName string `json:"new_name"`
}

type ReorderFoldersRequest struct {
	ParentID *int  `json:"parent_id"`
	Items    []int `json:"items"`
}

type UpdateColorRequest struct {
	Color *string `json:"color"`
}

// createFolder creates a new folder.
func (s *Server) createFolder(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req CreateFolderRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := utils.ValidateFolderPath(req.Path); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Determine parent_id from path
	var parentID *int
	parentPath := utils.GetParentPath(req.Path)

	// CASE 1: Top-Level Folder (Virtual Root)
	if parentPath == "/" {
		parentID = nil // Top-level folders have no physical parent (virtual root)
	} else if parentPath != "" {
		// CASE 2: Nested Folder (Parent ist anderer Folder)
		parent, err := s.noteService.GetFolderByPath(userID, parentPath)
		if err != nil {
			respondError(w, http.StatusBadRequest, "parent folder not found")
			return
		}
		parentID = &parent.ID
	}
	// CASE 3: Root Folder selbst (parentPath == "")
	// parentID bleibt nil → nur für "/" beim Erstellen

	folder, err := s.noteService.CreateFolder(userID, req.Path, parentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, folder)
}

// getAllFolders returns all folders with note counts.
func (s *Server) getAllFolders(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	folders, err := s.noteService.GetAllFolders(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"folders": folders,
	})
}

// moveFolder moves a folder to a new parent.
func (s *Server) moveFolder(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid folder id")
		return
	}

	var req MoveFolderRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := utils.ValidateFolderPath(req.NewParentPath); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = s.noteService.MoveFolder(userID, id, req.NewParentPath)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// deleteFolder deletes a folder.
func (s *Server) deleteFolder(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid folder id")
		return
	}

	err = s.noteService.DeleteFolder(userID, id)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// renameFolder renames a folder.
func (s *Server) renameFolder(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid folder id")
		return
	}

	var req RenameFolderRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.NewName == "" {
		respondError(w, http.StatusBadRequest, "new_name cannot be empty")
		return
	}

	err = s.noteService.RenameFolder(userID, id, req.NewName)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// reorderFolders updates the display order of folders.
func (s *Server) reorderFolders(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req ReorderFoldersRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Items) == 0 {
		respondError(w, http.StatusBadRequest, "items cannot be empty")
		return
	}

	err := s.noteService.ReorderFolders(userID, req.ParentID, req.Items)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// updateFolderColor updates the color of a folder.
func (s *Server) updateFolderColor(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid folder id")
		return
	}

	var req UpdateColorRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = s.noteService.UpdateFolderColor(userID, id, req.Color)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ============================================================================
// AI-Enabled Default (Claude API Opt-In) Endpoints
// ============================================================================

// UpdateFolderAIEnabledRequest represents the request body for toggling ai_enabled_default.
type UpdateFolderAIEnabledRequest struct {
	AIEnabled bool `json:"ai_enabled"`
}

// updateFolderAIEnabledDefault toggles the ai_enabled_default flag for a folder.
// New notes created in this folder will inherit this setting.
func (s *Server) updateFolderAIEnabledDefault(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid folder id")
		return
	}

	var req UpdateFolderAIEnabledRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update ai_enabled_default flag
	if err := s.noteService.UpdateFolderAIEnabledDefault(userID, id, req.AIEnabled); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "ok",
		"ai_enabled": req.AIEnabled,
	})
}

// getFolderAIEnabledDefault returns the ai_enabled_default status for a folder.
func (s *Server) getFolderAIEnabledDefault(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid folder id")
		return
	}

	aiEnabled, err := s.noteService.GetFolderAIEnabledDefault(userID, id)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ai_enabled": aiEnabled,
	})
}

// ============================================================================
// Encryption Default Endpoints
// ============================================================================

// UpdateFolderEncryptionDefaultRequest represents the request body for toggling encryption_default.
type UpdateFolderEncryptionDefaultRequest struct {
	Encrypted bool `json:"encrypted"`
}

// updateFolderEncryptionDefault toggles the encryption_default flag for a folder.
// New notes created in this folder will inherit this setting.
func (s *Server) updateFolderEncryptionDefault(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid folder id")
		return
	}

	var req UpdateFolderEncryptionDefaultRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.noteService.UpdateFolderEncryptionDefault(userID, id, req.Encrypted); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"encrypted": req.Encrypted,
	})
}

// getFolderEncryptionDefault returns the encryption_default status for a folder.
func (s *Server) getFolderEncryptionDefault(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid folder id")
		return
	}

	encrypted, err := s.noteService.GetFolderEncryptionDefault(userID, id)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"encrypted": encrypted,
	})
}
