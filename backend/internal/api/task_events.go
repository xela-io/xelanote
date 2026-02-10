package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/db"
)

type recordTaskEventRequest struct {
	TaskText           *string `json:"task_text"`
	TaskIndex          int     `json:"task_index"`
	EncryptedTaskText  *string `json:"encrypted_task_text"`
	WrappedDEK         *string `json:"wrapped_dek"`
	EncryptionMetadata *string `json:"encryption_metadata"`
	EventType          string  `json:"event_type"`
}

func (s *Server) recordTaskEvent(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(int)
	noteID := chi.URLParam(r, "id")

	var req recordTaskEventRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate event_type
	if req.EventType != "completed" && req.EventType != "reopened" {
		respondError(w, http.StatusBadRequest, "event_type must be 'completed' or 'reopened'")
		return
	}

	// Validate task_index
	if req.TaskIndex < 0 {
		respondError(w, http.StatusBadRequest, "task_index must be >= 0")
		return
	}

	// Validate XOR: exactly one of task_text / encrypted_task_text must be set
	hasPlaintext := req.TaskText != nil
	hasEncrypted := req.EncryptedTaskText != nil

	if hasPlaintext == hasEncrypted {
		respondError(w, http.StatusBadRequest, "exactly one of task_text or encrypted_task_text must be set")
		return
	}

	// Validate plaintext fields
	if hasPlaintext {
		trimmed := strings.TrimSpace(*req.TaskText)
		if trimmed == "" {
			respondError(w, http.StatusBadRequest, "task_text must not be empty")
			return
		}
		if len(trimmed) > 500 {
			trimmed = trimmed[:500]
		}
		req.TaskText = &trimmed
	}

	// Validate encrypted fields
	if hasEncrypted {
		if *req.EncryptedTaskText == "" {
			respondError(w, http.StatusBadRequest, "encrypted_task_text must not be empty")
			return
		}
		if req.WrappedDEK == nil || *req.WrappedDEK == "" {
			respondError(w, http.StatusBadRequest, "wrapped_dek is required for encrypted events")
			return
		}
		if req.EncryptionMetadata == nil || *req.EncryptionMetadata == "" {
			respondError(w, http.StatusBadRequest, "encryption_metadata is required for encrypted events")
			return
		}
	}

	// Verify note exists and user is owner (Phase 1: no shared notes)
	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil || note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	// Build event
	event := db.TaskEvent{
		NoteID:      noteID,
		ActorUserID: userID,
		TaskText:    req.TaskText,
		TaskIndex:   req.TaskIndex,
		EventType:   req.EventType,
	}

	if hasEncrypted {
		event.TextEncrypted = true
		event.EncryptedTaskText = req.EncryptedTaskText
		event.WrappedDEK = req.WrappedDEK
		event.EncryptionMetadata = req.EncryptionMetadata
	}

	if err := s.noteService.GetDB().RecordTaskEvent(event); err != nil {
		s.logger().Error("failed to record task event", "error", err, "note_id", noteID)
		respondError(w, http.StatusInternalServerError, "failed to record task event")
		return
	}

	w.WriteHeader(http.StatusCreated)
}
