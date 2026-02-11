package db

import "testing"

func TestDeleteRecipeData(t *testing.T) {
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

	// Add some ingredients
	err = db.SetRecipeIngredients(note.ID, userID, []RecipeIngredient{
		{Name: "Mehl"},
	}, meta.UpdatedAt)
	if err != nil {
		t.Fatalf("SetRecipeIngredients failed: %v", err)
	}

	// Delete recipe data
	err = db.DeleteRecipeData(note.ID)
	if err != nil {
		t.Fatalf("DeleteRecipeData failed: %v", err)
	}

	// Verify all gone
	meta, err = db.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeMetadata failed: %v", err)
	}
	if meta != nil {
		t.Error("Expected nil metadata after delete")
	}

	ings, err := db.GetRecipeIngredients(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeIngredients failed: %v", err)
	}
	if len(ings) != 0 {
		t.Errorf("Expected 0 ingredients after delete, got %d", len(ings))
	}
}

func TestDeleteRecipeData_IncludesImages(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)
	note, _ := db.CreateRecipeNote(userID, "Test Recipe", "", "/Rezepte")

	// Add an image
	db.AddRecipeImage(note.ID, userID, "/api/uploads/1/test.jpg", nil)

	// DeleteRecipeData should also delete images
	err := db.DeleteRecipeData(note.ID)
	if err != nil {
		t.Fatalf("DeleteRecipeData failed: %v", err)
	}

	images, _ := db.GetRecipeImages(note.ID)
	if len(images) != 0 {
		t.Errorf("Expected 0 images after DeleteRecipeData, got %d", len(images))
	}
}
