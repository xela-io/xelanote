package api

import (
	"net/http"
	"strconv"

	"github.com/xela-io/xelanote/internal/db"
)

// Template API Request/Response types

type CreateTemplateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Title       string `json:"title"`
	Content     string `json:"content"`
}

type UpdateTemplateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Title       string `json:"title"`
	Content     string `json:"content"`
}

// Validation constants (match db layer)
const (
	MaxTemplateContentSize = 102400 // 100KB
	MaxTemplateTitleSize   = 200
	MaxTemplateNameSize    = 100
)

// validateTemplateRequest validates a template request.
func validateTemplateRequest(name, title, content string) error {
	if len(name) == 0 || len(name) > MaxTemplateNameSize {
		respondErrorMsg := "name must be 1-" + strconv.Itoa(MaxTemplateNameSize) + " characters"
		return &validationError{respondErrorMsg}
	}
	if len(title) == 0 || len(title) > MaxTemplateTitleSize {
		respondErrorMsg := "title must be 1-" + strconv.Itoa(MaxTemplateTitleSize) + " characters"
		return &validationError{respondErrorMsg}
	}
	if len(content) > MaxTemplateContentSize {
		respondErrorMsg := "content must not exceed " + strconv.Itoa(MaxTemplateContentSize) + " bytes"
		return &validationError{respondErrorMsg}
	}
	return nil
}

// getAllTemplates returns all templates for the authenticated user.
func (s *Server) getAllTemplates(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	templates, err := s.templateService.GetAllTemplates(userID)
	if err != nil {
		s.respondInternalErr(w, "failed to list templates", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"templates": templates,
	})
}

// getTemplate returns a single template by ID.
func (s *Server) getTemplate(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	templateID, ok := parseIntParam(w, r, "id", "invalid template id")
	if !ok {
		return
	}

	template, err := s.templateService.GetTemplate(userID, templateID)
	if err == db.ErrNotFound {
		respondError(w, http.StatusNotFound, "template not found")
		return
	}
	if err != nil {
		s.respondInternalErr(w, "failed to get template", err)
		return
	}

	respondJSON(w, http.StatusOK, template)
}

// createTemplate creates a new template.
func (s *Server) createTemplate(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req CreateTemplateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := validateTemplateRequest(req.Name, req.Title, req.Content); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	template, err := s.templateService.CreateTemplate(userID, req.Name, req.Description, req.Title, req.Content)
	if err != nil {
		s.respondInternalErr(w, "failed to create template", err)
		return
	}

	respondJSON(w, http.StatusCreated, template)
}

// updateTemplate updates an existing template.
func (s *Server) updateTemplate(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	templateID, ok := parseIntParam(w, r, "id", "invalid template id")
	if !ok {
		return
	}

	var req UpdateTemplateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if err := validateTemplateRequest(req.Name, req.Title, req.Content); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	err := s.templateService.UpdateTemplate(userID, templateID, req.Name, req.Description, req.Title, req.Content)
	if err == db.ErrNotFound {
		respondError(w, http.StatusNotFound, "template not found")
		return
	}
	if err != nil {
		s.respondInternalErr(w, "failed to update template", err)
		return
	}

	// Return updated template
	template, err := s.templateService.GetTemplate(userID, templateID)
	if err != nil {
		s.respondInternalErr(w, "failed to get updated template", err)
		return
	}

	respondJSON(w, http.StatusOK, template)
}

// deleteTemplate deletes a template.
func (s *Server) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	templateID, ok := parseIntParam(w, r, "id", "invalid template id")
	if !ok {
		return
	}

	err := s.templateService.DeleteTemplate(userID, templateID)
	if err == db.ErrNotFound {
		respondError(w, http.StatusNotFound, "template not found")
		return
	}
	if err != nil {
		s.respondInternalErr(w, "failed to delete template", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
