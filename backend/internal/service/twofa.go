package service

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/xela-io/xelanote/internal/db"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

const (
	// Setup expiry time
	setupExpiryDuration = 15 * time.Minute

	// Number of backup codes to generate
	backupCodeCount = 10

	// Backup code length (Base32)
	backupCodeLength = 8

	// bcrypt cost for backup codes
	backupCodeBcryptCost = 12
)

// TwoFactorService handles 2FA operations
type TwoFactorService struct {
	db     *db.DB
	logger *slog.Logger
}

// NewTwoFactorService creates a new 2FA service
func NewTwoFactorService(database *db.DB, logger *slog.Logger) *TwoFactorService {
	return &TwoFactorService{
		db:     database,
		logger: logger,
	}
}

// TOTPSetupData contains data for TOTP setup
type TOTPSetupData struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

// TwoFactorStatus represents 2FA status for a user
type TwoFactorStatus struct {
	Enabled           bool   `json:"enabled"`
	VerifiedAt        string `json:"verified_at"`
	UnusedBackupCodes int    `json:"unused_backup_codes"`
}

// GenerateTOTPSetup generates TOTP secret, QR code URL, and backup codes
func (s *TwoFactorService) GenerateTOTPSetup(userID int, email string) (*TOTPSetupData, error) {
	// Clear any old setup data first
	if err := s.db.ClearTOTPSetup(userID); err != nil {
		return nil, fmt.Errorf("failed to clear old setup: %w", err)
	}

	// Generate TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "xelanote",
		AccountName: email,
		SecretSize:  32,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	// Store secret with setup timestamp
	if err := s.db.SetTOTPSecret(userID, key.Secret()); err != nil {
		return nil, fmt.Errorf("failed to store TOTP secret: %w", err)
	}

	// Generate backup codes
	backupCodes, err := s.generateBackupCodes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	// Hash and store backup codes
	codeHashes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		hash, err := bcrypt.GenerateFromPassword([]byte(code), backupCodeBcryptCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash backup code: %w", err)
		}
		codeHashes[i] = string(hash)
	}

	if err := s.db.CreateBackupCodes(userID, codeHashes); err != nil {
		return nil, fmt.Errorf("failed to store backup codes: %w", err)
	}

	// Format backup codes for display (XXXX-XXXX)
	formattedCodes := formatBackupCodesForDisplay(backupCodes)

	// SEC-008: Use structured logging for security events
	// SECURITY: NEVER log the secret or QR URL!
	s.logger.Info("TOTP setup generated",
		slog.Int("user_id", userID),
		slog.String("event", "totp_setup_generated"),
		slog.Time("timestamp", time.Now()))

	return &TOTPSetupData{
		Secret:      key.Secret(),
		QRCodeURL:   key.URL(),
		BackupCodes: formattedCodes,
	}, nil
}

// VerifyAndEnableTOTP verifies a TOTP code and enables 2FA
func (s *TwoFactorService) VerifyAndEnableTOTP(userID int, code string) error {
	// Get 2FA status
	tfa, err := s.db.GetTwoFactorAuth(userID)
	if err != nil {
		return fmt.Errorf("failed to get 2FA status: %w", err)
	}

	// Check if setup exists
	if tfa.TOTPSecret == "" {
		return errors.New("no TOTP setup found, please start setup first")
	}

	// Check setup expiry
	if tfa.TOTPSetupStartedAt != "" {
		setupTime, err := time.Parse(time.RFC3339, tfa.TOTPSetupStartedAt)
		if err != nil {
			return fmt.Errorf("failed to parse setup time: %w", err)
		}

		if time.Since(setupTime) > setupExpiryDuration {
			// Cleanup expired setup
			if err := s.db.ClearTOTPSetup(userID); err != nil {
				return fmt.Errorf("failed to clear expired setup: %w", err)
			}
			return errors.New("setup expired, please restart setup process")
		}
	}

	// Check if already enabled
	if tfa.TOTPEnabled {
		return errors.New("2FA is already enabled")
	}

	// Verify TOTP code
	valid := totp.Validate(code, tfa.TOTPSecret)
	if !valid {
		return errors.New("invalid TOTP code")
	}

	// Enable 2FA
	if err := s.db.EnableTwoFactor(userID); err != nil {
		return fmt.Errorf("failed to enable 2FA: %w", err)
	}

	return nil
}

// VerifyTOTP verifies a TOTP code with replay protection
func (s *TwoFactorService) VerifyTOTP(userID int, code string) error {
	tfa, err := s.db.GetTwoFactorAuth(userID)
	if err != nil {
		return fmt.Errorf("failed to get 2FA status: %w", err)
	}

	if !tfa.TOTPEnabled || tfa.TOTPSecret == "" {
		return errors.New("2FA not enabled")
	}

	// Calculate current time step (30-second intervals)
	currentStep := time.Now().Unix() / 30

	// REPLAY POLICY: 60s window (current + previous step)
	// Reject if currentStep <= lastStep+1 (already used in 60s window)
	if tfa.LastTOTPStep > 0 && currentStep <= tfa.LastTOTPStep+1 {
		return errors.New("TOTP code already used in this time window")
	}

	// Validate TOTP
	valid := totp.Validate(code, tfa.TOTPSecret)
	if !valid {
		return errors.New("invalid TOTP code")
	}

	// Atomic "validate then conditional update" pattern
	// Update only if step changed (race protection)
	rowsAffected, err := s.db.UpdateLastTOTPStep(userID, currentStep)
	if err != nil {
		return fmt.Errorf("failed to update TOTP step: %w", err)
	}

	// If no update: parallel request already updated (race detected)
	if rowsAffected == 0 {
		return errors.New("TOTP code already used (race detected)")
	}

	return nil
}

// VerifyBackupCode verifies a backup code with constant-time comparison
func (s *TwoFactorService) VerifyBackupCode(userID int, code string) error {
	codes, err := s.db.GetBackupCodes(userID)
	if err != nil {
		return fmt.Errorf("failed to get backup codes: %w", err)
	}

	if len(codes) == 0 {
		return errors.New("no backup codes available")
	}

	// Normalize input BEFORE bcrypt (prevent timing leak on length)
	code = strings.ToUpper(strings.ReplaceAll(code, "-", ""))
	code = strings.TrimSpace(code)

	var matchedCodeID int
	foundMatch := false

	// Validate exact length (8 characters Base32)
	// If wrong length, still do dummy bcrypt for constant-time
	if len(code) != backupCodeLength {
		// Execute dummy bcrypt for each code to maintain constant timing
		dummyHash := []byte(dummyBcryptHash)
		for range codes {
			bcrypt.CompareHashAndPassword(dummyHash, []byte("DUMMYCODE"))
		}
		return errors.New("invalid backup code")
	}

	// CRITICAL: Loop MUST always run the same number of iterations
	// - No continue/break statements
	// - No early returns
	// - Check ALL hashes (including used=1) for constant-time
	for _, c := range codes {
		// Check hash even if code is already used (constant-time)
		err := bcrypt.CompareHashAndPassword([]byte(c.CodeHash), []byte(code))

		// Only store first match, but DON'T break
		if err == nil && !c.Used && !foundMatch {
			matchedCodeID = c.ID
			foundMatch = true
		}
		// Loop continues to the end!
	}

	// Only after loop: evaluate result
	if foundMatch {
		return s.db.MarkBackupCodeUsed(matchedCodeID)
	}

	return errors.New("invalid backup code")
}

// DisableTwoFactor disables 2FA for a user
func (s *TwoFactorService) DisableTwoFactor(userID int) error {
	return s.db.DisableTwoFactor(userID)
}

// GetTwoFactorStatus returns 2FA status for a user
func (s *TwoFactorService) GetTwoFactorStatus(userID int) (*TwoFactorStatus, error) {
	tfa, err := s.db.GetTwoFactorAuth(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get 2FA status: %w", err)
	}

	unusedCount, err := s.db.CountUnusedBackupCodes(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count backup codes: %w", err)
	}

	return &TwoFactorStatus{
		Enabled:           tfa.TOTPEnabled,
		VerifiedAt:        tfa.TOTPVerifiedAt,
		UnusedBackupCodes: unusedCount,
	}, nil
}

// RegenerateBackupCodes generates new backup codes for a user
func (s *TwoFactorService) RegenerateBackupCodes(userID int) ([]string, error) {
	// Check if 2FA is enabled
	tfa, err := s.db.GetTwoFactorAuth(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get 2FA status: %w", err)
	}

	if !tfa.TOTPEnabled {
		return nil, errors.New("2FA is not enabled")
	}

	// Generate new backup codes
	backupCodes, err := s.generateBackupCodes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	// Hash and store backup codes
	codeHashes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		hash, err := bcrypt.GenerateFromPassword([]byte(code), backupCodeBcryptCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash backup code: %w", err)
		}
		codeHashes[i] = string(hash)
	}

	if err := s.db.CreateBackupCodes(userID, codeHashes); err != nil {
		return nil, fmt.Errorf("failed to store backup codes: %w", err)
	}

	// Format backup codes for display (XXXX-XXXX)
	formattedCodes := formatBackupCodesForDisplay(backupCodes)

	return formattedCodes, nil
}

// RegenerateBackupCodesForFIDO2 generates backup codes for a user who registers their first FIDO2 key.
// Unlike RegenerateBackupCodes, this does not require TOTP to be enabled.
func (s *TwoFactorService) RegenerateBackupCodesForFIDO2(userID int) ([]string, error) {
	backupCodes, err := s.generateBackupCodes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	codeHashes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		hash, err := bcrypt.GenerateFromPassword([]byte(code), backupCodeBcryptCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash backup code: %w", err)
		}
		codeHashes[i] = string(hash)
	}

	if err := s.db.CreateBackupCodes(userID, codeHashes); err != nil {
		return nil, fmt.Errorf("failed to store backup codes: %w", err)
	}

	return formatBackupCodesForDisplay(backupCodes), nil
}

// generateBackupCodes generates random backup codes
func (s *TwoFactorService) generateBackupCodes() ([]string, error) {
	codes := make([]string, backupCodeCount)

	for i := 0; i < backupCodeCount; i++ {
		// Generate 5 random bytes (40 bits) -> 8 Base32 characters
		b := make([]byte, 5)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("failed to generate random bytes: %w", err)
		}

		// Encode to Base32 and take first 8 characters
		code := base32.StdEncoding.EncodeToString(b)[:backupCodeLength]
		codes[i] = code
	}

	return codes, nil
}

func formatBackupCodesForDisplay(backupCodes []string) []string {
	formattedCodes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		formattedCodes[i] = fmt.Sprintf("%s-%s", code[:4], code[4:])
	}
	return formattedCodes
}
