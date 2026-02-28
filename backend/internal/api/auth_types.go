package api

// RegisterRequest represents the request body for user registration
type RegisterRequest struct {
	Username       string `json:"username"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	CaptchaToken   string `json:"captcha_token,omitempty"`
	BootstrapToken string `json:"bootstrap_token,omitempty"`
}

// LoginRequest represents the request body for user login
type LoginRequest struct {
	UsernameOrEmail string `json:"username_or_email"`
	Password        string `json:"password"`
	TOTPCode        string `json:"totp_code,omitempty"`
	BackupCode      string `json:"backup_code,omitempty"`
	CaptchaToken    string `json:"captcha_token,omitempty"`
}

// RefreshRequest represents the request body for token refresh
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RecoverySaltRequest represents the request to get recovery key salt
type RecoverySaltRequest struct {
	Email string `json:"email"`
}

// RecoveryVerifyRequest verifies email + recovery key and issues a short-lived reset token.
type RecoveryVerifyRequest struct {
	Email       string `json:"email"`
	RecoveryKey string `json:"recovery_key"`
}

// RecoveryResetPasswordRequest represents the request to reset password with recovery key
type RecoveryResetPasswordRequest struct {
	Email       string `json:"email"`
	RecoveryKey string `json:"recovery_key"`
	NewPassword string `json:"new_password"`
}

// RecoveryResetPasswordWithTokenRequest finalizes password recovery with a one-time reset token.
type RecoveryResetPasswordWithTokenRequest struct {
	RecoveryResetToken   string            `json:"recovery_reset_token"`
	NewPassword          string            `json:"new_password"`
	ReWrappedNoteDEKs    map[string]string `json:"re_wrapped_note_deks,omitempty"`    // noteID -> wrapped_dek
	ReWrappedVersionDEKs map[string]string `json:"re_wrapped_version_deks,omitempty"` // versionID -> wrapped_dek
}

type recoveryVerifyResponse struct {
	RecoveryResetToken string `json:"recovery_reset_token"`
	EncryptionSalt     string `json:"encryption_salt,omitempty"`
}

type recoveryWrappedDEKResponseItem struct {
	ID                 string `json:"id"`
	WrappedDEKRecovery string `json:"wrapped_dek_recovery"`
}

type recoveryWrappedDEKsResponse struct {
	Notes    []recoveryWrappedDEKResponseItem `json:"notes"`
	Versions []recoveryWrappedDEKResponseItem `json:"versions"`
}

// AuthResponse represents the response for successful authentication
type AuthResponse struct {
	AccessToken       string       `json:"access_token,omitempty"`
	RefreshToken      string       `json:"refresh_token,omitempty"`
	User              UserResponse `json:"user,omitempty"`
	RequiresTwoFactor bool         `json:"requires_two_factor,omitempty"`
	TwoFactorMethods  []string     `json:"two_factor_methods,omitempty"`
	PendingLoginToken string       `json:"pending_login_token,omitempty"`
	EncryptionSalt    string       `json:"encryption_salt,omitempty"` // Base64-encoded salt for E2E encryption
}

// TokenResponse represents the response for token refresh
// SEC-001: omitempty ensures web clients receive empty JSON (tokens only in cookies)
type TokenResponse struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// UserResponse represents user information (without sensitive data)
type UserResponse struct {
	ID             int     `json:"id"`
	Username       string  `json:"username"`
	Email          string  `json:"email"`
	IsAdmin        bool    `json:"is_admin"`
	EncryptionSalt *string `json:"encryption_salt,omitempty"` // Base64-encoded salt for E2E encryption
}
