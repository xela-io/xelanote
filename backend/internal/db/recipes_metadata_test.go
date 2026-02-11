package db

import "testing"

func TestRecipeMetadata_CRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	note, err := db.CreateRecipeNote(userID, "Test", "", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	// Read default metadata
	meta, err := db.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeMetadata failed: %v", err)
	}
	if meta.Servings != 4 {
		t.Errorf("Expected default servings 4, got %d", meta.Servings)
	}

	// Update metadata
	prepTime := 15
	difficulty := "medium"
	updatedMeta := &RecipeMetadata{
		Servings:        6,
		PrepTimeMinutes: &prepTime,
		Difficulty:      &difficulty,
	}
	err = db.SetRecipeMetadata(note.ID, userID, updatedMeta, meta.UpdatedAt)
	if err != nil {
		t.Fatalf("SetRecipeMetadata failed: %v", err)
	}

	// Read again
	meta, err = db.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeMetadata after update failed: %v", err)
	}
	if meta.Servings != 6 {
		t.Errorf("Expected servings 6, got %d", meta.Servings)
	}
	if meta.PrepTimeMinutes == nil || *meta.PrepTimeMinutes != 15 {
		t.Errorf("Expected prep_time 15, got %v", meta.PrepTimeMinutes)
	}
	if meta.Difficulty == nil || *meta.Difficulty != "medium" {
		t.Errorf("Expected difficulty 'medium', got %v", meta.Difficulty)
	}
}

func TestRecipeMetadata_OptimisticLock(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	note, err := db.CreateRecipeNote(userID, "Test", "", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	meta, err := db.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeMetadata failed: %v", err)
	}

	// First update succeeds
	err = db.SetRecipeMetadata(note.ID, userID, &RecipeMetadata{Servings: 2}, meta.UpdatedAt)
	if err != nil {
		t.Fatalf("First SetRecipeMetadata failed: %v", err)
	}

	// Second update with stale timestamp should fail
	err = db.SetRecipeMetadata(note.ID, userID, &RecipeMetadata{Servings: 8}, meta.UpdatedAt)
	if err != ErrVersionMismatch {
		t.Errorf("Expected ErrVersionMismatch, got %v", err)
	}
}

func TestRecipeMetadata_UpsertWhenMissing(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	note, err := db.CreateRecipeNote(userID, "Test", "", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	// Delete metadata (simulating encryption)
	db.DeleteRecipeData(note.ID)

	// Verify it's gone
	meta, err := db.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeMetadata failed: %v", err)
	}
	if meta != nil {
		t.Fatal("Expected metadata to be nil after delete")
	}

	// Upsert without expected_updated_at (decrypt fallback)
	err = db.SetRecipeMetadata(note.ID, userID, &RecipeMetadata{Servings: 6}, "")
	if err != nil {
		t.Fatalf("SetRecipeMetadata upsert failed: %v", err)
	}

	meta, err = db.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeMetadata after upsert failed: %v", err)
	}
	if meta == nil || meta.Servings != 6 {
		t.Errorf("Expected servings 6 after upsert, got %v", meta)
	}
}
