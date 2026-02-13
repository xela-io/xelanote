package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/service"
	"github.com/xela-io/xelanote/internal/websocket"
)

// DecryptNoteRequest represents the request body for decrypting a note.
type DecryptNoteRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	// Recipe fields (optional, only for recipe notes)
	RecipeMetadata    *service.RecipeMetadata    `json:"recipe_metadata,omitempty"`
	RecipeIngredients []service.RecipeIngredient `json:"recipe_ingredients,omitempty"`
}

func (s *Server) decryptNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	// Check If-Match header for optimistic locking
	ifMatch := r.Header.Get("If-Match")
	if ifMatch == "" {
		respondError(w, http.StatusBadRequest, "If-Match header required")
		return
	}

	version, ok2 := s.resolveETagVersion(w, userID, id, ifMatch)
	if !ok2 {
		return
	}

	var req DecryptNoteRequest
	if err := decodeJSONWithLimit(w, r, &req, MaxLargeJSONBodySize); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateNoteFields(req.Title, req.Content, ""); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	var note *service.Note
	var err error
	if req.RecipeMetadata != nil || len(req.RecipeIngredients) > 0 {
		note, err = s.noteService.DecryptRecipeNote(userID, id, req.Title, req.Content, version,
			req.RecipeMetadata, req.RecipeIngredients)
	} else {
		note, err = s.noteService.DecryptNote(userID, id, req.Title, req.Content, version)
	}
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		if errors.Is(err, service.ErrVersionMismatch) {
			respondError(w, http.StatusConflict, "version mismatch - note was modified")
			return
		}
		if err.Error() == "note is not encrypted" {
			respondError(w, http.StatusBadRequest, "note is not encrypted")
			return
		}
		s.respondInternalErr(w, "failed to decrypt note", err)
		return
	}

	// Broadcast update to WebSocket clients
	payload, err := json.Marshal(note)
	if err != nil {
		s.logger().Error("failed to encode note.updated payload", "err", err, "note_id", note.ID)
	} else {
		s.wsManager.BroadcastToUser(userID, websocket.Message{
			Type:    "note.updated",
			Payload: payload,
		})
	}

	w.Header().Set("ETag", generateETag(note.ID, note.Version))
	respondJSON(w, http.StatusOK, note)
}

// BatchReencryptDEKsRequest represents a batch update of wrapped DEKs
type BatchReencryptDEKsRequest struct {
	Updates []struct {
		NoteID     string `json:"note_id"`
		WrappedDEK string `json:"wrapped_dek"` // Base64-encoded
	} `json:"updates"`
}

// batchReencryptDEKs updates wrapped DEKs for multiple notes in a single transaction
func (s *Server) batchReencryptDEKs(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req BatchReencryptDEKsRequest
	if err := decodeJSONWithLimit(w, r, &req, MaxLargeJSONBodySize); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Updates) == 0 {
		respondError(w, http.StatusBadRequest, "no updates provided")
		return
	}

	// Validate all wrapped DEKs
	for i, update := range req.Updates {
		if update.NoteID == "" {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("note_id required for update %d", i))
			return
		}
		if err := ValidateWrappedDEK(update.WrappedDEK); err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid wrapped_dek for update %d: %v", i, err))
			return
		}
	}

	// Perform batch update
	updatedCount, err := s.noteService.BatchUpdateWrappedDEKs(userID, req.Updates)
	if err != nil {
		s.logger().Error("failed to batch reencrypt DEKs", "error", err, "user_id", userID)
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "one or more notes not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to update DEKs")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"updated_count": updatedCount,
		"message":       "DEKs successfully re-encrypted",
	})
}
