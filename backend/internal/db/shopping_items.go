package db

import (
	"database/sql"
	"fmt"
	"time"
)

// AddShoppingItem inserts a single item into a shopping list.
func (db *DB) AddShoppingItem(item *ShoppingItem) (*ShoppingItem, error) {
	now := time.Now().Format(time.RFC3339)

	// Get next display_order
	var maxOrder int
	err := db.QueryRow(`
		SELECT COALESCE(MAX(display_order), -1) FROM shopping_items
		WHERE list_id = ? AND is_checked = 0
	`, item.ListID).Scan(&maxOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to get max display order: %w", err)
	}

	result, err := db.Exec(`
		INSERT INTO shopping_items
			(list_id, name, quantity, unit, category, category_order, parent_id,
			 display_order, added_by_user_id, source_recipe_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ListID, item.Name, item.Quantity, item.Unit, item.Category,
		item.CategoryOrder, item.ParentID, maxOrder+1,
		item.AddedByUserID, item.SourceRecipeID, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to add shopping item: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	item.ID = int(id)
	item.DisplayOrder = maxOrder + 1
	item.Version = 1
	item.CreatedAt = now
	item.UpdatedAt = now
	return item, nil
}

// AddShoppingItems inserts multiple items in a single transaction.
func (db *DB) AddShoppingItems(items []ShoppingItem) ([]ShoppingItem, error) {
	if len(items) == 0 {
		return nil, nil
	}

	tx, err := db.BeginImmediate()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().Format(time.RFC3339)
	listID := items[0].ListID

	var maxOrder int
	err = tx.QueryRow(`
		SELECT COALESCE(MAX(display_order), -1) FROM shopping_items
		WHERE list_id = ? AND is_checked = 0
	`, listID).Scan(&maxOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to get max display order: %w", err)
	}

	result := make([]ShoppingItem, 0, len(items))
	for i, item := range items {
		order := maxOrder + 1 + i
		res, err := tx.Exec(`
			INSERT INTO shopping_items
				(list_id, name, quantity, unit, category, category_order, parent_id,
				 display_order, added_by_user_id, source_recipe_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, listID, item.Name, item.Quantity, item.Unit, item.Category,
			item.CategoryOrder, item.ParentID, order,
			item.AddedByUserID, item.SourceRecipeID, now, now)
		if err != nil {
			return nil, fmt.Errorf("failed to add shopping item %d: %w", i, err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert id: %w", err)
		}

		item.ID = int(id)
		item.ListID = listID
		item.DisplayOrder = order
		item.Version = 1
		item.CreatedAt = now
		item.UpdatedAt = now
		result = append(result, item)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit batch insert: %w", err)
	}
	return result, nil
}

// GetShoppingItems returns all items for a list, ordered by category then display_order.
func (db *DB) GetShoppingItems(listID int) ([]ShoppingItem, error) {
	rows, err := db.Query(`
		SELECT id, list_id, name, quantity, unit, category, category_order,
		       parent_id, is_checked, checked_at, display_order, version,
		       added_by_user_id, source_recipe_id, created_at, updated_at
		FROM shopping_items
		WHERE list_id = ?
		ORDER BY is_checked ASC, category_order ASC, display_order ASC
	`, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shopping items: %w", err)
	}
	defer rows.Close()

	return scanShoppingItems(rows)
}

// GetShoppingItem retrieves a single item by ID.
func (db *DB) GetShoppingItem(itemID int) (*ShoppingItem, error) {
	var item ShoppingItem
	var isChecked int
	err := db.QueryRow(`
		SELECT id, list_id, name, quantity, unit, category, category_order,
		       parent_id, is_checked, checked_at, display_order, version,
		       added_by_user_id, source_recipe_id, created_at, updated_at
		FROM shopping_items WHERE id = ?
	`, itemID).Scan(
		&item.ID, &item.ListID, &item.Name, &item.Quantity, &item.Unit,
		&item.Category, &item.CategoryOrder, &item.ParentID,
		&isChecked, &item.CheckedAt, &item.DisplayOrder, &item.Version,
		&item.AddedByUserID, &item.SourceRecipeID, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get shopping item: %w", err)
	}
	item.IsChecked = isChecked == 1
	return &item, nil
}

// UpdateShoppingItem updates an item with optimistic locking.
func (db *DB) UpdateShoppingItem(itemID int, name *string, quantity *float64, unit *string, category *string, categoryOrder *int, expectedVersion int) (*ShoppingItem, error) {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE shopping_items
		SET name = COALESCE(?, name),
		    quantity = COALESCE(?, quantity),
		    unit = COALESCE(?, unit),
		    category = COALESCE(?, category),
		    category_order = COALESCE(?, category_order),
		    version = version + 1,
		    updated_at = ?
		WHERE id = ? AND version = ?
	`, name, quantity, unit, category, categoryOrder, now, itemID, expectedVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to update shopping item: %w", err)
	}

	rows, err := rowsAffectedCount(result, "")
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrVersionMismatch
	}

	return db.GetShoppingItem(itemID)
}

// CheckShoppingItem sets the checked state of a single item (no version check).
func (db *DB) CheckShoppingItem(itemID int, isChecked bool) (*ShoppingItem, error) {
	now := time.Now().Format(time.RFC3339)

	var checkedAt *string
	if isChecked {
		checkedAt = &now
	}

	result, err := db.Exec(`
		UPDATE shopping_items
		SET is_checked = ?, checked_at = ?, updated_at = ?
		WHERE id = ?
	`, boolToInt(isChecked), checkedAt, now, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to check shopping item: %w", err)
	}

	if err := ensureRowsAffected(result); err != nil {
		return nil, err
	}

	return db.GetShoppingItem(itemID)
}

// CheckShoppingItems sets the checked state for multiple items in a transaction.
func (db *DB) CheckShoppingItems(itemIDs []int, isChecked bool) error {
	if len(itemIDs) == 0 {
		return nil
	}

	tx, err := db.BeginImmediate()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().Format(time.RFC3339)
	var checkedAt *string
	if isChecked {
		checkedAt = &now
	}

	for _, id := range itemIDs {
		_, err := tx.Exec(`
			UPDATE shopping_items
			SET is_checked = ?, checked_at = ?, updated_at = ?
			WHERE id = ?
		`, boolToInt(isChecked), checkedAt, now, id)
		if err != nil {
			return fmt.Errorf("failed to check item %d: %w", id, err)
		}
	}

	return tx.Commit()
}

// DeleteShoppingItem removes a single item.
func (db *DB) DeleteShoppingItem(itemID int) error {
	result, err := db.Exec(`DELETE FROM shopping_items WHERE id = ?`, itemID)
	if err != nil {
		return fmt.Errorf("failed to delete shopping item: %w", err)
	}
	return ensureRowsAffected(result)
}

// ClearCheckedItems removes all checked items from a list. Returns count of removed items.
func (db *DB) ClearCheckedItems(listID int) (int, error) {
	tx, err := db.BeginImmediate()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		DELETE FROM shopping_items
		WHERE list_id = ? AND is_checked = 1
	`, listID)
	if err != nil {
		return 0, fmt.Errorf("failed to clear checked items: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit clear checked: %w", err)
	}
	return int(rows), nil
}

// ReorderShoppingItems updates display_order for items.
func (db *DB) ReorderShoppingItems(listID int, itemIDs []int) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for i, id := range itemIDs {
		_, err := tx.Exec(`
			UPDATE shopping_items SET display_order = ?
			WHERE id = ? AND list_id = ?
		`, i, id, listID)
		if err != nil {
			return fmt.Errorf("failed to update item display order: %w", err)
		}
	}

	return tx.Commit()
}

// UpdateShoppingItemCategories batch-updates categories for items (used by AI sort).
func (db *DB) UpdateShoppingItemCategories(listID int, updates []ShoppingItemCategoryUpdate) error {
	tx, err := db.BeginImmediate()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().Format(time.RFC3339)
	for _, u := range updates {
		_, err := tx.Exec(`
			UPDATE shopping_items
			SET category = ?, category_order = ?, display_order = ?, updated_at = ?
			WHERE id = ? AND list_id = ?
		`, u.Category, u.CategoryOrder, u.DisplayOrder, now, u.ItemID, listID)
		if err != nil {
			return fmt.Errorf("failed to update item category: %w", err)
		}
	}

	return tx.Commit()
}

// ShoppingItemCategoryUpdate holds category update data for a single item.
type ShoppingItemCategoryUpdate struct {
	ItemID        int    `json:"item_id"`
	Category      string `json:"category"`
	CategoryOrder int    `json:"category_order"`
	DisplayOrder  int    `json:"display_order"`
}

// scanShoppingItems scans rows into a slice of ShoppingItem.
func scanShoppingItems(rows *sql.Rows) ([]ShoppingItem, error) {
	var items []ShoppingItem
	for rows.Next() {
		var item ShoppingItem
		var isChecked int
		if err := rows.Scan(
			&item.ID, &item.ListID, &item.Name, &item.Quantity, &item.Unit,
			&item.Category, &item.CategoryOrder, &item.ParentID,
			&isChecked, &item.CheckedAt, &item.DisplayOrder, &item.Version,
			&item.AddedByUserID, &item.SourceRecipeID, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan shopping item: %w", err)
		}
		item.IsChecked = isChecked == 1
		items = append(items, item)
	}
	return items, rows.Err()
}
