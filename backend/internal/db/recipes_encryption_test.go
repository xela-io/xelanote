package db

import "testing"

// --- Encryption Tests ---

func TestEncryptRecipe_DeletesMetadataAndIngredients(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	note, err := db.CreateRecipeNote(userID, "Encrypt Test", "content", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	meta, err := db.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeMetadata failed: %v", err)
	}

	// Add ingredients
	err = db.SetRecipeIngredients(note.ID, userID, []RecipeIngredient{
		{Name: "Mehl"},
		{Name: "Wasser"},
	}, meta.UpdatedAt)
	if err != nil {
		t.Fatalf("SetRecipeIngredients failed: %v", err)
	}

	// Encrypt the note
	_, err = db.UpdateEncryptedNote(
		userID, note.ID, "Encrypted Recipe", nil, false,
		[]byte("encrypted-content"), "wrapped-dek", "{}", "", note.Version,
	)
	if err != nil {
		t.Fatalf("UpdateEncryptedNote failed: %v", err)
	}

	// Simulate what the service layer does: delete recipe data
	err = db.DeleteRecipeData(note.ID)
	if err != nil {
		t.Fatalf("DeleteRecipeData failed: %v", err)
	}

	// Verify metadata and ingredients are gone
	meta, err = db.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeMetadata failed: %v", err)
	}
	if meta != nil {
		t.Error("Expected nil metadata after encryption")
	}

	ings, err := db.GetRecipeIngredients(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeIngredients failed: %v", err)
	}
	if len(ings) != 0 {
		t.Errorf("Expected 0 ingredients after encryption, got %d", len(ings))
	}
}

func TestDecryptRecipe_RestoresMetadataAndIngredients(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	note, err := db.CreateRecipeNote(userID, "Decrypt Test", "content", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	meta, err := db.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeMetadata failed: %v", err)
	}

	// Set custom metadata
	prepTime := 30
	difficulty := "hard"
	err = db.SetRecipeMetadata(note.ID, userID, &RecipeMetadata{
		Servings:        8,
		PrepTimeMinutes: &prepTime,
		Difficulty:      &difficulty,
	}, meta.UpdatedAt)
	if err != nil {
		t.Fatalf("SetRecipeMetadata failed: %v", err)
	}

	meta, err = db.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeMetadata after update failed: %v", err)
	}

	amt := 500.0
	err = db.SetRecipeIngredients(note.ID, userID, []RecipeIngredient{
		{Name: "Mehl", Amount: &amt, Scalable: true},
	}, meta.UpdatedAt)
	if err != nil {
		t.Fatalf("SetRecipeIngredients failed: %v", err)
	}

	// Encrypt + delete recipe data
	encNote, err := db.UpdateEncryptedNote(
		userID, note.ID, "Encrypted", nil, false,
		[]byte("enc"), "dek", "{}", "", note.Version,
	)
	if err != nil {
		t.Fatalf("UpdateEncryptedNote failed: %v", err)
	}
	db.DeleteRecipeData(note.ID)

	// Decrypt
	_, err = db.DecryptNote(userID, note.ID, "Decrypt Test", "content", encNote.Version)
	if err != nil {
		t.Fatalf("DecryptNote failed: %v", err)
	}

	// Restore recipe data (upsert without expected_updated_at)
	err = db.SetRecipeMetadata(note.ID, userID, &RecipeMetadata{
		Servings:        8,
		PrepTimeMinutes: &prepTime,
		Difficulty:      &difficulty,
	}, "")
	if err != nil {
		t.Fatalf("SetRecipeMetadata restore failed: %v", err)
	}

	restoredMeta, err := db.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeMetadata after restore failed: %v", err)
	}
	if restoredMeta == nil {
		t.Fatal("Expected metadata to be restored")
	}
	if restoredMeta.Servings != 8 {
		t.Errorf("Expected servings 8, got %d", restoredMeta.Servings)
	}
	if restoredMeta.Difficulty == nil || *restoredMeta.Difficulty != "hard" {
		t.Errorf("Expected difficulty 'hard', got %v", restoredMeta.Difficulty)
	}

	// Restore ingredients
	err = db.SetRecipeIngredients(note.ID, userID, []RecipeIngredient{
		{Name: "Mehl", Amount: &amt, Scalable: true},
	}, restoredMeta.UpdatedAt)
	if err != nil {
		t.Fatalf("SetRecipeIngredients restore failed: %v", err)
	}

	ings, err := db.GetRecipeIngredients(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeIngredients after restore failed: %v", err)
	}
	if len(ings) != 1 {
		t.Fatalf("Expected 1 ingredient after restore, got %d", len(ings))
	}
	if ings[0].Name != "Mehl" {
		t.Errorf("Expected 'Mehl', got '%s'", ings[0].Name)
	}
}

func TestDecryptRecipe_WithoutRecipeData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	note, err := db.CreateRecipeNote(userID, "No Data", "content", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	// Encrypt + delete recipe data
	encNote, err := db.UpdateEncryptedNote(
		userID, note.ID, "Encrypted", nil, false,
		[]byte("enc"), "dek", "{}", "", note.Version,
	)
	if err != nil {
		t.Fatalf("UpdateEncryptedNote failed: %v", err)
	}
	db.DeleteRecipeData(note.ID)

	// Decrypt WITHOUT restoring recipe data
	_, err = db.DecryptNote(userID, note.ID, "No Data", "content", encNote.Version)
	if err != nil {
		t.Fatalf("DecryptNote failed: %v", err)
	}

	// Metadata should be nil (I5 fallback)
	meta, err := db.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeMetadata failed: %v", err)
	}
	if meta != nil {
		t.Error("Expected nil metadata after decrypt without recipe data")
	}

	// Ingredients should be empty
	ings, err := db.GetRecipeIngredients(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeIngredients failed: %v", err)
	}
	if len(ings) != 0 {
		t.Errorf("Expected 0 ingredients, got %d", len(ings))
	}
}

func TestEncryptedRecipeGet_EncryptedNote(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	note, err := db.CreateRecipeNote(userID, "Test", "content", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	// Encrypt
	_, err = db.UpdateEncryptedNote(
		userID, note.ID, "Encrypted", nil, false,
		[]byte("enc"), "dek", "{}", "", note.Version,
	)
	if err != nil {
		t.Fatalf("UpdateEncryptedNote failed: %v", err)
	}

	// Verify note is encrypted
	encrypted, err := db.IsNoteEncrypted(note.ID)
	if err != nil {
		t.Fatalf("IsNoteEncrypted failed: %v", err)
	}
	if !encrypted {
		t.Error("Expected note to be encrypted")
	}
}
