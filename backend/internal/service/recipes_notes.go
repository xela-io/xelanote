package service

import (
	"fmt"

	"github.com/xela-io/xelanote/internal/db"
)

// CreateRecipeNote creates a new recipe note with default metadata.
func (s *RecipeService) CreateRecipeNote(userID int, title, content, folderPath string) (*db.Note, error) {
	if err := s.notes.checkNoteLimit(userID); err != nil {
		return nil, err
	}

	// Check feature enabled
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}

	note, err := s.db.CreateRecipeNote(userID, title, content, folderPath)
	if err != nil {
		return nil, err
	}

	// Invalidate caches
	s.notes.invalidateFolderCache(userID)
	s.notes.invalidateQuickSearchCache(userID)
	s.notes.invalidateNotesByFolderCache(userID, folderPath)
	if s.notes.graphService != nil {
		s.notes.graphService.InvalidateGraphCache(userID)
	}

	return note, nil
}

// CreateEncryptedRecipeNote creates a new encrypted recipe note with default metadata.
func (s *RecipeService) CreateEncryptedRecipeNote(
	userID int,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	keywords []string,
	folderPath string,
) (*db.Note, error) {
	return s.CreateEncryptedRecipeNoteWithID(
		userID,
		"",
		title,
		encryptedTitle,
		titleEncrypted,
		encryptedContent,
		wrappedDEK,
		encryptionMetadata,
		keywords,
		folderPath,
	)
}

// CreateEncryptedRecipeNoteWithID creates a new encrypted recipe note with
// optional client-provided note ID.
func (s *RecipeService) CreateEncryptedRecipeNoteWithID(
	userID int,
	noteID string,
	title string,
	encryptedTitle *string,
	titleEncrypted bool,
	encryptedContent []byte,
	wrappedDEK string,
	encryptionMetadata string,
	keywords []string,
	folderPath string,
) (*db.Note, error) {
	if err := s.notes.checkNoteLimit(userID); err != nil {
		return nil, err
	}

	// Check feature enabled
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}

	note, err := s.db.CreateEncryptedRecipeNoteWithID(
		userID, noteID, title, encryptedTitle, titleEncrypted,
		encryptedContent, wrappedDEK, encryptionMetadata, folderPath,
	)
	if err != nil {
		return nil, err
	}

	// Insert keywords if enabled
	if len(keywords) > 0 {
		prefs, err := s.db.GetUserPreferences(userID)
		if err == nil && prefs.KeywordsEnabled {
			if err := s.db.InsertNoteKeywords(note.ID, keywords); err != nil {
				s.notes.logger.Warn("failed to insert keywords", "error", err)
			}
		}
	}

	// Invalidate caches
	s.notes.invalidateFolderCache(userID)
	s.notes.invalidateQuickSearchCache(userID)
	s.notes.invalidateNotesByFolderCache(userID, folderPath)
	if s.notes.graphService != nil {
		s.notes.graphService.InvalidateGraphCache(userID)
	}

	return note, nil
}

// ListRecipes returns all recipe notes for the owner.
func (s *RecipeService) ListRecipes(userID int, fields string) ([]db.Note, error) {
	return s.db.ListRecipes(userID, fields)
}

// GetRecipeDetail returns the full recipe detail for a note.
// Handles encryption (I2) and sharing access checks.
func (s *RecipeService) GetRecipeDetail(callerUserID int, noteID string) (*db.RecipeDetail, error) {
	ownerID, perm, err := s.resolveOwnerID(noteID, callerUserID)
	if err != nil {
		return nil, err
	}

	// Get the note (as owner)
	note, err := s.db.GetNote(ownerID, noteID)
	if err != nil {
		return nil, err
	}

	isOwner := callerUserID == ownerID
	if !isOwner {
		note.IsShared = true
		note.ShareRole = perm
	}

	// Encryption check (I2)
	if note.ContentEncrypted {
		if !isOwner {
			return nil, ErrRecipeEncrypted
		}
		// Owner sees minimal payload
		return &db.RecipeDetail{
			Note:        *note,
			Metadata:    nil,
			Ingredients: []db.RecipeIngredient{},
			Images:      []db.RecipeImage{},
			Collections: []db.RecipeCollection{},
			Encrypted:   true,
		}, nil
	}

	// Get metadata and ingredients
	metadata, err := s.db.GetRecipeMetadata(noteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe metadata: %w", err)
	}

	ingredients, err := s.db.GetRecipeIngredients(noteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe ingredients: %w", err)
	}

	images, err := s.db.GetRecipeImages(noteID)
	if err != nil {
		s.logger.Warn("failed to get recipe images", "error", err, "note_id", noteID)
		images = []db.RecipeImage{}
	}

	// Collections are owner-only
	var collections []db.RecipeCollection
	if isOwner {
		collections, err = s.db.GetCollectionsForRecipe(ownerID, noteID)
		if err != nil {
			s.logger.Warn("failed to get collections for recipe", "error", err, "note_id", noteID)
			collections = []db.RecipeCollection{}
		}
	} else {
		collections = []db.RecipeCollection{}
	}

	return &db.RecipeDetail{
		Note:        *note,
		Metadata:    metadata,
		Ingredients: ingredients,
		Images:      images,
		Collections: collections,
		Encrypted:   false,
	}, nil
}

// UpdateRecipeMetadata updates recipe metadata with optimistic locking.
// Enforces I1 (ownership) and I2 (encryption guard).
func (s *RecipeService) UpdateRecipeMetadata(callerUserID int, noteID string, meta *db.RecipeMetadata, expectedUpdatedAt string) error {
	ownerID, err := s.checkRecipeWriteAccess(callerUserID, noteID)
	if err != nil {
		return err
	}

	if err := validateRecipeMetadata(meta); err != nil {
		return err
	}

	return s.db.SetRecipeMetadata(noteID, ownerID, meta, expectedUpdatedAt)
}

// SetRecipeIngredients replaces all ingredients for a recipe.
// Enforces I1 (ownership) and I2 (encryption guard).
func (s *RecipeService) SetRecipeIngredients(callerUserID int, noteID string,
	ingredients []db.RecipeIngredient, expectedUpdatedAt string) error {

	ownerID, err := s.checkRecipeWriteAccess(callerUserID, noteID)
	if err != nil {
		return err
	}

	if err := validateIngredients(ingredients); err != nil {
		return err
	}

	return s.db.SetRecipeIngredients(noteID, ownerID, ingredients, expectedUpdatedAt)
}

// GetScaledIngredients returns scaled ingredients for a recipe.
func (s *RecipeService) GetScaledIngredients(callerUserID int, noteID string, targetServings int) ([]db.ScaledIngredient, error) {
	_, err := s.checkRecipeReadAccess(callerUserID, noteID)
	if err != nil {
		return nil, err
	}

	meta, err := s.db.GetRecipeMetadata(noteID)
	if err != nil {
		return nil, err
	}

	baseServings := defaultBaseServings
	if meta != nil {
		baseServings = meta.Servings
	}

	ingredients, err := s.db.GetRecipeIngredients(noteID)
	if err != nil {
		return nil, err
	}

	return db.ScaleIngredients(ingredients, baseServings, targetServings), nil
}
