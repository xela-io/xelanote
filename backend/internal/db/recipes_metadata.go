package db

import (
	"database/sql"
	"fmt"
	"time"
)

// GetRecipeMetadata retrieves metadata for a recipe note.
func (db *DB) GetRecipeMetadata(noteID string) (*RecipeMetadata, error) {
	var m RecipeMetadata
	var prepTime, cookTime sql.NullInt64
	var sourceURL, difficulty sql.NullString

	err := db.QueryRow(`
		SELECT note_id, user_id, servings, prep_time_minutes, cook_time_minutes,
		       source_url, difficulty, created_at, updated_at
		FROM recipe_metadata WHERE note_id = ?
	`, noteID).Scan(
		&m.NoteID, &m.UserID, &m.Servings,
		&prepTime, &cookTime, &sourceURL, &difficulty,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe metadata: %w", err)
	}

	if prepTime.Valid {
		v := int(prepTime.Int64)
		m.PrepTimeMinutes = &v
	}
	if cookTime.Valid {
		v := int(cookTime.Int64)
		m.CookTimeMinutes = &v
	}
	if sourceURL.Valid {
		m.SourceURL = &sourceURL.String
	}
	if difficulty.Valid {
		m.Difficulty = &difficulty.String
	}

	return &m, nil
}

// SetRecipeMetadata creates or updates recipe metadata with optimistic locking.
// If expectedUpdatedAt is empty, it performs an upsert (INSERT or UPDATE without lock check).
// If expectedUpdatedAt is set, it performs an UPDATE with optimistic lock check.
func (db *DB) SetRecipeMetadata(noteID string, ownerUserID int, m *RecipeMetadata, expectedUpdatedAt string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	if expectedUpdatedAt != "" {
		// Update with optimistic lock
		result, err := db.Exec(`
			UPDATE recipe_metadata
			SET servings = ?, prep_time_minutes = ?, cook_time_minutes = ?,
			    source_url = ?, difficulty = ?, updated_at = ?
			WHERE note_id = ? AND updated_at = ?
		`, m.Servings, m.PrepTimeMinutes, m.CookTimeMinutes,
			m.SourceURL, m.Difficulty, now, noteID, expectedUpdatedAt)
		if err != nil {
			return fmt.Errorf("update recipe metadata: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("check rows affected: %w", err)
		}
		if rows == 0 {
			// Check if metadata exists at all
			var exists bool
			db.QueryRow("SELECT 1 FROM recipe_metadata WHERE note_id = ?", noteID).Scan(&exists)
			if !exists {
				return fmt.Errorf("recipe metadata not found")
			}
			return ErrVersionMismatch
		}
		return nil
	}

	// Upsert: try UPDATE first, then INSERT
	result, err := db.Exec(`
		UPDATE recipe_metadata
		SET servings = ?, prep_time_minutes = ?, cook_time_minutes = ?,
		    source_url = ?, difficulty = ?, updated_at = ?
		WHERE note_id = ?
	`, m.Servings, m.PrepTimeMinutes, m.CookTimeMinutes,
		m.SourceURL, m.Difficulty, now, noteID)
	if err != nil {
		return fmt.Errorf("update recipe metadata: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		// INSERT (decrypt fallback — metadata was deleted during encryption)
		_, err = db.Exec(`
			INSERT INTO recipe_metadata (note_id, user_id, servings, prep_time_minutes,
			    cook_time_minutes, source_url, difficulty, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, noteID, ownerUserID, m.Servings, m.PrepTimeMinutes, m.CookTimeMinutes,
			m.SourceURL, m.Difficulty, now)
		if err != nil {
			return fmt.Errorf("insert recipe metadata: %w", err)
		}
	}
	return nil
}
