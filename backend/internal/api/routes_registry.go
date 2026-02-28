package api

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// registerProtectedResourceRoutes wires feature/resource endpoints that represent core domain objects.
// SSE streaming routes are registered without timeout; everything else gets a 30s timeout.
func (s *Server) registerProtectedResourceRoutes(r chi.Router) {
	// Long-lived SSE endpoints (no timeout)
	s.registerNotesStreamingRoutes(r)

	// All other resource routes with request timeout
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(30 * time.Second))
		s.registerNotesRoutes(r)
		s.registerSharedRoutes(r)
		s.registerFolderRoutes(r)
		s.registerTagsRoutes(r)
		s.registerTemplatesRoutes(r)
		s.registerSnippetsRoutes(r)
		s.registerFeaturesRoutes(r)
		s.registerJournalRoutes(r)
		s.registerRecipeRoutes(r)
		s.registerCanvasRoutes(r)
		s.registerShoppingRoutes(r)
		s.registerUserRoutes(r)
	})
}

// registerProtectedUtilityRoutes wires cross-cutting endpoints (search, jobs, realtime, admin).
// WebSocket routes are registered without timeout; everything else gets a 30s timeout.
func (s *Server) registerProtectedUtilityRoutes(r chi.Router) {
	// Long-lived WebSocket connections (no timeout)
	s.registerWebsocketRoutes(r)

	// Transcription needs a longer timeout (audio upload + Whisper processing)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(120 * time.Second))
		s.registerTranscribeRoutes(r)
	})

	// All other utility routes with request timeout
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(30 * time.Second))
		s.registerSearchExportRoutes(r)
		s.registerTrashRoutes(r)
		s.registerLLMRoutes(r)
		s.registerJobRoutes(r)
		s.registerGraphRoutes(r)
		s.registerAdminRoutes(r)
		s.registerTelemetryRoutes(r)
	})
}
