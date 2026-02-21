package service

import (
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/xela-io/xelanote/internal/db"
)

func setupTwoFactorTest(t *testing.T) (*db.DB, *TwoFactorService, *db.User) {
	t.Helper()

	database, err := db.Open(":memory:", "")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := NewTwoFactorService(database, logger)

	user, err := database.CreateUser("testuser", "test@example.com", "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5aeKXJ8F3xJPi")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return database, svc, user
}

func TestTwoFactorService_GenerateTOTPSetup(t *testing.T) {
	_, svc, user := setupTwoFactorTest(t)

	t.Run("generates setup data successfully", func(t *testing.T) {
		setup, err := svc.GenerateTOTPSetup(user.ID, user.Email)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if setup.Secret == "" {
			t.Error("expected non-empty secret")
		}
		if setup.QRCodeURL == "" {
			t.Error("expected non-empty QR code URL")
		}
		if !strings.Contains(setup.QRCodeURL, "xelanote") {
			t.Errorf("QR URL should contain issuer 'xelanote', got: %s", setup.QRCodeURL)
		}
		if len(setup.BackupCodes) != backupCodeCount {
			t.Errorf("expected %d backup codes, got %d", backupCodeCount, len(setup.BackupCodes))
		}
		// Verify backup code format: XXXX-XXXX
		for _, code := range setup.BackupCodes {
			if len(code) != 9 || code[4] != '-' {
				t.Errorf("backup code should be XXXX-XXXX format, got: %s", code)
			}
		}
	})

	t.Run("clears previous setup on regeneration", func(t *testing.T) {
		setup1, err := svc.GenerateTOTPSetup(user.ID, user.Email)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		setup2, err := svc.GenerateTOTPSetup(user.ID, user.Email)
		if err != nil {
			t.Fatalf("unexpected error on re-setup: %v", err)
		}
		if setup1.Secret == setup2.Secret {
			t.Error("expected different secrets on re-setup")
		}
	})
}

func TestTwoFactorService_VerifyAndEnableTOTP(t *testing.T) {
	_, svc, user := setupTwoFactorTest(t)

	t.Run("rejects without setup", func(t *testing.T) {
		err := svc.VerifyAndEnableTOTP(user.ID, "123456")
		if err == nil {
			t.Fatal("expected error without setup")
		}
		if !strings.Contains(err.Error(), "no TOTP setup found") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("rejects invalid code", func(t *testing.T) {
		_, err := svc.GenerateTOTPSetup(user.ID, user.Email)
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		err = svc.VerifyAndEnableTOTP(user.ID, "000000")
		if err == nil {
			t.Fatal("expected error for invalid code")
		}
		if !strings.Contains(err.Error(), "invalid TOTP code") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("enables with valid code", func(t *testing.T) {
		setup, err := svc.GenerateTOTPSetup(user.ID, user.Email)
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		code, err := totp.GenerateCode(setup.Secret, time.Now())
		if err != nil {
			t.Fatalf("failed to generate TOTP code: %v", err)
		}

		err = svc.VerifyAndEnableTOTP(user.ID, code)
		if err != nil {
			t.Fatalf("unexpected error enabling TOTP: %v", err)
		}

		// Verify status
		status, err := svc.GetTwoFactorStatus(user.ID)
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		if !status.Enabled {
			t.Error("expected 2FA to be enabled")
		}
		if status.UnusedBackupCodes != backupCodeCount {
			t.Errorf("expected %d unused backup codes, got %d", backupCodeCount, status.UnusedBackupCodes)
		}
	})

	t.Run("rejects double enable", func(t *testing.T) {
		err := svc.VerifyAndEnableTOTP(user.ID, "123456")
		if err == nil {
			t.Fatal("expected error for double enable")
		}
		if !strings.Contains(err.Error(), "already enabled") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestTwoFactorService_VerifyTOTP(t *testing.T) {
	_, svc, user := setupTwoFactorTest(t)

	// Setup and enable TOTP
	setup, err := svc.GenerateTOTPSetup(user.ID, user.Email)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}
	if err := svc.VerifyAndEnableTOTP(user.ID, code); err != nil {
		t.Fatalf("enable failed: %v", err)
	}

	t.Run("rejects invalid code", func(t *testing.T) {
		err := svc.VerifyTOTP(user.ID, "000000")
		if err == nil {
			t.Fatal("expected error for invalid code")
		}
		if !strings.Contains(err.Error(), "invalid TOTP code") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestTwoFactorService_VerifyBackupCode(t *testing.T) {
	_, svc, user := setupTwoFactorTest(t)

	// Setup and enable TOTP
	setup, err := svc.GenerateTOTPSetup(user.ID, user.Email)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}
	if err := svc.VerifyAndEnableTOTP(user.ID, code); err != nil {
		t.Fatalf("enable failed: %v", err)
	}

	t.Run("accepts valid backup code", func(t *testing.T) {
		// Use the first backup code (formatted as XXXX-XXXX)
		err := svc.VerifyBackupCode(user.ID, setup.BackupCodes[0])
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects already used backup code", func(t *testing.T) {
		err := svc.VerifyBackupCode(user.ID, setup.BackupCodes[0])
		if err == nil {
			t.Fatal("expected error for reused backup code")
		}
		if !strings.Contains(err.Error(), "invalid backup code") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("rejects invalid backup code", func(t *testing.T) {
		err := svc.VerifyBackupCode(user.ID, "ZZZZ-ZZZZ")
		if err == nil {
			t.Fatal("expected error for invalid code")
		}
	})

	t.Run("rejects wrong length code", func(t *testing.T) {
		err := svc.VerifyBackupCode(user.ID, "SHORT")
		if err == nil {
			t.Fatal("expected error for short code")
		}
		if !strings.Contains(err.Error(), "invalid backup code") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("handles hyphenated input", func(t *testing.T) {
		// Use second backup code with hyphen (standard format)
		err := svc.VerifyBackupCode(user.ID, setup.BackupCodes[1])
		if err != nil {
			t.Fatalf("unexpected error with hyphenated code: %v", err)
		}
	})

	t.Run("handles lowercase input", func(t *testing.T) {
		err := svc.VerifyBackupCode(user.ID, strings.ToLower(setup.BackupCodes[2]))
		if err != nil {
			t.Fatalf("unexpected error with lowercase code: %v", err)
		}
	})
}

func TestTwoFactorService_DisableTwoFactor(t *testing.T) {
	_, svc, user := setupTwoFactorTest(t)

	// Setup and enable TOTP
	setup, err := svc.GenerateTOTPSetup(user.ID, user.Email)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}
	if err := svc.VerifyAndEnableTOTP(user.ID, code); err != nil {
		t.Fatalf("enable failed: %v", err)
	}

	t.Run("disables 2FA successfully", func(t *testing.T) {
		err := svc.DisableTwoFactor(user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		status, err := svc.GetTwoFactorStatus(user.ID)
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		if status.Enabled {
			t.Error("expected 2FA to be disabled")
		}
		if status.UnusedBackupCodes != 0 {
			t.Errorf("expected 0 backup codes after disable, got %d", status.UnusedBackupCodes)
		}
	})
}

func TestTwoFactorService_GetTwoFactorStatus(t *testing.T) {
	_, svc, user := setupTwoFactorTest(t)

	t.Run("returns disabled status for new user", func(t *testing.T) {
		status, err := svc.GetTwoFactorStatus(user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status.Enabled {
			t.Error("expected 2FA to be disabled for new user")
		}
		if status.UnusedBackupCodes != 0 {
			t.Errorf("expected 0 backup codes, got %d", status.UnusedBackupCodes)
		}
	})
}

func TestTwoFactorService_RegenerateBackupCodes(t *testing.T) {
	_, svc, user := setupTwoFactorTest(t)

	t.Run("rejects without 2FA enabled", func(t *testing.T) {
		_, err := svc.RegenerateBackupCodes(user.ID)
		if err == nil {
			t.Fatal("expected error without 2FA")
		}
		if !strings.Contains(err.Error(), "not enabled") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("generates new codes with 2FA enabled", func(t *testing.T) {
		// Enable 2FA first
		setup, err := svc.GenerateTOTPSetup(user.ID, user.Email)
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		code, err := totp.GenerateCode(setup.Secret, time.Now())
		if err != nil {
			t.Fatalf("failed to generate code: %v", err)
		}
		if err := svc.VerifyAndEnableTOTP(user.ID, code); err != nil {
			t.Fatalf("enable failed: %v", err)
		}

		// Regenerate
		newCodes, err := svc.RegenerateBackupCodes(user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(newCodes) != backupCodeCount {
			t.Errorf("expected %d codes, got %d", backupCodeCount, len(newCodes))
		}

		// Old codes should be invalid now
		err = svc.VerifyBackupCode(user.ID, setup.BackupCodes[0])
		if err == nil {
			t.Error("expected old backup code to be invalid after regeneration")
		}

		// New codes should work
		err = svc.VerifyBackupCode(user.ID, newCodes[0])
		if err != nil {
			t.Fatalf("new backup code should be valid: %v", err)
		}
	})
}

func TestTwoFactorService_RegenerateBackupCodesForFIDO2(t *testing.T) {
	_, svc, user := setupTwoFactorTest(t)

	t.Run("generates codes without TOTP requirement", func(t *testing.T) {
		codes, err := svc.RegenerateBackupCodesForFIDO2(user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(codes) != backupCodeCount {
			t.Errorf("expected %d codes, got %d", backupCodeCount, len(codes))
		}
		for _, c := range codes {
			if len(c) != 9 || c[4] != '-' {
				t.Errorf("backup code should be XXXX-XXXX format, got: %s", c)
			}
		}
	})
}
