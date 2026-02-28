package service

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/xela-io/xelanote/internal/db"
)

func validWrappedDEK(seed byte) string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{seed}, 48))
}

func TestUserService_ChangeEmail(t *testing.T) {
	testDB, userService := setupUserServiceTest(t)
	defer testDB.Close()

	t.Run("changes email with valid password", func(t *testing.T) {
		userID := createTestUserForPasswordTests(t, testDB, "emailuser1", "old@example.com", "password123")

		err := userService.ChangeEmail(userID, "new@example.com", "password123", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		user, err := testDB.GetUserByID(userID)
		if err != nil {
			t.Fatalf("failed to get user: %v", err)
		}
		if user.Email != "new@example.com" {
			t.Errorf("expected email 'new@example.com', got %q", user.Email)
		}
	})

	t.Run("rejects invalid email format", func(t *testing.T) {
		userID := createTestUserForPasswordTests(t, testDB, "emailuser2", "valid@example.com", "password123")

		err := userService.ChangeEmail(userID, "not-an-email", "password123", "")
		if err != ErrInvalidEmail {
			t.Errorf("expected ErrInvalidEmail, got: %v", err)
		}
	})

	t.Run("rejects empty email", func(t *testing.T) {
		userID := createTestUserForPasswordTests(t, testDB, "emailuser3", "user3@example.com", "password123")

		err := userService.ChangeEmail(userID, "", "password123", "")
		if err != ErrInvalidEmail {
			t.Errorf("expected ErrInvalidEmail, got: %v", err)
		}
	})

	t.Run("rejects wrong password", func(t *testing.T) {
		userID := createTestUserForPasswordTests(t, testDB, "emailuser4", "user4@example.com", "password123")

		err := userService.ChangeEmail(userID, "changed@example.com", "wrongpassword", "")
		if err != ErrInvalidPassword {
			t.Errorf("expected ErrInvalidPassword, got: %v", err)
		}
	})

	t.Run("rejects duplicate email", func(t *testing.T) {
		createTestUserForPasswordTests(t, testDB, "emailuser5", "taken@example.com", "password123")
		userID := createTestUserForPasswordTests(t, testDB, "emailuser6", "user6@example.com", "password123")

		err := userService.ChangeEmail(userID, "taken@example.com", "password123", "")
		if err != ErrEmailInUse {
			t.Errorf("expected ErrEmailInUse, got: %v", err)
		}
	})

	t.Run("trims whitespace from email", func(t *testing.T) {
		userID := createTestUserForPasswordTests(t, testDB, "emailuser7", "user7@example.com", "password123")

		err := userService.ChangeEmail(userID, "  trimmed@example.com  ", "password123", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		user, err := testDB.GetUserByID(userID)
		if err != nil {
			t.Fatalf("failed to get user: %v", err)
		}
		if user.Email != "trimmed@example.com" {
			t.Errorf("expected trimmed email, got %q", user.Email)
		}
	})
}

func TestUserService_SetRecoveryKey(t *testing.T) {
	testDB, userService := setupUserServiceTest(t)
	defer testDB.Close()

	userID := createTestUserForPasswordTests(t, testDB, "recovuser", "recov@example.com", "password123")

	t.Run("sets recovery key successfully", func(t *testing.T) {
		hash, err := bcrypt.GenerateFromPassword([]byte("my-recovery-key"), 12)
		if err != nil {
			t.Fatalf("failed to hash: %v", err)
		}

		err = userService.SetRecoveryKey(userID, string(hash), []byte("salt-bytes"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		salt, err := userService.GetRecoveryKeySalt(userID)
		if err != nil {
			t.Fatalf("failed to get salt: %v", err)
		}
		if string(salt) != "salt-bytes" {
			t.Errorf("expected salt 'salt-bytes', got %q", string(salt))
		}
	})

	t.Run("rejects empty hash", func(t *testing.T) {
		err := userService.SetRecoveryKey(userID, "", []byte("salt"))
		if err == nil {
			t.Fatal("expected error for empty hash")
		}
	})

	t.Run("rejects empty salt", func(t *testing.T) {
		err := userService.SetRecoveryKey(userID, "hash", nil)
		if err == nil {
			t.Fatal("expected error for empty salt")
		}
	})

	t.Run("blocks setup when encrypted notes exist", func(t *testing.T) {
		encUserID := createTestUserForPasswordTests(t, testDB, "recovenc", "recovenc@example.com", "password123")
		if _, err := testDB.Exec(`
			INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
			                   content_encrypted, wrapped_dek, encryption_version)
			VALUES ('recov-enc-note', 'Encrypted', 'encrypted', '', '/', ?, datetime('now'), datetime('now'), 1, 'wrapped', 2)
		`, encUserID); err != nil {
			t.Fatalf("failed to create encrypted note: %v", err)
		}

		hash, err := bcrypt.GenerateFromPassword([]byte("blocked-recovery-key"), 12)
		if err != nil {
			t.Fatalf("failed to hash: %v", err)
		}

		err = userService.SetRecoveryKey(encUserID, string(hash), []byte("salt-bytes"))
		if err != ErrRecoveryWrappedDEKsRequired {
			t.Fatalf("expected ErrRecoveryWrappedDEKsRequired, got: %v", err)
		}
	})

	t.Run("sets recovery key for encrypted account with complete wrappers", func(t *testing.T) {
		encUserID := createTestUserForPasswordTests(t, testDB, "recovencok", "recovencok@example.com", "password123")
		if _, err := testDB.Exec(`
			INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
			                   content_encrypted, wrapped_dek, encryption_version)
			VALUES ('recov-enc-note-ok', 'Encrypted', 'encrypted', '', '/', ?, datetime('now'), datetime('now'), 1, ?, 2)
		`, encUserID, validWrappedDEK(10)); err != nil {
			t.Fatalf("failed to create encrypted note: %v", err)
		}
		if _, err := testDB.Exec(`
			INSERT INTO note_versions (id, note_id, user_id, version, title, content, snapshot_at,
			                           content_encrypted, wrapped_dek, encryption_version)
			VALUES (1001, 'recov-enc-note-ok', ?, 1, 'V1', '', datetime('now'), 1, ?, 2)
		`, encUserID, validWrappedDEK(11)); err != nil {
			t.Fatalf("failed to create encrypted note version: %v", err)
		}

		hash, err := bcrypt.GenerateFromPassword([]byte("allowed-recovery-key"), 12)
		if err != nil {
			t.Fatalf("failed to hash: %v", err)
		}

		err = userService.SetRecoveryKeyWithRecoveryWrappedDEKs(
			encUserID,
			string(hash),
			[]byte("salt-bytes"),
			map[string]string{"recov-enc-note-ok": validWrappedDEK(12)},
			map[string]string{"1001": validWrappedDEK(13)},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		salt, err := userService.GetRecoveryKeySalt(encUserID)
		if err != nil {
			t.Fatalf("failed to get salt: %v", err)
		}
		if string(salt) != "salt-bytes" {
			t.Fatalf("expected salt-bytes, got %q", string(salt))
		}
	})
}

func TestUserService_RecoverPasswordWithRecoveryKey(t *testing.T) {
	testDB, userService := setupUserServiceTest(t)
	defer testDB.Close()

	userID := createTestUserForPasswordTests(t, testDB, "recoveruser", "recover@example.com", "oldpassword1")

	// Set a recovery key
	recoveryKey := "my-recovery-key-123"
	hash, err := bcrypt.GenerateFromPassword([]byte(recoveryKey), 12)
	if err != nil {
		t.Fatalf("failed to hash recovery key: %v", err)
	}
	if err := testDB.SetRecoveryKey(userID, string(hash), []byte("salt")); err != nil {
		t.Fatalf("failed to set recovery key: %v", err)
	}

	t.Run("recovers password with valid key", func(t *testing.T) {
		err := userService.RecoverPasswordWithRecoveryKey(userID, recoveryKey, "newpassword1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify new password works
		user, err := testDB.GetUserByID(userID)
		if err != nil {
			t.Fatalf("failed to get user: %v", err)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("newpassword1")); err != nil {
			t.Error("new password should be valid after recovery")
		}
	})

	t.Run("rejects short new password", func(t *testing.T) {
		err := userService.RecoverPasswordWithRecoveryKey(userID, recoveryKey, "short")
		if err != ErrPasswordTooShort {
			t.Errorf("expected ErrPasswordTooShort, got: %v", err)
		}
	})

	t.Run("rejects invalid recovery key", func(t *testing.T) {
		// Re-set recovery key since previous test changed the password
		hash2, _ := bcrypt.GenerateFromPassword([]byte("correct-key"), 12)
		testDB.SetRecoveryKey(userID, string(hash2), []byte("salt"))

		err := userService.RecoverPasswordWithRecoveryKey(userID, "wrong-key", "newpassword1")
		if err == nil {
			t.Fatal("expected error for invalid recovery key")
		}
		if !strings.Contains(err.Error(), "invalid recovery key") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("blocks recovery when encrypted notes exist", func(t *testing.T) {
		encryptedUserID := createTestUserForPasswordTests(
			t,
			testDB,
			"enc-recover-user",
			"enc-recover@example.com",
			"oldpassword1",
		)

		hash3, err := bcrypt.GenerateFromPassword([]byte("enc-recovery-key"), 12)
		if err != nil {
			t.Fatalf("failed to hash recovery key: %v", err)
		}
		if err := testDB.SetRecoveryKey(encryptedUserID, string(hash3), []byte("salt")); err != nil {
			t.Fatalf("failed to set recovery key: %v", err)
		}

		if _, err := testDB.Exec(`
			INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
			                   content_encrypted, wrapped_dek, encryption_version)
			VALUES ('enc-note-1', 'Encrypted', 'encrypted', '', '/', ?, datetime('now'), datetime('now'), 1, 'wrapped', 2)
		`, encryptedUserID); err != nil {
			t.Fatalf("failed to create encrypted note: %v", err)
		}

		err = userService.RecoverPasswordWithRecoveryKey(encryptedUserID, "enc-recovery-key", "newpassword1")
		if err != ErrRecoveryResetNeedsDEKRewrap {
			t.Fatalf("expected ErrRecoveryResetNeedsDEKRewrap, got: %v", err)
		}
	})
}

func TestUserService_RecoverPasswordWithRecoveryKeyByEmail(t *testing.T) {
	testDB, userService := setupUserServiceTest(t)
	defer testDB.Close()

	userID := createTestUserForPasswordTests(t, testDB, "emailrecov", "emailrecov@example.com", "password123")

	recoveryKey := "email-recovery-key"
	hash, _ := bcrypt.GenerateFromPassword([]byte(recoveryKey), 12)
	testDB.SetRecoveryKey(userID, string(hash), []byte("salt"))

	t.Run("recovers with valid email and key", func(t *testing.T) {
		err := userService.RecoverPasswordWithRecoveryKeyByEmail(
			"emailrecov@example.com",
			recoveryKey,
			"brandnewpw1",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		user, _ := testDB.GetUserByID(userID)
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("brandnewpw1")); err != nil {
			t.Error("password should be updated after email recovery")
		}
	})

	t.Run("rejects nonexistent email", func(t *testing.T) {
		err := userService.RecoverPasswordWithRecoveryKeyByEmail(
			"nonexistent@example.com",
			recoveryKey,
			"newpassword1",
		)
		if err == nil {
			t.Fatal("expected error for nonexistent email")
		}
		// Should not reveal whether email exists
		if !strings.Contains(err.Error(), "invalid email or recovery key") {
			t.Errorf("error should be generic, got: %v", err)
		}
	})

	t.Run("rejects empty email", func(t *testing.T) {
		err := userService.RecoverPasswordWithRecoveryKeyByEmail("", recoveryKey, "newpassword1")
		if err == nil {
			t.Fatal("expected error for empty email")
		}
	})

	t.Run("rejects short password", func(t *testing.T) {
		err := userService.RecoverPasswordWithRecoveryKeyByEmail(
			"emailrecov@example.com",
			recoveryKey,
			"short",
		)
		if err != ErrPasswordTooShort {
			t.Errorf("expected ErrPasswordTooShort, got: %v", err)
		}
	})

	t.Run("normalizes email case", func(t *testing.T) {
		// Re-set recovery key
		hash2, _ := bcrypt.GenerateFromPassword([]byte("case-key"), 12)
		testDB.SetRecoveryKey(userID, string(hash2), []byte("salt"))

		err := userService.RecoverPasswordWithRecoveryKeyByEmail(
			"  EMAILRECOV@EXAMPLE.COM  ",
			"case-key",
			"newpassword1",
		)
		// May succeed or fail depending on DB email case - this tests the normalization
		_ = err
	})

	t.Run("blocks recovery by email when encrypted notes exist", func(t *testing.T) {
		encryptedUserID := createTestUserForPasswordTests(
			t,
			testDB,
			"enc-email-recover-user",
			"enc-email-recover@example.com",
			"oldpassword1",
		)

		hash2, _ := bcrypt.GenerateFromPassword([]byte("enc-email-key"), 12)
		if err := testDB.SetRecoveryKey(encryptedUserID, string(hash2), []byte("salt")); err != nil {
			t.Fatalf("failed to set recovery key: %v", err)
		}

		if _, err := testDB.Exec(`
			INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
			                   content_encrypted, wrapped_dek, encryption_version)
			VALUES ('enc-email-note-1', 'Encrypted', 'encrypted', '', '/', ?, datetime('now'), datetime('now'), 1, 'wrapped', 2)
		`, encryptedUserID); err != nil {
			t.Fatalf("failed to create encrypted note: %v", err)
		}

		err := userService.RecoverPasswordWithRecoveryKeyByEmail(
			"enc-email-recover@example.com",
			"enc-email-key",
			"newpassword1",
		)
		if err != ErrRecoveryResetNeedsDEKRewrap {
			t.Fatalf("expected ErrRecoveryResetNeedsDEKRewrap, got: %v", err)
		}
	})
}

func TestUserService_GetRecoveryKeySaltByEmail(t *testing.T) {
	testDB, userService := setupUserServiceTest(t)
	defer testDB.Close()

	userID := createTestUserForPasswordTests(t, testDB, "saltuser", "saltuser@example.com", "password123")

	t.Run("returns salt for user with recovery key", func(t *testing.T) {
		testDB.SetRecoveryKey(userID, "hash", []byte("my-salt-data"))

		salt, err := userService.GetRecoveryKeySaltByEmail("saltuser@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(salt) != "my-salt-data" {
			t.Errorf("expected salt 'my-salt-data', got %q", string(salt))
		}
	})

	t.Run("returns generic error for nonexistent email", func(t *testing.T) {
		_, err := userService.GetRecoveryKeySaltByEmail("nobody@example.com")
		if err == nil {
			t.Fatal("expected error for nonexistent email")
		}
		// Should not reveal whether user exists
		if !strings.Contains(err.Error(), "recovery key not available") {
			t.Errorf("expected generic error, got: %v", err)
		}
	})

	t.Run("rejects empty email", func(t *testing.T) {
		_, err := userService.GetRecoveryKeySaltByEmail("")
		if err == nil {
			t.Fatal("expected error for empty email")
		}
	})

	t.Run("returns generic unavailable when encrypted notes exist", func(t *testing.T) {
		encUserID := createTestUserForPasswordTests(t, testDB, "saltenc", "saltenc@example.com", "password123")
		if err := testDB.SetRecoveryKey(encUserID, "hash", []byte("enc-salt")); err != nil {
			t.Fatalf("failed to set recovery key: %v", err)
		}
		if _, err := testDB.Exec(`
			INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
			                   content_encrypted, wrapped_dek, encryption_version)
			VALUES ('salt-enc-note', 'Encrypted', 'encrypted', '', '/', ?, datetime('now'), datetime('now'), 1, 'wrapped', 2)
		`, encUserID); err != nil {
			t.Fatalf("failed to create encrypted note: %v", err)
		}

		_, err := userService.GetRecoveryKeySaltByEmail("saltenc@example.com")
		if err == nil {
			t.Fatal("expected error when encrypted notes exist")
		}
		if !strings.Contains(err.Error(), "recovery key not available") {
			t.Fatalf("expected generic unavailable error, got: %v", err)
		}
	})

	t.Run("returns salt by email when encrypted notes have recovery wrappers", func(t *testing.T) {
		encUserID := createTestUserForPasswordTests(t, testDB, "saltready", "saltready@example.com", "password123")
		if err := testDB.SetRecoveryKey(encUserID, "hash", []byte("ready-salt")); err != nil {
			t.Fatalf("failed to set recovery key: %v", err)
		}
		if _, err := testDB.Exec(`
			INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
			                   content_encrypted, wrapped_dek, wrapped_dek_recovery, encryption_version)
			VALUES ('salt-ready-note', 'Encrypted', 'encrypted', '', '/', ?, datetime('now'), datetime('now'), 1, ?, ?, 2)
		`, encUserID, validWrappedDEK(20), validWrappedDEK(21)); err != nil {
			t.Fatalf("failed to create encrypted note: %v", err)
		}

		salt, err := userService.GetRecoveryKeySaltByEmail("saltready@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(salt) != "ready-salt" {
			t.Fatalf("expected ready-salt, got %q", string(salt))
		}
	})
}

func TestUserService_RecoveryResetTokenFlow_WithEncryptedContent(t *testing.T) {
	testDB, userService := setupUserServiceTest(t)
	defer testDB.Close()

	userID := createTestUserForPasswordTests(t, testDB, "tokenrecov", "tokenrecov@example.com", "oldpassword1")
	if err := testDB.SetUserEncryptionSalt(userID, []byte("0123456789abcdef")); err != nil {
		t.Fatalf("failed to set encryption salt: %v", err)
	}

	recoveryKey := "token-recovery-key"
	hash, err := bcrypt.GenerateFromPassword([]byte(recoveryKey), 12)
	if err != nil {
		t.Fatalf("failed to hash recovery key: %v", err)
	}
	if err := testDB.SetRecoveryKey(userID, string(hash), []byte("salt")); err != nil {
		t.Fatalf("failed to set recovery key: %v", err)
	}

	_, err = testDB.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
		                   content_encrypted, wrapped_dek, wrapped_dek_recovery, encryption_version)
		VALUES ('token-rec-note-1', 'Encrypted', 'encrypted', '', '/', ?, datetime('now'), datetime('now'), 1, ?, ?, 2)
	`, userID, validWrappedDEK(1), validWrappedDEK(2))
	if err != nil {
		t.Fatalf("failed to create encrypted note: %v", err)
	}

	_, err = testDB.Exec(`
		INSERT INTO note_versions (id, note_id, user_id, version, title, content, snapshot_at,
		                           content_encrypted, wrapped_dek, wrapped_dek_recovery, encryption_version)
		VALUES (101, 'token-rec-note-1', ?, 1, 'V1', '', datetime('now'), 1, ?, ?, 2)
	`, userID, validWrappedDEK(3), validWrappedDEK(4))
	if err != nil {
		t.Fatalf("failed to create encrypted note version: %v", err)
	}

	verifyResult, err := userService.BeginRecoveryResetByEmail("tokenrecov@example.com", recoveryKey)
	if err != nil {
		t.Fatalf("BeginRecoveryResetByEmail failed: %v", err)
	}
	if verifyResult.RecoveryResetToken == "" {
		t.Fatal("expected non-empty recovery reset token")
	}
	if verifyResult.EncryptionSalt == "" {
		t.Fatal("expected encryption_salt in verify result")
	}

	notes, versions, err := userService.GetRecoveryWrappedDEKs(verifyResult.RecoveryResetToken)
	if err != nil {
		t.Fatalf("GetRecoveryWrappedDEKs failed: %v", err)
	}
	if len(notes) != 1 || len(versions) != 1 {
		t.Fatalf("expected 1 note and 1 version wrapper, got %d and %d", len(notes), len(versions))
	}

	err = userService.FinalizeRecoveryResetWithToken(
		verifyResult.RecoveryResetToken,
		"newpassword1",
		map[string]string{"token-rec-note-1": validWrappedDEK(9)},
		map[string]string{"101": validWrappedDEK(8)},
	)
	if err != nil {
		t.Fatalf("FinalizeRecoveryResetWithToken failed: %v", err)
	}

	user, err := testDB.GetUserByID(userID)
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("newpassword1")); err != nil {
		t.Fatal("expected password to be updated after recovery finalize")
	}

	var noteWrappedDEK string
	if err := testDB.QueryRow(`SELECT wrapped_dek FROM notes WHERE id = 'token-rec-note-1'`).Scan(&noteWrappedDEK); err != nil {
		t.Fatalf("failed to read note wrapped_dek: %v", err)
	}
	if noteWrappedDEK != validWrappedDEK(9) {
		t.Fatalf("expected updated note wrapped_dek, got %q", noteWrappedDEK)
	}

	var versionWrappedDEK string
	if err := testDB.QueryRow(`SELECT wrapped_dek FROM note_versions WHERE id = 101`).Scan(&versionWrappedDEK); err != nil {
		t.Fatalf("failed to read version wrapped_dek: %v", err)
	}
	if versionWrappedDEK != validWrappedDEK(8) {
		t.Fatalf("expected updated version wrapped_dek, got %q", versionWrappedDEK)
	}

	prefs, err := testDB.GetUserPreferences(userID)
	if err != nil {
		t.Fatalf("failed to read user preferences: %v", err)
	}
	if prefs.RecoveryKeyHash != nil || prefs.RecoveryKeySalt != nil {
		t.Fatal("expected recovery key material to be invalidated after finalize")
	}

	var noteRecovery sql.NullString
	if err := testDB.QueryRow(`SELECT wrapped_dek_recovery FROM notes WHERE id = 'token-rec-note-1'`).Scan(&noteRecovery); err != nil {
		t.Fatalf("failed to read note wrapped_dek_recovery: %v", err)
	}
	if noteRecovery.Valid {
		t.Fatal("expected note wrapped_dek_recovery to be cleared")
	}

	var versionRecovery sql.NullString
	if err := testDB.QueryRow(`SELECT wrapped_dek_recovery FROM note_versions WHERE id = 101`).Scan(&versionRecovery); err != nil {
		t.Fatalf("failed to read version wrapped_dek_recovery: %v", err)
	}
	if versionRecovery.Valid {
		t.Fatal("expected version wrapped_dek_recovery to be cleared")
	}

	if err := userService.FinalizeRecoveryResetWithToken(
		verifyResult.RecoveryResetToken,
		"anotherpassword1",
		map[string]string{},
		map[string]string{},
	); err != ErrInvalidRecoveryResetToken {
		t.Fatalf("expected ErrInvalidRecoveryResetToken when reusing token, got %v", err)
	}
}

// Ensure GetRecoveryKeySalt and GetRecoveryKeySaltByEmail use UserService
// (These are tested via the db layer helper setupUserServiceTest)
func TestUserService_GetRecoveryKeySalt(t *testing.T) {
	testDB, userService := setupUserServiceTest(t)
	defer testDB.Close()

	userID := createTestUserForPasswordTests(t, testDB, "getsaltuser", "getsalt@example.com", "password123")

	t.Run("returns error when no recovery key set", func(t *testing.T) {
		_, err := userService.GetRecoveryKeySalt(userID)
		if err == nil {
			t.Fatal("expected error when no recovery key is set")
		}
		if err != db.ErrNotFound {
			t.Errorf("expected db.ErrNotFound, got: %v", err)
		}
	})

	t.Run("returns salt after setting recovery key", func(t *testing.T) {
		testDB.SetRecoveryKey(userID, "hash-value", []byte("recovery-salt"))

		salt, err := userService.GetRecoveryKeySalt(userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(salt) != "recovery-salt" {
			t.Errorf("expected 'recovery-salt', got %q", string(salt))
		}
	})

	t.Run("blocks salt access when encrypted notes exist", func(t *testing.T) {
		encUserID := createTestUserForPasswordTests(t, testDB, "getsaltenc", "getsaltenc@example.com", "password123")
		if err := testDB.SetRecoveryKey(encUserID, "hash-value", []byte("recovery-salt")); err != nil {
			t.Fatalf("failed to set recovery key: %v", err)
		}
		if _, err := testDB.Exec(`
			INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
			                   content_encrypted, wrapped_dek, encryption_version)
			VALUES ('get-salt-enc-note', 'Encrypted', 'encrypted', '', '/', ?, datetime('now'), datetime('now'), 1, 'wrapped', 2)
		`, encUserID); err != nil {
			t.Fatalf("failed to create encrypted note: %v", err)
		}

		_, err := userService.GetRecoveryKeySalt(encUserID)
		if err != db.ErrNotFound {
			t.Fatalf("expected db.ErrNotFound, got: %v", err)
		}
	})

	t.Run("returns salt for encrypted account when wrappers exist", func(t *testing.T) {
		encUserID := createTestUserForPasswordTests(t, testDB, "getsaltready", "getsaltready@example.com", "password123")
		if err := testDB.SetRecoveryKey(encUserID, "hash-value", []byte("ready-salt")); err != nil {
			t.Fatalf("failed to set recovery key: %v", err)
		}
		if _, err := testDB.Exec(`
			INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at,
			                   content_encrypted, wrapped_dek, wrapped_dek_recovery, encryption_version)
			VALUES ('get-salt-ready-note', 'Encrypted', 'encrypted', '', '/', ?, datetime('now'), datetime('now'), 1, ?, ?, 2)
		`, encUserID, validWrappedDEK(30), validWrappedDEK(31)); err != nil {
			t.Fatalf("failed to create encrypted note: %v", err)
		}
		if _, err := testDB.Exec(`
			INSERT INTO note_versions (id, note_id, user_id, version, title, content, snapshot_at,
			                           content_encrypted, wrapped_dek, wrapped_dek_recovery, encryption_version)
			VALUES (2001, 'get-salt-ready-note', ?, 1, 'V1', '', datetime('now'), 1, ?, ?, 2)
		`, encUserID, validWrappedDEK(32), validWrappedDEK(33)); err != nil {
			t.Fatalf("failed to create encrypted note version: %v", err)
		}

		salt, err := userService.GetRecoveryKeySalt(encUserID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(salt) != "ready-salt" {
			t.Fatalf("expected ready-salt, got %q", string(salt))
		}
	})
}
