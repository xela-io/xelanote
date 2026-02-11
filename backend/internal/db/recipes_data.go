package db

import "fmt"

// DeleteRecipeData removes all recipe metadata, ingredients, and images for a note.
// Used when encrypting a recipe note.
func (db *DB) DeleteRecipeData(noteID string) error {
	if _, err := db.Exec("DELETE FROM recipe_images WHERE note_id = ?", noteID); err != nil {
		return fmt.Errorf("delete recipe images: %w", err)
	}
	if _, err := db.Exec("DELETE FROM recipe_ingredients WHERE note_id = ?", noteID); err != nil {
		return fmt.Errorf("delete recipe ingredients: %w", err)
	}
	if _, err := db.Exec("DELETE FROM recipe_metadata WHERE note_id = ?", noteID); err != nil {
		return fmt.Errorf("delete recipe metadata: %w", err)
	}
	return nil
}
