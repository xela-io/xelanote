package db

import (
	"database/sql"
	"time"
)

// AddWebAuthnCredential adds a new WebAuthn credential for a user
func (db *DB) AddWebAuthnCredential(userID int64, credentialID, deviceName string) (*WebAuthnCredential, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	result, err := db.Exec(`
		INSERT INTO webauthn_credentials (user_id, credential_id, device_name, created_at)
		VALUES (?, ?, ?, ?)
	`, userID, credentialID, deviceName, now)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &WebAuthnCredential{
		ID:           id,
		UserID:       userID,
		CredentialID: credentialID,
		DeviceName:   deviceName,
		CreatedAt:    now,
		LastUsedAt:   nil,
	}, nil
}

// GetWebAuthnCredentials retrieves all WebAuthn credentials for a user
func (db *DB) GetWebAuthnCredentials(userID int64) ([]WebAuthnCredential, error) {
	rows, err := db.Query(`
		SELECT id, user_id, credential_id, device_name, created_at, last_used_at
		FROM webauthn_credentials
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credentials []WebAuthnCredential
	for rows.Next() {
		var c WebAuthnCredential
		var lastUsed sql.NullString

		err := rows.Scan(&c.ID, &c.UserID, &c.CredentialID, &c.DeviceName, &c.CreatedAt, &lastUsed)
		if err != nil {
			return nil, err
		}

		if lastUsed.Valid {
			c.LastUsedAt = &lastUsed.String
		}

		credentials = append(credentials, c)
	}

	return credentials, nil
}

// DeleteWebAuthnCredential deletes a WebAuthn credential
func (db *DB) DeleteWebAuthnCredential(userID int64, credentialID string) error {
	_, err := db.Exec(`
		DELETE FROM webauthn_credentials
		WHERE user_id = ? AND credential_id = ?
	`, userID, credentialID)
	return err
}

// TouchWebAuthnCredential updates the last_used_at timestamp for a credential
func (db *DB) TouchWebAuthnCredential(userID int64, credentialID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		UPDATE webauthn_credentials
		SET last_used_at = ?
		WHERE user_id = ? AND credential_id = ?
	`, now, userID, credentialID)
	return err
}
