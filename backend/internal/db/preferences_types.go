package db

// UserPreferences represents user settings stored in the database
type UserPreferences struct {
	ID               int     `json:"id"`
	UserID           int     `json:"user_id"`
	Theme            string  `json:"theme"`
	EditorMode       string  `json:"editor_mode"`
	KeywordsEnabled  bool    `json:"keywords_enabled"`
	EncryptTitles    bool    `json:"encrypt_titles"`
	SecurityLevel    string  `json:"security_level"`    // NEW: paranoid | balanced | convenient
	AutoLockTimeout  int     `json:"auto_lock_timeout"` // NEW: minutes (0 = never)
	ActiveAIProvider string  `json:"active_ai_provider"`
	RecoveryKeyHash  *string `json:"-"` // Not exposed in JSON for security
	RecoveryKeySalt  []byte  `json:"-"` // Not exposed in JSON for security
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// WebAuthnCredential represents a WebAuthn credential (imported from service package)
type WebAuthnCredential struct {
	ID           int64
	UserID       int64
	CredentialID string
	DeviceName   string
	CreatedAt    string
	LastUsedAt   *string
}
