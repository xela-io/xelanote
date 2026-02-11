package api

import "net/http"

// listNoteTitlesAIEnabled returns titles of notes with ai_enabled=true.
// Used for Claude API link suggestions (only AI-enabled notes are included).
func (s *Server) listNoteTitlesAIEnabled(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	titles, err := s.noteService.GetNoteTitlesAIEnabled(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to list AI-enabled note titles", err)
		return
	}

	if titles == nil {
		titles = []string{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"titles": titles,
	})
}
