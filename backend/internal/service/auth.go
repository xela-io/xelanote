package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/xela-io/xelanote/internal/auth"
	"github.com/xela-io/xelanote/internal/db"
)

const (
	// dummyBcryptHash is used for constant-time comparison
	// Used in Login, LoginWithTwoFactor, and RecoverPasswordWithRecoveryKey
	// to prevent timing attacks when user doesn't exist
	// Generated with: bcrypt.GenerateFromPassword([]byte("DUMMYPASSWORD"), 12)
	dummyBcryptHash = "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5aeKXJ8F3xJPi"

	// Input length limits to prevent memory exhaustion and database issues
	MaxUsernameLength = 100
	MaxEmailLength    = 255
	MaxPasswordLength = 128 // bcrypt truncates at 72 bytes, but allow more for Unicode
)

// AuthService handles user authentication and authorization
type AuthService struct {
	db         *db.DB
	jwtSecret  []byte
	tfaService *TwoFactorService
}

// NewAuthService creates a new AuthService
func NewAuthService(database *db.DB, jwtSecret []byte, tfaService *TwoFactorService) *AuthService {
	return &AuthService{
		db:         database,
		jwtSecret:  jwtSecret,
		tfaService: tfaService,
	}
}

// ErrRegistrationDisabled is returned when registration is disabled
var ErrRegistrationDisabled = errors.New("registration is currently disabled")

// ErrRefreshTokenReuseDetected signals attempted reuse of a rotated/revoked refresh token.
var ErrRefreshTokenReuseDetected = errors.New("refresh token reuse detected")

func validationErr(msg string) *ValidationError {
	return &ValidationError{Message: msg}
}

func validateRegistrationInput(username, email, password string) (string, string, error) {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)

	if username == "" {
		return "", "", validationErr("username is required")
	}
	if len(username) > MaxUsernameLength {
		return "", "", validationErr("username too long")
	}
	if email == "" {
		return "", "", validationErr("email is required")
	}
	if len(email) > MaxEmailLength {
		return "", "", validationErr("email too long")
	}
	if len(password) < 8 {
		return "", "", validationErr("password must be at least 8 characters")
	}
	if len(password) > MaxPasswordLength {
		return "", "", validationErr("password too long")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", "", validationErr("invalid email format")
	}

	return username, email, nil
}

func (s *AuthService) createUser(username, email, password string, forceAdmin bool) (*db.User, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}

	user, err := s.db.CreateUser(username, email, string(passwordHash))
	if err != nil {
		if err == db.ErrDuplicate {
			return nil, validationErr("unable to complete registration")
		}
		return nil, err
	}

	if forceAdmin {
		if err := s.db.SetUserAdmin(user.ID, true); err != nil {
			return nil, err
		}
		user.IsAdmin = true
	}

	return user, nil
}

// Register creates a new user account with password hashing
func (s *AuthService) Register(ctx context.Context, username, email, password string) (*db.User, error) {
	// Check if registration is enabled
	registrationEnabled, err := s.db.IsRegistrationEnabled()
	if err != nil {
		return nil, err
	}
	if !registrationEnabled {
		return nil, ErrRegistrationDisabled
	}

	validUsername, validEmail, err := validateRegistrationInput(username, email, password)
	if err != nil {
		return nil, err
	}

	// Check if this will be the first user (should become admin)
	userCount, err := s.db.CountUsers()
	if err != nil {
		return nil, err
	}
	isFirstUser := userCount == 0

	return s.createUser(validUsername, validEmail, password, isFirstUser)
}

// BootstrapAdmin creates the first admin account even when registration is disabled.
// It is only allowed if the instance has no users yet.
func (s *AuthService) BootstrapAdmin(ctx context.Context, username, email, password string) (*db.User, error) {
	validUsername, validEmail, err := validateRegistrationInput(username, email, password)
	if err != nil {
		return nil, err
	}

	userCount, err := s.db.CountUsers()
	if err != nil {
		return nil, err
	}
	if userCount > 0 {
		return nil, errors.New("bootstrap is only allowed on fresh instances")
	}
	return s.createUser(validUsername, validEmail, password, true)
}

// Login authenticates a user and returns JWT tokens.
// If 2FA is enabled (TOTP or FIDO2), returns requiresTwoFactor=true, methods list, and no tokens.
func (s *AuthService) Login(ctx context.Context, usernameOrEmail, password string) (accessToken, refreshToken string, requiresTwoFactor bool, methods []string, err error) {
	// Get user by username or email
	user, err := s.db.GetUserByUsernameOrEmail(usernameOrEmail)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// Constant-time: run bcrypt even for non-existent users to prevent timing attacks
			bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
			return "", "", false, nil, errors.New("invalid credentials")
		}
		return "", "", false, nil, err
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		// Always return generic error to prevent username enumeration
		return "", "", false, nil, errors.New("invalid credentials")
	}

	// Check if any 2FA method is enabled
	tfa, tfaErr := s.db.GetTwoFactorAuth(user.ID)
	totpEnabled := tfaErr == nil && tfa.TOTPEnabled
	hasFido2, _ := s.db.HasFIDO2Credentials(user.ID)

	if totpEnabled || hasFido2 {
		methods = []string{}
		if hasFido2 {
			methods = append(methods, "fido2")
		}
		if totpEnabled {
			methods = append(methods, "totp")
		}
		methods = append(methods, "backup_code")
		return "", "", true, methods, nil
	}

	// Generate access token (JWT, 15 minutes)
	accessToken, err = auth.GenerateAccessToken(user.ID, user.Username, s.jwtSecret)
	if err != nil {
		return "", "", false, nil, err
	}

	// Generate refresh token (random string, 30 days)
	refreshToken, err = auth.GenerateRefreshToken()
	if err != nil {
		return "", "", false, nil, err
	}

	// Store refresh token in database
	err = s.db.CreateRefreshToken(user.ID, refreshToken)
	if err != nil {
		return "", "", false, nil, err
	}

	return accessToken, refreshToken, false, nil, nil
}

// RefreshAccessToken issues a new access token using a valid refresh token
// Implements refresh token rotation for enhanced security
func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (newAccessToken, newRefreshToken string, err error) {
	// Validate refresh token in database
	userID, err := s.db.ValidateRefreshToken(refreshToken)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return "", "", errors.New("invalid refresh token")
		}
		if err == db.ErrRefreshTokenReuse {
			_ = s.db.RevokeRefreshTokenFamilyByToken(refreshToken)
			return "", "", ErrRefreshTokenReuseDetected
		}
		return "", "", err
	}

	// Get user details
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return "", "", err
	}

	// Generate new access token
	newAccessToken, err = auth.GenerateAccessToken(user.ID, user.Username, s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	// Rotate refresh token (security best practice)
	// Old token is deleted, new token is created atomically
	newRefreshToken, err = auth.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}

	// Atomic rotation: delete old, insert new
	err = s.db.RotateRefreshToken(refreshToken, userID, newRefreshToken)
	if err != nil {
		if err == db.ErrRefreshTokenReuse {
			_ = s.db.RevokeRefreshTokenFamilyByToken(refreshToken)
			return "", "", ErrRefreshTokenReuseDetected
		}
		return "", "", err
	}

	return newAccessToken, newRefreshToken, nil
}

// Logout revokes a refresh token (removes it from database)
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.db.DeleteRefreshToken(refreshToken)
}

// GetUserByID retrieves user information by ID
func (s *AuthService) GetUserByID(userID int) (*db.User, error) {
	return s.db.GetUserByID(userID)
}

// GetUserByUsernameOrEmail retrieves user information by username or email
func (s *AuthService) GetUserByUsernameOrEmail(usernameOrEmail string) (*db.User, error) {
	return s.db.GetUserByUsernameOrEmail(usernameOrEmail)
}

// LoginWithTwoFactor authenticates a user with 2FA and returns JWT tokens
func (s *AuthService) LoginWithTwoFactor(ctx context.Context, usernameOrEmail, password, totpCode, backupCode string) (accessToken, refreshToken string, err error) {
	// Get user by username or email
	user, err := s.db.GetUserByUsernameOrEmail(usernameOrEmail)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// Constant-time: run bcrypt even for non-existent users to prevent timing attacks
			bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
			return "", "", errors.New("invalid credentials")
		}
		return "", "", err
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		// Always return generic error to prevent username enumeration
		return "", "", errors.New("invalid credentials")
	}

	// Check if 2FA is enabled
	tfa, err := s.db.GetTwoFactorAuth(user.ID)
	if err != nil || !tfa.TOTPEnabled {
		return "", "", errors.New("2FA not enabled for this account")
	}

	// Verify TOTP code or backup code
	if totpCode != "" {
		// Verify TOTP code (with replay protection)
		err = s.tfaService.VerifyTOTP(user.ID, totpCode)
		if err != nil {
			return "", "", errors.New("invalid 2FA code")
		}
	} else if backupCode != "" {
		// Verify backup code (constant-time)
		err = s.tfaService.VerifyBackupCode(user.ID, backupCode)
		if err != nil {
			return "", "", errors.New("invalid backup code")
		}
	} else {
		return "", "", errors.New("2FA code or backup code required")
	}

	// Generate access token (JWT, 15 minutes)
	accessToken, err = auth.GenerateAccessToken(user.ID, user.Username, s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	// Generate refresh token (random string, 30 days)
	refreshToken, err = auth.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}

	// Store refresh token in database
	err = s.db.CreateRefreshToken(user.ID, refreshToken)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// IssueTokens generates access and refresh tokens for a user.
// Used by both TOTP and FIDO2 login flows after 2FA verification.
func (s *AuthService) IssueTokens(ctx context.Context, userID int) (accessToken, refreshToken string, err error) {
	user, err := s.db.GetUserByID(userID)
	if err != nil {
		return "", "", err
	}

	accessToken, err = auth.GenerateAccessToken(user.ID, user.Username, s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = auth.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}

	if err = s.db.CreateRefreshToken(user.ID, refreshToken); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// GetUserEncryptionSalt retrieves a user's encryption salt
func (s *AuthService) GetUserEncryptionSalt(userID int) ([]byte, error) {
	return s.db.GetUserEncryptionSalt(userID)
}

// SetUserEncryptionSalt stores a user's encryption salt
func (s *AuthService) SetUserEncryptionSalt(userID int, salt []byte) error {
	return s.db.SetUserEncryptionSalt(userID, salt)
}
