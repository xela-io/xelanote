package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/service"
	"github.com/xela-io/xelanote/internal/websocket"
)

// --- List handlers ---

func (s *Server) listShoppingLists(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	lists, err := s.shoppingService.ListShoppingLists(userID)
	if err != nil {
		s.handleShoppingError(w, "failed to list shopping lists", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"lists": ensureSlice(lists),
	})
}

func (s *Server) createShoppingList(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req struct {
		Name  string  `json:"name"`
		Color *string `json:"color,omitempty"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	list, err := s.shoppingService.CreateShoppingList(userID, req.Name, req.Color)
	if err != nil {
		s.handleShoppingError(w, "failed to create shopping list", err)
		return
	}

	respondJSON(w, http.StatusCreated, list)
}

func (s *Server) getShoppingList(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	detail, err := s.shoppingService.GetShoppingList(userID, listID)
	if err != nil {
		s.handleShoppingError(w, "failed to get shopping list", err)
		return
	}

	respondJSON(w, http.StatusOK, detail)
}

func (s *Server) updateShoppingList(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	var req struct {
		Name            *string `json:"name,omitempty"`
		Color           *string `json:"color,omitempty"`
		ExpectedVersion int     `json:"expected_version"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ExpectedVersion == 0 {
		respondError(w, http.StatusBadRequest, "expected_version is required")
		return
	}

	list, err := s.shoppingService.UpdateShoppingList(userID, listID, req.Name, req.Color, req.ExpectedVersion)
	if err != nil {
		s.handleShoppingError(w, "failed to update shopping list", err)
		return
	}

	s.broadcastToShoppingList(listID, "shopping.list.updated", map[string]interface{}{
		"list_id": listID,
		"list":    list,
	})

	respondJSON(w, http.StatusOK, list)
}

func (s *Server) deleteShoppingList(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	if err := s.shoppingService.DeleteShoppingList(userID, listID); err != nil {
		s.handleShoppingError(w, "failed to delete shopping list", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) archiveShoppingList(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	var req struct {
		ExpectedVersion int `json:"expected_version"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.shoppingService.ArchiveShoppingList(userID, listID, req.ExpectedVersion); err != nil {
		s.handleShoppingError(w, "failed to archive shopping list", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Item handlers ---

func (s *Server) addShoppingItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	var req db.ShoppingItem
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	item, err := s.shoppingService.AddShoppingItem(userID, listID, &req)
	if err != nil {
		s.handleShoppingError(w, "failed to add shopping item", err)
		return
	}

	s.broadcastToShoppingList(listID, "shopping.item.added", map[string]interface{}{
		"list_id": listID,
		"item":    item,
	})

	respondJSON(w, http.StatusCreated, item)
}

func (s *Server) addShoppingItems(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	var req struct {
		Items []db.ShoppingItem `json:"items"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Items) == 0 {
		respondError(w, http.StatusBadRequest, "items array is required")
		return
	}

	items, err := s.shoppingService.AddShoppingItems(userID, listID, req.Items)
	if err != nil {
		s.handleShoppingError(w, "failed to add shopping items", err)
		return
	}

	for _, item := range items {
		s.broadcastToShoppingList(listID, "shopping.item.added", map[string]interface{}{
			"list_id": listID,
			"item":    item,
		})
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"items": ensureSlice(items),
	})
}

func (s *Server) updateShoppingItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	itemID, ok := parseIntParam(w, r, "itemId", "invalid item id")
	if !ok {
		return
	}

	var req struct {
		Name            *string  `json:"name,omitempty"`
		Quantity        *float64 `json:"quantity,omitempty"`
		Unit            *string  `json:"unit,omitempty"`
		Category        *string  `json:"category,omitempty"`
		CategoryOrder   *int     `json:"category_order,omitempty"`
		ExpectedVersion int      `json:"expected_version"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ExpectedVersion == 0 {
		respondError(w, http.StatusBadRequest, "expected_version is required")
		return
	}

	item, err := s.shoppingService.UpdateShoppingItem(userID, listID, itemID, req.Name, req.Quantity, req.Unit, req.Category, req.CategoryOrder, req.ExpectedVersion)
	if err != nil {
		s.handleShoppingError(w, "failed to update shopping item", err)
		return
	}

	s.broadcastToShoppingList(listID, "shopping.item.updated", map[string]interface{}{
		"list_id": listID,
		"item":    item,
	})

	respondJSON(w, http.StatusOK, item)
}

func (s *Server) deleteShoppingItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	itemID, ok := parseIntParam(w, r, "itemId", "invalid item id")
	if !ok {
		return
	}

	if err := s.shoppingService.DeleteShoppingItem(userID, listID, itemID); err != nil {
		s.handleShoppingError(w, "failed to delete shopping item", err)
		return
	}

	s.broadcastToShoppingList(listID, "shopping.item.removed", map[string]interface{}{
		"list_id": listID,
		"item_id": itemID,
	})

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) setShoppingItemChecked(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	itemID, ok := parseIntParam(w, r, "itemId", "invalid item id")
	if !ok {
		return
	}

	var req struct {
		IsChecked bool `json:"is_checked"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := s.shoppingService.SetItemChecked(userID, listID, itemID, req.IsChecked)
	if err != nil {
		s.handleShoppingError(w, "failed to check shopping item", err)
		return
	}

	s.broadcastToShoppingList(listID, "shopping.item.checked", map[string]interface{}{
		"list_id":    listID,
		"item_id":    itemID,
		"is_checked": item.IsChecked,
		"checked_at": item.CheckedAt,
	})

	respondJSON(w, http.StatusOK, item)
}

func (s *Server) setShoppingItemsChecked(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	var req struct {
		ItemIDs   []int `json:"item_ids"`
		IsChecked bool  `json:"is_checked"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.ItemIDs) == 0 {
		respondError(w, http.StatusBadRequest, "item_ids is required")
		return
	}

	if err := s.shoppingService.SetItemsChecked(userID, listID, req.ItemIDs, req.IsChecked); err != nil {
		s.handleShoppingError(w, "failed to batch check items", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) clearCheckedItems(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	count, err := s.shoppingService.ClearCheckedItems(userID, listID)
	if err != nil {
		s.handleShoppingError(w, "failed to clear checked items", err)
		return
	}

	s.broadcastToShoppingList(listID, "shopping.items.cleared", map[string]interface{}{
		"list_id":       listID,
		"cleared_count": count,
	})

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"cleared_count": count,
	})
}

func (s *Server) reorderShoppingItems(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	var req struct {
		ItemIDs []int `json:"item_ids"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.shoppingService.ReorderShoppingItems(userID, listID, req.ItemIDs); err != nil {
		s.handleShoppingError(w, "failed to reorder items", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Favorites handlers ---

func (s *Server) getShoppingFavorites(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	favorites, err := s.shoppingService.GetShoppingFavorites(userID)
	if err != nil {
		s.handleShoppingError(w, "failed to get favorites", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"favorites": ensureSlice(favorites),
	})
}

func (s *Server) addShoppingFavorite(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req struct {
		Name     string   `json:"name"`
		Quantity *float64 `json:"default_quantity,omitempty"`
		Unit     *string  `json:"default_unit,omitempty"`
		Category *string  `json:"category,omitempty"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	fav, err := s.shoppingService.AddShoppingFavorite(userID, req.Name, req.Quantity, req.Unit, req.Category)
	if err != nil {
		s.handleShoppingError(w, "failed to add favorite", err)
		return
	}

	respondJSON(w, http.StatusCreated, fav)
}

func (s *Server) removeShoppingFavorite(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	favID, ok := parseIntParam(w, r, "id", "invalid favorite id")
	if !ok {
		return
	}

	if err := s.shoppingService.RemoveShoppingFavorite(userID, favID); err != nil {
		s.handleShoppingError(w, "failed to remove favorite", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- AI Sort handler ---

func (s *Server) sortShoppingList(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	if err := s.shoppingService.SortByCategory(r.Context(), userID, listID); err != nil {
		s.handleShoppingError(w, "failed to sort shopping list", err)
		return
	}

	s.broadcastToShoppingList(listID, "shopping.items.sorted", map[string]interface{}{
		"list_id": listID,
	})

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Recipe import handler ---

func (s *Server) importRecipeToShoppingList(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	var req struct {
		RecipeNoteID string `json:"recipe_note_id"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.RecipeNoteID == "" {
		respondError(w, http.StatusBadRequest, "recipe_note_id is required")
		return
	}

	items, err := s.shoppingService.ImportFromRecipe(userID, listID, req.RecipeNoteID)
	if err != nil {
		s.handleShoppingError(w, "failed to import recipe", err)
		return
	}

	for _, item := range items {
		s.broadcastToShoppingList(listID, "shopping.item.added", map[string]interface{}{
			"list_id": listID,
			"item":    item,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": ensureSlice(items),
	})
}

// --- Sharing handlers ---

func (s *Server) shareShoppingList(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	var req struct {
		UserID int    `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == 0 {
		respondError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	share, err := s.shoppingService.ShareShoppingList(userID, listID, req.UserID, req.Role)
	if err != nil {
		s.handleShoppingError(w, "failed to share shopping list", err)
		return
	}

	respondJSON(w, http.StatusCreated, share)
}

func (s *Server) getShoppingListShares(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	shares, err := s.shoppingService.GetShoppingListShares(userID, listID)
	if err != nil {
		s.handleShoppingError(w, "failed to get shares", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"shares": ensureSlice(shares),
	})
}

func (s *Server) updateShoppingListShareRole(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	targetUserID, ok := parseIntParam(w, r, "userId", "invalid user id")
	if !ok {
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.shoppingService.UpdateShoppingListShareRole(userID, listID, targetUserID, req.Role); err != nil {
		s.handleShoppingError(w, "failed to update share role", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) removeShoppingListShare(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	listID, ok := parseIntParam(w, r, "id", "invalid list id")
	if !ok {
		return
	}

	targetUserID, ok := parseIntParam(w, r, "userId", "invalid user id")
	if !ok {
		return
	}

	if err := s.shoppingService.RemoveShoppingListShare(userID, listID, targetUserID); err != nil {
		s.handleShoppingError(w, "failed to remove share", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Helper functions ---

// handleShoppingError maps service errors to HTTP responses.
func (s *Server) handleShoppingError(w http.ResponseWriter, msg string, err error) {
	switch {
	case errors.Is(err, service.ErrShoppingFeatureNotEnabled):
		respondError(w, http.StatusForbidden, "shopping feature not enabled")
	case errors.Is(err, service.ErrNoListAccess):
		respondError(w, http.StatusForbidden, "no access to this shopping list")
	case errors.Is(err, service.ErrNotListOwner):
		respondError(w, http.StatusForbidden, "only the list owner can perform this action")
	case errors.Is(err, db.ErrNotFound):
		respondError(w, http.StatusNotFound, "not found")
	case errors.Is(err, db.ErrDuplicate):
		respondError(w, http.StatusConflict, "duplicate entry")
	case errors.Is(err, db.ErrVersionMismatch):
		respondError(w, http.StatusConflict, "item was modified by another user, please refresh")
	case errors.Is(err, service.ErrCannotShareWithSelf):
		respondError(w, http.StatusBadRequest, "cannot share a list with yourself")
	case errors.Is(err, service.ErrListAlreadyShared):
		respondError(w, http.StatusConflict, "list is already shared with this user")
	case errors.Is(err, service.ErrShoppingRecipeNotFound):
		respondError(w, http.StatusNotFound, "recipe not found")
	case errors.Is(err, service.ErrShoppingRecipeNoAccess):
		respondError(w, http.StatusForbidden, "no access to this recipe")
	case errors.Is(err, service.ErrRecipeEncrypted):
		respondError(w, http.StatusBadRequest, "cannot import from encrypted recipe")
	case errors.Is(err, service.ErrInvalidInput):
		respondError(w, http.StatusBadRequest, "invalid input")
	default:
		s.respondInternalErr(w, msg, err)
	}
}

// broadcastToShoppingList sends a WebSocket message to all users with access to a list.
func (s *Server) broadcastToShoppingList(listID int, eventType string, payload interface{}) {
	if s.wsManager == nil {
		return
	}

	userIDs, err := s.shoppingService.GetShoppingListUserIDs(listID)
	if err != nil {
		s.logger().Error("failed to get shopping list user IDs for broadcast", "error", err, "list_id", listID)
		return
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		s.logger().Error("failed to marshal shopping broadcast payload", "error", err)
		return
	}

	msg := websocket.Message{
		Type:    eventType,
		Payload: payloadBytes,
	}

	for _, uid := range userIDs {
		s.wsManager.BroadcastToUser(uid, msg)
	}
}
