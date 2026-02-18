package api

import "github.com/go-chi/chi/v5"

func (s *Server) registerFeaturesRoutes(r chi.Router) {
	r.Route("/features", func(r chi.Router) {
		r.Get("/", s.listFeatures)
		r.Get("/{feature}", s.getFeature)
		r.Put("/{feature}", s.setFeature)
	})
}

func (s *Server) registerJournalRoutes(r chi.Router) {
	// Journal endpoints (lookup only, creation via POST /notes)
	r.Route("/journal", func(r chi.Router) {
		r.Get("/", s.getJournalLookup)
		r.Get("/calendar", s.getJournalCalendar)
		r.Get("/calendar/year", s.getJournalCalendarYear)
		r.Get("/entries", s.listJournalEntries)
	})
}

func (s *Server) registerRecipeRoutes(r chi.Router) {
	r.Route("/recipes", func(r chi.Router) {
		r.Get("/", s.listRecipes)
		r.Get("/{id}", s.getRecipeDetail)
		r.Put("/{id}/metadata", s.updateRecipeMetadata)
		r.Put("/{id}/ingredients", s.setRecipeIngredients)
		r.Get("/{id}/scaled", s.getScaledIngredients)

		// Recipe images
		r.Post("/{id}/images", s.addRecipeImage)
		r.Put("/{id}/images/order", s.reorderRecipeImages)
		r.Put("/{id}/images/{imageId}", s.updateRecipeImageCaption)
		r.Delete("/{id}/images/{imageId}", s.deleteRecipeImage)

		// Collections (owner-only)
		r.Get("/collections", s.listRecipeCollections)
		r.Post("/collections", s.createRecipeCollection)
		r.Put("/collections/{id}", s.updateRecipeCollection)
		r.Delete("/collections/{id}", s.deleteRecipeCollection)
		r.Post("/collections/{id}/items", s.addRecipeToCollection)
		r.Delete("/collections/{id}/items/{noteId}", s.removeRecipeFromCollection)
		r.Get("/collections/{id}/items", s.listCollectionItems)

		// Collection sharing (owner-only management)
		r.With(rateLimitMiddleware(s.shareLimiter)).Post("/collections/{id}/shares", s.shareCollection)
		r.Get("/collections/{id}/shares", s.getCollectionShares)
		r.Put("/collections/{id}/shares/{userId}", s.updateCollectionShareRole)
		r.Delete("/collections/{id}/shares/{userId}", s.removeCollectionShare)

		// AI-powered recipe suggestions (rate-limited)
		r.Route("/suggestions", func(r chi.Router) {
			r.With(rateLimitMiddleware(s.llmLimiter)).Post("/similar", s.findSimilarRecipes)
			r.With(rateLimitMiddleware(s.llmLimiter)).Post("/by-ingredients", s.suggestByIngredients)
			r.With(rateLimitMiddleware(s.llmLimiter)).Post("/save-generated", s.saveGeneratedRecipe)
			r.With(rateLimitMiddleware(s.llmLimiter)).Post("/extract-ingredients", s.extractIngredientsFromPhoto)
			r.With(rateLimitMiddleware(s.llmLimiter)).Post("/import-from-image", s.importRecipeFromImage)
			r.With(rateLimitMiddleware(s.llmLimiter)).Post("/import-from-url", s.importRecipeFromURL)
		})
	})
}
