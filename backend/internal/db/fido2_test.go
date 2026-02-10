package db

import (
	"testing"
)

func setupFIDO2TestDB(t *testing.T) *DB {
	testDB, err := Open(":memory:", "")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	if err := testDB.Migrate(); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	// Enable registration and create a test user
	testDB.Exec("INSERT INTO users (username, email, password_hash) VALUES ('testuser', 'test@example.com', '$2a$12$dummy')")

	return testDB
}

func TestAddAndGetFIDO2Credential(t *testing.T) {
	db := setupFIDO2TestDB(t)
	defer db.Close()

	cred := &FIDO2Credential{
		CredentialID:    []byte("cred-id-123"),
		PublicKey:       []byte("pubkey-data"),
		AttestationType: "none",
		AAGUID:          []byte("aaguid-data"),
		SignCount:       0,
		DeviceName:      "Test YubiKey",
		Transports:      `["usb"]`,
	}

	// Add credential
	err := db.AddFIDO2Credential(1, cred)
	if err != nil {
		t.Fatalf("AddFIDO2Credential failed: %v", err)
	}
	if cred.ID == 0 {
		t.Error("Expected non-zero credential ID")
	}

	// Get all credentials for user
	creds, err := db.GetFIDO2Credentials(1)
	if err != nil {
		t.Fatalf("GetFIDO2Credentials failed: %v", err)
	}
	if len(creds) != 1 {
		t.Fatalf("Expected 1 credential, got %d", len(creds))
	}
	if creds[0].DeviceName != "Test YubiKey" {
		t.Errorf("Expected device name 'Test YubiKey', got '%s'", creds[0].DeviceName)
	}
	if string(creds[0].CredentialID) != "cred-id-123" {
		t.Errorf("Expected credential ID 'cred-id-123', got '%s'", string(creds[0].CredentialID))
	}

	// Get by credential ID
	found, err := db.GetFIDO2CredentialByCredentialID([]byte("cred-id-123"))
	if err != nil {
		t.Fatalf("GetFIDO2CredentialByCredentialID failed: %v", err)
	}
	if found.DeviceName != "Test YubiKey" {
		t.Errorf("Expected device name 'Test YubiKey', got '%s'", found.DeviceName)
	}

	// Not found
	_, err = db.GetFIDO2CredentialByCredentialID([]byte("nonexistent"))
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestUpdateFIDO2SignCount(t *testing.T) {
	db := setupFIDO2TestDB(t)
	defer db.Close()

	cred := &FIDO2Credential{
		CredentialID:    []byte("cred-sign-test"),
		PublicKey:       []byte("pubkey"),
		AttestationType: "none",
		SignCount:       0,
		DeviceName:      "Test Key",
	}
	db.AddFIDO2Credential(1, cred)

	// Update sign count
	err := db.UpdateFIDO2SignCount([]byte("cred-sign-test"), 42)
	if err != nil {
		t.Fatalf("UpdateFIDO2SignCount failed: %v", err)
	}

	// Verify
	found, _ := db.GetFIDO2CredentialByCredentialID([]byte("cred-sign-test"))
	if found.SignCount != 42 {
		t.Errorf("Expected sign count 42, got %d", found.SignCount)
	}
}

func TestTouchFIDO2Credential(t *testing.T) {
	db := setupFIDO2TestDB(t)
	defer db.Close()

	cred := &FIDO2Credential{
		CredentialID:    []byte("cred-touch-test"),
		PublicKey:       []byte("pubkey"),
		AttestationType: "none",
		DeviceName:      "Test Key",
	}
	db.AddFIDO2Credential(1, cred)

	// Touch should set last_used_at
	err := db.TouchFIDO2Credential([]byte("cred-touch-test"))
	if err != nil {
		t.Fatalf("TouchFIDO2Credential failed: %v", err)
	}

	found, _ := db.GetFIDO2CredentialByCredentialID([]byte("cred-touch-test"))
	if found.LastUsedAt == nil {
		t.Error("Expected last_used_at to be set after touch")
	}
}

func TestDeleteFIDO2Credential(t *testing.T) {
	db := setupFIDO2TestDB(t)
	defer db.Close()

	cred := &FIDO2Credential{
		CredentialID:    []byte("cred-delete-test"),
		PublicKey:       []byte("pubkey"),
		AttestationType: "none",
		DeviceName:      "Delete Me",
	}
	db.AddFIDO2Credential(1, cred)

	// Delete
	err := db.DeleteFIDO2Credential(1, cred.ID)
	if err != nil {
		t.Fatalf("DeleteFIDO2Credential failed: %v", err)
	}

	// Verify deleted
	creds, _ := db.GetFIDO2Credentials(1)
	if len(creds) != 0 {
		t.Errorf("Expected 0 credentials after delete, got %d", len(creds))
	}

	// Delete nonexistent should return ErrNotFound
	err = db.DeleteFIDO2Credential(1, 99999)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound for nonexistent credential, got %v", err)
	}

	// Delete wrong user should return ErrNotFound
	cred2 := &FIDO2Credential{
		CredentialID:    []byte("cred-wrong-user"),
		PublicKey:       []byte("pubkey"),
		AttestationType: "none",
		DeviceName:      "Wrong User",
	}
	db.AddFIDO2Credential(1, cred2)
	err = db.DeleteFIDO2Credential(999, cred2.ID) // wrong user
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound for wrong user, got %v", err)
	}
}

func TestCountAndHasFIDO2Credentials(t *testing.T) {
	db := setupFIDO2TestDB(t)
	defer db.Close()

	// Initially no credentials
	count, err := db.CountFIDO2Credentials(1)
	if err != nil {
		t.Fatalf("CountFIDO2Credentials failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 credentials, got %d", count)
	}

	has, err := db.HasFIDO2Credentials(1)
	if err != nil {
		t.Fatalf("HasFIDO2Credentials failed: %v", err)
	}
	if has {
		t.Error("Expected HasFIDO2Credentials to return false")
	}

	// Add one
	db.AddFIDO2Credential(1, &FIDO2Credential{
		CredentialID:    []byte("cred-count-1"),
		PublicKey:       []byte("pk"),
		AttestationType: "none",
		DeviceName:      "Key 1",
	})

	count, _ = db.CountFIDO2Credentials(1)
	if count != 1 {
		t.Errorf("Expected 1 credential, got %d", count)
	}

	has, _ = db.HasFIDO2Credentials(1)
	if !has {
		t.Error("Expected HasFIDO2Credentials to return true")
	}

	// Add another
	db.AddFIDO2Credential(1, &FIDO2Credential{
		CredentialID:    []byte("cred-count-2"),
		PublicKey:       []byte("pk"),
		AttestationType: "none",
		DeviceName:      "Key 2",
	})

	count, _ = db.CountFIDO2Credentials(1)
	if count != 2 {
		t.Errorf("Expected 2 credentials, got %d", count)
	}
}

func TestDeleteAllFIDO2Credentials(t *testing.T) {
	db := setupFIDO2TestDB(t)
	defer db.Close()

	for i := 0; i < 3; i++ {
		db.AddFIDO2Credential(1, &FIDO2Credential{
			CredentialID:    []byte("cred-all-" + string(rune('0'+i))),
			PublicKey:       []byte("pk"),
			AttestationType: "none",
			DeviceName:      "Key",
		})
	}

	count, _ := db.CountFIDO2Credentials(1)
	if count != 3 {
		t.Fatalf("Expected 3 credentials, got %d", count)
	}

	err := db.DeleteAllFIDO2Credentials(1)
	if err != nil {
		t.Fatalf("DeleteAllFIDO2Credentials failed: %v", err)
	}

	count, _ = db.CountFIDO2Credentials(1)
	if count != 0 {
		t.Errorf("Expected 0 credentials after delete all, got %d", count)
	}
}
