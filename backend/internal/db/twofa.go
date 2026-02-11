package db

import (
	"database/sql"
	"errors"
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
		return nil, err
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

	_, err := db.Exec(`
		UPDATE users
		SET totp_secret = ?,
		    totp_setup_started_at = ?,
		    updated_at = ?
		WHERE id = ?
	`, secret, now, now, userID)

	return err
}

// ClearTOTPSetup removes expired/old TOTP setup data
func (db *DB) ClearTOTPSetup(userID int) error {
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(`
		UPDATE users
		SET totp_secret = NULL,
		    totp_setup_started_at = NULL,
		    updated_at = ?
		WHERE id = ?
	`, now, userID)

	return err
}

// EnableTwoFactor activates 2FA after successful verification
func (db *DB) EnableTwoFactor(userID int) error {
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(`
		UPDATE users
		SET totp_enabled = 1,
		    totp_verified_at = ?,
		    totp_setup_started_at = NULL,
		    updated_at = ?
		WHERE id = ?
	`, now, now, userID)

	return err
}

// DisableTwoFactor deactivates 2FA and removes all related data
func (db *DB) DisableTwoFactor(userID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
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
		return err
	}

	// Delete all backup codes
	_, err = tx.Exec(`DELETE FROM backup_codes WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateLastTOTPStep stores the last used TOTP time step for replay protection
// Returns number of rows affected (0 if step was already used)
func (db *DB) UpdateLastTOTPStep(userID int, step int64) (int64, error) {
	result, err := db.Exec(`
		UPDATE users
		SET last_totp_step = ?
		WHERE id = ? AND (last_totp_step IS NULL OR last_totp_step < ?)
	`, step, userID, step)

	if err != nil {
		return 0, err
	}
	return rowsAffectedCount(result, "")
}

// CreateBackupCodes stores hashed backup codes
func (db *DB) CreateBackupCodes(userID int, codeHashes []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete old backup codes
	_, err = tx.Exec(`DELETE FROM backup_codes WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	// Insert new codes
	stmt, err := tx.Prepare(`
		INSERT INTO backup_codes (user_id, code_hash)
		VALUES (?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, hash := range codeHashes {
		_, err = stmt.Exec(userID, hash)
		if err != nil {
			return err
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
		return nil, err
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
			return nil, err
		}

		code.UsedAt = usedAt.String
		codes = append(codes, code)
	}

	return codes, rows.Err()
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
		return err
	}

	rowsAffected, err := rowsAffectedCount(result, "")
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("code already used or invalid")
	}

	return nil
}

// CountUnusedBackupCodes returns the number of unused backup codes
func (db *DB) CountUnusedBackupCodes(userID int) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM backup_codes
		WHERE user_id = ? AND used = 0
	`, userID).Scan(&count)

	return count, err
}
