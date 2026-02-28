package api

import "github.com/go-chi/chi/v5"

func (s *Server) registerShoppingRoutes(r chi.Router) {
	r.Route("/shopping", func(r chi.Router) {
		// Lists
		r.Get("/lists", s.listShoppingLists)
		r.Post("/lists", s.createShoppingList)
		r.Get("/lists/{id}", s.getShoppingList)
		r.Put("/lists/{id}", s.updateShoppingList)
		r.Delete("/lists/{id}", s.deleteShoppingList)
		r.Post("/lists/{id}/archive", s.archiveShoppingList)

		// Items
		r.Post("/lists/{id}/items", s.addShoppingItem)
		r.Post("/lists/{id}/items/batch", s.addShoppingItems)
		r.Put("/lists/{id}/items/{itemId}", s.updateShoppingItem)
		r.Delete("/lists/{id}/items/{itemId}", s.deleteShoppingItem)
		r.Put("/lists/{id}/items/{itemId}/checked", s.setShoppingItemChecked)
		r.Put("/lists/{id}/items/checked", s.setShoppingItemsChecked)
		r.Delete("/lists/{id}/items/checked", s.clearCheckedItems)
		r.Put("/lists/{id}/items/reorder", s.reorderShoppingItems)

		// Favorites
		r.Get("/favorites", s.getShoppingFavorites)
		r.Post("/favorites", s.addShoppingFavorite)
		r.Delete("/favorites/{id}", s.removeShoppingFavorite)

		// AI sort (rate-limited)
		r.With(rateLimitMiddleware(s.llmLimiter)).Post("/lists/{id}/sort", s.sortShoppingList)

		// Recipe import
		r.Post("/lists/{id}/import-recipe", s.importRecipeToShoppingList)

		// Sharing (rate-limited for share creation)
		r.With(rateLimitMiddleware(s.shareLimiter)).Post("/lists/{id}/shares", s.shareShoppingList)
		r.Get("/lists/{id}/shares", s.getShoppingListShares)
		r.Put("/lists/{id}/shares/{userId}", s.updateShoppingListShareRole)
		r.Delete("/lists/{id}/shares/{userId}", s.removeShoppingListShare)
	})
}
