package db

// ShoppingList represents a user's shopping list.
type ShoppingList struct {
	ID           int     `json:"id"`
	UserID       int     `json:"user_id"`
	Name         string  `json:"name"`
	Color        *string `json:"color,omitempty"`
	IsArchived   bool    `json:"is_archived"`
	DisplayOrder int     `json:"display_order"`
	Version      int     `json:"version"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// ShoppingItem represents an item in a shopping list.
type ShoppingItem struct {
	ID             int      `json:"id"`
	ListID         int      `json:"list_id"`
	Name           string   `json:"name"`
	Quantity       *float64 `json:"quantity,omitempty"`
	Unit           *string  `json:"unit,omitempty"`
	Category       *string  `json:"category,omitempty"`
	CategoryOrder  int      `json:"category_order"`
	ParentID       *int     `json:"parent_id,omitempty"`
	IsChecked      bool     `json:"is_checked"`
	CheckedAt      *string  `json:"checked_at,omitempty"`
	DisplayOrder   int      `json:"display_order"`
	Version        int      `json:"version"`
	AddedByUserID  *int     `json:"added_by_user_id,omitempty"`
	SourceRecipeID *string  `json:"source_recipe_id,omitempty"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// ShoppingFavorite represents a frequently used item template.
type ShoppingFavorite struct {
	ID              int      `json:"id"`
	UserID          int      `json:"user_id"`
	Name            string   `json:"name"`
	DefaultQuantity *float64 `json:"default_quantity,omitempty"`
	DefaultUnit     *string  `json:"default_unit,omitempty"`
	Category        *string  `json:"category,omitempty"`
	UsageCount      int      `json:"usage_count"`
	CreatedAt       string   `json:"created_at"`
}

// ShoppingListShare represents a sharing record for a shopping list.
type ShoppingListShare struct {
	ID               int    `json:"id"`
	ListID           int    `json:"list_id"`
	OwnerUserID      int    `json:"owner_user_id"`
	SharedWithUserID int    `json:"shared_with_user_id"`
	SharedWithName   string `json:"shared_with_name,omitempty"`
	Role             string `json:"role"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// ShoppingListDetail is the full response for GET /api/shopping/lists/{id}.
type ShoppingListDetail struct {
	ShoppingList
	Items      []ShoppingItem      `json:"items"`
	ItemCount  int                 `json:"item_count"`
	SharedWith []ShoppingListShare `json:"shared_with,omitempty"`
	Role       string              `json:"role"`
}

// ShoppingListSummary is a lightweight list representation for the list overview.
type ShoppingListSummary struct {
	ShoppingList
	ItemCount    int    `json:"item_count"`
	CheckedCount int    `json:"checked_count"`
	Role         string `json:"role"`
	SharedBy     string `json:"shared_by,omitempty"`
}
