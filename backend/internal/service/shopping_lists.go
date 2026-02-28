package service

import (
	"errors"

	"github.com/xela-io/xelanote/internal/db"
)

// CreateShoppingList creates a new shopping list.
func (s *ShoppingService) CreateShoppingList(userID int, name string, color *string) (*ShoppingList, error) {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}

	list, err := s.db.CreateShoppingList(userID, name, color)
	if err != nil {
		if errors.Is(err, db.ErrDuplicate) {
			return nil, db.ErrDuplicate
		}
		return nil, err
	}
	return list, nil
}

// GetShoppingList retrieves a list with items (access-checked).
func (s *ShoppingService) GetShoppingList(userID, listID int) (*ShoppingListDetail, error) {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}

	role, err := s.requireAccess(listID, userID, "viewer")
	if err != nil {
		return nil, err
	}

	detail, err := s.db.GetShoppingListDetail(listID, userID)
	if err != nil {
		return nil, err
	}
	detail.Role = role
	return detail, nil
}

// ListShoppingLists returns all lists accessible to a user.
func (s *ShoppingService) ListShoppingLists(userID int) ([]ShoppingListSummary, error) {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}
	return s.db.ListShoppingLists(userID)
}

// UpdateShoppingList updates list name/color with optimistic locking.
func (s *ShoppingService) UpdateShoppingList(userID, listID int, name *string, color *string, expectedVersion int) (*ShoppingList, error) {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}

	if _, err := s.requireAccess(listID, userID, "owner"); err != nil {
		return nil, err
	}

	return s.db.UpdateShoppingList(listID, name, color, expectedVersion)
}

// ArchiveShoppingList archives a list (owner only).
func (s *ShoppingService) ArchiveShoppingList(userID, listID int, expectedVersion int) error {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return err
	}

	if _, err := s.requireAccess(listID, userID, "owner"); err != nil {
		return err
	}

	return s.db.ArchiveShoppingList(listID, expectedVersion)
}

// DeleteShoppingList permanently deletes a list (owner only).
func (s *ShoppingService) DeleteShoppingList(userID, listID int) error {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return err
	}

	if _, err := s.requireAccess(listID, userID, "owner"); err != nil {
		return err
	}

	return s.db.DeleteShoppingList(listID)
}

// checkFeatureEnabled verifies the shopping feature is active for the user.
func (s *ShoppingService) checkFeatureEnabled(userID int) error {
	feature, err := s.db.GetUserFeature(userID, "shopping")
	if err != nil {
		return err
	}
	if !feature.Enabled {
		return ErrShoppingFeatureNotEnabled
	}
	return nil
}

// requireAccess checks that a user has at least the specified role.
// Returns the actual role if access is granted.
func (s *ShoppingService) requireAccess(listID, userID int, minRole string) (string, error) {
	role, err := s.db.GetShoppingListAccess(listID, userID)
	if err != nil {
		return "", err
	}

	if role == "" {
		return "", ErrNoListAccess
	}

	// Check minimum role level: owner > editor > viewer
	roleLevel := map[string]int{"viewer": 1, "editor": 2, "owner": 3}
	if roleLevel[role] < roleLevel[minRole] {
		if minRole == "owner" {
			return "", ErrNotListOwner
		}
		return "", ErrNoListAccess
	}

	return role, nil
}
