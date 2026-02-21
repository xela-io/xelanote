package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/xela-io/xelanote/internal/utils"
)

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
	titleNorm := utils.NormalizeTitle(title)
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
	titleNorm := utils.NormalizeTitle(title)
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
	titleNorm := utils.NormalizeTitle(title)
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
		    cook_time_minutes, difficulty, source_url)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, userID, metadata.Servings, metadata.PrepTimeMinutes,
		metadata.CookTimeMinutes, metadata.Difficulty, metadata.SourceURL)
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
			(note_id, user_id, amount, unit, name, group_name, display_order, optional, scalable)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, userID, ing.Amount, unit, name, ing.GroupName, i,
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

// ListRecipes returns all recipe notes for a user (owner-only).
func (db *DB) ListRecipes(userID int, fields string) ([]Note, error) {
	var query string
	if fields == "slim" {
		query = `
		SELECT id, title, '', folder_path, version, display_order, color,
		       created_at, updated_at,
		       NULL, content_encrypted, encrypted_title, title_encrypted,
		       wrapped_dek, encryption_version, encryption_metadata,
		       note_type, journal_date, ai_enabled
		FROM notes
		WHERE user_id = ? AND note_type = 'recipe' AND is_deleted = 0
		ORDER BY updated_at DESC`
	} else {
		query = `
		SELECT id, title, content, folder_path, version, display_order, color,
		       created_at, updated_at,
		       encrypted_content, content_encrypted, encrypted_title, title_encrypted,
		       wrapped_dek, encryption_version, encryption_metadata,
		       note_type, journal_date, ai_enabled
		FROM notes
		WHERE user_id = ? AND note_type = 'recipe' AND is_deleted = 0
		ORDER BY updated_at DESC`
	}
	rows, err := db.Query(query, userID)
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
