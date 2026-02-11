package db

import (
	"database/sql"
	"fmt"
	"time"
)

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
	collectionID, err := validateLastInsertID(id, "recipe collection id")
	if err != nil {
		return nil, err
	}
	return &RecipeCollection{
		ID:          collectionID,
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
	return ensureRowsAffectedWithContext(result, "check rows affected")
}

// DeleteRecipeCollection deletes a collection owned by the user.
func (db *DB) DeleteRecipeCollection(userID, collectionID int) error {
	result, err := db.Exec(`DELETE FROM recipe_collections WHERE id = ? AND user_id = ?`,
		collectionID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete recipe collection: %w", err)
	}
	return ensureRowsAffectedWithContext(result, "check rows affected")
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
	return ensureRowsAffectedWithContext(result, "check rows affected")
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
