package db

// UpdateFolderColor updates the color of a folder.
// Pass nil to remove the color.
func (d *DB) UpdateFolderColor(userID int, folderID int, color *string) error {
	// Validate color format if provided
	if color != nil {
		if err := validateHexColor(*color); err != nil {
			return err
		}
	}

	result, err := d.Exec(`
		UPDATE folders
		SET color = ?, updated_at = datetime('now')
		WHERE id = ? AND user_id = ?
	`, color, folderID, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}
