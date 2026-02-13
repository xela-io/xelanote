package api

import "github.com/xela-io/xelanote/internal/service"

// --- Request/Response types ---

type UpdateRecipeMetadataRequest struct {
	Servings          int     `json:"servings"`
	PrepTimeMinutes   *int    `json:"prep_time_minutes,omitempty"`
	CookTimeMinutes   *int    `json:"cook_time_minutes,omitempty"`
	SourceURL         *string `json:"source_url,omitempty"`
	Difficulty        *string `json:"difficulty,omitempty"`
	ExpectedUpdatedAt string  `json:"expected_updated_at"`
}

type SetRecipeIngredientsRequest struct {
	Ingredients       []service.RecipeIngredient `json:"ingredients"`
	ExpectedUpdatedAt string                `json:"expected_updated_at"`
}

type CreateCollectionRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
}

type UpdateCollectionRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
}

type AddToCollectionRequest struct {
	NoteID string `json:"note_id"`
}

// --- Collection Sharing request types ---

// ShareCollectionRequest represents the request to share a collection.
type ShareCollectionRequest struct {
	Identifier string `json:"identifier"` // Username or email
	Role       string `json:"role"`       // "viewer" or "editor"
}

// UpdateCollectionShareRoleRequest represents the request to update a collection share role.
type UpdateCollectionShareRoleRequest struct {
	Role string `json:"role"` // "viewer" or "editor"
}

// --- Recipe Image request types ---

// AddRecipeImageRequest represents the request to add an image to a recipe.
type AddRecipeImageRequest struct {
	ImageURL string  `json:"image_url"`
	Caption  *string `json:"caption,omitempty"`
}

// UpdateRecipeImageCaptionRequest represents the request to update an image caption.
type UpdateRecipeImageCaptionRequest struct {
	Caption *string `json:"caption"`
}

// ReorderRecipeImagesRequest represents the request to reorder images.
type ReorderRecipeImagesRequest struct {
	ImageIDs []int `json:"image_ids"`
}
