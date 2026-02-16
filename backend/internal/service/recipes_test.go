package service

import (
	"fmt"
	"testing"

	"github.com/xela-io/xelanote/internal/db"
)

func TestRecipeService_CreateRecipeNote_FeatureDisabled(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "user1")

	if err := database.SetUserFeature(user.ID, "recipe", false, nil); err != nil {
		t.Fatalf("set feature: %v", err)
	}

	service := NewRecipeService(database, NewNoteService(database))
	if _, err := service.CreateRecipeNote(user.ID, "Title", "Content", "/"); err != ErrRecipeFeatureNotEnabled {
		t.Fatalf("expected ErrRecipeFeatureNotEnabled, got %v", err)
	}
}

func TestRecipeService_UpdateRecipeMetadata_EncryptedNote(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "user1")
	service := NewRecipeService(database, NewNoteService(database))

	note, err := service.CreateRecipeNote(user.ID, "Title", "Content", "/")
	if err != nil {
		t.Fatalf("create recipe note: %v", err)
	}
	if _, err := database.Exec(`UPDATE notes SET content_encrypted = 1 WHERE id = ?`, note.ID); err != nil {
		t.Fatalf("mark encrypted: %v", err)
	}

	meta := &db.RecipeMetadata{Servings: 2}
	if err := service.UpdateRecipeMetadata(user.ID, note.ID, meta, ""); err != ErrRecipeEncrypted {
		t.Fatalf("expected ErrRecipeEncrypted, got %v", err)
	}
}

func TestRecipeService_UpdateRecipeMetadata_ViewerForbidden(t *testing.T) {
	database := setupTestDB(t)
	owner := createTestUser(t, database, "owner")
	viewer := createTestUser(t, database, "viewer")
	service := NewRecipeService(database, NewNoteService(database))

	note, err := service.CreateRecipeNote(owner.ID, "Title", "Content", "/")
	if err != nil {
		t.Fatalf("create recipe note: %v", err)
	}
	if _, err := database.CreateNoteShare(owner.ID, note.ID, viewer.ID, "viewer"); err != nil {
		t.Fatalf("create share: %v", err)
	}

	meta := &db.RecipeMetadata{Servings: 2}
	if err := service.UpdateRecipeMetadata(viewer.ID, note.ID, meta, ""); err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestRecipeService_UpdateRecipeMetadata_EditorAllowed(t *testing.T) {
	database := setupTestDB(t)
	owner := createTestUser(t, database, "owner")
	editor := createTestUser(t, database, "editor")
	service := NewRecipeService(database, NewNoteService(database))

	note, err := service.CreateRecipeNote(owner.ID, "Title", "Content", "/")
	if err != nil {
		t.Fatalf("create recipe note: %v", err)
	}
	if _, err := database.CreateNoteShare(owner.ID, note.ID, editor.ID, "editor"); err != nil {
		t.Fatalf("create share: %v", err)
	}

	meta := &db.RecipeMetadata{Servings: 3}
	if err := service.UpdateRecipeMetadata(editor.ID, note.ID, meta, ""); err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	stored, err := database.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	if stored == nil || stored.Servings != 3 {
		t.Fatalf("expected servings updated, got %+v", stored)
	}
}

func TestRecipeService_SetRecipeIngredients_ValidatesInput(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "user1")
	service := NewRecipeService(database, NewNoteService(database))

	note, err := service.CreateRecipeNote(user.ID, "Title", "Content", "/")
	if err != nil {
		t.Fatalf("create recipe note: %v", err)
	}

	err = service.SetRecipeIngredients(user.ID, note.ID, []db.RecipeIngredient{
		{Name: ""},
	}, "")
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestRecipeService_AddRecipeImage_InvalidURL(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "user1")
	service := NewRecipeService(database, NewNoteService(database))

	note, err := service.CreateRecipeNote(user.ID, "Title", "Content", "/")
	if err != nil {
		t.Fatalf("create recipe note: %v", err)
	}

	if _, err := service.AddRecipeImage(user.ID, note.ID, "http://example.com/img.png", nil); err != ErrInvalidImageURL {
		t.Fatalf("expected ErrInvalidImageURL, got %v", err)
	}
}

// SEC-003: Cross-user upload URL must be rejected
func TestRecipeService_AddRecipeImage_CrossUserRejected(t *testing.T) {
	database := setupTestDB(t)
	owner := createTestUser(t, database, "owner")
	attacker := createTestUser(t, database, "attacker")
	svc := NewRecipeService(database, NewNoteService(database))

	note, err := svc.CreateRecipeNote(owner.ID, "My Recipe", "Instructions", "/")
	if err != nil {
		t.Fatalf("create recipe note: %v", err)
	}

	// Owner tries to reference attacker's upload — must fail
	crossUserURL := fmt.Sprintf("/api/uploads/%d/secret.jpg", attacker.ID)
	if _, err := svc.AddRecipeImage(owner.ID, note.ID, crossUserURL, nil); err != ErrForbidden {
		t.Fatalf("expected ErrForbidden for cross-user URL, got %v", err)
	}
}

// SEC-003: Own upload URL must succeed
func TestRecipeService_AddRecipeImage_OwnURLAllowed(t *testing.T) {
	database := setupTestDB(t)
	owner := createTestUser(t, database, "owner")
	svc := NewRecipeService(database, NewNoteService(database))

	note, err := svc.CreateRecipeNote(owner.ID, "My Recipe", "Instructions", "/")
	if err != nil {
		t.Fatalf("create recipe note: %v", err)
	}

	ownURL := fmt.Sprintf("/api/uploads/%d/photo.jpg", owner.ID)
	img, err := svc.AddRecipeImage(owner.ID, note.ID, ownURL, nil)
	if err != nil {
		t.Fatalf("expected success for own URL, got %v", err)
	}
	if img == nil {
		t.Fatal("expected non-nil image")
	}
}
