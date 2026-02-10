package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/fido2"
)

// FIDO2Service handles FIDO2/WebAuthn 2FA operations.
type FIDO2Service struct {
	db      *db.DB
	manager *fido2.Manager
	tfa     *TwoFactorService
	logger  *slog.Logger
}

// RegistrationResult is returned by FinishRegistration.
type RegistrationResult struct {
	CredentialID int64
	BackupCodes  []string // Only set when first key registered and no codes exist
}

// FIDO2CredentialInfo is the public representation of a FIDO2 credential.
type FIDO2CredentialInfo struct {
	ID         int64    `json:"id"`
	DeviceName string   `json:"device_name"`
	CreatedAt  string   `json:"created_at"`
	LastUsedAt *string  `json:"last_used_at,omitempty"`
	Transports []string `json:"transports,omitempty"`
}

// NewFIDO2Service creates a new FIDO2 service.
func NewFIDO2Service(database *db.DB, manager *fido2.Manager, tfa *TwoFactorService, logger *slog.Logger) *FIDO2Service {
	return &FIDO2Service{
		db:      database,
		manager: manager,
		tfa:     tfa,
		logger:  logger,
	}
}

// BeginRegistration starts a FIDO2 credential registration.
func (s *FIDO2Service) BeginRegistration(userID int, username string) (*protocol.CredentialCreation, error) {
	existing, err := s.db.GetFIDO2Credentials(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing credentials: %w", err)
	}

	user := s.buildWebAuthnUser(userID, username, existing)
	creation, err := s.manager.BeginRegistration(user)
	if err != nil {
		return nil, err
	}

	return creation, nil
}

// FinishRegistration completes a FIDO2 credential registration.
func (s *FIDO2Service) FinishRegistration(userID int, username, deviceName string, r *http.Request) (*RegistrationResult, error) {
	existing, err := s.db.GetFIDO2Credentials(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing credentials: %w", err)
	}

	user := s.buildWebAuthnUser(userID, username, existing)
	credential, err := s.manager.FinishRegistration(user, r)
	if err != nil {
		return nil, err
	}

	// Serialize transports to JSON
	var transportsJSON string
	if len(credential.Transport) > 0 {
		transports := make([]string, len(credential.Transport))
		for i, t := range credential.Transport {
			transports[i] = string(t)
		}
		b, _ := json.Marshal(transports)
		transportsJSON = string(b)
	}

	dbCred := &db.FIDO2Credential{
		CredentialID:    credential.ID,
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		AAGUID:          credential.Authenticator.AAGUID,
		SignCount:       credential.Authenticator.SignCount,
		DeviceName:      deviceName,
		Transports:      transportsJSON,
	}

	if err := s.db.AddFIDO2Credential(userID, dbCred); err != nil {
		return nil, fmt.Errorf("failed to store credential: %w", err)
	}

	s.logger.Info("fido2_credential_registered",
		slog.Int("user_id", userID),
		slog.String("device_name", deviceName),
		slog.String("event", "fido2_registration"))

	result := &RegistrationResult{
		CredentialID: dbCred.ID,
	}

	// If this is the first FIDO2 key and no backup codes exist, generate them
	if len(existing) == 0 {
		unusedCodes, err := s.db.CountUnusedBackupCodes(userID)
		if err == nil && unusedCodes == 0 {
			codes, err := s.tfa.RegenerateBackupCodesForFIDO2(userID)
			if err != nil {
				s.logger.Warn("failed to generate backup codes for first FIDO2 key",
					slog.Int("user_id", userID),
					slog.Any("error", err))
			} else {
				result.BackupCodes = codes
			}
		}
	}

	return result, nil
}

// BeginAuthentication starts a FIDO2 authentication ceremony.
func (s *FIDO2Service) BeginAuthentication(userID int, username string) (*protocol.CredentialAssertion, error) {
	existing, err := s.db.GetFIDO2Credentials(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	if len(existing) == 0 {
		return nil, errors.New("no FIDO2 credentials registered")
	}

	user := s.buildWebAuthnUser(userID, username, existing)
	assertion, err := s.manager.BeginLogin(user)
	if err != nil {
		return nil, err
	}

	return assertion, nil
}

// FinishAuthentication completes a FIDO2 authentication ceremony.
func (s *FIDO2Service) FinishAuthentication(userID int, username string, r *http.Request) error {
	existing, err := s.db.GetFIDO2Credentials(userID)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	user := s.buildWebAuthnUser(userID, username, existing)
	credential, err := s.manager.FinishLogin(user, r)
	if err != nil {
		return err
	}

	// Sign count validation
	var storedCred *db.FIDO2Credential
	for i := range existing {
		if byteSliceEqual(existing[i].CredentialID, credential.ID) {
			storedCred = &existing[i]
			break
		}
	}

	if storedCred != nil {
		newCount := credential.Authenticator.SignCount
		oldCount := storedCred.SignCount

		if oldCount == 0 {
			// First use: accept any value
		} else if newCount == 0 && oldCount > 0 {
			// Some platforms reset counter - warn but accept
			s.logger.Warn("fido2_sign_count_reset",
				slog.Int("user_id", userID),
				slog.Uint64("old_count", uint64(oldCount)),
				slog.String("event", "sign_count_reset"))
		} else if newCount <= oldCount && newCount != 0 {
			// Potential cloning - reject
			s.logger.Error("fido2_possible_clone",
				slog.Int("user_id", userID),
				slog.Uint64("old_count", uint64(oldCount)),
				slog.Uint64("new_count", uint64(newCount)),
				slog.String("event", "possible_credential_clone"))
			return errors.New("security key may have been cloned")
		}

		// Update sign count and last used
		if err := s.db.UpdateFIDO2SignCount(credential.ID, newCount); err != nil {
			s.logger.Error("failed to update sign count", slog.Any("error", err))
		}
		if err := s.db.TouchFIDO2Credential(credential.ID); err != nil {
			s.logger.Error("failed to touch credential", slog.Any("error", err))
		}
	}

	return nil
}

// ListCredentials returns all FIDO2 credentials for a user.
func (s *FIDO2Service) ListCredentials(userID int) ([]FIDO2CredentialInfo, error) {
	creds, err := s.db.GetFIDO2Credentials(userID)
	if err != nil {
		return nil, err
	}

	result := make([]FIDO2CredentialInfo, len(creds))
	for i, c := range creds {
		info := FIDO2CredentialInfo{
			ID:         c.ID,
			DeviceName: c.DeviceName,
			CreatedAt:  c.CreatedAt,
			LastUsedAt: c.LastUsedAt,
		}

		if c.Transports != "" {
			var transports []string
			if err := json.Unmarshal([]byte(c.Transports), &transports); err == nil {
				info.Transports = transports
			}
		}

		result[i] = info
	}

	return result, nil
}

// DeleteCredential removes a FIDO2 credential.
func (s *FIDO2Service) DeleteCredential(userID int, credID int64) error {
	if err := s.db.DeleteFIDO2Credential(userID, credID); err != nil {
		return err
	}

	s.logger.Info("fido2_credential_deleted",
		slog.Int("user_id", userID),
		slog.Int64("credential_id", credID),
		slog.String("event", "fido2_deletion"))

	return nil
}

// buildWebAuthnUser constructs a WebAuthnUser from DB credentials.
func (s *FIDO2Service) buildWebAuthnUser(userID int, username string, dbCreds []db.FIDO2Credential) *fido2.WebAuthnUser {
	creds := make([]webauthn.Credential, len(dbCreds))
	for i, c := range dbCreds {
		var transports []protocol.AuthenticatorTransport
		if c.Transports != "" {
			var ts []string
			if err := json.Unmarshal([]byte(c.Transports), &ts); err == nil {
				for _, t := range ts {
					transports = append(transports, protocol.AuthenticatorTransport(t))
				}
			}
		}

		creds[i] = webauthn.Credential{
			ID:              c.CredentialID,
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Transport:       transports,
			Authenticator: webauthn.Authenticator{
				AAGUID:    c.AAGUID,
				SignCount: c.SignCount,
			},
		}
	}

	return &fido2.WebAuthnUser{
		ID:          userID,
		Name:        username,
		DisplayName: username,
		Credentials: creds,
	}
}

// StorePendingLogin stores a pending login state in the session store.
func (s *FIDO2Service) StorePendingLogin(token string, pending any) error {
	return s.manager.Sessions.Store("pending:"+token, pending, fido2.PendingLoginTTL)
}

// GetPendingLogin retrieves and removes a pending login from the session store.
func (s *FIDO2Service) GetPendingLogin(token string) (any, error) {
	return s.manager.Sessions.Get("pending:" + token)
}

func byteSliceEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
