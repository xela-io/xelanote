package api

import "github.com/go-chi/chi/v5"

// registerProtectedResourceRoutes wires feature/resource endpoints that represent core domain objects.
func (s *Server) registerProtectedResourceRoutes(r chi.Router) {
	s.registerNotesRoutes(r)
	s.registerSharedRoutes(r)
	s.registerFolderRoutes(r)
	s.registerTagsRoutes(r)
	s.registerTemplatesRoutes(r)
	s.registerSnippetsRoutes(r)
	s.registerFeaturesRoutes(r)
	s.registerJournalRoutes(r)
	s.registerRecipeRoutes(r)
	s.registerUserRoutes(r)
}

// registerProtectedUtilityRoutes wires cross-cutting endpoints (search, jobs, realtime, admin).
func (s *Server) registerProtectedUtilityRoutes(r chi.Router) {
	s.registerSearchExportRoutes(r)
	s.registerTrashRoutes(r)
	s.registerLLMRoutes(r)
	s.registerJobRoutes(r)
	s.registerGraphRoutes(r)
	s.registerWebsocketRoutes(r)
	s.registerAdminRoutes(r)
	s.registerTelemetryRoutes(r)
}
