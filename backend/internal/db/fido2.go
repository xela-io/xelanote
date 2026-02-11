package db

import (
	"database/sql"
	"errors"
	"time"
)

// FIDO2Credential represents a registered FIDO2/WebAuthn credential for 2FA
type FIDO2Credential struct {
	ID              int64
	UserID          int
	CredentialID    []byte
	PublicKey       []byte
	AttestationType string
	AAGUID          []byte
	SignCount       uint32
	DeviceName      string
	Transports      string // JSON array
	CreatedAt       string
	LastUsedAt      *string
}

// AddFIDO2Credential stores a new FIDO2 credential
func (db *DB) AddFIDO2Credential(userID int, cred *FIDO2Credential) error {
	result, err := db.Exec(`
		INSERT INTO fido2_credentials (user_id, credential_id, public_key, attestation_type, aaguid, sign_count, device_name, transports)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, cred.CredentialID, cred.PublicKey, cred.AttestationType, cred.AAGUID, cred.SignCount, cred.DeviceName, cred.Transports)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	cred.ID = id
	cred.UserID = userID
	return nil
}

// GetFIDO2Credentials retrieves all FIDO2 credentials for a user
func (db *DB) GetFIDO2Credentials(userID int) ([]FIDO2Credential, error) {
	rows, err := db.Query(`
		SELECT id, user_id, credential_id, public_key, attestation_type, aaguid, sign_count, device_name, transports, created_at, last_used_at
		FROM fido2_credentials
		WHERE user_id = ?
		ORDER BY created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []FIDO2Credential
	for rows.Next() {
		var c FIDO2Credential
		var lastUsed sql.NullString
		var transports sql.NullString
		var aaguid []byte

		err := rows.Scan(
			&c.ID, &c.UserID, &c.CredentialID, &c.PublicKey,
			&c.AttestationType, &aaguid, &c.SignCount,
			&c.DeviceName, &transports, &c.CreatedAt, &lastUsed,
		)
		if err != nil {
			return nil, err
		}

		c.AAGUID = aaguid
		if transports.Valid {
			c.Transports = transports.String
		}
		if lastUsed.Valid {
			c.LastUsedAt = &lastUsed.String
		}
		creds = append(creds, c)
	}

	return creds, rows.Err()
}

// GetFIDO2CredentialByCredentialID retrieves a single credential by its WebAuthn credential ID
func (db *DB) GetFIDO2CredentialByCredentialID(credentialID []byte) (*FIDO2Credential, error) {
	var c FIDO2Credential
	var lastUsed sql.NullString
	var transports sql.NullString
	var aaguid []byte

	err := db.QueryRow(`
		SELECT id, user_id, credential_id, public_key, attestation_type, aaguid, sign_count, device_name, transports, created_at, last_used_at
		FROM fido2_credentials
		WHERE credential_id = ?
	`, credentialID).Scan(
		&c.ID, &c.UserID, &c.CredentialID, &c.PublicKey,
		&c.AttestationType, &aaguid, &c.SignCount,
		&c.DeviceName, &transports, &c.CreatedAt, &lastUsed,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	c.AAGUID = aaguid
	if transports.Valid {
		c.Transports = transports.String
	}
	if lastUsed.Valid {
		c.LastUsedAt = &lastUsed.String
	}
	return &c, nil
}

// UpdateFIDO2SignCount updates the sign count for a credential
func (db *DB) UpdateFIDO2SignCount(credentialID []byte, newSignCount uint32) error {
	_, err := db.Exec(`
		UPDATE fido2_credentials SET sign_count = ? WHERE credential_id = ?
	`, newSignCount, credentialID)
	return err
}

// TouchFIDO2Credential sets last_used_at to now
func (db *DB) TouchFIDO2Credential(credentialID []byte) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		UPDATE fido2_credentials SET last_used_at = ? WHERE credential_id = ?
	`, now, credentialID)
	return err
}

// DeleteFIDO2Credential removes a specific credential by its DB ID, scoped to user
func (db *DB) DeleteFIDO2Credential(userID int, credID int64) error {
	result, err := db.Exec(`
		DELETE FROM fido2_credentials WHERE id = ? AND user_id = ?
	`, credID, userID)
	if err != nil {
		return err
	}
	return ensureRowsAffected(result)
}

// CountFIDO2Credentials returns the number of FIDO2 credentials for a user
func (db *DB) CountFIDO2Credentials(userID int) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM fido2_credentials WHERE user_id = ?
	`, userID).Scan(&count)
	return count, err
}

// HasFIDO2Credentials returns true if a user has at least one FIDO2 credential
func (db *DB) HasFIDO2Credentials(userID int) (bool, error) {
	count, err := db.CountFIDO2Credentials(userID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeleteAllFIDO2Credentials removes all FIDO2 credentials for a user
func (db *DB) DeleteAllFIDO2Credentials(userID int) error {
	_, err := db.Exec(`
		DELETE FROM fido2_credentials WHERE user_id = ?
	`, userID)
	return err
}
