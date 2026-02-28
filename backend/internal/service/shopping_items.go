package service

import (
	"github.com/xela-io/xelanote/internal/db"
)

// AddShoppingItem adds a single item to a list.
func (s *ShoppingService) AddShoppingItem(userID, listID int, item *ShoppingItem) (*ShoppingItem, error) {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}

	if _, err := s.requireAccess(listID, userID, "editor"); err != nil {
		return nil, err
	}

	item.ListID = listID
	item.AddedByUserID = &userID

	result, err := s.db.AddShoppingItem(item)
	if err != nil {
		return nil, err
	}

	// Increment favorite usage if the item name matches
	_ = s.db.IncrementFavoriteUsage(userID, result.Name)

	return result, nil
}

// AddShoppingItems adds multiple items in a batch.
func (s *ShoppingService) AddShoppingItems(userID, listID int, items []ShoppingItem) ([]ShoppingItem, error) {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}

	if _, err := s.requireAccess(listID, userID, "editor"); err != nil {
		return nil, err
	}

	for i := range items {
		items[i].ListID = listID
		items[i].AddedByUserID = &userID
	}

	return s.db.AddShoppingItems(items)
}

// UpdateShoppingItem updates an item with optimistic locking.
func (s *ShoppingService) UpdateShoppingItem(userID, listID, itemID int, name *string, quantity *float64, unit *string, category *string, categoryOrder *int, expectedVersion int) (*ShoppingItem, error) {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}

	if _, err := s.requireAccess(listID, userID, "editor"); err != nil {
		return nil, err
	}

	// Verify item belongs to the list
	item, err := s.db.GetShoppingItem(itemID)
	if err != nil {
		return nil, err
	}
	if item.ListID != listID {
		return nil, ErrNoListAccess
	}

	return s.db.UpdateShoppingItem(itemID, name, quantity, unit, category, categoryOrder, expectedVersion)
}

// SetItemChecked explicitly sets the checked state (no toggle).
func (s *ShoppingService) SetItemChecked(userID, listID, itemID int, isChecked bool) (*ShoppingItem, error) {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}

	if _, err := s.requireAccess(listID, userID, "editor"); err != nil {
		return nil, err
	}

	// Verify item belongs to the list
	item, err := s.db.GetShoppingItem(itemID)
	if err != nil {
		return nil, err
	}
	if item.ListID != listID {
		return nil, ErrNoListAccess
	}

	result, err := s.db.CheckShoppingItem(itemID, isChecked)
	if err != nil {
		return nil, err
	}

	// Check-propagation: if checking a parent, check all children
	if isChecked && item.ParentID == nil {
		children, err := s.db.GetShoppingItems(listID)
		if err == nil {
			var childIDs []int
			for _, child := range children {
				if child.ParentID != nil && *child.ParentID == itemID && !child.IsChecked {
					childIDs = append(childIDs, child.ID)
				}
			}
			if len(childIDs) > 0 {
				_ = s.db.CheckShoppingItems(childIDs, true)
			}
		}
	}

	return result, nil
}

// SetItemsChecked sets checked state for multiple items.
func (s *ShoppingService) SetItemsChecked(userID, listID int, itemIDs []int, isChecked bool) error {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return err
	}

	if _, err := s.requireAccess(listID, userID, "editor"); err != nil {
		return err
	}

	return s.db.CheckShoppingItems(itemIDs, isChecked)
}

// DeleteShoppingItem removes an item.
func (s *ShoppingService) DeleteShoppingItem(userID, listID, itemID int) error {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return err
	}

	if _, err := s.requireAccess(listID, userID, "editor"); err != nil {
		return err
	}

	// Verify item belongs to the list
	item, err := s.db.GetShoppingItem(itemID)
	if err != nil {
		return err
	}
	if item.ListID != listID {
		return ErrNoListAccess
	}

	return s.db.DeleteShoppingItem(itemID)
}

// ClearCheckedItems removes all checked items from a list.
func (s *ShoppingService) ClearCheckedItems(userID, listID int) (int, error) {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return 0, err
	}

	if _, err := s.requireAccess(listID, userID, "editor"); err != nil {
		return 0, err
	}

	return s.db.ClearCheckedItems(listID)
}

// ReorderShoppingItems updates the display order for items.
func (s *ShoppingService) ReorderShoppingItems(userID, listID int, itemIDs []int) error {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return err
	}

	if _, err := s.requireAccess(listID, userID, "editor"); err != nil {
		return err
	}

	return s.db.ReorderShoppingItems(listID, itemIDs)
}

// ImportFromRecipe imports ingredients from a recipe into a shopping list.
func (s *ShoppingService) ImportFromRecipe(userID, listID int, recipeNoteID string) ([]ShoppingItem, error) {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}

	if _, err := s.requireAccess(listID, userID, "editor"); err != nil {
		return nil, err
	}

	// Load recipe ingredients
	ingredients, err := s.db.GetRecipeIngredients(recipeNoteID)
	if err != nil {
		return nil, ErrShoppingRecipeNotFound
	}
	if len(ingredients) == 0 {
		return nil, ErrShoppingRecipeNotFound
	}

	// Check recipe access: user must own the recipe note or it must be shared with them
	note, err := s.db.GetNote(userID, recipeNoteID)
	if err != nil {
		return nil, ErrShoppingRecipeNoAccess
	}
	if note.ContentEncrypted {
		return nil, ErrRecipeEncrypted
	}

	// Convert ingredients to shopping items
	items := make([]db.ShoppingItem, 0, len(ingredients))
	for _, ing := range ingredients {
		if ing.Optional {
			continue
		}

		item := db.ShoppingItem{
			ListID:         listID,
			Name:           ing.Name,
			Quantity:       ing.Amount,
			Unit:           ing.Unit,
			Category:       ing.GroupName,
			AddedByUserID:  &userID,
			SourceRecipeID: &recipeNoteID,
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		return []db.ShoppingItem{}, nil
	}

	return s.db.AddShoppingItems(items)
}
