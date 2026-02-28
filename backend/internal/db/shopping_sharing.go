package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// GetShoppingListAccess returns the role of a user for a shopping list.
// Returns "owner", "editor", "viewer", or "" (no access).
func (db *DB) GetShoppingListAccess(listID, userID int) (string, error) {
	// Check ownership
	var ownerID int
	err := db.QueryRow(`SELECT user_id FROM shopping_lists WHERE id = ?`, listID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to check list ownership: %w", err)
	}
	if ownerID == userID {
		return "owner", nil
	}

	// Check shares
	var role string
	err = db.QueryRow(`
		SELECT role FROM shopping_list_shares
		WHERE list_id = ? AND shared_with_user_id = ?
	`, listID, userID).Scan(&role)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to check share access: %w", err)
	}
	return role, nil
}

// ShareShoppingList creates a share record for a shopping list.
func (db *DB) ShareShoppingList(listID, ownerUserID, sharedWithUserID int, role string) (*ShoppingListShare, error) {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		INSERT INTO shopping_list_shares (list_id, owner_user_id, shared_with_user_id, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, listID, ownerUserID, sharedWithUserID, role, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("failed to share shopping list: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &ShoppingListShare{
		ID:               int(id),
		ListID:           listID,
		OwnerUserID:      ownerUserID,
		SharedWithUserID: sharedWithUserID,
		Role:             role,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// GetShoppingListShares returns all shares for a list.
func (db *DB) GetShoppingListShares(listID int) ([]ShoppingListShare, error) {
	rows, err := db.Query(`
		SELECT sls.id, sls.list_id, sls.owner_user_id, sls.shared_with_user_id,
		       COALESCE(u.username, '') AS shared_with_name,
		       sls.role, sls.created_at, sls.updated_at
		FROM shopping_list_shares sls
		LEFT JOIN users u ON u.id = sls.shared_with_user_id
		WHERE sls.list_id = ?
		ORDER BY sls.created_at ASC
	`, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shopping list shares: %w", err)
	}
	defer rows.Close()

	var shares []ShoppingListShare
	for rows.Next() {
		var s ShoppingListShare
		if err := rows.Scan(
			&s.ID, &s.ListID, &s.OwnerUserID, &s.SharedWithUserID,
			&s.SharedWithName, &s.Role, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan shopping list share: %w", err)
		}
		shares = append(shares, s)
	}
	return shares, rows.Err()
}

// UpdateShoppingListShareRole updates the role for an existing share.
func (db *DB) UpdateShoppingListShareRole(listID, sharedWithUserID int, role string) error {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE shopping_list_shares
		SET role = ?, updated_at = ?
		WHERE list_id = ? AND shared_with_user_id = ?
	`, role, now, listID, sharedWithUserID)
	if err != nil {
		return fmt.Errorf("failed to update share role: %w", err)
	}
	return ensureRowsAffected(result)
}

// RemoveShoppingListShare removes a share record.
func (db *DB) RemoveShoppingListShare(listID, sharedWithUserID int) error {
	result, err := db.Exec(`
		DELETE FROM shopping_list_shares
		WHERE list_id = ? AND shared_with_user_id = ?
	`, listID, sharedWithUserID)
	if err != nil {
		return fmt.Errorf("failed to remove share: %w", err)
	}
	return ensureRowsAffected(result)
}

// GetShoppingListUserIDs returns all user IDs with access to a list (owner + shared).
func (db *DB) GetShoppingListUserIDs(listID int) ([]int, error) {
	rows, err := db.Query(`
		SELECT user_id FROM shopping_lists WHERE id = ?
		UNION
		SELECT shared_with_user_id FROM shopping_list_shares WHERE list_id = ?
	`, listID, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list user ids: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan user id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
