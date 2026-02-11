package db

import "testing"

// --- Validation Tests ---

func TestRecipeValidation_Difficulty(t *testing.T) {
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

	// Invalid difficulty should fail at DB constraint level
	invalidDifficulty := "impossible"
	_, err = db.Exec(`UPDATE recipe_metadata SET difficulty = ? WHERE note_id = ?`,
		invalidDifficulty, note.ID)
	if err == nil {
		t.Error("Expected error for invalid difficulty, got nil")
	}

	// Valid difficulties should work
	for _, d := range []string{"easy", "medium", "hard"} {
		diff := d
		err := db.SetRecipeMetadata(note.ID, userID, &RecipeMetadata{
			Servings:   4,
			Difficulty: &diff,
		}, meta.UpdatedAt)
		if err != nil {
			t.Errorf("SetRecipeMetadata with difficulty '%s' failed: %v", d, err)
		}
		meta, _ = db.GetRecipeMetadata(note.ID)
	}
}

func TestRecipeValidation_Servings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Servings 0 should fail at DB constraint
	_, err := db.Exec(`
		INSERT INTO recipe_metadata (note_id, user_id, servings) VALUES ('fake', ?, 0)
	`, userID)
	if err == nil {
		t.Error("Expected error for servings=0, got nil")
	}

	// Servings 1000 should fail at DB constraint
	_, err = db.Exec(`
		INSERT INTO recipe_metadata (note_id, user_id, servings) VALUES ('fake2', ?, 1000)
	`, userID)
	if err == nil {
		t.Error("Expected error for servings=1000, got nil")
	}
}
