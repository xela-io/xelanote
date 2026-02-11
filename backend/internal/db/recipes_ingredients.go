package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// GetRecipeIngredients retrieves all ingredients for a recipe note.
func (db *DB) GetRecipeIngredients(noteID string) ([]RecipeIngredient, error) {
	rows, err := db.Query(`
		SELECT id, note_id, amount, amount_text, unit, name, group_name,
		       display_order, optional, scalable
		FROM recipe_ingredients
		WHERE note_id = ?
		ORDER BY display_order, id
	`, noteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe ingredients: %w", err)
	}
	defer rows.Close()

	var ingredients []RecipeIngredient
	for rows.Next() {
		var ing RecipeIngredient
		var amount sql.NullFloat64
		var amountText, unit, groupName sql.NullString
		var optional, scalable int

		if err := rows.Scan(
			&ing.ID, &ing.NoteID, &amount, &amountText, &unit, &ing.Name,
			&groupName, &ing.DisplayOrder, &optional, &scalable,
		); err != nil {
			return nil, fmt.Errorf("failed to scan ingredient: %w", err)
		}

		if amount.Valid {
			ing.Amount = &amount.Float64
		}
		if amountText.Valid {
			ing.AmountText = &amountText.String
		}
		if unit.Valid {
			ing.Unit = &unit.String
		}
		if groupName.Valid {
			ing.GroupName = &groupName.String
		}
		ing.Optional = optional == 1
		ing.Scalable = scalable == 1

		ingredients = append(ingredients, ing)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ingredients: %w", err)
	}

	if ingredients == nil {
		ingredients = []RecipeIngredient{}
	}
	return ingredients, nil
}

// SetRecipeIngredients atomically replaces all ingredients for a recipe note.
// Uses optimistic locking via recipe_metadata.updated_at.
func (db *DB) SetRecipeIngredients(noteID string, ownerUserID int,
	ingredients []RecipeIngredient, expectedUpdatedAt string) error {

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Optimistic Lock: check updated_at of recipe_metadata
	var currentUpdatedAt string
	err = tx.QueryRow("SELECT updated_at FROM recipe_metadata WHERE note_id = ?", noteID).Scan(&currentUpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("recipe metadata not found - create metadata first")
	}
	if err != nil {
		return fmt.Errorf("check metadata version: %w", err)
	}
	if currentUpdatedAt != expectedUpdatedAt {
		return ErrVersionMismatch
	}

	// Delete all existing ingredients
	_, err = tx.Exec("DELETE FROM recipe_ingredients WHERE note_id = ? AND user_id = ?",
		noteID, ownerUserID)
	if err != nil {
		return fmt.Errorf("delete ingredients: %w", err)
	}

	// Insert new ingredients with consistent display_order
	for i, ing := range ingredients {
		_, err = tx.Exec(`INSERT INTO recipe_ingredients
			(note_id, user_id, amount, amount_text, unit, name, group_name,
			 display_order, optional, scalable)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			noteID, ownerUserID, ing.Amount, ing.AmountText, ing.Unit,
			ing.Name, ing.GroupName, i, boolToInt(ing.Optional), boolToInt(ing.Scalable))
		if err != nil {
			return fmt.Errorf("insert ingredient %d: %w", i, err)
		}
	}

	// Update metadata timestamp explicitly (no trigger — managed by app layer)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = tx.Exec("UPDATE recipe_metadata SET updated_at = ? WHERE note_id = ?", now, noteID)
	if err != nil {
		return fmt.Errorf("update metadata timestamp: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
