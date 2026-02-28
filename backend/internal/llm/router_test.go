package llm

import (
	"context"
	"os"
	"testing"

	"github.com/xela-io/xelanote/internal/crypto"
	"github.com/xela-io/xelanote/internal/db"
)

func init() {
	_ = os.Setenv("XELANOTE_API_KEY_SECRET", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
}

func setupRouterTestDB(t *testing.T) (*db.DB, int) {
	t.Helper()

	database, err := db.Open(":memory:", "")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	userID := 1
	if _, err := database.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (?, 'testuser', 'test@example.com', 'hash', datetime('now'))
	`, userID); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return database, userID
}

func insertTestNote(t *testing.T, database *db.DB, userID int, noteID string, aiEnabled bool) {
	t.Helper()

	ai := 0
	if aiEnabled {
		ai = 1
	}
	if _, err := database.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, ai_enabled, created_at, updated_at)
		VALUES (?, 'Note', 'note', 'Content', '/', ?, ?, datetime('now'), datetime('now'))
	`, noteID, userID, ai); err != nil {
		t.Fatalf("failed to create note: %v", err)
	}
}

func setClaudeKey(t *testing.T, database *db.DB, userID int, apiKey string) {
	t.Helper()

	encrypted, err := crypto.EncryptAPIKey(apiKey)
	if err != nil {
		t.Fatalf("failed to encrypt api key: %v", err)
	}
	if err := database.SetClaudeAPIKey(userID, encrypted); err != nil {
		t.Fatalf("failed to set claude api key: %v", err)
	}
}

func setGeminiKey(t *testing.T, database *db.DB, userID int, apiKey string) {
	t.Helper()

	encrypted, err := crypto.EncryptAPIKey(apiKey)
	if err != nil {
		t.Fatalf("failed to encrypt api key: %v", err)
	}
	if err := database.SetGeminiAPIKey(userID, encrypted); err != nil {
		t.Fatalf("failed to set gemini api key: %v", err)
	}
}

func setOpenAIKey(t *testing.T, database *db.DB, userID int, apiKey string) {
	t.Helper()

	encrypted, err := crypto.EncryptAPIKey(apiKey)
	if err != nil {
		t.Fatalf("failed to encrypt api key: %v", err)
	}
	if err := database.SetOpenAIAPIKey(userID, encrypted); err != nil {
		t.Fatalf("failed to set openai api key: %v", err)
	}
}

func TestGetProviderForNote_NotAIEnabled(t *testing.T) {
	t.Parallel()

	database, userID := setupRouterTestDB(t)
	defer database.Close()

	insertTestNote(t, database, userID, "note1", false)
	router := NewProviderRouter(database)

	_, err := router.GetProviderForNote(context.Background(), userID, "note1")
	if err != ErrNoteNotAIEnabled {
		t.Fatalf("expected ErrNoteNotAIEnabled, got %v", err)
	}
}

func TestGetProviderForNote_ClaudePreferred(t *testing.T) {
	t.Parallel()

	database, userID := setupRouterTestDB(t)
	defer database.Close()

	insertTestNote(t, database, userID, "note1", true)
	setClaudeKey(t, database, userID, "sk-ant-test-claude-key-1234567890")
	setGeminiKey(t, database, userID, "AIza-test-gemini-key-1234567890")

	router := NewProviderRouter(database)
	provider, err := router.GetProviderForNote(context.Background(), userID, "note1")
	if err != nil {
		t.Fatalf("expected provider, got %v", err)
	}
	if provider.Name() != string(ProviderTypeClaude) {
		t.Fatalf("expected claude provider, got %s", provider.Name())
	}
}

func TestGetProviderForNote_GeminiFallback(t *testing.T) {
	t.Parallel()

	database, userID := setupRouterTestDB(t)
	defer database.Close()

	insertTestNote(t, database, userID, "note1", true)
	setGeminiKey(t, database, userID, "AIza-test-gemini-key-1234567890")

	router := NewProviderRouter(database)
	provider, err := router.GetProviderForNote(context.Background(), userID, "note1")
	if err != nil {
		t.Fatalf("expected provider, got %v", err)
	}
	if provider.Name() != string(ProviderTypeGemini) {
		t.Fatalf("expected gemini provider, got %s", provider.Name())
	}
}

func TestGetProviderForNote_NoProvidersConfigured(t *testing.T) {
	t.Parallel()

	database, userID := setupRouterTestDB(t)
	defer database.Close()

	insertTestNote(t, database, userID, "note1", true)
	router := NewProviderRouter(database)

	_, err := router.GetProviderForNote(context.Background(), userID, "note1")
	if err != ErrNoProviderAvailable {
		t.Fatalf("expected ErrNoProviderAvailable, got %v", err)
	}
}

func TestGetAnyProvider_ClaudePreferred(t *testing.T) {
	t.Parallel()

	database, userID := setupRouterTestDB(t)
	defer database.Close()

	setClaudeKey(t, database, userID, "sk-ant-test-claude-key-1234567890")
	setGeminiKey(t, database, userID, "AIza-test-gemini-key-1234567890")

	router := NewProviderRouter(database)
	provider, err := router.GetAnyProvider(context.Background(), userID)
	if err != nil {
		t.Fatalf("expected provider, got %v", err)
	}
	if provider.Name() != string(ProviderTypeClaude) {
		t.Fatalf("expected claude provider, got %s", provider.Name())
	}
}

func TestGetAnyProvider_UsesSelectedProvider(t *testing.T) {
	t.Parallel()

	database, userID := setupRouterTestDB(t)
	defer database.Close()

	setClaudeKey(t, database, userID, "sk-ant-test-claude-key-1234567890")
	setGeminiKey(t, database, userID, "AIza-test-gemini-key-1234567890")
	setOpenAIKey(t, database, userID, "sk-test-openai-key-1234567890")

	if err := database.SetActiveAIProvider(userID, "chatgpt"); err != nil {
		t.Fatalf("failed to set active provider: %v", err)
	}

	router := NewProviderRouter(database)
	provider, err := router.GetAnyProvider(context.Background(), userID)
	if err != nil {
		t.Fatalf("expected provider, got %v", err)
	}
	if provider.Name() != string(ProviderTypeChatGPT) {
		t.Fatalf("expected chatgpt provider, got %s", provider.Name())
	}
}

func TestGetClaudeProvider_NotConfigured(t *testing.T) {
	t.Parallel()

	database, userID := setupRouterTestDB(t)
	defer database.Close()

	router := NewProviderRouter(database)
	_, err := router.GetClaudeProvider(context.Background(), userID)
	if err != ErrClaudeNotConfigured {
		t.Fatalf("expected ErrClaudeNotConfigured, got %v", err)
	}
}

func TestInvalidateClaudeClient(t *testing.T) {
	t.Parallel()

	database, userID := setupRouterTestDB(t)
	defer database.Close()

	setClaudeKey(t, database, userID, "sk-ant-test-claude-key-1234567890")
	router := NewProviderRouter(database)

	first, err := router.getClaudeClient(userID)
	if err != nil {
		t.Fatalf("expected client, got %v", err)
	}
	router.InvalidateClaudeClient(userID)
	second, err := router.getClaudeClient(userID)
	if err != nil {
		t.Fatalf("expected client, got %v", err)
	}
	if first == second {
		t.Fatalf("expected new client after invalidation")
	}
}
