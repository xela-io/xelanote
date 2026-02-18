package api

import (
	"net/http"

	"github.com/xela-io/xelanote/internal/service"
)

// ============================================================================
// Folder Sharing Handlers
// ============================================================================

// shareFolderHandler handles POST /api/folders/{id}/shares
func (s *Server) shareFolderHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	folderID, ok := parseIntParam(w, r, "id", "invalid folder ID")
	if !ok {
		return
	}

	var req ShareFolderRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if msg := validateShareCreateInput(req.Identifier, req.Role); msg != "" {
		respondError(w, http.StatusBadRequest, msg)
		return
	}

	share, err := s.sharingService.ShareFolder(userID, folderID, req.Identifier, req.Role)
	if err != nil {
		if status, msg, handled := mapShareCreateError(shareResourceFolder, err); handled {
			respondError(w, status, msg)
		} else {
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

	folderID, ok := parseIntParam(w, r, "id", "invalid folder ID")
	if !ok {
		return
	}

	shares, err := s.sharingService.GetFolderShares(userID, folderID)
	if err != nil {
		if status, msg, handled := mapShareAccessError(shareResourceFolder, err); handled {
			respondError(w, status, msg)
			return
		}
		s.respondInternalErr(w, "failed to get folder shares", err)
		return
	}

	if shares == nil {
		shares = []service.FolderShare{}
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

	folderID, ok := parseIntParam(w, r, "id", "invalid folder ID")
	if !ok {
		return
	}

	targetUserID, ok := parseIntParam(w, r, "userId", "invalid user ID")
	if !ok {
		return
	}

	var req UpdateShareRoleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if msg := validateShareRoleInput(req.Role); msg != "" {
		respondError(w, http.StatusBadRequest, msg)
		return
	}

	err := s.sharingService.UpdateFolderShareRole(userID, folderID, targetUserID, req.Role)
	if err != nil {
		if status, msg, handled := mapShareMutateError(shareResourceFolder, "update", err); handled {
			respondError(w, status, msg)
			return
		}
		s.respondInternalErr(w, "failed to update folder share role", err)
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

	folderID, ok := parseIntParam(w, r, "id", "invalid folder ID")
	if !ok {
		return
	}

	targetUserID, ok := parseIntParam(w, r, "userId", "invalid user ID")
	if !ok {
		return
	}

	err := s.sharingService.UnshareFolder(userID, folderID, targetUserID)
	if err != nil {
		if status, msg, handled := mapShareMutateError(shareResourceFolder, "remove", err); handled {
			respondError(w, status, msg)
			return
		}
		s.respondInternalErr(w, "failed to remove folder share", err)
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
		s.respondInternalErr(w, "failed to get shared folders", err)
		return
	}

	if folders == nil {
		folders = []service.SharedFolder{}
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

	folderID, ok := parseIntParam(w, r, "id", "invalid folder ID")
	if !ok {
		return
	}

	notes, err := s.sharingService.GetSharedFolderNotes(userID, folderID)
	if err != nil {
		s.respondInternalErr(w, "failed to get shared folder notes", err)
		return
	}

	if notes == nil {
		notes = []service.SharedNote{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"notes": notes})
}
