package db

import (
	"database/sql"
	"fmt"
	"strings"
)

// GetRecipeSummaries returns lightweight summaries of all unencrypted recipes for a user.
// Used for building LLM prompts (similar recipes, ingredient matching).
func (db *DB) GetRecipeSummaries(userID int, snippetLength int) ([]RecipeSummary, error) {
	if snippetLength <= 0 {
		snippetLength = 200
	}

	rows, err := db.Query(`
		SELECT n.id, n.title, SUBSTR(n.content, 1, ?) as snippet,
		       rm.difficulty, rm.servings,
		       COALESCE(
		           (SELECT GROUP_CONCAT(ri.name, ',')
		            FROM recipe_ingredients ri WHERE ri.note_id = n.id),
		           ''
		       ) as ingredient_names
		FROM notes n
		LEFT JOIN recipe_metadata rm ON rm.note_id = n.id
		WHERE n.user_id = ? AND n.note_type = 'recipe'
		  AND n.content_encrypted = 0 AND n.is_deleted = 0
		ORDER BY n.updated_at DESC
	`, snippetLength, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe summaries: %w", err)
	}
	defer rows.Close()

	var summaries []RecipeSummary
	for rows.Next() {
		var s RecipeSummary
		var difficulty sql.NullString
		var ingredientCSV string

		if err := rows.Scan(&s.NoteID, &s.Title, &s.ContentSnippet,
			&difficulty, &s.Servings, &ingredientCSV); err != nil {
			return nil, fmt.Errorf("failed to scan recipe summary: %w", err)
		}

		if difficulty.Valid {
			s.Difficulty = &difficulty.String
		}
		if ingredientCSV != "" {
			s.IngredientNames = strings.Split(ingredientCSV, ",")
		}

		summaries = append(summaries, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recipe summaries: %w", err)
	}
	if summaries == nil {
		summaries = []RecipeSummary{}
	}
	return summaries, nil
}

// GetRecipeSummariesInCollection returns summaries for recipes in a specific collection.
func (db *DB) GetRecipeSummariesInCollection(userID, collectionID int, snippetLength int) ([]RecipeSummary, error) {
	if snippetLength <= 0 {
		snippetLength = 200
	}

	rows, err := db.Query(`
		SELECT n.id, n.title, SUBSTR(n.content, 1, ?) as snippet,
		       rm.difficulty, rm.servings,
		       COALESCE(
		           (SELECT GROUP_CONCAT(ri.name, ',')
		            FROM recipe_ingredients ri WHERE ri.note_id = n.id),
		           ''
		       ) as ingredient_names
		FROM notes n
		JOIN recipe_collection_items ci ON ci.note_id = n.id
		LEFT JOIN recipe_metadata rm ON rm.note_id = n.id
		WHERE ci.collection_id = ? AND n.user_id = ?
		  AND n.note_type = 'recipe'
		  AND n.content_encrypted = 0 AND n.is_deleted = 0
		ORDER BY n.updated_at DESC
	`, snippetLength, collectionID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe summaries in collection: %w", err)
	}
	defer rows.Close()

	var summaries []RecipeSummary
	for rows.Next() {
		var s RecipeSummary
		var difficulty sql.NullString
		var ingredientCSV string

		if err := rows.Scan(&s.NoteID, &s.Title, &s.ContentSnippet,
			&difficulty, &s.Servings, &ingredientCSV); err != nil {
			return nil, fmt.Errorf("failed to scan recipe summary: %w", err)
		}

		if difficulty.Valid {
			s.Difficulty = &difficulty.String
		}
		if ingredientCSV != "" {
			s.IngredientNames = strings.Split(ingredientCSV, ",")
		}

		summaries = append(summaries, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recipe summaries: %w", err)
	}
	if summaries == nil {
		summaries = []RecipeSummary{}
	}
	return summaries, nil
}
