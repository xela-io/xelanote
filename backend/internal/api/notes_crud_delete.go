package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/websocket"
)

func (s *Server) deleteNote(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	id := chi.URLParam(r, "id")

	err := s.noteService.DeleteNote(userID, id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		s.respondInternalErr(w, "failed to delete note", err)
		return
	}

	// Broadcast deletion to WebSocket clients
	payload, err := json.Marshal(map[string]string{"id": id})
	if err != nil {
		s.logger().Error("failed to encode note.deleted payload", "err", err, "note_id", id)
	} else {
		s.wsManager.BroadcastToUser(userID, websocket.Message{
			Type:    "note.deleted",
			Payload: payload,
		})
	}

	w.WriteHeader(http.StatusNoContent)
}
