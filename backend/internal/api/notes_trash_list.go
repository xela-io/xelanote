package api

import (
	"net/http"
	"strconv"
)

func (s *Server) listTrash(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = min(parsed, 500) // Cap at 500 to prevent memory exhaustion
		}
	}
	cursor := r.URL.Query().Get("cursor")

	notes, nextCursor, err := s.noteService.ListDeletedNotes(userID, limit, cursor)
	if err != nil {
		s.respondInternalErr(w, "failed to list trash", err)
		return
	}

	respondJSON(w, http.StatusOK, NoteListResponse{
		Notes:      ensureSlice(notes),
		NextCursor: nextCursor,
	})
}

func (s *Server) getTrashCount(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	count, err := s.noteService.GetDeletedNotesCount(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to get trash count", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"count": count,
	})
}
