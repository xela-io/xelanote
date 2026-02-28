package db

import (
	"database/sql"
	"fmt"
	"time"
)

// CreateRecoveryResetToken stores a short-lived password-recovery reset token hash.
func (db *DB) CreateRecoveryResetToken(userID int, tokenHash string, expiresAt time.Time) error {
	now := time.Now().UTC().Format(time.RFC3339)
	expiresAtRFC3339 := expiresAt.UTC().Format(time.RFC3339)

	if _, err := db.Exec(`
		INSERT INTO recovery_reset_tokens (token_hash, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, tokenHash, userID, expiresAtRFC3339, now); err != nil {
		return fmt.Errorf("insert recovery reset token: %w", err)
	}

	return nil
}

// ValidateRecoveryResetToken returns the associated user if token is valid and unconsumed.
func (db *DB) ValidateRecoveryResetToken(tokenHash string) (int, error) {
	var userID int
	var expiresAt string
	var consumedAt sql.NullString

	err := db.QueryRow(`
		SELECT user_id, expires_at, consumed_at
		FROM recovery_reset_tokens
		WHERE token_hash = ?
	`, tokenHash).Scan(&userID, &expiresAt, &consumedAt)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("query recovery reset token: %w", err)
	}

	if consumedAt.Valid {
		return 0, ErrNotFound
	}

	expiresTime, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return 0, fmt.Errorf("parse recovery token expiry: %w", err)
	}
	if time.Now().After(expiresTime) {
		if _, delErr := db.Exec(`DELETE FROM recovery_reset_tokens WHERE token_hash = ?`, tokenHash); delErr != nil {
			return 0, fmt.Errorf("delete expired recovery token: %w", delErr)
		}
		return 0, ErrNotFound
	}

	return userID, nil
}

// ConsumeRecoveryResetTokenTx marks a valid token as consumed and returns its user ID.
// It is single-use by design; once consumed it cannot be reused.
func (tx *Tx) ConsumeRecoveryResetTokenTx(tokenHash string) (int, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	var userID int
	err := tx.QueryRow(`
		SELECT user_id
		FROM recovery_reset_tokens
		WHERE token_hash = ? AND consumed_at IS NULL AND expires_at > ?
	`, tokenHash, now).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("query recovery reset token in tx: %w", err)
	}

	result, err := tx.Exec(`
		UPDATE recovery_reset_tokens
		SET consumed_at = ?
		WHERE token_hash = ? AND consumed_at IS NULL AND expires_at > ?
	`, now, tokenHash, now)
	if err != nil {
		return 0, fmt.Errorf("consume recovery reset token in tx: %w", err)
	}

	rows, err := rowsAffectedCount(result, "failed to check recovery token rows affected")
	if err != nil {
		return 0, err
	}
	if rows != 1 {
		return 0, ErrNotFound
	}

	return userID, nil
}
