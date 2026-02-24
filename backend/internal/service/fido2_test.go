package service

import (
	"log/slog"
	"os"
	"testing"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/fido2"
)

func setupFIDO2Test(t *testing.T) (*db.DB, *FIDO2Service, *db.User) {
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

	manager, err := fido2.NewManager("xelanote-test", "localhost", []string{"http://localhost:8080"})
	if err != nil {
		t.Fatalf("failed to create fido2 manager: %v", err)
	}

	tfaService := NewTwoFactorService(database, logger)
	svc := NewFIDO2Service(database, manager, tfaService, logger)

	user, err := database.CreateUser("fido2user", "fido2@example.com", "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5aeKXJ8F3xJPi")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return database, svc, user
}

// addTestFIDO2Credential inserts a fake FIDO2 credential directly into the DB.
func addTestFIDO2Credential(t *testing.T, database *db.DB, userID int, deviceName string) *db.FIDO2Credential {
	t.Helper()

	cred := &db.FIDO2Credential{
		CredentialID:    []byte("test-credential-id-" + deviceName),
		PublicKey:       []byte("test-public-key-" + deviceName),
		AttestationType: "none",
		AAGUID:          []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		SignCount:       0,
		DeviceName:      deviceName,
		Transports:      `["usb","nfc"]`,
	}

	if err := database.AddFIDO2Credential(userID, cred); err != nil {
		t.Fatalf("failed to add FIDO2 credential: %v", err)
	}

	return cred
}

func TestFIDO2Service_ListCredentials(t *testing.T) {
	database, svc, user := setupFIDO2Test(t)

	t.Run("returns empty list for user with no credentials", func(t *testing.T) {
		creds, err := svc.ListCredentials(user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(creds) != 0 {
			t.Errorf("expected 0 credentials, got %d", len(creds))
		}
	})

	t.Run("returns credentials with parsed transports", func(t *testing.T) {
		addTestFIDO2Credential(t, database, user.ID, "YubiKey 5")

		creds, err := svc.ListCredentials(user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(creds) != 1 {
			t.Fatalf("expected 1 credential, got %d", len(creds))
		}
		if creds[0].DeviceName != "YubiKey 5" {
			t.Errorf("expected device name 'YubiKey 5', got %q", creds[0].DeviceName)
		}
		if len(creds[0].Transports) != 2 {
			t.Errorf("expected 2 transports, got %d", len(creds[0].Transports))
		}
		if creds[0].Transports[0] != "usb" || creds[0].Transports[1] != "nfc" {
			t.Errorf("unexpected transports: %v", creds[0].Transports)
		}
	})

	t.Run("returns multiple credentials in order", func(t *testing.T) {
		addTestFIDO2Credential(t, database, user.ID, "TouchID")

		creds, err := svc.ListCredentials(user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(creds) != 2 {
			t.Fatalf("expected 2 credentials, got %d", len(creds))
		}
		// First registered should come first (ORDER BY created_at ASC)
		if creds[0].DeviceName != "YubiKey 5" {
			t.Errorf("expected first credential 'YubiKey 5', got %q", creds[0].DeviceName)
		}
		if creds[1].DeviceName != "TouchID" {
			t.Errorf("expected second credential 'TouchID', got %q", creds[1].DeviceName)
		}
	})

	t.Run("isolates credentials between users", func(t *testing.T) {
		otherUser, err := database.CreateUser("otheruser", "other@example.com", "hash")
		if err != nil {
			t.Fatalf("failed to create other user: %v", err)
		}

		creds, err := svc.ListCredentials(otherUser.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(creds) != 0 {
			t.Errorf("expected 0 credentials for other user, got %d", len(creds))
		}
	})
}

func TestFIDO2Service_DeleteCredential(t *testing.T) {
	database, svc, user := setupFIDO2Test(t)

	t.Run("deletes own credential", func(t *testing.T) {
		cred := addTestFIDO2Credential(t, database, user.ID, "ToDelete")

		err := svc.DeleteCredential(user.ID, cred.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify it's gone
		creds, _ := svc.ListCredentials(user.ID)
		for _, c := range creds {
			if c.ID == cred.ID {
				t.Error("deleted credential should not appear in list")
			}
		}
	})

	t.Run("rejects deleting nonexistent credential", func(t *testing.T) {
		err := svc.DeleteCredential(user.ID, 99999)
		if err == nil {
			t.Fatal("expected error for nonexistent credential")
		}
	})

	t.Run("rejects deleting another user's credential", func(t *testing.T) {
		otherUser, _ := database.CreateUser("deleteother", "deleteother@example.com", "hash")
		cred := addTestFIDO2Credential(t, database, otherUser.ID, "OtherKey")

		err := svc.DeleteCredential(user.ID, cred.ID)
		if err == nil {
			t.Fatal("expected error when deleting another user's credential")
		}
	})
}

func TestFIDO2Service_BeginRegistration(t *testing.T) {
	_, svc, user := setupFIDO2Test(t)

	t.Run("returns credential creation options", func(t *testing.T) {
		creation, err := svc.BeginRegistration(user.ID, user.Username)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if creation == nil {
			t.Fatal("expected non-nil CredentialCreation")
		}
		if creation.Response.RelyingParty.Name != "xelanote-test" {
			t.Errorf("expected RP name 'xelanote-test', got %q", creation.Response.RelyingParty.Name)
		}
		if string(creation.Response.User.Name) != user.Username {
			t.Errorf("expected user name %q, got %q", user.Username, creation.Response.User.Name)
		}
	})
}

func TestFIDO2Service_BeginAuthentication(t *testing.T) {
	_, svc, user := setupFIDO2Test(t)

	t.Run("rejects when no credentials registered", func(t *testing.T) {
		_, err := svc.BeginAuthentication(user.ID, user.Username)
		if err == nil {
			t.Fatal("expected error when no credentials registered")
		}
	})

	t.Run("returns assertion options with registered credentials", func(t *testing.T) {
		// Register a credential first via BeginRegistration (stores challenge)
		_, err := svc.BeginRegistration(user.ID, user.Username)
		if err != nil {
			t.Fatalf("begin registration failed: %v", err)
		}

		// Add a credential directly to DB (simulating a completed registration)
		addTestFIDO2Credential(t, svc.db, user.ID, "AuthKey")

		assertion, err := svc.BeginAuthentication(user.ID, user.Username)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if assertion == nil {
			t.Fatal("expected non-nil CredentialAssertion")
		}
		if len(assertion.Response.AllowedCredentials) == 0 {
			t.Error("expected at least one allowed credential")
		}
	})
}

func TestFIDO2Service_PendingLogin(t *testing.T) {
	_, svc, _ := setupFIDO2Test(t)

	t.Run("stores and retrieves pending login", func(t *testing.T) {
		type pendingData struct {
			UserID int
			Token  string
		}

		data := &pendingData{UserID: 42, Token: "test-token"}
		err := svc.StorePendingLogin("session-123", data)
		if err != nil {
			t.Fatalf("unexpected error storing: %v", err)
		}

		retrieved, err := svc.GetPendingLogin("session-123")
		if err != nil {
			t.Fatalf("unexpected error retrieving: %v", err)
		}

		result, ok := retrieved.(*pendingData)
		if !ok {
			t.Fatalf("expected *pendingData, got %T", retrieved)
		}
		if result.UserID != 42 || result.Token != "test-token" {
			t.Errorf("unexpected data: %+v", result)
		}
	})

	t.Run("returns error for nonexistent token", func(t *testing.T) {
		_, err := svc.GetPendingLogin("nonexistent")
		if err == nil {
			t.Fatal("expected error for nonexistent token")
		}
	})

	t.Run("pending login is one-time use", func(t *testing.T) {
		err := svc.StorePendingLogin("one-time", "data")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// First retrieval should succeed
		_, err = svc.GetPendingLogin("one-time")
		if err != nil {
			t.Fatalf("first get failed: %v", err)
		}

		// Second retrieval should fail (one-time use)
		_, err = svc.GetPendingLogin("one-time")
		if err == nil {
			t.Fatal("expected error on second retrieval (one-time use)")
		}
	})
}
