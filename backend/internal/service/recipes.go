package service

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/xela-io/xelanote/internal/db"
)

var (
	ErrRecipeFeatureNotEnabled       = errors.New("recipe feature not enabled")
	ErrRecipeEncrypted               = errors.New("recipe is encrypted")
	ErrRecipeMetadataNotFound        = errors.New("recipe metadata not found - create metadata first")
	ErrForbidden                     = errors.New("forbidden")
	ErrNotCollectionOwner            = errors.New("only the collection owner can manage shares")
	ErrCollectionAlreadyShared       = errors.New("collection is already shared with this user")
	ErrCollectionHasEncryptedRecipes = errors.New("collection contains encrypted recipes and cannot be shared")
	ErrNotRecipeNote                 = errors.New("only recipe notes can be added to collections")
	ErrMaxImagesReached              = errors.New("maximum number of images reached (50)")
	ErrInvalidImageURL               = errors.New("image URL must start with /api/uploads/")
	ErrInvalidInput                  = errors.New("invalid input")
)

// RecipeService handles recipe-specific business logic.
type RecipeService struct {
	db     *db.DB
	logger *slog.Logger
	notes  *NoteService
}

// NewRecipeService creates a new RecipeService.
func NewRecipeService(database *db.DB, noteService *NoteService) *RecipeService {
	return &RecipeService{
		db:     database,
		logger: slog.Default(),
		notes:  noteService,
	}
}

// resolveOwnerID determines the owner user_id for a recipe note (I1).
// If callerUserID is the owner, returns callerUserID.
// If callerUserID has share access (editor), returns the actual owner.
func (s *RecipeService) resolveOwnerID(noteID string, callerUserID int) (int, string, error) {
	// Try as owner first
	note, err := s.db.GetNote(callerUserID, noteID)
	if err == nil && note != nil {
		return note.UserID, "", nil
	}

	// Not found as owner — check share permission
	perm, err := s.db.GetSharePermission(callerUserID, noteID)
	if err != nil {
		return 0, "", fmt.Errorf("failed to check share permission: %w", err)
	}
	if perm == "" {
		return 0, "", ErrForbidden
	}

	ownerID, err := s.db.GetNoteOwnerUserID(noteID)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get note owner: %w", err)
	}

	return ownerID, perm, nil
}

// checkFeatureEnabled verifies the recipe feature is enabled for the user.
func (s *RecipeService) checkFeatureEnabled(userID int) error {
	feature, err := s.db.GetUserFeature(userID, "recipe")
	if err != nil {
		return err
	}
	if !feature.Enabled {
		return ErrRecipeFeatureNotEnabled
	}
	return nil
}

// CreateRecipeNote creates a new recipe note with default metadata.
func (s *RecipeService) CreateRecipeNote(userID int, title, content, folderPath string) (*db.Note, error) {
	// Check note limit
	maxNotes, err := s.db.GetMaxNotesPerUser()
	if err != nil {
		return nil, err
	}
	if maxNotes > 0 {
		currentCount, err := s.db.GetNoteCountForUser(userID)
		if err != nil {
			return nil, err
		}
		if currentCount >= maxNotes {
			return nil, ErrNoteLimitExceeded
		}
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
	// Check note limit
	maxNotes, err := s.db.GetMaxNotesPerUser()
	if err != nil {
		return nil, err
	}
	if maxNotes > 0 {
		currentCount, err := s.db.GetNoteCountForUser(userID)
		if err != nil {
			return nil, err
		}
		if currentCount >= maxNotes {
			return nil, ErrNoteLimitExceeded
		}
	}

	// Check feature enabled
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}

	note, err := s.db.CreateEncryptedRecipeNote(
		userID, title, encryptedTitle, titleEncrypted,
		encryptedContent, wrappedDEK, encryptionMetadata, folderPath,
	)
	if err != nil {
		return nil, err
	}

	// Insert keywords if enabled
	if len(keywords) > 0 {
		prefs, err := s.db.GetUserPreferences(userID)
		if err == nil && prefs.KeywordsEnabled {
			for _, kw := range keywords {
				if err := s.db.InsertNoteKeyword(note.ID, kw); err != nil {
					s.notes.logger.Warn("failed to insert keyword", "error", err)
				}
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
func (s *RecipeService) ListRecipes(userID int) ([]db.Note, error) {
	return s.db.ListRecipes(userID)
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

	_ = perm // perm checked implicitly in resolveOwnerID

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
	ownerID, perm, err := s.resolveOwnerID(noteID, callerUserID)
	if err != nil {
		return err
	}

	// Check write permission for shared users
	if callerUserID != ownerID && perm != "editor" {
		return ErrForbidden
	}

	// Encryption check (I2)
	encrypted, err := s.db.IsNoteEncrypted(noteID)
	if err != nil {
		return err
	}
	if encrypted {
		return ErrRecipeEncrypted
	}

	// Validate
	if err := validateRecipeMetadata(meta); err != nil {
		return err
	}

	return s.db.SetRecipeMetadata(noteID, ownerID, meta, expectedUpdatedAt)
}

// SetRecipeIngredients replaces all ingredients for a recipe.
// Enforces I1 (ownership) and I2 (encryption guard).
func (s *RecipeService) SetRecipeIngredients(callerUserID int, noteID string,
	ingredients []db.RecipeIngredient, expectedUpdatedAt string) error {

	ownerID, perm, err := s.resolveOwnerID(noteID, callerUserID)
	if err != nil {
		return err
	}

	// Check write permission for shared users
	if callerUserID != ownerID && perm != "editor" {
		return ErrForbidden
	}

	// Encryption check (I2)
	encrypted, err := s.db.IsNoteEncrypted(noteID)
	if err != nil {
		return err
	}
	if encrypted {
		return ErrRecipeEncrypted
	}

	// Validate ingredients
	for i, ing := range ingredients {
		if strings.TrimSpace(ing.Name) == "" {
			return fmt.Errorf("ingredient %d: name is required", i)
		}
		if len(ing.Name) > 200 {
			return fmt.Errorf("ingredient %d: name too long (max 200)", i)
		}
		if ing.Unit != nil && len(*ing.Unit) > 50 {
			return fmt.Errorf("ingredient %d: unit too long (max 50)", i)
		}
		if ing.GroupName != nil && len(*ing.GroupName) > 100 {
			return fmt.Errorf("ingredient %d: group_name too long (max 100)", i)
		}
		if ing.AmountText != nil && len(*ing.AmountText) > 100 {
			return fmt.Errorf("ingredient %d: amount_text too long (max 100)", i)
		}
	}

	return s.db.SetRecipeIngredients(noteID, ownerID, ingredients, expectedUpdatedAt)
}

// GetScaledIngredients returns scaled ingredients for a recipe.
func (s *RecipeService) GetScaledIngredients(callerUserID int, noteID string, targetServings int) ([]db.ScaledIngredient, error) {
	ownerID, _, err := s.resolveOwnerID(noteID, callerUserID)
	if err != nil {
		return nil, err
	}

	// Encryption check
	encrypted, err := s.db.IsNoteEncrypted(noteID)
	if err != nil {
		return nil, err
	}
	if encrypted {
		return nil, ErrRecipeEncrypted
	}

	meta, err := s.db.GetRecipeMetadata(noteID)
	if err != nil {
		return nil, err
	}

	baseServings := 4 // Default (I5)
	if meta != nil {
		baseServings = meta.Servings
	}

	ingredients, err := s.db.GetRecipeIngredients(noteID)
	if err != nil {
		return nil, err
	}

	_ = ownerID
	return db.ScaleIngredients(ingredients, baseServings, targetServings), nil
}

// --- Recipe Images ---

// AddRecipeImage adds an image to a recipe.
func (s *RecipeService) AddRecipeImage(callerUserID int, noteID string, imageURL string, caption *string) (*db.RecipeImage, error) {
	ownerID, perm, err := s.resolveOwnerID(noteID, callerUserID)
	if err != nil {
		return nil, err
	}
	if callerUserID != ownerID && perm != "editor" {
		return nil, ErrForbidden
	}

	// Encryption guard
	encrypted, err := s.db.IsNoteEncrypted(noteID)
	if err != nil {
		return nil, err
	}
	if encrypted {
		return nil, ErrRecipeEncrypted
	}

	// URL validation
	if !strings.HasPrefix(imageURL, "/api/uploads/") {
		return nil, ErrInvalidImageURL
	}

	// Max 50 images
	existing, err := s.db.GetRecipeImages(noteID)
	if err != nil {
		return nil, err
	}
	if len(existing) >= 50 {
		return nil, ErrMaxImagesReached
	}

	return s.db.AddRecipeImage(noteID, callerUserID, imageURL, caption)
}

// DeleteRecipeImage deletes an image from a recipe.
// Owner can delete any image, editor can only delete their own.
func (s *RecipeService) DeleteRecipeImage(callerUserID int, noteID string, imageID int) error {
	ownerID, perm, err := s.resolveOwnerID(noteID, callerUserID)
	if err != nil {
		return err
	}
	if callerUserID != ownerID && perm != "editor" {
		return ErrForbidden
	}

	// Encryption guard
	encrypted, err := s.db.IsNoteEncrypted(noteID)
	if err != nil {
		return err
	}
	if encrypted {
		return ErrRecipeEncrypted
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
	ownerID, perm, err := s.resolveOwnerID(noteID, callerUserID)
	if err != nil {
		return err
	}
	if callerUserID != ownerID && perm != "editor" {
		return ErrForbidden
	}

	// Encryption guard
	encrypted, err := s.db.IsNoteEncrypted(noteID)
	if err != nil {
		return err
	}
	if encrypted {
		return ErrRecipeEncrypted
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
	ownerID, perm, err := s.resolveOwnerID(noteID, callerUserID)
	if err != nil {
		return err
	}
	if callerUserID != ownerID && perm != "editor" {
		return ErrForbidden
	}

	// Encryption guard
	encrypted, err := s.db.IsNoteEncrypted(noteID)
	if err != nil {
		return err
	}
	if encrypted {
		return ErrRecipeEncrypted
	}

	if len(imageIDs) == 0 {
		return ErrInvalidInput
	}

	return s.db.ReorderRecipeImages(noteID, imageIDs)
}

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

// --- Validation helpers ---

func validateRecipeMetadata(m *db.RecipeMetadata) error {
	if m.Servings < 1 || m.Servings > 999 {
		return fmt.Errorf("servings must be between 1 and 999")
	}
	if m.PrepTimeMinutes != nil && (*m.PrepTimeMinutes < 0 || *m.PrepTimeMinutes > 99999) {
		return fmt.Errorf("prep_time_minutes must be between 0 and 99999")
	}
	if m.CookTimeMinutes != nil && (*m.CookTimeMinutes < 0 || *m.CookTimeMinutes > 99999) {
		return fmt.Errorf("cook_time_minutes must be between 0 and 99999")
	}
	if m.Difficulty != nil {
		d := *m.Difficulty
		if d != "easy" && d != "medium" && d != "hard" {
			return fmt.Errorf("difficulty must be 'easy', 'medium', or 'hard'")
		}
	}
	if m.SourceURL != nil {
		u := *m.SourceURL
		if len(u) > 2048 {
			return fmt.Errorf("source_url too long (max 2048)")
		}
		if u != "" {
			parsed, err := url.Parse(u)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				return fmt.Errorf("source_url must be a valid http/https URL")
			}
		}
	}
	return nil
}
