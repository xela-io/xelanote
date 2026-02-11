package service

import (
	"errors"

	"github.com/xela-io/xelanote/internal/db"
)

// --- Collection Sharing ---

// ShareCollection shares a collection with another user.
func (s *RecipeService) ShareCollection(ownerUserID, collectionID int, targetIdentifier, role string) (*db.CollectionShare, error) {
	// Check feature enabled
	if err := s.checkFeatureEnabled(ownerUserID); err != nil {
		return nil, err
	}

	// Check ownership
	actualOwnerID, err := s.db.GetCollectionOwnerUserID(collectionID)
	if err != nil {
		return nil, err
	}
	if actualOwnerID != ownerUserID {
		return nil, ErrNotCollectionOwner
	}

	// R4: Check if collection has encrypted recipes
	hasEncrypted, err := s.db.CollectionHasEncryptedRecipes(collectionID)
	if err != nil {
		return nil, err
	}
	if hasEncrypted {
		return nil, ErrCollectionHasEncryptedRecipes
	}

	// Find target user
	targetUser, err := s.db.GetUserByUsernameOrEmail(targetIdentifier)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Prevent self-sharing
	if targetUser.ID == ownerUserID {
		return nil, ErrCannotShareWithSelf
	}

	// Create the share
	share, err := s.db.CreateCollectionShare(ownerUserID, collectionID, targetUser.ID, role)
	if err != nil {
		if errors.Is(err, db.ErrDuplicate) {
			return nil, ErrCollectionAlreadyShared
		}
		return nil, err
	}

	s.logger.Info("collection shared",
		"collection_id", collectionID,
		"owner_id", ownerUserID,
		"shared_with_id", targetUser.ID,
		"role", role,
	)

	return share, nil
}

// UnshareCollection removes a collection share.
func (s *RecipeService) UnshareCollection(ownerUserID, collectionID, targetUserID int) error {
	// Check ownership
	actualOwnerID, err := s.db.GetCollectionOwnerUserID(collectionID)
	if err != nil {
		return err
	}
	if actualOwnerID != ownerUserID {
		return ErrNotCollectionOwner
	}

	return s.db.DeleteCollectionShare(ownerUserID, collectionID, targetUserID)
}

// GetCollectionShares returns all shares for a collection (owner-only).
func (s *RecipeService) GetCollectionShares(ownerUserID, collectionID int) ([]db.CollectionShare, error) {
	// Check ownership
	actualOwnerID, err := s.db.GetCollectionOwnerUserID(collectionID)
	if err != nil {
		return nil, err
	}
	if actualOwnerID != ownerUserID {
		return nil, ErrNotCollectionOwner
	}

	return s.db.GetCollectionShares(ownerUserID, collectionID)
}

// UpdateCollectionShareRole updates the role for a collection share.
func (s *RecipeService) UpdateCollectionShareRole(ownerUserID, collectionID, targetUserID int, role string) error {
	// Check ownership
	actualOwnerID, err := s.db.GetCollectionOwnerUserID(collectionID)
	if err != nil {
		return err
	}
	if actualOwnerID != ownerUserID {
		return ErrNotCollectionOwner
	}

	return s.db.UpdateCollectionShareRole(ownerUserID, collectionID, targetUserID, role)
}

// GetSharedCollectionsForUser returns all collections shared with a user.
func (s *RecipeService) GetSharedCollectionsForUser(userID int) ([]db.SharedCollection, error) {
	return s.db.GetSharedCollectionsForUser(userID)
}

// ListSharedCollectionItems returns recipes in a shared collection.
// Permission check: caller must have a share on the collection.
func (s *RecipeService) ListSharedCollectionItems(callerUserID, collectionID int) ([]db.Note, error) {
	perm, err := s.db.GetCollectionSharePermission(callerUserID, collectionID)
	if err != nil {
		return nil, err
	}
	if perm == "" {
		return nil, ErrForbidden
	}

	return s.db.ListRecipesInSharedCollection(collectionID)
}

// AddToSharedCollection adds a recipe to a shared collection (editor only).
func (s *RecipeService) AddToSharedCollection(callerUserID, collectionID int, noteID string) error {
	perm, err := s.db.GetCollectionSharePermission(callerUserID, collectionID)
	if err != nil {
		return err
	}
	if perm != "editor" {
		return ErrForbidden
	}

	// R5: Only recipe notes can be added
	ownerID, err := s.db.GetNoteOwnerUserID(noteID)
	if err != nil {
		return err
	}
	note, err := s.db.GetNote(ownerID, noteID)
	if err != nil {
		return err
	}
	if note.NoteType != db.NoteTypeRecipe {
		return ErrNotRecipeNote
	}

	// R4: Encrypted recipes cannot be added to shared collections
	if note.ContentEncrypted {
		return ErrRecipeEncrypted
	}

	return s.db.AddRecipeToCollection(ownerID, collectionID, noteID)
}

// RemoveFromSharedCollection removes a recipe from a shared collection (editor only).
func (s *RecipeService) RemoveFromSharedCollection(callerUserID, collectionID int, noteID string) error {
	perm, err := s.db.GetCollectionSharePermission(callerUserID, collectionID)
	if err != nil {
		return err
	}
	if perm != "editor" {
		return ErrForbidden
	}

	ownerID, err := s.db.GetCollectionOwnerUserID(collectionID)
	if err != nil {
		return err
	}

	return s.db.RemoveRecipeFromCollection(ownerID, collectionID, noteID)
}

// ListSharedRecipes returns all recipe notes shared with the given user.
func (s *RecipeService) ListSharedRecipes(userID int) ([]db.SharedNote, error) {
	return s.db.GetSharedRecipesForUser(userID)
}
