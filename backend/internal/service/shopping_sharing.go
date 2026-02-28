package service

import (
	"errors"

	"github.com/xela-io/xelanote/internal/db"
)

// ShareShoppingList shares a list with another user (owner only).
func (s *ShoppingService) ShareShoppingList(userID, listID, targetUserID int, role string) (*ShoppingListShare, error) {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}

	if _, err := s.requireAccess(listID, userID, "owner"); err != nil {
		return nil, err
	}

	if userID == targetUserID {
		return nil, ErrCannotShareWithSelf
	}

	if role != "viewer" && role != "editor" {
		role = "editor"
	}

	share, err := s.db.ShareShoppingList(listID, userID, targetUserID, role)
	if err != nil {
		if errors.Is(err, db.ErrDuplicate) {
			return nil, ErrListAlreadyShared
		}
		return nil, err
	}
	return share, nil
}

// GetShoppingListShares returns all shares for a list (any role can view).
func (s *ShoppingService) GetShoppingListShares(userID, listID int) ([]ShoppingListShare, error) {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}

	if _, err := s.requireAccess(listID, userID, "viewer"); err != nil {
		return nil, err
	}

	return s.db.GetShoppingListShares(listID)
}

// UpdateShoppingListShareRole changes a share's role (owner only).
func (s *ShoppingService) UpdateShoppingListShareRole(userID, listID, targetUserID int, role string) error {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return err
	}

	if _, err := s.requireAccess(listID, userID, "owner"); err != nil {
		return err
	}

	if role != "viewer" && role != "editor" {
		return ErrInvalidInput
	}

	return s.db.UpdateShoppingListShareRole(listID, targetUserID, role)
}

// RemoveShoppingListShare removes a share (owner only).
func (s *ShoppingService) RemoveShoppingListShare(userID, listID, targetUserID int) error {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return err
	}

	if _, err := s.requireAccess(listID, userID, "owner"); err != nil {
		return err
	}

	return s.db.RemoveShoppingListShare(listID, targetUserID)
}

// GetShoppingListUserIDs returns all user IDs with access to a list (for WS broadcast).
func (s *ShoppingService) GetShoppingListUserIDs(listID int) ([]int, error) {
	return s.db.GetShoppingListUserIDs(listID)
}
