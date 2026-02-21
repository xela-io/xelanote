package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// TwoFactorAuth represents 2FA status for a user
type TwoFactorAuth struct {
	UserID             int
	TOTPSecret         string
	TOTPEnabled        bool
	TOTPVerifiedAt     string
	TOTPDisabledAt     string
	TOTPSetupStartedAt string
	LastTOTPStep       int64
}

// BackupCode represents a recovery code
type BackupCode struct {
	ID        int
	UserID    int
	CodeHash  string
	Used      bool
	UsedAt    string
	CreatedAt string
}

// GetTwoFactorAuth retrieves 2FA status for a user
func (db *DB) GetTwoFactorAuth(userID int) (*TwoFactorAuth, error) {
	var tfa TwoFactorAuth
	var totpSecret, totpVerifiedAt, totpDisabledAt, totpSetupStartedAt sql.NullString
	var lastTOTPStep sql.NullInt64

	err := db.QueryRow(`
		SELECT
			id,
			COALESCE(totp_secret, ''),
			COALESCE(totp_enabled, 0),
			COALESCE(totp_verified_at, ''),
			COALESCE(totp_disabled_at, ''),
			COALESCE(totp_setup_started_at, ''),
			last_totp_step
		FROM users
		WHERE id = ?
	`, userID).Scan(
		&tfa.UserID,
		&totpSecret,
		&tfa.TOTPEnabled,
		&totpVerifiedAt,
		&totpDisabledAt,
		&totpSetupStartedAt,
		&lastTOTPStep,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("query 2fa status: %w", err)
	}

	tfa.TOTPSecret = totpSecret.String
	tfa.TOTPVerifiedAt = totpVerifiedAt.String
	tfa.TOTPDisabledAt = totpDisabledAt.String
	tfa.TOTPSetupStartedAt = totpSetupStartedAt.String
	if lastTOTPStep.Valid {
		tfa.LastTOTPStep = lastTOTPStep.Int64
	}

	return &tfa, nil
}

// SetTOTPSecret stores the TOTP secret and sets setup timestamp
func (db *DB) SetTOTPSecret(userID int, secret string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	if _, err := db.Exec(`
		UPDATE users
		SET totp_secret = ?,
		    totp_setup_started_at = ?,
		    updated_at = ?
		WHERE id = ?
	`, secret, now, now, userID); err != nil {
		return fmt.Errorf("set totp secret: %w", err)
	}

	return nil
}

// ClearTOTPSetup removes expired/old TOTP setup data
func (db *DB) ClearTOTPSetup(userID int) error {
	now := time.Now().UTC().Format(time.RFC3339)

	if _, err := db.Exec(`
		UPDATE users
		SET totp_secret = NULL,
		    totp_setup_started_at = NULL,
		    updated_at = ?
		WHERE id = ?
	`, now, userID); err != nil {
		return fmt.Errorf("clear totp setup: %w", err)
	}

	return nil
}

// EnableTwoFactor activates 2FA after successful verification.
// Uses BEGIN IMMEDIATE to prevent concurrent enable race conditions (F2-03).
func (db *DB) EnableTwoFactor(userID int) error {
	tx, err := db.BeginImmediate()
	if err != nil {
		return fmt.Errorf("begin enable 2fa tx: %w", err)
	}
	defer tx.Rollback()

	// Re-check state inside transaction to prevent double-enable
	var alreadyEnabled bool
	if err := tx.QueryRow(`SELECT COALESCE(totp_enabled, 0) FROM users WHERE id = ?`, userID).Scan(&alreadyEnabled); err != nil {
		return fmt.Errorf("check 2fa state: %w", err)
	}
	if alreadyEnabled {
		return nil // idempotent
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.Exec(`
		UPDATE users
		SET totp_enabled = 1,
		    totp_verified_at = ?,
		    totp_setup_started_at = NULL,
		    updated_at = ?
		WHERE id = ?
	`, now, now, userID); err != nil {
		return fmt.Errorf("enable 2fa: %w", err)
	}

	return tx.Commit()
}

// DisableTwoFactor deactivates 2FA and removes all related data
func (db *DB) DisableTwoFactor(userID int) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin disable 2fa tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)

	// Clear 2FA fields and set disabled timestamp
	_, err = tx.Exec(`
		UPDATE users
		SET totp_secret = NULL,
		    totp_enabled = 0,
		    totp_verified_at = NULL,
		    totp_disabled_at = ?,
		    totp_setup_started_at = NULL,
		    last_totp_step = NULL,
		    updated_at = ?
		WHERE id = ?
	`, now, now, userID)
	if err != nil {
		return fmt.Errorf("clear 2fa fields: %w", err)
	}

	// Delete all backup codes
	_, err = tx.Exec(`DELETE FROM backup_codes WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("delete backup codes: %w", err)
	}

	return tx.Commit()
}

// UpdateLastTOTPStep stores the last used TOTP time step for replay protection.
// Uses BEGIN IMMEDIATE to close the race window between reading and updating
// the step value (F2-07). Returns number of rows affected (0 if step was already used).
func (db *DB) UpdateLastTOTPStep(userID int, step int64) (int64, error) {
	tx, err := db.BeginImmediate()
	if err != nil {
		return 0, fmt.Errorf("begin totp step tx: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		UPDATE users
		SET last_totp_step = ?
		WHERE id = ? AND (last_totp_step IS NULL OR last_totp_step < ?)
	`, step, userID, step)
	if err != nil {
		return 0, fmt.Errorf("update last totp step: %w", err)
	}

	affected, err := rowsAffectedCount(result, "")
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit totp step tx: %w", err)
	}
	return affected, nil
}

// CreateBackupCodes stores hashed backup codes
func (db *DB) CreateBackupCodes(userID int, codeHashes []string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin create backup codes tx: %w", err)
	}
	defer tx.Rollback()

	// Delete old backup codes
	_, err = tx.Exec(`DELETE FROM backup_codes WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("delete old backup codes: %w", err)
	}

	// Insert new codes
	stmt, err := tx.Prepare(`
		INSERT INTO backup_codes (user_id, code_hash)
		VALUES (?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare backup code insert: %w", err)
	}
	defer stmt.Close()

	for _, hash := range codeHashes {
		_, err = stmt.Exec(userID, hash)
		if err != nil {
			return fmt.Errorf("insert backup code: %w", err)
		}
	}

	return tx.Commit()
}

// GetBackupCodes retrieves all backup codes for a user
func (db *DB) GetBackupCodes(userID int) ([]BackupCode, error) {
	rows, err := db.Query(`
		SELECT id, user_id, code_hash, used, COALESCE(used_at, ''), created_at
		FROM backup_codes
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query backup codes: %w", err)
	}
	defer rows.Close()

	var codes []BackupCode
	for rows.Next() {
		var code BackupCode
		var usedAt sql.NullString

		err := rows.Scan(
			&code.ID,
			&code.UserID,
			&code.CodeHash,
			&code.Used,
			&usedAt,
			&code.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan backup code: %w", err)
		}

		code.UsedAt = usedAt.String
		codes = append(codes, code)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backup codes: %w", err)
	}

	return codes, nil
}

// MarkBackupCodeUsed marks a code as used (atomic operation)
func (db *DB) MarkBackupCodeUsed(codeID int) error {
	now := time.Now().UTC().Format(time.RFC3339)

	result, err := db.Exec(`
		UPDATE backup_codes
		SET used = 1, used_at = ?
		WHERE id = ? AND used = 0
	`, now, codeID)

	if err != nil {
		return fmt.Errorf("mark backup code used: %w", err)
	}

	rowsAffected, err := rowsAffectedCount(result, "")
	if err != nil {
		return fmt.Errorf("check backup code rows: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("code already used or invalid")
	}

	return nil
}

// CountUnusedBackupCodes returns the number of unused backup codes
func (db *DB) CountUnusedBackupCodes(userID int) (int, error) {
	var count int
	if err := db.QueryRow(`
		SELECT COUNT(*)
		FROM backup_codes
		WHERE user_id = ? AND used = 0
	`, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unused backup codes: %w", err)
	}

	return count, nil
}
