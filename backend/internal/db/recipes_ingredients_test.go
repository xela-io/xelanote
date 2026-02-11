package db

import (
	"sync"
	"testing"
)

func TestSetRecipeIngredients_Atomic(t *testing.T) {
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

	amt := 500.0
	ingredients := []RecipeIngredient{
		{Name: "Mehl", Amount: &amt, Scalable: true},
		{Name: "Salz", Scalable: false},
	}

	err = db.SetRecipeIngredients(note.ID, userID, ingredients, meta.UpdatedAt)
	if err != nil {
		t.Fatalf("SetRecipeIngredients failed: %v", err)
	}

	// Verify
	ings, err := db.GetRecipeIngredients(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeIngredients failed: %v", err)
	}
	if len(ings) != 2 {
		t.Fatalf("Expected 2 ingredients, got %d", len(ings))
	}
	if ings[0].Name != "Mehl" {
		t.Errorf("Expected first ingredient 'Mehl', got '%s'", ings[0].Name)
	}
	if ings[0].DisplayOrder != 0 {
		t.Errorf("Expected display_order 0, got %d", ings[0].DisplayOrder)
	}
	if ings[1].DisplayOrder != 1 {
		t.Errorf("Expected display_order 1, got %d", ings[1].DisplayOrder)
	}
}

func TestSetRecipeIngredients_ReplaceAll(t *testing.T) {
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

	// Set initial ingredients
	err = db.SetRecipeIngredients(note.ID, userID, []RecipeIngredient{
		{Name: "Mehl"},
		{Name: "Wasser"},
	}, meta.UpdatedAt)
	if err != nil {
		t.Fatalf("SetRecipeIngredients failed: %v", err)
	}

	// Get updated timestamp
	meta, err = db.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeMetadata failed: %v", err)
	}

	// Replace with different ingredients
	err = db.SetRecipeIngredients(note.ID, userID, []RecipeIngredient{
		{Name: "Butter"},
	}, meta.UpdatedAt)
	if err != nil {
		t.Fatalf("SetRecipeIngredients replace failed: %v", err)
	}

	ings, err := db.GetRecipeIngredients(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeIngredients failed: %v", err)
	}
	if len(ings) != 1 {
		t.Fatalf("Expected 1 ingredient after replace, got %d", len(ings))
	}
	if ings[0].Name != "Butter" {
		t.Errorf("Expected 'Butter', got '%s'", ings[0].Name)
	}
}

func TestSetRecipeIngredients_ConcurrentUpdate(t *testing.T) {
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

	// Both use the same timestamp
	staleTimestamp := meta.UpdatedAt

	// First update succeeds
	err = db.SetRecipeIngredients(note.ID, userID, []RecipeIngredient{
		{Name: "Mehl"},
	}, staleTimestamp)
	if err != nil {
		t.Fatalf("First SetRecipeIngredients failed: %v", err)
	}

	// Second update with stale timestamp fails
	err = db.SetRecipeIngredients(note.ID, userID, []RecipeIngredient{
		{Name: "Zucker"},
	}, staleTimestamp)
	if err != ErrVersionMismatch {
		t.Errorf("Expected ErrVersionMismatch, got %v", err)
	}
}

func TestSetRecipeIngredients_MetadataNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	note, err := db.CreateRecipeNote(userID, "Test", "", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	// Delete metadata
	db.DeleteRecipeData(note.ID)

	// SetRecipeIngredients should fail with metadata not found
	err = db.SetRecipeIngredients(note.ID, userID, []RecipeIngredient{
		{Name: "Mehl"},
	}, "anything")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// Test concurrent access (goroutine-based, verifies no data corruption)
func TestSetRecipeIngredients_ConcurrentAccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	note, err := db.CreateRecipeNote(userID, "Concurrent Test", "", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	meta, err := db.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeMetadata failed: %v", err)
	}

	// Two concurrent updates with same timestamp - one should fail
	var wg sync.WaitGroup
	errors := make([]error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errors[idx] = db.SetRecipeIngredients(note.ID, userID, []RecipeIngredient{
				{Name: "Ingredient from goroutine"},
			}, meta.UpdatedAt)
		}(i)
	}
	wg.Wait()

	// One should succeed, one should fail with version mismatch
	successes := 0
	for _, err := range errors {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Errorf("Expected exactly 1 success, got %d (errors: %v)", successes, errors)
	}
}
