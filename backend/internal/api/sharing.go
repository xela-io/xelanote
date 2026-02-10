package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/service"
)

// ShareNoteRequest represents the request to share a note.
type ShareNoteRequest struct {
	Identifier string `json:"identifier"` // Username or email
	Role       string `json:"role"`       // "viewer" or "editor"
}

// UpdateShareRoleRequest represents the request to update a share role.
type UpdateShareRoleRequest struct {
	Role string `json:"role"` // "viewer" or "editor"
}

// SharedNoteUpdateRequest represents the request to update a shared note.
type SharedNoteUpdateRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Version int    `json:"version"`
}

// shareNote handles POST /api/notes/{id}/shares
func (s *Server) shareNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	var req ShareNoteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Identifier == "" {
		respondError(w, http.StatusBadRequest, "identifier (username or email) required")
		return
	}
	if req.Role != "viewer" && req.Role != "editor" {
		respondError(w, http.StatusBadRequest, "role must be 'viewer' or 'editor'")
		return
	}

	share, err := s.sharingService.ShareNote(userID, noteID, req.Identifier, req.Role)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrNotFound):
			respondError(w, http.StatusNotFound, "note not found")
		case errors.Is(err, service.ErrCannotShareEncrypted):
			respondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrCannotShareWithSelf):
			respondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrNotNoteOwner):
			respondError(w, http.StatusForbidden, "only the note owner can share")
		case errors.Is(err, service.ErrUserNotFound):
			respondError(w, http.StatusBadRequest, "unable to share with specified user")
		default:
			s.logger().Error("unexpected sharing error", "error", err)
			respondError(w, http.StatusInternalServerError, "an unexpected error occurred")
		}
		return
	}

	respondJSON(w, http.StatusCreated, share)
}

// getNoteShares handles GET /api/notes/{id}/shares
func (s *Server) getNoteShares(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	shares, err := s.sharingService.GetNoteShares(userID, noteID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		if errors.Is(err, service.ErrNotNoteOwner) {
			respondError(w, http.StatusForbidden, "only the note owner can view shares")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Ensure JSON array, not null
	if shares == nil {
		shares = []db.NoteShare{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"shares": shares})
}

// updateShareRole handles PUT /api/notes/{id}/shares/{userId}
func (s *Server) updateShareRole(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	targetUserIDStr := chi.URLParam(r, "userId")
	targetUserID, err := strconv.Atoi(targetUserIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req UpdateShareRoleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Role != "viewer" && req.Role != "editor" {
		respondError(w, http.StatusBadRequest, "role must be 'viewer' or 'editor'")
		return
	}

	err = s.sharingService.UpdateShareRole(userID, noteID, targetUserID, req.Role)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "share not found")
			return
		}
		if errors.Is(err, service.ErrNotNoteOwner) {
			respondError(w, http.StatusForbidden, "only the note owner can update shares")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// removeShare handles DELETE /api/notes/{id}/shares/{userId}
func (s *Server) removeShare(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	targetUserIDStr := chi.URLParam(r, "userId")
	targetUserID, err := strconv.Atoi(targetUserIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	err = s.sharingService.UnshareNote(userID, noteID, targetUserID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "share not found")
			return
		}
		if errors.Is(err, service.ErrNotNoteOwner) {
			respondError(w, http.StatusForbidden, "only the note owner can remove shares")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getSharedNotes handles GET /api/shared
func (s *Server) getSharedNotes(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	notes, err := s.sharingService.GetSharedNotesForUser(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Ensure JSON array, not null
	if notes == nil {
		notes = []db.SharedNote{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"notes": notes})
}

// getSharedNote handles GET /api/shared/{id}
func (s *Server) getSharedNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	note, err := s.sharingService.GetSharedNote(userID, noteID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "shared note not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, note)
}

// updateSharedNote handles PUT /api/shared/{id}
func (s *Server) updateSharedNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	var req SharedNoteUpdateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Version <= 0 {
		respondError(w, http.StatusBadRequest, "version required")
		return
	}

	note, err := s.sharingService.UpdateSharedNote(userID, noteID, req.Title, req.Content, req.Version)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "shared note not found")
			return
		}
		if errors.Is(err, db.ErrVersionMismatch) {
			respondError(w, http.StatusConflict, "version conflict - note was modified")
			return
		}
		respondError(w, http.StatusForbidden, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, note)
}

// ============================================================================
// Folder Sharing Handlers
// ============================================================================

// ShareFolderRequest represents the request to share a folder.
type ShareFolderRequest struct {
	Identifier string `json:"identifier"` // Username or email
	Role       string `json:"role"`       // "viewer" or "editor"
}

// PlacementRequest represents the request to place a shared note in a folder.
type PlacementRequest struct {
	FolderID int `json:"folder_id"`
}

// shareFolderHandler handles POST /api/folders/{id}/shares
func (s *Server) shareFolderHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	folderIDStr := chi.URLParam(r, "id")
	folderID, err := strconv.Atoi(folderIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid folder ID")
		return
	}

	var req ShareFolderRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Identifier == "" {
		respondError(w, http.StatusBadRequest, "identifier (username or email) required")
		return
	}
	if req.Role != "viewer" && req.Role != "editor" {
		respondError(w, http.StatusBadRequest, "role must be 'viewer' or 'editor'")
		return
	}

	share, err := s.sharingService.ShareFolder(userID, folderID, req.Identifier, req.Role)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrNotFound):
			respondError(w, http.StatusNotFound, "folder not found")
		case errors.Is(err, service.ErrCannotShareEncryptedFolder):
			respondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrFolderHasEncryptedNotes):
			respondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrCannotShareWithSelf):
			respondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrNotFolderOwner):
			respondError(w, http.StatusForbidden, "only the folder owner can share")
		case errors.Is(err, service.ErrUserNotFound):
			respondError(w, http.StatusBadRequest, "unable to share with specified user")
		default:
			s.logger().Error("unexpected sharing error", "error", err)
			respondError(w, http.StatusInternalServerError, "an unexpected error occurred")
		}
		return
	}

	respondJSON(w, http.StatusCreated, share)
}

// getFolderSharesHandler handles GET /api/folders/{id}/shares
func (s *Server) getFolderSharesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	folderIDStr := chi.URLParam(r, "id")
	folderID, err := strconv.Atoi(folderIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid folder ID")
		return
	}

	shares, err := s.sharingService.GetFolderShares(userID, folderID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "folder not found")
			return
		}
		if errors.Is(err, service.ErrNotFolderOwner) {
			respondError(w, http.StatusForbidden, "only the folder owner can view shares")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if shares == nil {
		shares = []db.FolderShare{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"shares": shares})
}

// updateFolderShareRoleHandler handles PUT /api/folders/{id}/shares/{userId}
func (s *Server) updateFolderShareRoleHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	folderIDStr := chi.URLParam(r, "id")
	folderID, err := strconv.Atoi(folderIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid folder ID")
		return
	}

	targetUserIDStr := chi.URLParam(r, "userId")
	targetUserID, err := strconv.Atoi(targetUserIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req UpdateShareRoleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Role != "viewer" && req.Role != "editor" {
		respondError(w, http.StatusBadRequest, "role must be 'viewer' or 'editor'")
		return
	}

	err = s.sharingService.UpdateFolderShareRole(userID, folderID, targetUserID, req.Role)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "share not found")
			return
		}
		if errors.Is(err, service.ErrNotFolderOwner) {
			respondError(w, http.StatusForbidden, "only the folder owner can update shares")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// removeFolderShareHandler handles DELETE /api/folders/{id}/shares/{userId}
func (s *Server) removeFolderShareHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	folderIDStr := chi.URLParam(r, "id")
	folderID, err := strconv.Atoi(folderIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid folder ID")
		return
	}

	targetUserIDStr := chi.URLParam(r, "userId")
	targetUserID, err := strconv.Atoi(targetUserIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	err = s.sharingService.UnshareFolder(userID, folderID, targetUserID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "share not found")
			return
		}
		if errors.Is(err, service.ErrNotFolderOwner) {
			respondError(w, http.StatusForbidden, "only the folder owner can remove shares")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getSharedFoldersHandler handles GET /api/shared/folders
func (s *Server) getSharedFoldersHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	folders, err := s.sharingService.GetSharedFoldersForUser(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if folders == nil {
		folders = []db.SharedFolder{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"folders": folders})
}

// getSharedFolderNotesHandler handles GET /api/shared/folders/{id}/notes
func (s *Server) getSharedFolderNotesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	folderIDStr := chi.URLParam(r, "id")
	folderID, err := strconv.Atoi(folderIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid folder ID")
		return
	}

	notes, err := s.sharingService.GetSharedFolderNotes(userID, folderID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if notes == nil {
		notes = []db.SharedNote{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"notes": notes})
}

// placeSharedNoteHandler handles POST /api/shared/{id}/placement
func (s *Server) placeSharedNoteHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	var req PlacementRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.FolderID <= 0 {
		respondError(w, http.StatusBadRequest, "folder_id required")
		return
	}

	err := s.sharingService.PlaceSharedNote(userID, noteID, req.FolderID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNoShareAccess):
			respondError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, service.ErrCannotPlaceOwnNote):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			s.logger().Error("unexpected placement error", "error", err)
			respondError(w, http.StatusInternalServerError, "an unexpected error occurred")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// removePlacementHandler handles DELETE /api/shared/{id}/placement
func (s *Server) removePlacementHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	if noteID == "" {
		respondError(w, http.StatusBadRequest, "note ID required")
		return
	}

	err := s.sharingService.RemovePlacement(userID, noteID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "placement not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// searchUsers handles GET /api/users/search
func (s *Server) searchUsers(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	query := r.URL.Query().Get("q")
	if len(query) < 3 {
		respondError(w, http.StatusBadRequest, "query must be at least 3 characters")
		return
	}

	users, err := s.sharingService.SearchUsers(query, userID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Ensure JSON array, not null
	if users == nil {
		users = []db.UserSearchResult{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"users": users})
}
