//go:build fts5

package api

import (
	"context"
	"crypto/rand"
	"io"
	"log/slog"
	"testing"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/service"
)

// TestSaltOverwritePrevention tests the critical fix that prevents
// generating a new encryption salt when a user already has encrypted notes.
// This would cause permanent data loss.
func TestSaltOverwritePrevention(t *testing.T) {
	// Setup test database
	database, err := db.Open(":memory:", "")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}
	defer database.Close()

	// Generate JWT secret
	jwtSecret := make([]byte, 32)
	_, err = rand.Read(jwtSecret)
	if err != nil {
		t.Fatalf("failed to generate jwt secret: %v", err)
	}

	// Create test logger (discard output)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create services
	tfaService := service.NewTwoFactorService(database, logger)
	authService := service.NewAuthService(database, jwtSecret, tfaService)
	noteService := service.NewNoteService(database)

	// Create test server with noteService (important for the check!)
	server := NewServer(ServerConfig{
		NoteService: noteService,
		AuthService: authService,
		TFAService:  tfaService,
		Logger:      logger,
		JWTSecret:   jwtSecret,
	})

	// 1. Create a user
	user, err := authService.Register(context.Background(), "testuser", "test@example.com", "TestPassword123!")
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}
	userID := user.ID

	// 2. Create an encrypted note for this user
	_, err = database.Exec(`
		INSERT INTO notes (id, user_id, title, title_norm, content, encrypted_content)
		VALUES ('test-note-uuid', ?, 'Test Note', 'test note', '', 'encrypted_data_here')
	`, userID)
	if err != nil {
		t.Fatalf("failed to create encrypted note: %v", err)
	}

	// 3. Verify user has encrypted notes
	hasEncrypted, err := noteService.UserHasEncryptedNotes(userID)
	if err != nil {
		t.Fatalf("failed to check encrypted notes: %v", err)
	}
	if !hasEncrypted {
		t.Fatal("expected user to have encrypted notes")
	}

	// 4. Delete the encryption salt (simulating the bug scenario)
	// The encryption_salt is in the users table, not a separate table
	_, err = database.Exec(`UPDATE users SET encryption_salt = NULL WHERE id = ?`, userID)
	if err != nil {
		t.Fatalf("failed to delete salt: %v", err)
	}

	// 5. Try to get/generate salt - this should FAIL with error (the fix!)
	_, saltErr := server.getOrGenerateUserSalt(userID)

	// The fix: This should return an error, NOT silently generate a new salt
	if saltErr == nil {
		t.Fatal("CRITICAL: getOrGenerateUserSalt should have returned an error when encrypted notes exist but salt is missing")
	}

	expectedErrMsg := "encryption salt missing but encrypted notes exist - contact administrator for data recovery"
	if saltErr.Error() != expectedErrMsg {
		t.Fatalf("unexpected error message: got %q, want %q", saltErr.Error(), expectedErrMsg)
	}

	t.Logf("✅ Salt overwrite prevention working correctly: %v", saltErr)
}

// TestSaltGenerationAllowedForNewUsers verifies that new users without
// encrypted notes CAN get a new salt generated (normal flow)
func TestSaltGenerationAllowedForNewUsers(t *testing.T) {
	// Setup test database
	database, err := db.Open(":memory:", "")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}
	defer database.Close()

	// Generate JWT secret
	jwtSecret := make([]byte, 32)
	_, err = rand.Read(jwtSecret)
	if err != nil {
		t.Fatalf("failed to generate jwt secret: %v", err)
	}

	// Create test logger (discard output)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create services
	tfaService := service.NewTwoFactorService(database, logger)
	authService := service.NewAuthService(database, jwtSecret, tfaService)
	noteService := service.NewNoteService(database)

	// Create test server
	server := NewServer(ServerConfig{
		NoteService: noteService,
		AuthService: authService,
		TFAService:  tfaService,
		Logger:      logger,
		JWTSecret:   jwtSecret,
	})

	// 1. Create a user
	user, err := authService.Register(context.Background(), "newuser", "new@example.com", "TestPassword123!")
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}
	userID := user.ID

	// 2. User has NO encrypted notes (just registered)
	hasEncrypted, err := noteService.UserHasEncryptedNotes(userID)
	if err != nil {
		t.Fatalf("failed to check encrypted notes: %v", err)
	}
	if hasEncrypted {
		t.Fatal("new user should not have encrypted notes")
	}

	// 3. Delete any existing salt (simulating fresh state)
	_, _ = database.Exec(`UPDATE users SET encryption_salt = NULL WHERE id = ?`, userID)

	// 4. Generate salt - this should SUCCEED (no encrypted notes to protect)
	salt, saltErr := server.getOrGenerateUserSalt(userID)
	if saltErr != nil {
		t.Fatalf("salt generation should succeed for new user: %v", saltErr)
	}
	if len(salt) != 16 {
		t.Fatalf("expected 16-byte salt, got %d bytes", len(salt))
	}

	t.Logf("✅ Salt generation allowed for new users: %d bytes generated", len(salt))
}
