package api

import (
	"net/http"

	"github.com/xela-io/xelanote/internal/db"
)

// getGlobalGraph handles GET /api/graph
// Returns the global graph for the authenticated user.
// Query parameters:
//   - folder: Filter by folder path (optional)
func (s *Server) getGlobalGraph(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	// Parse query params
	folderPath := r.URL.Query().Get("folder")

	var graphData *db.GraphData
	var err error

	// Get filtered or global graph
	if folderPath != "" {
		graphData, err = s.graphService.GetFilteredGraph(userID, folderPath)
	} else {
		graphData, err = s.graphService.GetGlobalGraph(userID)
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load graph")
		return
	}

	respondJSON(w, http.StatusOK, graphData)
}
