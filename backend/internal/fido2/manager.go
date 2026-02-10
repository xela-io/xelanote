package fido2

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

const (
	// ChallengeTTL is how long a WebAuthn challenge is valid
	ChallengeTTL = 5 * time.Minute

	// PendingLoginTTL is how long a pending login token is valid
	PendingLoginTTL = 5 * time.Minute
)

// Manager wraps the go-webauthn library and provides session management.
type Manager struct {
	wa       *webauthn.WebAuthn
	Sessions SessionStore
}

// NewManager creates a new WebAuthn manager.
func NewManager(rpDisplayName, rpID string, rpOrigins []string) (*Manager, error) {
	cfg := &webauthn.Config{
		RPID:                  rpID,
		RPDisplayName:         rpDisplayName,
		RPOrigins:             rpOrigins,
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationPreferred,
		},
	}

	wa, err := webauthn.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create webauthn: %w", err)
	}

	return &Manager{
		wa:       wa,
		Sessions: NewInMemorySessionStore(),
	}, nil
}

// BeginRegistration starts a WebAuthn registration ceremony.
func (m *Manager) BeginRegistration(user *WebAuthnUser) (*protocol.CredentialCreation, error) {
	// Exclude already-registered credentials to prevent duplicate registration
	excludeList := make([]protocol.CredentialDescriptor, len(user.Credentials))
	for i, cred := range user.Credentials {
		excludeList[i] = protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: cred.ID,
			Transport:    cred.Transport,
		}
	}

	creation, session, err := m.wa.BeginRegistration(
		user,
		webauthn.WithExclusions(excludeList),
	)
	if err != nil {
		return nil, fmt.Errorf("begin registration failed: %w", err)
	}

	key := fmt.Sprintf("register:%d", user.ID)
	if err := m.Sessions.Store(key, session, ChallengeTTL); err != nil {
		return nil, fmt.Errorf("failed to store registration session: %w", err)
	}

	return creation, nil
}

// FinishRegistration completes a WebAuthn registration ceremony.
func (m *Manager) FinishRegistration(user *WebAuthnUser, r *http.Request) (*webauthn.Credential, error) {
	key := fmt.Sprintf("register:%d", user.ID)
	data, err := m.Sessions.Get(key)
	if err != nil {
		return nil, fmt.Errorf("registration session not found or expired: %w", err)
	}

	session, ok := data.(*webauthn.SessionData)
	if !ok {
		return nil, fmt.Errorf("invalid session data type")
	}

	credential, err := m.wa.FinishRegistration(user, *session, r)
	if err != nil {
		return nil, fmt.Errorf("finish registration failed: %w", err)
	}

	return credential, nil
}

// BeginLogin starts a WebAuthn authentication ceremony.
func (m *Manager) BeginLogin(user *WebAuthnUser) (*protocol.CredentialAssertion, error) {
	assertion, session, err := m.wa.BeginLogin(user)
	if err != nil {
		return nil, fmt.Errorf("begin login failed: %w", err)
	}

	key := fmt.Sprintf("login:%d", user.ID)
	if err := m.Sessions.Store(key, session, ChallengeTTL); err != nil {
		return nil, fmt.Errorf("failed to store login session: %w", err)
	}

	return assertion, nil
}

// FinishLogin completes a WebAuthn authentication ceremony.
func (m *Manager) FinishLogin(user *WebAuthnUser, r *http.Request) (*webauthn.Credential, error) {
	key := fmt.Sprintf("login:%d", user.ID)
	data, err := m.Sessions.Get(key)
	if err != nil {
		return nil, fmt.Errorf("login session not found or expired: %w", err)
	}

	session, ok := data.(*webauthn.SessionData)
	if !ok {
		return nil, fmt.Errorf("invalid session data type")
	}

	credential, err := m.wa.FinishLogin(user, *session, r)
	if err != nil {
		return nil, fmt.Errorf("finish login failed: %w", err)
	}

	return credential, nil
}
