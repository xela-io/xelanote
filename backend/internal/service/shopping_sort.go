package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/llm"
)

// SortByCategory uses an LLM to categorize and sort items by German supermarket layout.
func (s *ShoppingService) SortByCategory(ctx context.Context, userID, listID int) error {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return err
	}

	if _, err := s.requireAccess(listID, userID, "editor"); err != nil {
		return err
	}

	if s.router == nil {
		return fmt.Errorf("AI sorting not available: no LLM provider configured")
	}

	// Get unchecked items
	items, err := s.db.GetShoppingItems(listID)
	if err != nil {
		return err
	}

	var unchecked []db.ShoppingItem
	for _, item := range items {
		if !item.IsChecked {
			unchecked = append(unchecked, item)
		}
	}

	if len(unchecked) == 0 {
		return nil
	}

	// Build item names for the prompt
	itemNames := make([]string, len(unchecked))
	for i, item := range unchecked {
		name := item.Name
		if item.Quantity != nil {
			if item.Unit != nil {
				name = fmt.Sprintf("%.4g%s %s", *item.Quantity, *item.Unit, item.Name)
			} else {
				name = fmt.Sprintf("%.4gx %s", *item.Quantity, item.Name)
			}
		}
		itemNames[i] = name
	}

	// Get LLM provider
	provider, err := s.router.GetAnyProvider(ctx, userID)
	if err != nil {
		return fmt.Errorf("no AI provider available: %w", err)
	}

	// Build and send prompt
	prompt := llm.BuildShoppingListSortPrompt(itemNames)
	response, err := provider.Generate(ctx, prompt, 2048)
	if err != nil {
		return fmt.Errorf("AI sorting failed: %w", err)
	}

	// Parse response
	var sortResult []struct {
		Name     string `json:"name"`
		Category string `json:"category"`
		Order    int    `json:"order"`
	}
	if err := json.Unmarshal([]byte(response), &sortResult); err != nil {
		s.logger.Error("failed to parse AI sort response", "error", err, "response", response)
		return fmt.Errorf("failed to parse AI response")
	}

	// Build category order map
	categoryOrder := map[string]int{
		"Obst & Gemüse":       1,
		"Brot & Backwaren":    2,
		"Molkereiprodukte":    3,
		"Käse":                4,
		"Fleisch & Wurst":     5,
		"Fisch":               6,
		"Tiefkühl":            7,
		"Konserven & Gläser":  8,
		"Nudeln, Reis & Co.":  9,
		"Gewürze & Soßen":     10,
		"Getränke":            11,
		"Süßwaren & Snacks":   12,
		"Haushalt & Drogerie": 13,
		"Sonstiges":           14,
	}

	// Match AI results to items and build updates
	nameToItem := make(map[string]db.ShoppingItem)
	for _, item := range unchecked {
		nameToItem[item.Name] = item
	}

	var updates []db.ShoppingItemCategoryUpdate
	for _, sr := range sortResult {
		// Find the original item by name
		item, found := nameToItem[sr.Name]
		if !found {
			// Try partial match
			for _, candidate := range unchecked {
				if candidate.Name == sr.Name {
					item = candidate
					found = true
					break
				}
			}
		}
		if !found {
			continue
		}

		catOrder, ok := categoryOrder[sr.Category]
		if !ok {
			catOrder = 99
		}

		updates = append(updates, db.ShoppingItemCategoryUpdate{
			ItemID:        item.ID,
			Category:      sr.Category,
			CategoryOrder: catOrder,
			DisplayOrder:  sr.Order,
		})
	}

	if len(updates) == 0 {
		return nil
	}

	return s.db.UpdateShoppingItemCategories(listID, updates)
}
