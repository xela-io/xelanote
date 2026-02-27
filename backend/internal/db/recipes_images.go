package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type sqlExecer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func touchRecipeMetadata(execer sqlExecer, noteID string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := execer.Exec("UPDATE recipe_metadata SET updated_at = ? WHERE note_id = ?", now, noteID); err != nil {
		return fmt.Errorf("touch recipe metadata updated_at: %w", err)
	}
	return nil
}

// GetRecipeImages retrieves all images for a recipe note.
func (db *DB) GetRecipeImages(noteID string) ([]RecipeImage, error) {
	rows, err := db.Query(`
		SELECT id, note_id, user_id, image_url, caption, display_order, created_at
		FROM recipe_images
		WHERE note_id = ?
		ORDER BY display_order, id
	`, noteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe images: %w", err)
	}
	defer rows.Close()

	var images []RecipeImage
	for rows.Next() {
		var img RecipeImage
		var caption sql.NullString
		if err := rows.Scan(
			&img.ID, &img.NoteID, &img.UserID, &img.ImageURL,
			&caption, &img.DisplayOrder, &img.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan recipe image: %w", err)
		}
		if caption.Valid {
			img.Caption = &caption.String
		}
		images = append(images, img)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recipe images: %w", err)
	}

	if images == nil {
		images = []RecipeImage{}
	}
	return images, nil
}

// AddRecipeImage adds a new image to a recipe. Sets display_order = MAX+1
// and bumps recipe_metadata.updated_at.
func (db *DB) AddRecipeImage(noteID string, userID int, imageURL string, caption *string) (*RecipeImage, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get next display_order
	var maxOrder sql.NullInt64
	err = tx.QueryRow("SELECT MAX(display_order) FROM recipe_images WHERE note_id = ?", noteID).Scan(&maxOrder)
	if err != nil {
		return nil, fmt.Errorf("get max display_order: %w", err)
	}
	nextOrder := 0
	if maxOrder.Valid {
		nextOrder = int(maxOrder.Int64) + 1
	}

	result, err := tx.Exec(`
		INSERT INTO recipe_images (note_id, user_id, image_url, caption, display_order)
		VALUES (?, ?, ?, ?, ?)
	`, noteID, userID, imageURL, caption, nextOrder)
	if err != nil {
		return nil, fmt.Errorf("insert recipe image: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}
	imageID, err := validateLastInsertID(id, "recipe image id")
	if err != nil {
		return nil, err
	}

	// Bump metadata.updated_at
	if err := touchRecipeMetadata(tx, noteID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	// Read back the inserted row
	var img RecipeImage
	var cap sql.NullString
	err = db.QueryRow(`
		SELECT id, note_id, user_id, image_url, caption, display_order, created_at
		FROM recipe_images WHERE id = ?
	`, imageID).Scan(&img.ID, &img.NoteID, &img.UserID, &img.ImageURL, &cap, &img.DisplayOrder, &img.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("read back recipe image: %w", err)
	}
	if cap.Valid {
		img.Caption = &cap.String
	}
	return &img, nil
}

// DeleteRecipeImage deletes a recipe image by its ID.
func (db *DB) DeleteRecipeImage(imageID int) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get note_id before deleting so we can bump metadata
	var noteID string
	err = tx.QueryRow("SELECT note_id FROM recipe_images WHERE id = ?", imageID).Scan(&noteID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get image note_id: %w", err)
	}

	_, err = tx.Exec("DELETE FROM recipe_images WHERE id = ?", imageID)
	if err != nil {
		return fmt.Errorf("delete recipe image: %w", err)
	}

	// Bump metadata.updated_at
	if err := touchRecipeMetadata(tx, noteID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// UpdateRecipeImageCaption updates the caption of a recipe image.
func (db *DB) UpdateRecipeImageCaption(imageID int, caption *string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get note_id before updating so we can bump metadata
	var noteID string
	err = tx.QueryRow("SELECT note_id FROM recipe_images WHERE id = ?", imageID).Scan(&noteID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get image note_id: %w", err)
	}

	_, err = tx.Exec("UPDATE recipe_images SET caption = ? WHERE id = ?", caption, imageID)
	if err != nil {
		return fmt.Errorf("update recipe image caption: %w", err)
	}

	// Bump metadata.updated_at
	if err := touchRecipeMetadata(tx, noteID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// ReorderRecipeImages sets display_order = 0, 1, 2, ... in the order of imageIDs.
// All imageIDs must belong to the given noteID. No duplicates allowed.
func (db *DB) ReorderRecipeImages(noteID string, imageIDs []int) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Validate all IDs belong to this note
	var count int
	placeholders := make([]string, len(imageIDs))
	args := make([]interface{}, 0, len(imageIDs)+1)
	args = append(args, noteID)
	for i, id := range imageIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf( //nolint:gosec // uses parametrized placeholders, no injection risk
		"SELECT COUNT(*) FROM recipe_images WHERE note_id = ? AND id IN (%s)",
		strings.Join(placeholders, ","),
	)
	err = tx.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return fmt.Errorf("validate image IDs: %w", err)
	}
	if count != len(imageIDs) {
		return fmt.Errorf("some image IDs do not belong to this recipe")
	}

	// Check for duplicates
	seen := make(map[int]bool, len(imageIDs))
	for _, id := range imageIDs {
		if seen[id] {
			return fmt.Errorf("duplicate image ID: %d", id)
		}
		seen[id] = true
	}

	// Update display_order
	for i, id := range imageIDs {
		_, err = tx.Exec("UPDATE recipe_images SET display_order = ? WHERE id = ?", i, id)
		if err != nil {
			return fmt.Errorf("update display_order for image %d: %w", id, err)
		}
	}

	// Bump metadata.updated_at
	if err := touchRecipeMetadata(tx, noteID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// GetRecipeImage retrieves a single recipe image by ID.
func (db *DB) GetRecipeImage(imageID int) (*RecipeImage, error) {
	var img RecipeImage
	var caption sql.NullString
	err := db.QueryRow(`
		SELECT id, note_id, user_id, image_url, caption, display_order, created_at
		FROM recipe_images WHERE id = ?
	`, imageID).Scan(&img.ID, &img.NoteID, &img.UserID, &img.ImageURL, &caption, &img.DisplayOrder, &img.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get recipe image: %w", err)
	}
	if caption.Valid {
		img.Caption = &caption.String
	}
	return &img, nil
}
