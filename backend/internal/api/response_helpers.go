package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// ensureSlice returns an empty slice if items is nil, otherwise returns items unchanged.
// This ensures JSON serialization produces [] instead of null.
func ensureSlice[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}

// parseIntParam extracts a chi URL parameter by name and parses it as an int.
// On failure it responds with 400 and the provided message, returning false.
func parseIntParam(w http.ResponseWriter, r *http.Request, param, errMsg string) (int, bool) {
	v, err := strconv.Atoi(chi.URLParam(r, param))
	if err != nil {
		respondError(w, http.StatusBadRequest, errMsg)
		return 0, false
	}
	return v, true
}
