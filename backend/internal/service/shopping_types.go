package service

import (
	"errors"
	"log/slog"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/llm"
)

// Type aliases for shopping-related DB types.
type ShoppingList = db.ShoppingList
type ShoppingItem = db.ShoppingItem
type ShoppingFavorite = db.ShoppingFavorite
type ShoppingListShare = db.ShoppingListShare
type ShoppingListDetail = db.ShoppingListDetail
type ShoppingListSummary = db.ShoppingListSummary
type ShoppingItemCategoryUpdate = db.ShoppingItemCategoryUpdate

var (
	ErrShoppingFeatureNotEnabled = errors.New("shopping feature not enabled")
	ErrNoListAccess              = errors.New("no access to this shopping list")
	ErrNotListOwner              = errors.New("only the list owner can perform this action")
	ErrListAlreadyShared         = errors.New("list is already shared with this user")
	ErrShoppingRecipeNotFound    = errors.New("recipe not found")
	ErrShoppingRecipeNoAccess    = errors.New("no access to this recipe")
)

// ShoppingService handles shopping list business logic.
type ShoppingService struct {
	db     *db.DB
	logger *slog.Logger
	router *llm.ProviderRouter
}

// NewShoppingService creates a new ShoppingService.
func NewShoppingService(database *db.DB, router *llm.ProviderRouter) *ShoppingService {
	return &ShoppingService{
		db:     database,
		logger: slog.Default(),
		router: router,
	}
}
