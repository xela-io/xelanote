package api

import (
	"net/http"

	"github.com/xela-io/xelanote/internal/utils"
)

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
		respondError(w, http.StatusBadRequest, "invalid folder path")
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
	// parentID bleibt nil -> nur fuer "/" beim Erstellen

	folder, err := s.noteService.CreateFolder(userID, req.Path, parentID)
	if err != nil {
		s.respondInternalErr(w, "failed to create folder", err)
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
		s.respondInternalErr(w, "failed to list folders", err)
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

	id, ok := parseIntParam(w, r, "id", "invalid folder id")
	if !ok {
		return
	}

	var req MoveFolderRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Virtual root ("/") is allowed as move target, but regular validation
	// still applies for non-root paths.
	if req.NewParentPath != "/" {
		if err := utils.ValidateFolderPath(req.NewParentPath); err != nil {
			respondError(w, http.StatusBadRequest, "invalid folder path")
			return
		}
	}

	err := s.noteService.MoveFolder(userID, id, req.NewParentPath)
	if err != nil {
		s.respondInternalErr(w, "failed to move folder", err)
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

	id, ok := parseIntParam(w, r, "id", "invalid folder id")
	if !ok {
		return
	}

	err := s.noteService.DeleteFolder(userID, id)
	if err != nil {
		s.respondInternalErr(w, "failed to delete folder", err)
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

	id, ok := parseIntParam(w, r, "id", "invalid folder id")
	if !ok {
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

	err := s.noteService.RenameFolder(userID, id, req.NewName)
	if err != nil {
		s.respondInternalErr(w, "failed to rename folder", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
