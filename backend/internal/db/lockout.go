package db

import (
	"time"
)

// LockoutRecord represents a persisted lockout entry.
type LockoutRecord struct {
	IdentifierHash string
	IP             string
	FailedAttempts int
	LockedUntil    time.Time
	LastAttempt    time.Time
}

const lockoutTimeFormat = time.RFC3339

// UpsertLockout persists or updates a lockout record.
func (db *DB) UpsertLockout(rec LockoutRecord) error {
	_, err := db.Exec(`
		INSERT INTO account_lockouts (identifier_hash, ip, failed_attempts, locked_until, last_attempt)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(identifier_hash, ip) DO UPDATE SET
			failed_attempts = excluded.failed_attempts,
			locked_until = excluded.locked_until,
			last_attempt = excluded.last_attempt
	`, rec.IdentifierHash, rec.IP, rec.FailedAttempts,
		rec.LockedUntil.UTC().Format(lockoutTimeFormat),
		rec.LastAttempt.UTC().Format(lockoutTimeFormat))
	return err
}

// DeleteLockout removes a lockout record for an identifier (all IPs).
func (db *DB) DeleteLockout(identifierHash string) error {
	_, err := db.Exec(`DELETE FROM account_lockouts WHERE identifier_hash = ?`, identifierHash)
	return err
}

// LoadActiveLockouts loads all lockout records that are still relevant
// (locked or had activity in the last 2 hours).
func (db *DB) LoadActiveLockouts() ([]LockoutRecord, error) {
	cutoff := time.Now().UTC().Add(-2 * time.Hour).Format(lockoutTimeFormat)
	rows, err := db.Query(`
		SELECT identifier_hash, ip, failed_attempts, locked_until, last_attempt
		FROM account_lockouts
		WHERE locked_until > ? OR last_attempt > ?
	`, cutoff, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []LockoutRecord
	for rows.Next() {
		var rec LockoutRecord
		var lockedUntilStr, lastAttemptStr string
		if err := rows.Scan(&rec.IdentifierHash, &rec.IP, &rec.FailedAttempts, &lockedUntilStr, &lastAttemptStr); err != nil {
			return nil, err
		}
		if t, err := time.Parse(lockoutTimeFormat, lockedUntilStr); err == nil {
			rec.LockedUntil = t
		}
		if t, err := time.Parse(lockoutTimeFormat, lastAttemptStr); err == nil {
			rec.LastAttempt = t
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// CleanupExpiredLockouts removes lockout records that are fully expired.
func (db *DB) CleanupExpiredLockouts() (int64, error) {
	cutoff := time.Now().UTC().Add(-2 * time.Hour).Format(lockoutTimeFormat)
	result, err := db.Exec(`
		DELETE FROM account_lockouts
		WHERE locked_until < ? AND last_attempt < ?
	`, cutoff, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
