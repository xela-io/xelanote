package api

import (
	"net/http"
	"os"
)

// getChangelog serves the CHANGELOG.md content as plain text.
// Tries multiple paths to support both development and Docker environments.
func (s *Server) getChangelog(w http.ResponseWriter, r *http.Request) {
	paths := []string{"CHANGELOG.md", "../CHANGELOG.md"}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write(data)
			return
		}
	}

	respondError(w, http.StatusNotFound, "changelog not found")
}
