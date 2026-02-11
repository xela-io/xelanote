package db

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

// RecipeSummary is a lightweight representation of a recipe for LLM prompts.
type RecipeSummary struct {
	NoteID          string
	Title           string
	IngredientNames []string
	ContentSnippet  string
	Difficulty      *string
	Servings        int
}

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
