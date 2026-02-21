package api

import (
	"net/http"
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

	if msg := validateShareCreateInput(req.Identifier, req.Role); msg != "" {
		respondError(w, http.StatusBadRequest, msg)
		return
	}

	share, err := s.recipeService.ShareCollection(userID, collID, req.Identifier, req.Role)
	if err != nil {
		if status, msg, handled := mapShareCreateError(shareResourceCollection, err); handled {
			respondError(w, status, msg)
		} else {
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
		if status, msg, handled := mapShareAccessError(shareResourceCollection, err); handled {
			respondError(w, status, msg)
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

	if msg := validateShareRoleInput(req.Role); msg != "" {
		respondError(w, http.StatusBadRequest, msg)
		return
	}

	if err := s.recipeService.UpdateCollectionShareRole(userID, collID, targetUserID, req.Role); err != nil {
		if status, msg, handled := mapShareMutateError(shareResourceCollection, "update", err); handled {
			respondError(w, status, msg)
			return
		}
		s.respondInternalErr(w, "failed to share collection", err)
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
		if status, msg, handled := mapShareMutateError(shareResourceCollection, "remove", err); handled {
			respondError(w, status, msg)
			return
		}
		s.respondInternalErr(w, "failed to remove collection share", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
