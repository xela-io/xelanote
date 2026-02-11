package api

import "github.com/xela-io/xelanote/internal/db"

// Request/Response types for user preferences

type webAuthnCredentialInfo struct {
	ID           int64   `json:"id"`
	CredentialID string  `json:"credential_id"`
	DeviceName   string  `json:"device_name"`
	CreatedAt    string  `json:"created_at"`
	LastUsedAt   *string `json:"last_used_at,omitempty"`
}

type preferencesResponse struct {
	Theme           string                   `json:"theme"`
	EditorMode      string                   `json:"editor_mode"`
	KeywordsEnabled bool                     `json:"keywords_enabled"`
	EncryptTitles   bool                     `json:"encrypt_titles"`
	SecurityLevel   string                   `json:"security_level"`
	AutoLockTimeout int                      `json:"auto_lock_timeout"`
	Credentials     []webAuthnCredentialInfo `json:"webauthn_credentials"`
	Created         bool                     `json:"created"`
}

type updatePreferencesRequest struct {
	Theme      string `json:"theme"`
	EditorMode string `json:"editor_mode"`
}

type updateEncryptionPreferencesRequest struct {
	KeywordsEnabled bool `json:"keywords_enabled"`
	EncryptTitles   bool `json:"encrypt_titles"`
}

type updateSecurityPreferencesRequest struct {
	SecurityLevel   *string `json:"security_level"`
	AutoLockTimeout *int    `json:"auto_lock_timeout"`
}

type addWebAuthnCredentialRequest struct {
	CredentialID string `json:"credential_id"`
	DeviceName   string `json:"device_name"`
}

type setRecoveryKeyRequest struct {
	RecoveryKeyHash string `json:"recovery_key_hash"` // bcrypt hash
	Salt            string `json:"salt"`              // Base64-encoded
}

type changeEmailRequest struct {
	NewEmail        string `json:"new_email"`
	CurrentPassword string `json:"current_password"`
}

type changePasswordRequest struct {
	CurrentPassword      string            `json:"current_password"`
	NewPassword          string            `json:"new_password"`
	ReWrappedNoteDEKs    map[string]string `json:"re_wrapped_note_deks,omitempty"`    // noteID -> wrapped_dek (optional)
	ReWrappedVersionDEKs map[string]string `json:"re_wrapped_version_deks,omitempty"` // versionID -> wrapped_dek (optional)
}

// apiKeyProvider configures generic API key handlers for a specific LLM provider.
type apiKeyProvider struct {
	name            string
	setKey          func(int, string) error
	deleteKey       func(int) error
	getKeyStatus    func(int) (any, error)
	invalidateCache func(int)
	validationErr   error
	invalidKeyMsg   string
}

// convertWebAuthnCredentials converts db.WebAuthnCredential slice to webAuthnCredentialInfo slice
func convertWebAuthnCredentials(creds []db.WebAuthnCredential) []webAuthnCredentialInfo {
	result := make([]webAuthnCredentialInfo, 0, len(creds))
	for _, c := range creds {
		result = append(result, webAuthnCredentialInfo{
			ID:           c.ID,
			CredentialID: c.CredentialID,
			DeviceName:   c.DeviceName,
			CreatedAt:    c.CreatedAt,
			LastUsedAt:   c.LastUsedAt,
		})
	}
	return result
}
