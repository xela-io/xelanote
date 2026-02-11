package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/xela-io/xelanote/internal/service"
)

// validateNoteFields checks common field constraints for note create/update/decrypt.
// folderPath is only validated if non-empty (decrypt doesn't send it).
func validateNoteFields(title, content, folderPath string) error {
	if title == "" {
		return fmt.Errorf("title is required")
	}
	if len(title) > MaxNoteTitleLength {
		return fmt.Errorf("title too long")
	}
	if len(content) > MaxNoteContentLength {
		return fmt.Errorf("content too long")
	}
	if len(folderPath) > MaxFolderPathLength {
		return fmt.Errorf("folder path too long")
	}
	return nil
}

// validateClientLinks validates client-provided links and returns an error response if invalid.
// Returns (linkTitles, true) on success, or (nil, false) if a validation error was sent.
func validateClientLinks(w http.ResponseWriter, links []ClientLink) ([]string, bool) {
	if len(links) > service.MaxLinksPerNote {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("too many links (max %d)", service.MaxLinksPerNote))
		return nil, false
	}
	for i, l := range links {
		if len(l.TargetTitle) > service.MaxLinkTitleLength {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("link %d title too long (max %d chars)", i, service.MaxLinkTitleLength))
			return nil, false
		}
	}
	titles := make([]string, len(links))
	for i, l := range links {
		titles[i] = l.TargetTitle
	}
	return titles, true
}

// handleCreateNoteError handles errors from note creation with appropriate HTTP responses.
func handleCreateNoteError(w http.ResponseWriter, err error, title string, isJournal bool) {
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		// SQLite reports: "UNIQUE constraint failed: notes.user_id, notes.journal_date"
		if isJournal && strings.Contains(err.Error(), "journal_date") {
			respondError(w, http.StatusConflict,
				"Journal für dieses Datum existiert bereits")
			return
		}
		// Fallback: title conflict
		respondError(w, http.StatusConflict,
			fmt.Sprintf("Eine Notiz mit dem Titel '%s' existiert bereits in diesem Ordner", title))
		return
	}
	// Feature not enabled
	if strings.Contains(err.Error(), "journal feature not enabled") {
		respondError(w, http.StatusForbidden, "journal feature not enabled")
		return
	}
	if strings.Contains(err.Error(), "recipe feature not enabled") {
		respondError(w, http.StatusForbidden, "recipe feature not enabled")
		return
	}
	respondError(w, http.StatusInternalServerError, "failed to create note")
}
