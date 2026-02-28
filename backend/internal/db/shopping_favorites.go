package db

import (
	"fmt"
	"strings"
	"time"
)

// GetShoppingFavorites returns all favorites for a user, ordered by usage.
func (db *DB) GetShoppingFavorites(userID int) ([]ShoppingFavorite, error) {
	rows, err := db.Query(`
		SELECT id, user_id, name, default_quantity, default_unit, category,
		       usage_count, created_at
		FROM shopping_favorites
		WHERE user_id = ?
		ORDER BY usage_count DESC, name ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shopping favorites: %w", err)
	}
	defer rows.Close()

	var favorites []ShoppingFavorite
	for rows.Next() {
		var f ShoppingFavorite
		if err := rows.Scan(
			&f.ID, &f.UserID, &f.Name, &f.DefaultQuantity, &f.DefaultUnit,
			&f.Category, &f.UsageCount, &f.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan shopping favorite: %w", err)
		}
		favorites = append(favorites, f)
	}
	return favorites, rows.Err()
}

// AddShoppingFavorite adds a new favorite item template.
func (db *DB) AddShoppingFavorite(userID int, name string, quantity *float64, unit *string, category *string) (*ShoppingFavorite, error) {
	now := time.Now().Format(time.RFC3339)

	result, err := db.Exec(`
		INSERT INTO shopping_favorites (user_id, name, default_quantity, default_unit, category, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, name, quantity, unit, category, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("failed to add shopping favorite: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &ShoppingFavorite{
		ID:              int(id),
		UserID:          userID,
		Name:            name,
		DefaultQuantity: quantity,
		DefaultUnit:     unit,
		Category:        category,
		CreatedAt:       now,
	}, nil
}

// RemoveShoppingFavorite deletes a favorite by ID (only if owned by user).
func (db *DB) RemoveShoppingFavorite(userID, favoriteID int) error {
	result, err := db.Exec(`
		DELETE FROM shopping_favorites WHERE id = ? AND user_id = ?
	`, favoriteID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove shopping favorite: %w", err)
	}
	return ensureRowsAffected(result)
}

// IncrementFavoriteUsage increments usage_count for a favorite matched by name.
func (db *DB) IncrementFavoriteUsage(userID int, name string) error {
	_, err := db.Exec(`
		UPDATE shopping_favorites
		SET usage_count = usage_count + 1
		WHERE user_id = ? AND name = ?
	`, userID, name)
	if err != nil {
		return fmt.Errorf("failed to increment favorite usage: %w", err)
	}
	return nil
}
