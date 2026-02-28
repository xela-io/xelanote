package api

import "github.com/go-chi/chi/v5"

func (s *Server) registerNotesRoutes(r chi.Router) {
	r.Route("/notes", func(r chi.Router) {
		r.Get("/", s.listNotes)
		r.Get("/titles", s.listNoteTitles)                     // Lightweight endpoint for link suggestions
		r.Get("/titles/ai-enabled", s.listNoteTitlesAIEnabled) // Only AI-enabled notes for Claude link suggestions
		r.Post("/", s.createNote)
		r.Post("/reorder", s.reorderNotes)
		r.Get("/{id}", s.getNote)
		r.Put("/{id}", s.updateNote)
		r.Delete("/{id}", s.deleteNote)
		r.Post("/{id}/rename", s.renameNote)
		r.Get("/{id}/backlinks", s.getBacklinks)
		r.Put("/{id}/color", s.updateNoteColor)
		r.Get("/{id}/user-state", s.getNoteUserState)
		r.Put("/{id}/user-state", s.updateNoteUserState)

		// AI-enabled (Claude API opt-in) endpoints
		r.Get("/{id}/ai-enabled", s.getNoteAIEnabled)
		r.Put("/{id}/ai-enabled", s.updateNoteAIEnabled)

		// LLM endpoints (rate-limited with shared limiter)
		// NOTE: GET /{id}/summarize/stream (SSE) is registered separately
		// in registerNotesStreamingRoutes without timeout middleware.
		r.With(rateLimitMiddleware(s.llmLimiter)).Post("/{id}/summarize", s.summarizeNote)
		r.With(rateLimitMiddleware(s.llmLimiter)).Post("/{id}/summarize/prepare", s.prepareSummarizeStream)
		r.With(rateLimitMiddleware(s.llmLimiter)).Post("/{id}/suggest-tags", s.suggestTags)
		r.With(rateLimitMiddleware(s.llmLimiter)).Post("/{id}/suggest-links", s.suggestLinks)
		r.With(rateLimitMiddleware(s.llmLimiter)).Post("/{id}/format-markdown", s.formatMarkdown)
		r.With(rateLimitMiddleware(s.llmLimiter)).Post("/{id}/ai-transform", s.aiTransform)

		// Encryption toggle
		r.Post("/{id}/decrypt", s.decryptNote)

		// Batch DEK re-encryption (for password changes)
		r.Post("/batch-reencrypt-deks", s.batchReencryptDEKs)

		// Tag endpoints for notes
		r.Get("/{id}/tags", s.getNoteTags)
		r.Put("/{id}/tags", s.setNoteTags)

		// Version history endpoints
		r.Get("/{id}/versions", s.listVersions)
		r.Get("/{id}/versions/compare", s.compareVersions)
		r.With(rateLimitMiddleware(s.llmLimiter)).Post("/{id}/versions/delta-summary", s.summarizeVersionDelta)
		r.Get("/{id}/versions/{version}", s.getVersion)
		r.Post("/{id}/versions/{version}/restore", s.restoreVersion)

		// Sharing endpoints (per-note)
		r.With(rateLimitMiddleware(s.shareLimiter)).Post("/{id}/shares", s.shareNote)
		r.Get("/{id}/shares", s.getNoteShares)
		r.Put("/{id}/shares/{userId}", s.updateShareRole)
		r.Delete("/{id}/shares/{userId}", s.removeShare)

		// Task event logging
		r.Post("/{id}/task-events", s.recordTaskEvent)
	})
}

func (s *Server) registerSharedRoutes(r chi.Router) {
	// Shared notes endpoints (notes shared WITH current user)
	r.Get("/shared", s.getSharedNotes)
	r.Get("/shared/folders", s.getSharedFoldersHandler)
	r.Get("/shared/folders/{id}/notes", s.getSharedFolderNotesHandler)
	r.Get("/shared/recipes", s.listSharedRecipes)
	r.Get("/shared/collections", s.listSharedCollections)
	r.Get("/shared/collections/{id}/items", s.listSharedCollectionItems)
	r.Post("/shared/collections/{id}/items", s.addToSharedCollection)
	r.Delete("/shared/collections/{id}/items/{noteId}", s.removeFromSharedCollection)
	r.Get("/shared/{id}", s.getSharedNote)
	r.Put("/shared/{id}", s.updateSharedNote)
	r.Post("/shared/{id}/placement", s.placeSharedNoteHandler)
	r.Delete("/shared/{id}/placement", s.removePlacementHandler)

	// User search for share dialog
	r.With(rateLimitMiddleware(s.userSearchLimiter)).Get("/users/search", s.searchUsers)
}

func (s *Server) registerFolderRoutes(r chi.Router) {
	r.Route("/folders", func(r chi.Router) {
		r.Get("/", s.getAllFolders)
		r.Post("/", s.createFolder)
		r.Post("/reorder", s.reorderFolders)
		r.Put("/{id}/move", s.moveFolder)
		r.Put("/{id}/rename", s.renameFolder)
		r.Put("/{id}/color", s.updateFolderColor)
		r.Delete("/{id}", s.deleteFolder)

		// AI-enabled default (Claude API opt-in) endpoints
		r.Get("/{id}/ai-enabled", s.getFolderAIEnabledDefault)
		r.Put("/{id}/ai-enabled", s.updateFolderAIEnabledDefault)

		// Encryption default endpoints
		r.Get("/{id}/encryption-default", s.getFolderEncryptionDefault)
		r.Put("/{id}/encryption-default", s.updateFolderEncryptionDefault)

		// Folder sharing endpoints
		r.With(rateLimitMiddleware(s.shareLimiter)).Post("/{id}/shares", s.shareFolderHandler)
		r.Get("/{id}/shares", s.getFolderSharesHandler)
		r.Put("/{id}/shares/{userId}", s.updateFolderShareRoleHandler)
		r.Delete("/{id}/shares/{userId}", s.removeFolderShareHandler)
	})
}

func (s *Server) registerTagsRoutes(r chi.Router) {
	r.Route("/tags", func(r chi.Router) {
		r.Get("/", s.getAllTags)
		r.Delete("/{tagId}", s.deleteTag)
	})
}

func (s *Server) registerTemplatesRoutes(r chi.Router) {
	r.Route("/templates", func(r chi.Router) {
		r.Get("/", s.getAllTemplates)
		r.Get("/{id}", s.getTemplate)
		r.Post("/", s.createTemplate)
		r.Put("/{id}", s.updateTemplate)
		r.Delete("/{id}", s.deleteTemplate)
	})
}

func (s *Server) registerSnippetsRoutes(r chi.Router) {
	r.Route("/snippets", func(r chi.Router) {
		r.Get("/", s.getAllSnippets)
		r.Get("/{id}", s.getSnippet)
		r.Post("/", s.createSnippet)
		r.Put("/{id}", s.updateSnippet)
		r.Delete("/{id}", s.deleteSnippet)
	})
}

// registerNotesStreamingRoutes registers SSE endpoints that must NOT have
// a timeout middleware (they are long-lived server-sent event streams).
func (s *Server) registerNotesStreamingRoutes(r chi.Router) {
	r.With(rateLimitMiddleware(s.llmLimiter)).Get("/notes/{id}/summarize/stream", s.summarizeNoteStream)
}
