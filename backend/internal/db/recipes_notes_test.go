package db

import "testing"

func TestCreateRecipeNote(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	note, err := db.CreateRecipeNote(userID, "Pasta Carbonara", "Instructions here", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	if note.NoteType != NoteTypeRecipe {
		t.Errorf("Expected note_type 'recipe', got '%s'", note.NoteType)
	}
	if note.FolderPath != "/Rezepte" {
		t.Errorf("Expected folder_path '/Rezepte', got '%s'", note.FolderPath)
	}

	// Verify metadata was auto-created (I6)
	meta, err := db.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeMetadata failed: %v", err)
	}
	if meta == nil {
		t.Fatal("Expected metadata to be auto-created, got nil")
	}
	if meta.Servings != 4 {
		t.Errorf("Expected default servings 4, got %d", meta.Servings)
	}
	if meta.UserID != userID {
		t.Errorf("Expected user_id %d, got %d", userID, meta.UserID)
	}
}

func TestCreateRecipeNote_AutoCreatesFolder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Folder should be created automatically
	_, err := db.CreateRecipeNote(userID, "Test Recipe", "", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	// Verify folder exists
	folders, err := db.GetFolders(userID)
	if err != nil {
		t.Fatalf("GetFolders failed: %v", err)
	}
	found := false
	for _, f := range folders {
		if f == "/Rezepte" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected /Rezepte folder to be auto-created")
	}
}

func TestRecipeNotIncludedInListNotes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create a regular note and a recipe
	_, err := db.CreateNote(userID, "Regular Note", "Content", "/")
	if err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}
	_, err = db.CreateRecipeNote(userID, "Recipe Note", "Instructions", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	// ListNotes should NOT include recipe notes
	notes, _, err := db.ListNotes(userID, 50, "", ListNotesOptions{})
	if err != nil {
		t.Fatalf("ListNotes failed: %v", err)
	}
	for _, n := range notes {
		if n.NoteType == NoteTypeRecipe {
			t.Error("ListNotes should not include recipe notes")
		}
	}
	if len(notes) != 1 {
		t.Errorf("Expected 1 note, got %d", len(notes))
	}
}

func TestRecipeVisibleInRezepteFolder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	recipe, err := db.CreateRecipeNote(userID, "My Recipe", "Steps", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	notes, err := db.ListNotesByFolder(userID, "/Rezepte", "")
	if err != nil {
		t.Fatalf("ListNotesByFolder failed: %v", err)
	}

	if len(notes) != 1 {
		t.Fatalf("Expected 1 recipe, got %d", len(notes))
	}
	if notes[0].ID != recipe.ID {
		t.Errorf("Expected recipe ID %s, got %s", recipe.ID, notes[0].ID)
	}
}

func TestListRecipes_OwnerOnly(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user1 := 1
	user2 := 2
	createTestUser(t, db, user1)
	createTestUserWithName(t, db, user2, "user2")

	_, err := db.CreateRecipeNote(user1, "User1 Recipe", "", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}
	_, err = db.CreateRecipeNote(user2, "User2 Recipe", "", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	recipes, err := db.ListRecipes(user1)
	if err != nil {
		t.Fatalf("ListRecipes failed: %v", err)
	}
	if len(recipes) != 1 {
		t.Errorf("Expected 1 recipe for user1, got %d", len(recipes))
	}
}
