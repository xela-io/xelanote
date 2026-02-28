package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// CreateShoppingList creates a new shopping list for a user.
func (db *DB) CreateShoppingList(userID int, name string, color *string) (*ShoppingList, error) {
	now := time.Now().Format(time.RFC3339)

	// Get next display_order
	var maxOrder int
	err := db.QueryRow(`
		SELECT COALESCE(MAX(display_order), -1) FROM shopping_lists
		WHERE user_id = ? AND is_archived = 0
	`, userID).Scan(&maxOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to get max display order: %w", err)
	}

	result, err := db.Exec(`
		INSERT INTO shopping_lists (user_id, name, color, display_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, name, color, maxOrder+1, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("failed to create shopping list: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &ShoppingList{
		ID:           int(id),
		UserID:       userID,
		Name:         name,
		Color:        color,
		DisplayOrder: maxOrder + 1,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// GetShoppingList retrieves a single shopping list by ID.
func (db *DB) GetShoppingList(listID int) (*ShoppingList, error) {
	var list ShoppingList
	var isArchived int
	err := db.QueryRow(`
		SELECT id, user_id, name, color, is_archived, display_order, version,
		       created_at, updated_at
		FROM shopping_lists WHERE id = ?
	`, listID).Scan(
		&list.ID, &list.UserID, &list.Name, &list.Color,
		&isArchived, &list.DisplayOrder, &list.Version,
		&list.CreatedAt, &list.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get shopping list: %w", err)
	}
	list.IsArchived = isArchived == 1
	return &list, nil
}

// GetShoppingListDetail retrieves a list with all its items.
func (db *DB) GetShoppingListDetail(listID, userID int) (*ShoppingListDetail, error) {
	list, err := db.GetShoppingList(listID)
	if err != nil {
		return nil, err
	}

	items, err := db.GetShoppingItems(listID)
	if err != nil {
		return nil, err
	}

	role, err := db.GetShoppingListAccess(listID, userID)
	if err != nil {
		return nil, err
	}

	shares, err := db.GetShoppingListShares(listID)
	if err != nil {
		return nil, err
	}

	if items == nil {
		items = []ShoppingItem{}
	}
	if shares == nil {
		shares = []ShoppingListShare{}
	}

	detail := &ShoppingListDetail{
		ShoppingList: *list,
		Items:        items,
		ItemCount:    len(items),
		SharedWith:   shares,
		Role:         role,
	}
	return detail, nil
}

// ListShoppingLists returns all lists accessible to a user (owned + shared).
func (db *DB) ListShoppingLists(userID int) ([]ShoppingListSummary, error) {
	rows, err := db.Query(`
		SELECT sl.id, sl.user_id, sl.name, sl.color, sl.is_archived,
		       sl.display_order, sl.version, sl.created_at, sl.updated_at,
		       COALESCE(ic.total, 0) AS item_count,
		       COALESCE(ic.checked, 0) AS checked_count,
		       'owner' AS role,
		       '' AS shared_by
		FROM shopping_lists sl
		LEFT JOIN (
			SELECT list_id, COUNT(*) AS total, SUM(is_checked) AS checked
			FROM shopping_items GROUP BY list_id
		) ic ON ic.list_id = sl.id
		WHERE sl.user_id = ? AND sl.is_archived = 0

		UNION ALL

		SELECT sl.id, sl.user_id, sl.name, sl.color, sl.is_archived,
		       sl.display_order, sl.version, sl.created_at, sl.updated_at,
		       COALESCE(ic.total, 0) AS item_count,
		       COALESCE(ic.checked, 0) AS checked_count,
		       sls.role,
		       COALESCE(u.username, '') AS shared_by
		FROM shopping_list_shares sls
		JOIN shopping_lists sl ON sl.id = sls.list_id
		LEFT JOIN users u ON u.id = sl.user_id
		LEFT JOIN (
			SELECT list_id, COUNT(*) AS total, SUM(is_checked) AS checked
			FROM shopping_items GROUP BY list_id
		) ic ON ic.list_id = sl.id
		WHERE sls.shared_with_user_id = ? AND sl.is_archived = 0

		ORDER BY display_order ASC
	`, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list shopping lists: %w", err)
	}
	defer rows.Close()

	var lists []ShoppingListSummary
	for rows.Next() {
		var s ShoppingListSummary
		var isArchived int
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.Name, &s.Color, &isArchived,
			&s.DisplayOrder, &s.Version, &s.CreatedAt, &s.UpdatedAt,
			&s.ItemCount, &s.CheckedCount, &s.Role, &s.SharedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan shopping list: %w", err)
		}
		s.IsArchived = isArchived == 1
		lists = append(lists, s)
	}
	return lists, rows.Err()
}

// UpdateShoppingList updates a list's name and/or color with optimistic locking.
func (db *DB) UpdateShoppingList(listID int, name *string, color *string, expectedVersion int) (*ShoppingList, error) {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE shopping_lists
		SET name = COALESCE(?, name),
		    color = COALESCE(?, color),
		    version = version + 1,
		    updated_at = ?
		WHERE id = ? AND version = ?
	`, name, color, now, listID, expectedVersion)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("failed to update shopping list: %w", err)
	}

	rows, err := rowsAffectedCount(result, "")
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrVersionMismatch
	}

	return db.GetShoppingList(listID)
}

// ArchiveShoppingList sets the is_archived flag (with version bump).
func (db *DB) ArchiveShoppingList(listID int, expectedVersion int) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE shopping_lists
		SET is_archived = 1, version = version + 1, updated_at = ?
		WHERE id = ? AND version = ?
	`, now, listID, expectedVersion)
	if err != nil {
		return fmt.Errorf("failed to archive shopping list: %w", err)
	}

	rows, err := rowsAffectedCount(result, "")
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrVersionMismatch
	}
	return nil
}

// DeleteShoppingList permanently deletes a shopping list and its items (CASCADE).
func (db *DB) DeleteShoppingList(listID int) error {
	result, err := db.Exec(`DELETE FROM shopping_lists WHERE id = ?`, listID)
	if err != nil {
		return fmt.Errorf("failed to delete shopping list: %w", err)
	}
	rows, err := rowsAffectedCount(result, "")
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ReorderShoppingLists updates display_order for multiple lists.
func (db *DB) ReorderShoppingLists(userID int, listIDs []int) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for i, id := range listIDs {
		_, err := tx.Exec(`
			UPDATE shopping_lists SET display_order = ?
			WHERE id = ? AND user_id = ?
		`, i, id, userID)
		if err != nil {
			return fmt.Errorf("failed to update display order: %w", err)
		}
	}

	return tx.Commit()
}
