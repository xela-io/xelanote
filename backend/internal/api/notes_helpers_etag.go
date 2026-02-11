package api

import "net/http"

func (s *Server) resolveETagVersion(w http.ResponseWriter, userID int, noteID, ifMatch string) (int, bool) {
	if ifMatch == "" {
		respondError(w, http.StatusBadRequest, "If-Match header required")
		return 0, false
	}

	note, err := s.noteService.GetNote(userID, noteID)
	if err != nil {
		s.respondInternalErr(w, "failed to get note", err)
		return 0, false
	}
	if note == nil {
		respondError(w, http.StatusNotFound, "note not found")
		return 0, false
	}

	version, err := parseETag(ifMatch, noteID, note.Version)
	if err != nil {
		respondError(w, http.StatusPreconditionFailed, "invalid or outdated ETag")
		return 0, false
	}

	return version, true
}
