package db

// ReorderNotes updates the display_order for notes within a folder.
// noteIDs is a list of note IDs (UUIDs) in the desired order.
func (d *DB) ReorderNotes(userID int, folderPath string, noteIDs []string) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, noteID := range noteIDs {
		newOrder := i * 100
		_, err := tx.Exec(`
			UPDATE notes
			SET display_order = ?, updated_at = datetime('now')
			WHERE id = ? AND user_id = ? AND folder_path = ? AND is_deleted = 0
		`, newOrder, noteID, userID, folderPath)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
