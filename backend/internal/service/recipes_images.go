package service

import (
	"github.com/xela-io/xelanote/internal/db"
)

// --- Recipe Images ---

// AddRecipeImage adds an image to a recipe.
func (s *RecipeService) AddRecipeImage(callerUserID int, noteID string, imageURL string, caption *string) (*db.RecipeImage, error) {
	_, err := s.checkRecipeWriteAccess(callerUserID, noteID)
	if err != nil {
		return nil, err
	}

	// SEC-003: Validate URL format and owner — prevents cross-user upload URL signing oracle
	ownerID, _, err := ParseUploadURL(imageURL)
	if err != nil {
		return nil, ErrInvalidImageURL
	}
	if ownerID != callerUserID {
		return nil, ErrForbidden
	}

	// Max recipe images
	existing, err := s.db.GetRecipeImages(noteID)
	if err != nil {
		return nil, err
	}
	if len(existing) >= maxRecipeImages {
		return nil, ErrMaxImagesReached
	}

	return s.db.AddRecipeImage(noteID, callerUserID, imageURL, caption)
}

// DeleteRecipeImage deletes an image from a recipe.
// Owner can delete any image, editor can only delete their own.
func (s *RecipeService) DeleteRecipeImage(callerUserID int, noteID string, imageID int) error {
	ownerID, err := s.checkRecipeWriteAccess(callerUserID, noteID)
	if err != nil {
		return err
	}

	// Ownership check: editor can only delete own images
	if callerUserID != ownerID {
		img, err := s.db.GetRecipeImage(imageID)
		if err != nil {
			return err
		}
		if img.UserID != callerUserID {
			return ErrForbidden
		}
	}

	return s.db.DeleteRecipeImage(imageID)
}

// UpdateRecipeImageCaption updates the caption of a recipe image.
// Owner can update any caption, editor can only update their own.
func (s *RecipeService) UpdateRecipeImageCaption(callerUserID int, noteID string, imageID int, caption *string) error {
	ownerID, err := s.checkRecipeWriteAccess(callerUserID, noteID)
	if err != nil {
		return err
	}

	// Ownership check: editor can only update own captions
	if callerUserID != ownerID {
		img, err := s.db.GetRecipeImage(imageID)
		if err != nil {
			return err
		}
		if img.UserID != callerUserID {
			return ErrForbidden
		}
	}

	return s.db.UpdateRecipeImageCaption(imageID, caption)
}

// ReorderRecipeImages reorders images for a recipe.
func (s *RecipeService) ReorderRecipeImages(callerUserID int, noteID string, imageIDs []int) error {
	_, err := s.checkRecipeWriteAccess(callerUserID, noteID)
	if err != nil {
		return err
	}

	if len(imageIDs) == 0 {
		return ErrInvalidInput
	}

	return s.db.ReorderRecipeImages(noteID, imageIDs)
}
