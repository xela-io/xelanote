package api

import (
	"errors"
	"net/http"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/service"
)

// --- Collection Sharing handlers ---

// shareCollection handles POST /api/recipes/collections/{id}/shares
func (s *Server) shareCollection(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collID, ok := parseIntParam(w, r, "id", "invalid collection id")
	if !ok {
		return
	}

	var req ShareCollectionRequest
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

	share, err := s.recipeService.ShareCollection(userID, collID, req.Identifier, req.Role)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrNotFound):
			respondError(w, http.StatusNotFound, "collection not found")
		case errors.Is(err, service.ErrNotCollectionOwner):
			respondError(w, http.StatusForbidden, "only the collection owner can share")
		case errors.Is(err, service.ErrCollectionHasEncryptedRecipes):
			respondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrCannotShareWithSelf):
			respondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrCollectionAlreadyShared):
			respondError(w, http.StatusConflict, err.Error())
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

// getCollectionShares handles GET /api/recipes/collections/{id}/shares
func (s *Server) getCollectionShares(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collID, ok := parseIntParam(w, r, "id", "invalid collection id")
	if !ok {
		return
	}

	shares, err := s.recipeService.GetCollectionShares(userID, collID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "collection not found")
			return
		}
		if errors.Is(err, service.ErrNotCollectionOwner) {
			respondError(w, http.StatusForbidden, "only the collection owner can view shares")
			return
		}
		s.respondInternalErr(w, "failed to get collection shares", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"shares": shares,
	})
}

// updateCollectionShareRole handles PUT /api/recipes/collections/{id}/shares/{userId}
func (s *Server) updateCollectionShareRole(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collID, ok := parseIntParam(w, r, "id", "invalid collection id")
	if !ok {
		return
	}

	targetUserID, ok := parseIntParam(w, r, "userId", "invalid user id")
	if !ok {
		return
	}

	var req UpdateCollectionShareRoleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Role != "viewer" && req.Role != "editor" {
		respondError(w, http.StatusBadRequest, "role must be 'viewer' or 'editor'")
		return
	}

	if err := s.recipeService.UpdateCollectionShareRole(userID, collID, targetUserID, req.Role); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "share not found")
			return
		}
		if errors.Is(err, service.ErrNotCollectionOwner) {
			respondError(w, http.StatusForbidden, "only the collection owner can manage shares")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// removeCollectionShare handles DELETE /api/recipes/collections/{id}/shares/{userId}
func (s *Server) removeCollectionShare(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	collID, ok := parseIntParam(w, r, "id", "invalid collection id")
	if !ok {
		return
	}

	targetUserID, ok := parseIntParam(w, r, "userId", "invalid user id")
	if !ok {
		return
	}

	if err := s.recipeService.UnshareCollection(userID, collID, targetUserID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "share not found")
			return
		}
		if errors.Is(err, service.ErrNotCollectionOwner) {
			respondError(w, http.StatusForbidden, "only the collection owner can manage shares")
			return
		}
		s.respondInternalErr(w, "failed to remove collection share", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
