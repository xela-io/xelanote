package service

// GetShoppingFavorites returns all favorites for a user.
func (s *ShoppingService) GetShoppingFavorites(userID int) ([]ShoppingFavorite, error) {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}
	return s.db.GetShoppingFavorites(userID)
}

// AddShoppingFavorite creates a new favorite item template.
func (s *ShoppingService) AddShoppingFavorite(userID int, name string, quantity *float64, unit *string, category *string) (*ShoppingFavorite, error) {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return nil, err
	}
	return s.db.AddShoppingFavorite(userID, name, quantity, unit, category)
}

// RemoveShoppingFavorite deletes a favorite.
func (s *ShoppingService) RemoveShoppingFavorite(userID, favoriteID int) error {
	if err := s.checkFeatureEnabled(userID); err != nil {
		return err
	}
	return s.db.RemoveShoppingFavorite(userID, favoriteID)
}
