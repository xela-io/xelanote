package service

import (
	"errors"
	"log/slog"

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

const (
	defaultBaseServings = 4
	maxRecipeImages     = 50
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
