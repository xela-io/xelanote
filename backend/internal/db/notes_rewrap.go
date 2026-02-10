package db

import "fmt"

// BulkUpdateWrappedDEKs updates wrapped_dek fields for multiple notes and versions in a transaction.
// noteUpdates maps noteID -> new_wrapped_dek
// versionUpdates maps versionID (as string) -> new_wrapped_dek
func (db *DB) BulkUpdateWrappedDEKs(userID int, noteUpdates map[string]string, versionUpdates map[string]string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update notes
	for noteID, newWrappedDEK := range noteUpdates {
		_, err := tx.Exec(`
			UPDATE notes
			SET wrapped_dek = ?
			WHERE id = ? AND user_id = ?
		`, newWrappedDEK, noteID, userID)
		if err != nil {
			return fmt.Errorf("failed to update wrapped_dek for note %s: %w", noteID, err)
		}
	}

	// Update note_versions
	for versionIDStr, newWrappedDEK := range versionUpdates {
		_, err := tx.Exec(`
			UPDATE note_versions
			SET wrapped_dek = ?
			WHERE id = ? AND user_id = ?
		`, newWrappedDEK, versionIDStr, userID)
		if err != nil {
			return fmt.Errorf("failed to update wrapped_dek for version %s: %w", versionIDStr, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BulkUpdateWrappedDEKsTx updates wrapped_dek fields within an existing transaction.
// Use this when atomically updating DEKs along with other operations (e.g., password change).
func (tx *Tx) BulkUpdateWrappedDEKsTx(userID int, noteUpdates map[string]string, versionUpdates map[string]string) error {
	// Update notes
	for noteID, newWrappedDEK := range noteUpdates {
		_, err := tx.Exec(`
			UPDATE notes
			SET wrapped_dek = ?
			WHERE id = ? AND user_id = ?
		`, newWrappedDEK, noteID, userID)
		if err != nil {
			return fmt.Errorf("failed to update wrapped_dek for note %s: %w", noteID, err)
		}
	}

	// Update note_versions
	for versionIDStr, newWrappedDEK := range versionUpdates {
		_, err := tx.Exec(`
			UPDATE note_versions
			SET wrapped_dek = ?
			WHERE id = ? AND user_id = ?
		`, newWrappedDEK, versionIDStr, userID)
		if err != nil {
			return fmt.Errorf("failed to update wrapped_dek for version %s: %w", versionIDStr, err)
		}
	}

	return nil
}
