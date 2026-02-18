package api

import (
	"net/http"
	"strconv"

	"github.com/xela-io/xelanote/internal/constraints"
	"github.com/xela-io/xelanote/internal/service"
)

// Snippet API Request/Response types

type CreateSnippetRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Shortcut    string `json:"shortcut"`
}

type UpdateSnippetRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Shortcut    string `json:"shortcut"`
}

// validateSnippetRequest validates a snippet request.
func validateSnippetRequest(name, content string) error {
	if len(name) == 0 || len(name) > constraints.MaxSnippetNameSize {
		respondErrorMsg := "name must be 1-" + strconv.Itoa(constraints.MaxSnippetNameSize) + " characters"
		return &validationError{respondErrorMsg}
	}
	if len(content) > constraints.MaxSnippetContentSize {
		respondErrorMsg := "content must not exceed " + strconv.Itoa(constraints.MaxSnippetContentSize) + " bytes"
		return &validationError{respondErrorMsg}
	}
	return nil
}

// getAllSnippets returns all snippets for the authenticated user.
func (s *Server) getAllSnippets(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	snippets, err := s.snippetService.GetAllSnippets(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to list snippets", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"snippets": snippets,
	})
}

// getSnippet returns a single snippet by ID.
func (s *Server) getSnippet(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	snippetID, ok := parseIntParam(w, r, "id", "invalid snippet id")
	if !ok {
		return
	}

	snippet, err := s.snippetService.GetSnippet(userID, snippetID)
	if err == service.ErrNotFound {
		respondError(w, http.StatusNotFound, "snippet not found")
		return
	}
	if err != nil {
		s.respondInternalErr(w, "failed to get snippet", err)
		return
	}

	respondJSON(w, http.StatusOK, snippet)
}

// createSnippet creates a new snippet.
func (s *Server) createSnippet(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req CreateSnippetRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := validateSnippetRequest(req.Name, req.Content); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	snippet, err := s.snippetService.CreateSnippet(userID, req.Name, req.Description, req.Content, req.Shortcut)
	if err != nil {
		s.respondInternalErr(w, "failed to create snippet", err)
		return
	}

	respondJSON(w, http.StatusCreated, snippet)
}

// updateSnippet updates an existing snippet.
func (s *Server) updateSnippet(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	snippetID, ok := parseIntParam(w, r, "id", "invalid snippet id")
	if !ok {
		return
	}

	var req UpdateSnippetRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := validateSnippetRequest(req.Name, req.Content); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	err := s.snippetService.UpdateSnippet(userID, snippetID, req.Name, req.Description, req.Content, req.Shortcut)
	if err == service.ErrNotFound {
		respondError(w, http.StatusNotFound, "snippet not found")
		return
	}
	if err != nil {
		s.respondInternalErr(w, "failed to update snippet", err)
		return
	}

	// Return updated snippet
	snippet, err := s.snippetService.GetSnippet(userID, snippetID)
	if err != nil {
		s.respondInternalErr(w, "failed to get updated snippet", err)
		return
	}

	respondJSON(w, http.StatusOK, snippet)
}

// deleteSnippet deletes a snippet.
func (s *Server) deleteSnippet(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	snippetID, ok := parseIntParam(w, r, "id", "invalid snippet id")
	if !ok {
		return
	}

	err := s.snippetService.DeleteSnippet(userID, snippetID)
	if err == service.ErrNotFound {
		respondError(w, http.StatusNotFound, "snippet not found")
		return
	}
	if err != nil {
		s.respondInternalErr(w, "failed to delete snippet", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
