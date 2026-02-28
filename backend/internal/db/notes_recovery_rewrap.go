package db

import "fmt"

// BulkUpdateRecoveryWrappedDEKs updates wrapped_dek_recovery fields for multiple notes and versions.
// noteUpdates maps noteID -> new wrapped_dek_recovery.
// versionUpdates maps versionID (as string) -> new wrapped_dek_recovery.
func (db *DB) BulkUpdateRecoveryWrappedDEKs(userID int, noteUpdates map[string]string, versionUpdates map[string]string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := (&Tx{Tx: tx}).BulkUpdateRecoveryWrappedDEKsTx(userID, noteUpdates, versionUpdates); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BulkUpdateRecoveryWrappedDEKsTx updates wrapped_dek_recovery fields within an existing transaction.
func (tx *Tx) BulkUpdateRecoveryWrappedDEKsTx(userID int, noteUpdates map[string]string, versionUpdates map[string]string) error {
	for noteID, newWrappedDEKRecovery := range noteUpdates {
		_, err := tx.Exec(`
			UPDATE notes
			SET wrapped_dek_recovery = ?
			WHERE id = ? AND user_id = ?
		`, newWrappedDEKRecovery, noteID, userID)
		if err != nil {
			return fmt.Errorf("failed to update wrapped_dek_recovery for note %s: %w", noteID, err)
		}
	}

	for versionIDStr, newWrappedDEKRecovery := range versionUpdates {
		_, err := tx.Exec(`
			UPDATE note_versions
			SET wrapped_dek_recovery = ?
			WHERE id = ? AND user_id = ?
		`, newWrappedDEKRecovery, versionIDStr, userID)
		if err != nil {
			return fmt.Errorf("failed to update wrapped_dek_recovery for version %s: %w", versionIDStr, err)
		}
	}

	return nil
}

// ClearRecoveryWrappedDEKs clears all wrapped_dek_recovery values for a user in notes and note_versions.
func (db *DB) ClearRecoveryWrappedDEKs(userID int) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		UPDATE notes
		SET wrapped_dek_recovery = NULL
		WHERE user_id = ?
	`, userID); err != nil {
		return fmt.Errorf("failed to clear note wrapped_dek_recovery values: %w", err)
	}

	if _, err := tx.Exec(`
		UPDATE note_versions
		SET wrapped_dek_recovery = NULL
		WHERE user_id = ?
	`, userID); err != nil {
		return fmt.Errorf("failed to clear version wrapped_dek_recovery values: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
