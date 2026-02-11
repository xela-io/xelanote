package db

// ReorderFolders updates the display_order for folders within a parent.
// items is a list of folder IDs in the desired order.
func (d *DB) ReorderFolders(userID int, parentID *int, items []int) error {
	// Start transaction
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update each folder's display_order
	// Use spacing of 100 to allow for future insertions
	for i, folderID := range items {
		newOrder := i * 100
		_, err := tx.Exec(`
			UPDATE folders
			SET display_order = ?, updated_at = datetime('now')
			WHERE id = ? AND user_id = ?
		`, newOrder, folderID, userID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
