package fido2

import (
	"encoding/binary"

	"github.com/go-webauthn/webauthn/webauthn"
)

// WebAuthnUser implements the webauthn.User interface for FIDO2 2FA.
type WebAuthnUser struct {
	ID          int
	Name        string
	DisplayName string
	Credentials []webauthn.Credential
}

// WebAuthnID returns the user's ID as bytes (required by webauthn.User interface).
func (u *WebAuthnUser) WebAuthnID() []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(u.ID)) //nolint:gosec // user IDs are always positive
	return buf
}

// WebAuthnName returns the user's login name.
func (u *WebAuthnUser) WebAuthnName() string {
	return u.Name
}

// WebAuthnDisplayName returns the user's display name.
func (u *WebAuthnUser) WebAuthnDisplayName() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Name
}

// WebAuthnCredentials returns the user's registered credentials.
func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}
