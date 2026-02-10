package db

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RecipeMetadata holds metadata for a recipe note.
type RecipeMetadata struct {
	NoteID          string  `json:"note_id"`
	UserID          int     `json:"user_id"`
	Servings        int     `json:"servings"`
	PrepTimeMinutes *int    `json:"prep_time_minutes,omitempty"`
	CookTimeMinutes *int    `json:"cook_time_minutes,omitempty"`
	SourceURL       *string `json:"source_url,omitempty"`
	Difficulty      *string `json:"difficulty,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// RecipeIngredient holds a single ingredient for a recipe.
type RecipeIngredient struct {
	ID           int      `json:"id"`
	NoteID       string   `json:"note_id"`
	Amount       *float64 `json:"amount,omitempty"`
	AmountText   *string  `json:"amount_text,omitempty"`
	Unit         *string  `json:"unit,omitempty"`
	Name         string   `json:"name"`
	GroupName    *string  `json:"group_name,omitempty"`
	DisplayOrder int      `json:"display_order"`
	Optional     bool     `json:"optional"`
	Scalable     bool     `json:"scalable"`
}

// ScaledIngredient extends RecipeIngredient with scaled values.
type ScaledIngredient struct {
	RecipeIngredient
	ScaledAmount  *float64 `json:"scaled_amount,omitempty"`
	DisplayAmount string   `json:"display_amount"`
}

// RecipeCollection represents a user-owned cookbook.
type RecipeCollection struct {
	ID           int     `json:"id"`
	UserID       int     `json:"user_id"`
	Name         string  `json:"name"`
	Description  *string `json:"description,omitempty"`
	Color        *string `json:"color,omitempty"`
	DisplayOrder int     `json:"display_order"`
	RecipeCount  int     `json:"recipe_count,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// RecipeImage holds a single image for a recipe.
type RecipeImage struct {
	ID           int     `json:"id"`
	NoteID       string  `json:"note_id"`
	UserID       int     `json:"user_id"`
	ImageURL     string  `json:"image_url"`
	Caption      *string `json:"caption,omitempty"`
	DisplayOrder int     `json:"display_order"`
	CreatedAt    string  `json:"created_at"`
}

// RecipeDetail is the full response for GET /api/recipes/{id}.
type RecipeDetail struct {
	Note        Note               `json:"note"`
	Metadata    *RecipeMetadata    `json:"metadata"`
	Ingredients []RecipeIngredient `json:"ingredients"`
	Images      []RecipeImage      `json:"images"`
	Collections []RecipeCollection `json:"collections"`
	Encrypted   bool               `json:"encrypted"`
}

// CreateRecipeNote creates a new plaintext recipe note with default metadata.
// Automatically creates the /Rezepte folder if it doesn't exist.
func (db *DB) CreateRecipeNote(userID int, title, content, folderPath string) (*Note, error) {
	if folderPath == "/Rezepte" {
		folder, err := db.CreateFolder(userID, "/Rezepte", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create recipe folder: %w", err)
		}
		// Recipes are unencrypted by default
		_ = db.UpdateFolderEncryptionDefault(userID, folder.ID, false)
	}

	id := uuid.New().String()
	titleNorm := NormalizeTitle(title)
	now := time.Now().UTC()

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create the note
	_, err = tx.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id,
		                  note_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'recipe', ?, ?)
	`, id, title, titleNorm, content, folderPath, userID,
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("failed to create recipe note: %w", err)
	}

	// Create default metadata (I6: Metadata-Existenz-Garantie)
	_, err = tx.Exec(`
		INSERT INTO recipe_metadata (note_id, user_id, servings)
		VALUES (?, ?, 4)
	`, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create recipe metadata: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &Note{
		ID:         id,
		Title:      title,
		Content:    content,
		FolderPath: folderPath,
		Version:    1,
		NoteType:   NoteTypeRecipe,
		CreatedAt:  now,
		UpdatedAt:  now,
		UserID:     userID,
	}, nil
}

// CreateEncryptedRecipeNote creates a new encrypted recipe note with default metadata.
// Automatically creates the /Rezepte folder if it doesn't exist.
func (db *DB) CreateEncryptedRecipeNote(
	userID int,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	folderPath string,
) (*Note, error) {
	if folderPath == "/Rezepte" {
		folder, err := db.CreateFolder(userID, "/Rezepte", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create recipe folder: %w", err)
		}
		// Recipes are unencrypted by default
		_ = db.UpdateFolderEncryptionDefault(userID, folder.ID, false)
	}

	id := uuid.New().String()
	titleNorm := NormalizeTitle(title)
	now := time.Now().UTC()

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create the encrypted note with note_type='recipe'
	_, err = tx.Exec(`
		INSERT INTO notes (
			id, title, title_norm, encrypted_title, title_encrypted,
			content, encrypted_content, content_encrypted,
			wrapped_dek, encryption_version, encryption_metadata,
			folder_path, user_id, version, note_type,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, '', ?, 1, ?, 1, ?, ?, ?, 1, 'recipe', ?, ?)
	`, id, title, titleNorm, encryptedTitle, titleEncrypted,
		encryptedContent, wrappedDEK, encryptionMetadata,
		folderPath, userID,
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("failed to create encrypted recipe note: %w", err)
	}

	// Create default metadata (I6: Metadata-Existenz-Garantie)
	// Note: For encrypted recipes, metadata will be deleted when encryption is applied
	// and restored when decrypted. But we create it initially for consistency.
	_, err = tx.Exec(`
		INSERT INTO recipe_metadata (note_id, user_id, servings)
		VALUES (?, ?, 4)
	`, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create recipe metadata: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return db.GetNote(userID, id)
}

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

// --- Recipe Images ---

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

	// Bump metadata.updated_at
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = tx.Exec("UPDATE recipe_metadata SET updated_at = ? WHERE note_id = ?", now, noteID)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	// Read back the inserted row
	var img RecipeImage
	var cap sql.NullString
	err = db.QueryRow(`
		SELECT id, note_id, user_id, image_url, caption, display_order, created_at
		FROM recipe_images WHERE id = ?
	`, id).Scan(&img.ID, &img.NoteID, &img.UserID, &img.ImageURL, &cap, &img.DisplayOrder, &img.CreatedAt)
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
	// Get note_id before deleting so we can bump metadata
	var noteID string
	err := db.QueryRow("SELECT note_id FROM recipe_images WHERE id = ?", imageID).Scan(&noteID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get image note_id: %w", err)
	}

	_, err = db.Exec("DELETE FROM recipe_images WHERE id = ?", imageID)
	if err != nil {
		return fmt.Errorf("delete recipe image: %w", err)
	}

	// Bump metadata.updated_at
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = db.Exec("UPDATE recipe_metadata SET updated_at = ? WHERE note_id = ?", now, noteID)

	return nil
}

// UpdateRecipeImageCaption updates the caption of a recipe image.
func (db *DB) UpdateRecipeImageCaption(imageID int, caption *string) error {
	// Get note_id before updating so we can bump metadata
	var noteID string
	err := db.QueryRow("SELECT note_id FROM recipe_images WHERE id = ?", imageID).Scan(&noteID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get image note_id: %w", err)
	}

	_, err = db.Exec("UPDATE recipe_images SET caption = ? WHERE id = ?", caption, imageID)
	if err != nil {
		return fmt.Errorf("update recipe image caption: %w", err)
	}

	// Bump metadata.updated_at
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = db.Exec("UPDATE recipe_metadata SET updated_at = ? WHERE note_id = ?", now, noteID)

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

	query := fmt.Sprintf(
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
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = tx.Exec("UPDATE recipe_metadata SET updated_at = ? WHERE note_id = ?", now, noteID)

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

// ListRecipes returns all recipe notes for a user (owner-only).
func (db *DB) ListRecipes(userID int) ([]Note, error) {
	rows, err := db.Query(`
		SELECT id, title, content, folder_path, version, display_order, color,
		       created_at, updated_at,
		       encrypted_content, content_encrypted, encrypted_title, title_encrypted,
		       wrapped_dek, encryption_version, encryption_metadata,
		       note_type, journal_date, ai_enabled
		FROM notes
		WHERE user_id = ? AND note_type = 'recipe' AND is_deleted = 0
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list recipes: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		var createdAt, updatedAt string
		var content, encryptedTitle, wrappedDEK, encryptionMetadata sql.NullString
		var encryptedContent []byte
		var noteType, journalDate sql.NullString

		if err := rows.Scan(
			&note.ID, &note.Title, &content, &note.FolderPath, &note.Version,
			&note.DisplayOrder, &note.Color, &createdAt, &updatedAt,
			&encryptedContent, &note.ContentEncrypted, &encryptedTitle, &note.TitleEncrypted,
			&wrappedDEK, &note.EncryptionVersion, &encryptionMetadata,
			&noteType, &journalDate, &note.AIEnabled,
		); err != nil {
			return nil, fmt.Errorf("failed to scan recipe note: %w", err)
		}

		note.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		note.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		note.Content = content.String
		note.EncryptedContent = encryptedContent
		if encryptedTitle.Valid {
			note.EncryptedTitle = &encryptedTitle.String
		}
		note.WrappedDEK = wrappedDEK.String
		note.EncryptionMetadata = encryptionMetadata.String
		if noteType.Valid {
			note.NoteType = noteType.String
		}
		if journalDate.Valid {
			note.JournalDate = &journalDate.String
		}
		note.UserID = userID
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recipes: %w", err)
	}

	if notes == nil {
		notes = []Note{}
	}
	return notes, nil
}

// --- Recipe Collections ---

// CreateRecipeCollection creates a new collection for a user.
func (db *DB) CreateRecipeCollection(userID int, name string, description, color *string) (*RecipeCollection, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := db.Exec(`
		INSERT INTO recipe_collections (user_id, name, description, color, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, name, description, color, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create recipe collection: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get collection id: %w", err)
	}
	return &RecipeCollection{
		ID:          int(id),
		UserID:      userID,
		Name:        name,
		Description: description,
		Color:       color,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// ListRecipeCollections returns all collections for a user with recipe counts.
func (db *DB) ListRecipeCollections(userID int) ([]RecipeCollection, error) {
	rows, err := db.Query(`
		SELECT c.id, c.user_id, c.name, c.description, c.color, c.display_order,
		       c.created_at, c.updated_at,
		       (SELECT COUNT(*) FROM recipe_collection_items ci WHERE ci.collection_id = c.id) as recipe_count
		FROM recipe_collections c
		WHERE c.user_id = ?
		ORDER BY c.display_order, c.name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list recipe collections: %w", err)
	}
	defer rows.Close()

	var collections []RecipeCollection
	for rows.Next() {
		var c RecipeCollection
		var description, color sql.NullString
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.Name, &description, &color,
			&c.DisplayOrder, &c.CreatedAt, &c.UpdatedAt, &c.RecipeCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan collection: %w", err)
		}
		if description.Valid {
			c.Description = &description.String
		}
		if color.Valid {
			c.Color = &color.String
		}
		collections = append(collections, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating collections: %w", err)
	}

	if collections == nil {
		collections = []RecipeCollection{}
	}
	return collections, nil
}

// UpdateRecipeCollection updates a collection's name, description, and color.
func (db *DB) UpdateRecipeCollection(userID, collectionID int, name string, description, color *string) error {
	result, err := db.Exec(`
		UPDATE recipe_collections SET name = ?, description = ?, color = ?
		WHERE id = ? AND user_id = ?
	`, name, description, color, collectionID, userID)
	if err != nil {
		return fmt.Errorf("failed to update recipe collection: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteRecipeCollection deletes a collection owned by the user.
func (db *DB) DeleteRecipeCollection(userID, collectionID int) error {
	result, err := db.Exec(`DELETE FROM recipe_collections WHERE id = ? AND user_id = ?`,
		collectionID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete recipe collection: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// AddRecipeToCollection adds a recipe to a collection.
func (db *DB) AddRecipeToCollection(userID, collectionID int, noteID string) error {
	// Verify collection belongs to user
	var exists bool
	err := db.QueryRow("SELECT 1 FROM recipe_collections WHERE id = ? AND user_id = ?",
		collectionID, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("check collection ownership: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO recipe_collection_items (collection_id, note_id)
		VALUES (?, ?)
		ON CONFLICT(collection_id, note_id) DO NOTHING
	`, collectionID, noteID)
	if err != nil {
		return fmt.Errorf("failed to add recipe to collection: %w", err)
	}
	return nil
}

// RemoveRecipeFromCollection removes a recipe from a collection.
func (db *DB) RemoveRecipeFromCollection(userID, collectionID int, noteID string) error {
	// Verify collection belongs to user
	var exists bool
	err := db.QueryRow("SELECT 1 FROM recipe_collections WHERE id = ? AND user_id = ?",
		collectionID, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("check collection ownership: %w", err)
	}

	result, err := db.Exec(`DELETE FROM recipe_collection_items WHERE collection_id = ? AND note_id = ?`,
		collectionID, noteID)
	if err != nil {
		return fmt.Errorf("failed to remove recipe from collection: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ListRecipesInCollection returns all recipe notes in a collection.
func (db *DB) ListRecipesInCollection(userID, collectionID int) ([]Note, error) {
	// Verify collection belongs to user
	var exists bool
	err := db.QueryRow("SELECT 1 FROM recipe_collections WHERE id = ? AND user_id = ?",
		collectionID, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("check collection ownership: %w", err)
	}

	rows, err := db.Query(`
		SELECT n.id, n.title, n.content, n.folder_path, n.version, n.display_order,
		       n.color, n.created_at, n.updated_at,
		       n.encrypted_content, n.content_encrypted, n.encrypted_title, n.title_encrypted,
		       n.wrapped_dek, n.encryption_version, n.encryption_metadata,
		       n.note_type, n.journal_date, n.ai_enabled
		FROM notes n
		JOIN recipe_collection_items ci ON ci.note_id = n.id
		WHERE ci.collection_id = ? AND n.is_deleted = 0 AND n.note_type = 'recipe'
		ORDER BY ci.display_order, n.title
	`, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list recipes in collection: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		var createdAt, updatedAt string
		var content, encryptedTitle, wrappedDEK, encryptionMetadata sql.NullString
		var encryptedContent []byte
		var noteType, journalDate sql.NullString

		if err := rows.Scan(
			&note.ID, &note.Title, &content, &note.FolderPath, &note.Version,
			&note.DisplayOrder, &note.Color, &createdAt, &updatedAt,
			&encryptedContent, &note.ContentEncrypted, &encryptedTitle, &note.TitleEncrypted,
			&wrappedDEK, &note.EncryptionVersion, &encryptionMetadata,
			&noteType, &journalDate, &note.AIEnabled,
		); err != nil {
			return nil, fmt.Errorf("failed to scan recipe note: %w", err)
		}

		note.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		note.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		note.Content = content.String
		note.EncryptedContent = encryptedContent
		if encryptedTitle.Valid {
			note.EncryptedTitle = &encryptedTitle.String
		}
		note.WrappedDEK = wrappedDEK.String
		note.EncryptionMetadata = encryptionMetadata.String
		if noteType.Valid {
			note.NoteType = noteType.String
		}
		if journalDate.Valid {
			note.JournalDate = &journalDate.String
		}
		note.UserID = userID
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recipes: %w", err)
	}

	if notes == nil {
		notes = []Note{}
	}
	return notes, nil
}

// GetCollectionsForRecipe returns all collections that contain a given recipe.
func (db *DB) GetCollectionsForRecipe(userID int, noteID string) ([]RecipeCollection, error) {
	rows, err := db.Query(`
		SELECT c.id, c.user_id, c.name, c.description, c.color, c.display_order,
		       c.created_at, c.updated_at
		FROM recipe_collections c
		JOIN recipe_collection_items ci ON ci.collection_id = c.id
		WHERE c.user_id = ? AND ci.note_id = ?
		ORDER BY c.display_order, c.name
	`, userID, noteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collections for recipe: %w", err)
	}
	defer rows.Close()

	var collections []RecipeCollection
	for rows.Next() {
		var c RecipeCollection
		var description, color sql.NullString
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.Name, &description, &color,
			&c.DisplayOrder, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan collection: %w", err)
		}
		if description.Valid {
			c.Description = &description.String
		}
		if color.Valid {
			c.Color = &color.String
		}
		collections = append(collections, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating collections: %w", err)
	}

	if collections == nil {
		collections = []RecipeCollection{}
	}
	return collections, nil
}

// --- Recipe Summaries for AI ---

// RecipeSummary is a lightweight representation of a recipe for LLM prompts.
type RecipeSummary struct {
	NoteID          string
	Title           string
	IngredientNames []string
	ContentSnippet  string
	Difficulty      *string
	Servings        int
}

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

// CreateRecipeNoteWithIngredients creates a new recipe note with metadata and ingredients
// in a single atomic transaction. Used for saving AI-generated recipes.
func (db *DB) CreateRecipeNoteWithIngredients(
	userID int, title, content, folderPath string,
	metadata RecipeMetadata, ingredients []RecipeIngredient,
) (*Note, error) {
	if folderPath == "" {
		folderPath = "/Rezepte"
	}
	if folderPath == "/Rezepte" {
		folder, err := db.CreateFolder(userID, "/Rezepte", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create recipe folder: %w", err)
		}
		_ = db.UpdateFolderEncryptionDefault(userID, folder.ID, false)
	}

	id := uuid.New().String()
	titleNorm := NormalizeTitle(title)
	now := time.Now().UTC()

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create the note
	_, err = tx.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id,
		                  note_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'recipe', ?, ?)
	`, id, title, titleNorm, content, folderPath, userID,
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("failed to create recipe note: %w", err)
	}

	// Create metadata
	_, err = tx.Exec(`
		INSERT INTO recipe_metadata (note_id, user_id, servings, prep_time_minutes,
		    cook_time_minutes, difficulty)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, userID, metadata.Servings, metadata.PrepTimeMinutes,
		metadata.CookTimeMinutes, metadata.Difficulty)
	if err != nil {
		return nil, fmt.Errorf("failed to create recipe metadata: %w", err)
	}

	// Insert ingredients
	for i, ing := range ingredients {
		name := strings.TrimSpace(ing.Name)
		if name == "" {
			continue
		}
		if len(name) > 200 {
			name = name[:200]
		}
		unit := ing.Unit
		if unit != nil {
			trimmed := strings.TrimSpace(*unit)
			if len(trimmed) > 50 {
				trimmed = trimmed[:50]
			}
			unit = &trimmed
		}
		_, err = tx.Exec(`INSERT INTO recipe_ingredients
			(note_id, user_id, amount, unit, name, display_order, optional, scalable)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			id, userID, ing.Amount, unit, name, i,
			boolToInt(ing.Optional), boolToInt(ing.Scalable))
		if err != nil {
			return nil, fmt.Errorf("insert ingredient %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &Note{
		ID:         id,
		Title:      title,
		Content:    content,
		FolderPath: folderPath,
		Version:    1,
		NoteType:   NoteTypeRecipe,
		CreatedAt:  now,
		UpdatedAt:  now,
		UserID:     userID,
	}, nil
}

// --- Scaling ---

// ScaleIngredients scales ingredients based on target servings.
// This is the canonical server-side implementation (I4).
func ScaleIngredients(ingredients []RecipeIngredient, baseServings, targetServings int) []ScaledIngredient {
	factor := float64(targetServings) / float64(baseServings)
	result := make([]ScaledIngredient, len(ingredients))

	for i, ing := range ingredients {
		result[i].RecipeIngredient = ing

		if !ing.Scalable || ing.Amount == nil {
			result[i].ScaledAmount = ing.Amount
			result[i].DisplayAmount = FormatAmount(ing.Amount, ing.AmountText)
			continue
		}

		scaled := math.Round(*ing.Amount*factor*100) / 100
		result[i].ScaledAmount = &scaled
		result[i].DisplayAmount = FormatDisplayAmount(scaled)
	}
	return result
}

// FormatDisplayAmount formats a float for display.
func FormatDisplayAmount(v float64) string {
	if v == math.Trunc(v) {
		return strconv.Itoa(int(v))
	}
	if v*10 == math.Trunc(v*10) {
		return fmt.Sprintf("%.1f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

// FormatAmount formats the original amount with optional text hint.
func FormatAmount(amount *float64, amountText *string) string {
	if amount == nil {
		return ""
	}
	if amountText != nil {
		return *amountText
	}
	return FormatDisplayAmount(*amount)
}

// --- Collection Sharing ---

// CollectionShare represents a sharing record for a recipe collection.
type CollectionShare struct {
	ID                 int    `json:"id"`
	CollectionID       int    `json:"collection_id"`
	CollectionName     string `json:"collection_name"`
	OwnerUserID        int    `json:"owner_user_id"`
	OwnerUsername      string `json:"owner_username"`
	SharedWithUserID   int    `json:"shared_with_user_id"`
	SharedWithUsername string `json:"shared_with_username"`
	Role               string `json:"role"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

// SharedCollection represents a collection shared with the current user (recipient view).
type SharedCollection struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
	RecipeCount int     `json:"recipe_count"`
	SharedBy    string  `json:"shared_by"`
	ShareRole   string  `json:"share_role"`
	ShareID     int     `json:"share_id"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// GetCollectionOwnerUserID returns the owner user_id for a collection.
func (db *DB) GetCollectionOwnerUserID(collectionID int) (int, error) {
	var userID int
	err := db.QueryRow(`SELECT user_id FROM recipe_collections WHERE id = ?`, collectionID).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get collection owner: %w", err)
	}
	return userID, nil
}

// CreateCollectionShare creates a new collection sharing record.
func (db *DB) CreateCollectionShare(ownerUserID, collectionID, sharedWithUserID int, role string) (*CollectionShare, error) {
	if role != "viewer" && role != "editor" {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	result, err := db.Exec(`
		INSERT INTO recipe_collection_shares (collection_id, owner_user_id, shared_with_user_id, role)
		VALUES (?, ?, ?, ?)
	`, collectionID, ownerUserID, sharedWithUserID, role)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("failed to create collection share: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get collection share id: %w", err)
	}
	return db.getCollectionShareByID(int(id))
}

// DeleteCollectionShare removes a collection sharing record.
func (db *DB) DeleteCollectionShare(ownerUserID, collectionID, sharedWithUserID int) error {
	result, err := db.Exec(`
		DELETE FROM recipe_collection_shares
		WHERE collection_id = ? AND owner_user_id = ? AND shared_with_user_id = ?
	`, collectionID, ownerUserID, sharedWithUserID)
	if err != nil {
		return fmt.Errorf("failed to delete collection share: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// GetCollectionShares returns all shares for a specific collection (owner view).
func (db *DB) GetCollectionShares(ownerUserID, collectionID int) ([]CollectionShare, error) {
	rows, err := db.Query(`
		SELECT rcs.id, rcs.collection_id, rc.name,
		       rcs.owner_user_id, ou.username,
		       rcs.shared_with_user_id, su.username, rcs.role,
		       rcs.created_at, rcs.updated_at
		FROM recipe_collection_shares rcs
		JOIN recipe_collections rc ON rc.id = rcs.collection_id
		JOIN users ou ON ou.id = rcs.owner_user_id
		JOIN users su ON su.id = rcs.shared_with_user_id
		WHERE rcs.collection_id = ? AND rcs.owner_user_id = ?
		ORDER BY rcs.created_at DESC
	`, collectionID, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection shares: %w", err)
	}
	defer rows.Close()

	var shares []CollectionShare
	for rows.Next() {
		var s CollectionShare
		if err := rows.Scan(
			&s.ID, &s.CollectionID, &s.CollectionName,
			&s.OwnerUserID, &s.OwnerUsername,
			&s.SharedWithUserID, &s.SharedWithUsername, &s.Role,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan collection share: %w", err)
		}
		shares = append(shares, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating collection shares: %w", err)
	}

	return shares, nil
}

// UpdateCollectionShareRole updates the role for a collection share record.
func (db *DB) UpdateCollectionShareRole(ownerUserID, collectionID, sharedWithUserID int, role string) error {
	if role != "viewer" && role != "editor" {
		return fmt.Errorf("invalid role: %s", role)
	}

	result, err := db.Exec(`
		UPDATE recipe_collection_shares
		SET role = ?, updated_at = datetime('now')
		WHERE collection_id = ? AND owner_user_id = ? AND shared_with_user_id = ?
	`, role, collectionID, ownerUserID, sharedWithUserID)
	if err != nil {
		return fmt.Errorf("failed to update collection share role: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// GetSharedCollectionsForUser returns all collections shared with a user.
func (db *DB) GetSharedCollectionsForUser(userID int) ([]SharedCollection, error) {
	rows, err := db.Query(`
		SELECT rc.id, rc.name, rc.description, rc.color,
		       (SELECT COUNT(*) FROM recipe_collection_items ci
		        JOIN notes n ON n.id = ci.note_id
		        WHERE ci.collection_id = rc.id AND n.is_deleted = 0 AND n.note_type = 'recipe') as recipe_count,
		       ou.username, rcs.role, rcs.id,
		       rc.created_at, rc.updated_at
		FROM recipe_collection_shares rcs
		JOIN recipe_collections rc ON rc.id = rcs.collection_id
		JOIN users ou ON ou.id = rcs.owner_user_id
		WHERE rcs.shared_with_user_id = ?
		ORDER BY ou.username, rc.name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared collections: %w", err)
	}
	defer rows.Close()

	var collections []SharedCollection
	for rows.Next() {
		var sc SharedCollection
		var description, color sql.NullString
		if err := rows.Scan(
			&sc.ID, &sc.Name, &description, &color,
			&sc.RecipeCount,
			&sc.SharedBy, &sc.ShareRole, &sc.ShareID,
			&sc.CreatedAt, &sc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan shared collection: %w", err)
		}
		if description.Valid {
			sc.Description = &description.String
		}
		if color.Valid {
			sc.Color = &color.String
		}
		collections = append(collections, sc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shared collections: %w", err)
	}

	return collections, nil
}

// GetCollectionSharePermission returns the role for a user on a collection,
// or empty string if no share exists.
func (db *DB) GetCollectionSharePermission(userID, collectionID int) (string, error) {
	var role string
	err := db.QueryRow(`
		SELECT role FROM recipe_collection_shares
		WHERE shared_with_user_id = ? AND collection_id = ?
	`, userID, collectionID).Scan(&role)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get collection share permission: %w", err)
	}
	return role, nil
}

// ListRecipesInSharedCollection returns all recipe notes in a shared collection.
// SECURITY: No user_id filter — must only be called after permission check in service layer.
func (db *DB) ListRecipesInSharedCollection(collectionID int) ([]Note, error) {
	rows, err := db.Query(`
		SELECT n.id, n.title, n.content, n.folder_path, n.version, n.display_order,
		       n.color, n.created_at, n.updated_at,
		       n.encrypted_content, n.content_encrypted, n.encrypted_title, n.title_encrypted,
		       n.wrapped_dek, n.encryption_version, n.encryption_metadata,
		       n.note_type, n.journal_date, n.ai_enabled, n.user_id
		FROM notes n
		JOIN recipe_collection_items ci ON ci.note_id = n.id
		WHERE ci.collection_id = ? AND n.is_deleted = 0 AND n.note_type = 'recipe'
		ORDER BY ci.display_order, n.title
	`, collectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list recipes in shared collection: %w", err)
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		var createdAt, updatedAt string
		var content, encryptedTitle, wrappedDEK, encryptionMetadata sql.NullString
		var encryptedContent []byte
		var noteType, journalDate sql.NullString

		if err := rows.Scan(
			&note.ID, &note.Title, &content, &note.FolderPath, &note.Version,
			&note.DisplayOrder, &note.Color, &createdAt, &updatedAt,
			&encryptedContent, &note.ContentEncrypted, &encryptedTitle, &note.TitleEncrypted,
			&wrappedDEK, &note.EncryptionVersion, &encryptionMetadata,
			&noteType, &journalDate, &note.AIEnabled, &note.UserID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan recipe note: %w", err)
		}

		note.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		note.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		note.Content = content.String
		note.EncryptedContent = encryptedContent
		if encryptedTitle.Valid {
			note.EncryptedTitle = &encryptedTitle.String
		}
		note.WrappedDEK = wrappedDEK.String
		note.EncryptionMetadata = encryptionMetadata.String
		if noteType.Valid {
			note.NoteType = noteType.String
		}
		if journalDate.Valid {
			note.JournalDate = &journalDate.String
		}
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recipes: %w", err)
	}

	if notes == nil {
		notes = []Note{}
	}
	return notes, nil
}

// CollectionHasEncryptedRecipes returns true if any recipe in the collection is encrypted.
func (db *DB) CollectionHasEncryptedRecipes(collectionID int) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM recipe_collection_items ci
		JOIN notes n ON n.id = ci.note_id
		WHERE ci.collection_id = ? AND n.is_deleted = 0 AND n.content_encrypted = 1
	`, collectionID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check collection encrypted recipes: %w", err)
	}
	return count > 0, nil
}

// getCollectionShareByID retrieves a collection share by its ID (internal helper).
func (db *DB) getCollectionShareByID(id int) (*CollectionShare, error) {
	var s CollectionShare
	err := db.QueryRow(`
		SELECT rcs.id, rcs.collection_id, rc.name,
		       rcs.owner_user_id, ou.username,
		       rcs.shared_with_user_id, su.username, rcs.role,
		       rcs.created_at, rcs.updated_at
		FROM recipe_collection_shares rcs
		JOIN recipe_collections rc ON rc.id = rcs.collection_id
		JOIN users ou ON ou.id = rcs.owner_user_id
		JOIN users su ON su.id = rcs.shared_with_user_id
		WHERE rcs.id = ?
	`, id).Scan(
		&s.ID, &s.CollectionID, &s.CollectionName,
		&s.OwnerUserID, &s.OwnerUsername,
		&s.SharedWithUserID, &s.SharedWithUsername, &s.Role,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get collection share: %w", err)
	}
	return &s, nil
}
