package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/service"
)

// VersionListResponse represents a paginated list of versions.
type VersionListResponse struct {
	Versions   []service.NoteVersion `json:"versions"`
	NextCursor string                `json:"next_cursor,omitempty"`
	Total      int                   `json:"total"`
}

// CompareResponse represents two versions for comparison.
type CompareResponse struct {
	Version1 *service.NoteVersion `json:"version1"`
	Version2 *service.NoteVersion `json:"version2"`
}

// listVersions returns a paginated list of versions for a note.
// GET /api/notes/{id}/versions
func (s *Server) listVersions(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")

	// Verify note exists and belongs to user
	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil {
		s.respondInternalErr(w, "failed to get note for version listing", err)
		return
	}
	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = min(parsed, 200) // Cap at 200 for version history
		}
	}
	cursor := r.URL.Query().Get("cursor")

	versions, nextCursor, total, err := s.noteService.GetNoteVersions(userID, noteID, limit, cursor)
	if err != nil {
		s.respondInternalErr(w, "failed to list note versions", err)
		return
	}

	if versions == nil {
		versions = []service.NoteVersion{}
	}

	respondJSON(w, http.StatusOK, VersionListResponse{
		Versions:   versions,
		NextCursor: nextCursor,
		Total:      total,
	})
}

// getVersion returns a specific version of a note.
// GET /api/notes/{id}/versions/{version}
func (s *Server) getVersion(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")

	version, ok := parseIntParam(w, r, "version", "invalid version number")
	if !ok {
		return
	}

	// Verify note exists and belongs to user
	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil {
		s.respondInternalErr(w, "failed to get note for version lookup", err)
		return
	}
	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	v, err := s.noteService.GetNoteVersion(userID, noteID, version)
	if err != nil {
		s.respondInternalErr(w, "failed to get note version", err)
		return
	}
	if v == nil {
		respondError(w, http.StatusNotFound, "version not found")
		return
	}

	respondJSON(w, http.StatusOK, v)
}

// compareVersions returns two versions for client-side diff comparison.
// GET /api/notes/{id}/versions/compare?v1=X&v2=Y
func (s *Server) compareVersions(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")

	v1Str := r.URL.Query().Get("v1")
	v2Str := r.URL.Query().Get("v2")

	if v1Str == "" || v2Str == "" {
		respondError(w, http.StatusBadRequest, "v1 and v2 query parameters are required")
		return
	}

	v1, err := strconv.Atoi(v1Str)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid v1 version number")
		return
	}

	v2, err := strconv.Atoi(v2Str)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid v2 version number")
		return
	}

	// Verify note exists and belongs to user
	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil {
		s.respondInternalErr(w, "failed to get note for version comparison", err)
		return
	}
	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return
	}

	version1, err := s.noteService.GetNoteVersion(userID, noteID, v1)
	if err != nil {
		s.respondInternalErr(w, "failed to get version v1", err)
		return
	}
	if version1 == nil {
		respondError(w, http.StatusNotFound, "version v1 not found")
		return
	}

	version2, err := s.noteService.GetNoteVersion(userID, noteID, v2)
	if err != nil {
		s.respondInternalErr(w, "failed to get version v2", err)
		return
	}
	if version2 == nil {
		respondError(w, http.StatusNotFound, "version v2 not found")
		return
	}

	respondJSON(w, http.StatusOK, CompareResponse{
		Version1: version1,
		Version2: version2,
	})
}

// restoreVersion restores a note to a previous version.
// POST /api/notes/{id}/versions/{version}/restore
func (s *Server) restoreVersion(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	noteID := chi.URLParam(r, "id")
	targetVersion, ok := parseIntParam(w, r, "version", "invalid version number")
	if !ok {
		return
	}

	// Check If-Match header for optimistic locking
	ifMatch := r.Header.Get("If-Match")
	if ifMatch == "" {
		respondError(w, http.StatusBadRequest, "If-Match header required")
		return
	}

	currentVersion, ok2 := s.resolveETagVersion(w, userID, noteID, ifMatch)
	if !ok2 {
		return
	}

	note, err := s.noteService.RestoreVersion(userID, noteID, targetVersion, currentVersion)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "note or version not found")
			return
		}
		if errors.Is(err, service.ErrVersionMismatch) {
			respondError(w, http.StatusConflict, "version mismatch - note was modified")
			return
		}
		s.respondInternalErr(w, "failed to restore version", err)
		return
	}

	w.Header().Set("ETag", generateETag(note.ID, note.Version))
	respondJSON(w, http.StatusOK, note)
}
