package service

import (
	"fmt"
	"strings"

	"github.com/xela-io/xelanote/internal/db"
)

// --- Collections (owner-only) ---

func (s *RecipeService) ListCollections(userID int) ([]db.RecipeCollection, error) {
	return s.db.ListRecipeCollections(userID)
}

func (s *RecipeService) CreateCollection(userID int, name string, description, color *string) (*db.RecipeCollection, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("collection name is required")
	}
	if len(name) > 200 {
		return nil, fmt.Errorf("collection name too long (max 200)")
	}
	if description != nil && len(*description) > 1000 {
		return nil, fmt.Errorf("description too long (max 1000)")
	}
	if color != nil && len(*color) > 20 {
		return nil, fmt.Errorf("color too long (max 20)")
	}
	return s.db.CreateRecipeCollection(userID, name, description, color)
}

func (s *RecipeService) UpdateCollection(userID, collectionID int, name string, description, color *string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("collection name is required")
	}
	if len(name) > 200 {
		return fmt.Errorf("collection name too long (max 200)")
	}
	return s.db.UpdateRecipeCollection(userID, collectionID, name, description, color)
}

func (s *RecipeService) DeleteCollection(userID, collectionID int) error {
	return s.db.DeleteRecipeCollection(userID, collectionID)
}

func (s *RecipeService) AddToCollection(userID, collectionID int, noteID string) error {
	// R5: Only recipe notes can be added to collections
	note, err := s.db.GetNote(userID, noteID)
	if err != nil {
		return err
	}
	if note.NoteType != db.NoteTypeRecipe {
		return ErrNotRecipeNote
	}
	return s.db.AddRecipeToCollection(userID, collectionID, noteID)
}

func (s *RecipeService) RemoveFromCollection(userID, collectionID int, noteID string) error {
	return s.db.RemoveRecipeFromCollection(userID, collectionID, noteID)
}

func (s *RecipeService) ListCollectionItems(userID, collectionID int) ([]db.Note, error) {
	return s.db.ListRecipesInCollection(userID, collectionID)
}
