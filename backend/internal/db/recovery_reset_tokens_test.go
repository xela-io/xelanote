package db

import (
	"context"
	"testing"
	"time"
)

func TestRecoveryResetTokenLifecycle(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	createTestUser(t, database, 1)

	const tokenHash = "test-recovery-token-hash"
	if err := database.CreateRecoveryResetToken(1, tokenHash, time.Now().UTC().Add(10*time.Minute)); err != nil {
		t.Fatalf("CreateRecoveryResetToken failed: %v", err)
	}

	userID, err := database.ValidateRecoveryResetToken(tokenHash)
	if err != nil {
		t.Fatalf("ValidateRecoveryResetToken failed: %v", err)
	}
	if userID != 1 {
		t.Fatalf("expected userID 1, got %d", userID)
	}

	tx, err := database.BeginTx(context.Background())
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}
	defer tx.Rollback()

	consumedUserID, err := tx.ConsumeRecoveryResetTokenTx(tokenHash)
	if err != nil {
		t.Fatalf("ConsumeRecoveryResetTokenTx failed: %v", err)
	}
	if consumedUserID != 1 {
		t.Fatalf("expected consumed userID 1, got %d", consumedUserID)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	if _, err := database.ValidateRecoveryResetToken(tokenHash); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for consumed token, got %v", err)
	}
}

func TestRecoveryResetTokenExpired(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	createTestUser(t, database, 1)

	const tokenHash = "expired-recovery-token-hash"
	if err := database.CreateRecoveryResetToken(1, tokenHash, time.Now().UTC().Add(-1*time.Minute)); err != nil {
		t.Fatalf("CreateRecoveryResetToken failed: %v", err)
	}

	if _, err := database.ValidateRecoveryResetToken(tokenHash); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for expired token, got %v", err)
	}
}
