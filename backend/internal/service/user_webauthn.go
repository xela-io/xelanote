package service

import "github.com/xela-io/xelanote/internal/db"

// AddWebAuthnCredential adds a new WebAuthn credential for a user
func (s *UserService) AddWebAuthnCredential(userID int64, credentialID, deviceName string) (*db.WebAuthnCredential, error) {
	return s.db.AddWebAuthnCredential(userID, credentialID, deviceName)
}

// GetWebAuthnCredentials retrieves all WebAuthn credentials for a user
func (s *UserService) GetWebAuthnCredentials(userID int64) ([]db.WebAuthnCredential, error) {
	return s.db.GetWebAuthnCredentials(userID)
}

// DeleteWebAuthnCredential deletes a WebAuthn credential
func (s *UserService) DeleteWebAuthnCredential(userID int64, credentialID string) error {
	return s.db.DeleteWebAuthnCredential(userID, credentialID)
}

// TouchWebAuthnCredential updates the last_used_at timestamp for a credential
func (s *UserService) TouchWebAuthnCredential(userID int64, credentialID string) error {
	return s.db.TouchWebAuthnCredential(userID, credentialID)
}
