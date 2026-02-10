package db

import (
	"math"
	"sync"
	"testing"
)

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
	notes, _, err := db.ListNotes(userID, 50, "")
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

	notes, err := db.ListNotesByFolder(userID, "/Rezepte")
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

func TestRecipeCollections_CRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create
	desc := "My first cookbook"
	color := "#ff0000"
	coll, err := db.CreateRecipeCollection(userID, "Favorites", &desc, &color)
	if err != nil {
		t.Fatalf("CreateRecipeCollection failed: %v", err)
	}
	if coll.Name != "Favorites" {
		t.Errorf("Expected name 'Favorites', got '%s'", coll.Name)
	}

	// List
	colls, err := db.ListRecipeCollections(userID)
	if err != nil {
		t.Fatalf("ListRecipeCollections failed: %v", err)
	}
	if len(colls) != 1 {
		t.Fatalf("Expected 1 collection, got %d", len(colls))
	}

	// Update
	newDesc := "Updated"
	err = db.UpdateRecipeCollection(userID, coll.ID, "Updated Name", &newDesc, nil)
	if err != nil {
		t.Fatalf("UpdateRecipeCollection failed: %v", err)
	}

	colls, err = db.ListRecipeCollections(userID)
	if err != nil {
		t.Fatalf("ListRecipeCollections failed: %v", err)
	}
	if colls[0].Name != "Updated Name" {
		t.Errorf("Expected updated name, got '%s'", colls[0].Name)
	}

	// Delete
	err = db.DeleteRecipeCollection(userID, coll.ID)
	if err != nil {
		t.Fatalf("DeleteRecipeCollection failed: %v", err)
	}

	colls, err = db.ListRecipeCollections(userID)
	if err != nil {
		t.Fatalf("ListRecipeCollections after delete failed: %v", err)
	}
	if len(colls) != 0 {
		t.Errorf("Expected 0 collections after delete, got %d", len(colls))
	}
}

func TestRecipeCollectionItems(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	recipe, err := db.CreateRecipeNote(userID, "Test Recipe", "", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	coll, err := db.CreateRecipeCollection(userID, "Favorites", nil, nil)
	if err != nil {
		t.Fatalf("CreateRecipeCollection failed: %v", err)
	}

	// Add recipe to collection
	err = db.AddRecipeToCollection(userID, coll.ID, recipe.ID)
	if err != nil {
		t.Fatalf("AddRecipeToCollection failed: %v", err)
	}

	// List recipes in collection
	recipes, err := db.ListRecipesInCollection(userID, coll.ID)
	if err != nil {
		t.Fatalf("ListRecipesInCollection failed: %v", err)
	}
	if len(recipes) != 1 {
		t.Fatalf("Expected 1 recipe, got %d", len(recipes))
	}

	// Check recipe count in collection
	colls, err := db.ListRecipeCollections(userID)
	if err != nil {
		t.Fatalf("ListRecipeCollections failed: %v", err)
	}
	if colls[0].RecipeCount != 1 {
		t.Errorf("Expected recipe_count 1, got %d", colls[0].RecipeCount)
	}

	// Check collections for recipe
	rColls, err := db.GetCollectionsForRecipe(userID, recipe.ID)
	if err != nil {
		t.Fatalf("GetCollectionsForRecipe failed: %v", err)
	}
	if len(rColls) != 1 {
		t.Fatalf("Expected 1 collection for recipe, got %d", len(rColls))
	}

	// Remove recipe from collection
	err = db.RemoveRecipeFromCollection(userID, coll.ID, recipe.ID)
	if err != nil {
		t.Fatalf("RemoveRecipeFromCollection failed: %v", err)
	}

	recipes, err = db.ListRecipesInCollection(userID, coll.ID)
	if err != nil {
		t.Fatalf("ListRecipesInCollection after remove failed: %v", err)
	}
	if len(recipes) != 0 {
		t.Errorf("Expected 0 recipes after remove, got %d", len(recipes))
	}
}

func TestCollections_OwnerOnly(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user1 := 1
	user2 := 2
	createTestUser(t, db, user1)
	createTestUserWithName(t, db, user2, "user2")

	_, err := db.CreateRecipeCollection(user1, "User1 Cookbook", nil, nil)
	if err != nil {
		t.Fatalf("CreateRecipeCollection failed: %v", err)
	}

	// User2 should see no collections
	colls, err := db.ListRecipeCollections(user2)
	if err != nil {
		t.Fatalf("ListRecipeCollections failed: %v", err)
	}
	if len(colls) != 0 {
		t.Errorf("Expected 0 collections for user2, got %d", len(colls))
	}
}

// --- Scaling Tests ---

func TestScaleIngredients(t *testing.T) {
	amt := 500.0
	ingredients := []RecipeIngredient{
		{Name: "Mehl", Amount: &amt, Scalable: true},
		{Name: "Salz", Scalable: false},
		{Name: "Wasser", Amount: nil, Scalable: true}, // nil amount
	}

	result := ScaleIngredients(ingredients, 4, 8)
	if len(result) != 3 {
		t.Fatalf("Expected 3 scaled ingredients, got %d", len(result))
	}

	// Scalable with amount
	if result[0].ScaledAmount == nil || *result[0].ScaledAmount != 1000.0 {
		t.Errorf("Expected scaled amount 1000, got %v", result[0].ScaledAmount)
	}
	if result[0].DisplayAmount != "1000" {
		t.Errorf("Expected display '1000', got '%s'", result[0].DisplayAmount)
	}

	// Not scalable
	if result[1].ScaledAmount != nil {
		t.Errorf("Expected nil scaled amount for non-scalable, got %v", result[1].ScaledAmount)
	}

	// Nil amount (scalable but no amount)
	if result[2].ScaledAmount != nil {
		t.Errorf("Expected nil scaled amount for nil amount, got %v", result[2].ScaledAmount)
	}
	if result[2].DisplayAmount != "" {
		t.Errorf("Expected empty display for nil amount, got '%s'", result[2].DisplayAmount)
	}
}

func TestScaleIngredients_Rounding(t *testing.T) {
	amt := 100.0
	ingredients := []RecipeIngredient{
		{Name: "Test", Amount: &amt, Scalable: true},
	}

	// 100 * 3/4 = 75 (exact)
	result := ScaleIngredients(ingredients, 4, 3)
	if *result[0].ScaledAmount != 75.0 {
		t.Errorf("Expected 75, got %v", *result[0].ScaledAmount)
	}

	// 100 * 1/3 = 33.33 (rounded to 2 decimals)
	result = ScaleIngredients(ingredients, 3, 1)
	expected := math.Round(100.0/3.0*100) / 100
	if *result[0].ScaledAmount != expected {
		t.Errorf("Expected %v, got %v", expected, *result[0].ScaledAmount)
	}
}

func TestScaleIngredients_DisplayAmount(t *testing.T) {
	tests := []struct {
		value    float64
		expected string
	}{
		{500.0, "500"},
		{2.5, "2.5"},
		{1.33, "1.33"},
		{0.0, "0"},
		{10.0, "10"},
	}

	for _, tt := range tests {
		result := FormatDisplayAmount(tt.value)
		if result != tt.expected {
			t.Errorf("FormatDisplayAmount(%v) = '%s', expected '%s'", tt.value, result, tt.expected)
		}
	}
}

func TestScaleIngredients_AmountTextIgnored(t *testing.T) {
	amt := 100.0
	text := "ca. 100"
	ingredients := []RecipeIngredient{
		{Name: "Mehl", Amount: &amt, AmountText: &text, Scalable: true},
	}

	// When scaling, amount_text should be ignored
	result := ScaleIngredients(ingredients, 4, 8)
	if result[0].DisplayAmount != "200" {
		t.Errorf("Expected '200' (amount_text ignored during scaling), got '%s'", result[0].DisplayAmount)
	}
}

func TestScaleIngredients_AmountNilAmountTextSet(t *testing.T) {
	text := "etwas"
	ingredients := []RecipeIngredient{
		{Name: "Salz", Amount: nil, AmountText: &text, Scalable: true},
	}

	result := ScaleIngredients(ingredients, 4, 8)
	if result[0].DisplayAmount != "" {
		t.Errorf("Expected empty display for nil amount, got '%s'", result[0].DisplayAmount)
	}
}

func TestFormatAmount_OriginalServings(t *testing.T) {
	amt := 100.0
	text := "ca. 100"

	// When not scaling (original servings), amount_text is used
	result := FormatAmount(&amt, &text)
	if result != "ca. 100" {
		t.Errorf("Expected 'ca. 100', got '%s'", result)
	}

	// Without amount_text, numeric format
	result = FormatAmount(&amt, nil)
	if result != "100" {
		t.Errorf("Expected '100', got '%s'", result)
	}

	// Nil amount
	result = FormatAmount(nil, &text)
	if result != "" {
		t.Errorf("Expected empty string for nil amount, got '%s'", result)
	}
}

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

// --- Sharing Tests ---

func TestShareRecipeNote(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner := 1
	viewer := 2
	createTestUser(t, db, owner)
	createTestUserWithName(t, db, viewer, "viewer")

	note, err := db.CreateRecipeNote(owner, "Shared Recipe", "content", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	// Share the note
	_, err = db.Exec(`
		INSERT INTO note_shares (note_id, owner_user_id, shared_with_user_id, role, created_at, updated_at)
		VALUES (?, ?, ?, 'viewer', datetime('now'), datetime('now'))
	`, note.ID, owner, viewer)
	if err != nil {
		t.Fatalf("Share creation failed: %v", err)
	}

	// Viewer should have permission
	perm, err := db.GetSharePermission(viewer, note.ID)
	if err != nil {
		t.Fatalf("GetSharePermission failed: %v", err)
	}
	if perm != "viewer" {
		t.Errorf("Expected 'viewer' permission, got '%s'", perm)
	}
}

func TestExistingShareThenEncrypt_SharesRemoved(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner := 1
	viewer := 2
	createTestUser(t, db, owner)
	createTestUserWithName(t, db, viewer, "viewer")

	note, err := db.CreateRecipeNote(owner, "Share Then Encrypt", "content", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	// Share the note
	_, err = db.Exec(`
		INSERT INTO note_shares (note_id, owner_user_id, shared_with_user_id, role, created_at, updated_at)
		VALUES (?, ?, ?, 'editor', datetime('now'), datetime('now'))
	`, note.ID, owner, viewer)
	if err != nil {
		t.Fatalf("Share creation failed: %v", err)
	}

	// Encrypt the note
	_, err = db.UpdateEncryptedNote(
		owner, note.ID, "Encrypted", nil, false,
		[]byte("enc"), "dek", "{}", "", note.Version,
	)
	if err != nil {
		t.Fatalf("UpdateEncryptedNote failed: %v", err)
	}

	// Delete shares (as service layer does)
	err = db.DeleteAllSharesForNote(note.ID)
	if err != nil {
		t.Fatalf("DeleteAllSharesForNote failed: %v", err)
	}

	// Delete recipe data (as service layer does)
	err = db.DeleteRecipeData(note.ID)
	if err != nil {
		t.Fatalf("DeleteRecipeData failed: %v", err)
	}

	// Verify: no more shares
	perm, err := db.GetSharePermission(viewer, note.ID)
	if err != nil {
		t.Fatalf("GetSharePermission failed: %v", err)
	}
	if perm != "" {
		t.Errorf("Expected empty permission after encryption, got '%s'", perm)
	}

	// Verify: recipe data gone
	meta, _ := db.GetRecipeMetadata(note.ID)
	if meta != nil {
		t.Error("Expected nil metadata after encryption")
	}
}

// === Collection Sharing Tests ===

func TestCollectionShare_CRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner := 1
	viewer := 2
	createTestUser(t, db, owner)
	createTestUserWithName(t, db, viewer, "viewer")

	coll, err := db.CreateRecipeCollection(owner, "Shared Cookbook", nil, nil)
	if err != nil {
		t.Fatalf("CreateRecipeCollection failed: %v", err)
	}

	// Create share
	share, err := db.CreateCollectionShare(owner, coll.ID, viewer, "viewer")
	if err != nil {
		t.Fatalf("CreateCollectionShare failed: %v", err)
	}
	if share.Role != "viewer" {
		t.Errorf("Expected role 'viewer', got '%s'", share.Role)
	}

	// Get shares
	shares, err := db.GetCollectionShares(owner, coll.ID)
	if err != nil {
		t.Fatalf("GetCollectionShares failed: %v", err)
	}
	if len(shares) != 1 {
		t.Fatalf("Expected 1 share, got %d", len(shares))
	}
	if shares[0].SharedWithUsername != "viewer" {
		t.Errorf("Expected shared_with_username 'viewer', got '%s'", shares[0].SharedWithUsername)
	}

	// Update role
	err = db.UpdateCollectionShareRole(owner, coll.ID, viewer, "editor")
	if err != nil {
		t.Fatalf("UpdateCollectionShareRole failed: %v", err)
	}
	shares, _ = db.GetCollectionShares(owner, coll.ID)
	if shares[0].Role != "editor" {
		t.Errorf("Expected updated role 'editor', got '%s'", shares[0].Role)
	}

	// Delete share
	err = db.DeleteCollectionShare(owner, coll.ID, viewer)
	if err != nil {
		t.Fatalf("DeleteCollectionShare failed: %v", err)
	}
	shares, _ = db.GetCollectionShares(owner, coll.ID)
	if len(shares) != 0 {
		t.Errorf("Expected 0 shares after delete, got %d", len(shares))
	}
}

func TestCollectionShare_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner := 1
	viewer := 2
	createTestUser(t, db, owner)
	createTestUserWithName(t, db, viewer, "viewer")

	coll, err := db.CreateRecipeCollection(owner, "Test", nil, nil)
	if err != nil {
		t.Fatalf("CreateRecipeCollection failed: %v", err)
	}

	_, err = db.CreateCollectionShare(owner, coll.ID, viewer, "viewer")
	if err != nil {
		t.Fatalf("First CreateCollectionShare failed: %v", err)
	}

	// Duplicate should fail
	_, err = db.CreateCollectionShare(owner, coll.ID, viewer, "editor")
	if err != ErrDuplicate {
		t.Errorf("Expected ErrDuplicate, got %v", err)
	}
}

func TestGetSharedCollectionsForUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner := 1
	viewer := 2
	createTestUser(t, db, owner)
	createTestUserWithName(t, db, viewer, "viewer")

	coll, err := db.CreateRecipeCollection(owner, "Shared Cookbook", nil, nil)
	if err != nil {
		t.Fatalf("CreateRecipeCollection failed: %v", err)
	}

	// Add a recipe to get recipe_count > 0
	recipe, err := db.CreateRecipeNote(owner, "Test Recipe", "", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}
	err = db.AddRecipeToCollection(owner, coll.ID, recipe.ID)
	if err != nil {
		t.Fatalf("AddRecipeToCollection failed: %v", err)
	}

	// Share collection
	_, err = db.CreateCollectionShare(owner, coll.ID, viewer, "editor")
	if err != nil {
		t.Fatalf("CreateCollectionShare failed: %v", err)
	}

	// Viewer should see the shared collection
	shared, err := db.GetSharedCollectionsForUser(viewer)
	if err != nil {
		t.Fatalf("GetSharedCollectionsForUser failed: %v", err)
	}
	if len(shared) != 1 {
		t.Fatalf("Expected 1 shared collection, got %d", len(shared))
	}
	if shared[0].Name != "Shared Cookbook" {
		t.Errorf("Expected name 'Shared Cookbook', got '%s'", shared[0].Name)
	}
	if shared[0].ShareRole != "editor" {
		t.Errorf("Expected share_role 'editor', got '%s'", shared[0].ShareRole)
	}
	if shared[0].RecipeCount != 1 {
		t.Errorf("Expected recipe_count 1, got %d", shared[0].RecipeCount)
	}

	// Owner should NOT see it as shared-with-me
	ownerShared, err := db.GetSharedCollectionsForUser(owner)
	if err != nil {
		t.Fatalf("GetSharedCollectionsForUser for owner failed: %v", err)
	}
	if len(ownerShared) != 0 {
		t.Errorf("Expected 0 shared collections for owner, got %d", len(ownerShared))
	}
}

func TestCollectionSharePermission(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner := 1
	viewer := 2
	stranger := 3
	createTestUser(t, db, owner)
	createTestUserWithName(t, db, viewer, "viewer")
	createTestUserWithName(t, db, stranger, "stranger")

	coll, err := db.CreateRecipeCollection(owner, "Test", nil, nil)
	if err != nil {
		t.Fatalf("CreateRecipeCollection failed: %v", err)
	}

	_, err = db.CreateCollectionShare(owner, coll.ID, viewer, "editor")
	if err != nil {
		t.Fatalf("CreateCollectionShare failed: %v", err)
	}

	// Viewer has permission
	perm, err := db.GetCollectionSharePermission(viewer, coll.ID)
	if err != nil {
		t.Fatalf("GetCollectionSharePermission failed: %v", err)
	}
	if perm != "editor" {
		t.Errorf("Expected 'editor', got '%s'", perm)
	}

	// Stranger has no permission
	perm, err = db.GetCollectionSharePermission(stranger, coll.ID)
	if err != nil {
		t.Fatalf("GetCollectionSharePermission failed: %v", err)
	}
	if perm != "" {
		t.Errorf("Expected empty permission for stranger, got '%s'", perm)
	}
}

func TestListRecipesInSharedCollection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner := 1
	createTestUser(t, db, owner)

	coll, err := db.CreateRecipeCollection(owner, "Test", nil, nil)
	if err != nil {
		t.Fatalf("CreateRecipeCollection failed: %v", err)
	}

	recipe, err := db.CreateRecipeNote(owner, "My Recipe", "steps", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	err = db.AddRecipeToCollection(owner, coll.ID, recipe.ID)
	if err != nil {
		t.Fatalf("AddRecipeToCollection failed: %v", err)
	}

	// ListRecipesInSharedCollection has no user_id filter
	items, err := db.ListRecipesInSharedCollection(coll.ID)
	if err != nil {
		t.Fatalf("ListRecipesInSharedCollection failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 recipe, got %d", len(items))
	}
	if items[0].ID != recipe.ID {
		t.Errorf("Expected recipe ID %s, got %s", recipe.ID, items[0].ID)
	}
}

func TestCollectionHasEncryptedRecipes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner := 1
	createTestUser(t, db, owner)

	coll, err := db.CreateRecipeCollection(owner, "Test", nil, nil)
	if err != nil {
		t.Fatalf("CreateRecipeCollection failed: %v", err)
	}

	recipe, err := db.CreateRecipeNote(owner, "Plain Recipe", "content", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	err = db.AddRecipeToCollection(owner, coll.ID, recipe.ID)
	if err != nil {
		t.Fatalf("AddRecipeToCollection failed: %v", err)
	}

	// No encrypted recipes
	hasEncrypted, err := db.CollectionHasEncryptedRecipes(coll.ID)
	if err != nil {
		t.Fatalf("CollectionHasEncryptedRecipes failed: %v", err)
	}
	if hasEncrypted {
		t.Error("Expected no encrypted recipes")
	}

	// Encrypt the recipe
	_, err = db.UpdateEncryptedNote(
		owner, recipe.ID, "Encrypted", nil, false,
		[]byte("enc"), "dek", "{}", "", recipe.Version,
	)
	if err != nil {
		t.Fatalf("UpdateEncryptedNote failed: %v", err)
	}

	// Now should have encrypted recipes
	hasEncrypted, err = db.CollectionHasEncryptedRecipes(coll.ID)
	if err != nil {
		t.Fatalf("CollectionHasEncryptedRecipes failed: %v", err)
	}
	if !hasEncrypted {
		t.Error("Expected encrypted recipes after encryption")
	}
}

func TestGetSharePermission_CollectionShareBranch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner := 1
	viewer := 2
	createTestUser(t, db, owner)
	createTestUserWithName(t, db, viewer, "viewer")

	// Create collection with recipe
	coll, err := db.CreateRecipeCollection(owner, "Shared Cookbook", nil, nil)
	if err != nil {
		t.Fatalf("CreateRecipeCollection failed: %v", err)
	}

	recipe, err := db.CreateRecipeNote(owner, "Recipe In Collection", "content", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	err = db.AddRecipeToCollection(owner, coll.ID, recipe.ID)
	if err != nil {
		t.Fatalf("AddRecipeToCollection failed: %v", err)
	}

	// Share collection with viewer
	_, err = db.CreateCollectionShare(owner, coll.ID, viewer, "viewer")
	if err != nil {
		t.Fatalf("CreateCollectionShare failed: %v", err)
	}

	// GetSharePermission should find the collection share (3rd branch)
	perm, err := db.GetSharePermission(viewer, recipe.ID)
	if err != nil {
		t.Fatalf("GetSharePermission failed: %v", err)
	}
	if perm != "viewer" {
		t.Errorf("Expected 'viewer' via collection share, got '%s'", perm)
	}
}

func TestGetSharePermission_PriorityChain(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner := 1
	user := 2
	createTestUser(t, db, owner)
	createTestUserWithName(t, db, user, "user2")

	// Create collection with recipe
	coll, err := db.CreateRecipeCollection(owner, "Cookbook", nil, nil)
	if err != nil {
		t.Fatalf("CreateRecipeCollection failed: %v", err)
	}

	recipe, err := db.CreateRecipeNote(owner, "Recipe", "content", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	err = db.AddRecipeToCollection(owner, coll.ID, recipe.ID)
	if err != nil {
		t.Fatalf("AddRecipeToCollection failed: %v", err)
	}

	// Share via collection as viewer
	_, err = db.CreateCollectionShare(owner, coll.ID, user, "viewer")
	if err != nil {
		t.Fatalf("CreateCollectionShare failed: %v", err)
	}

	// Also share directly as editor (higher priority)
	_, err = db.Exec(`
		INSERT INTO note_shares (note_id, owner_user_id, shared_with_user_id, role, created_at, updated_at)
		VALUES (?, ?, ?, 'editor', datetime('now'), datetime('now'))
	`, recipe.ID, owner, user)
	if err != nil {
		t.Fatalf("Direct note share failed: %v", err)
	}

	// note_share (editor) should win over collection_share (viewer)
	perm, err := db.GetSharePermission(user, recipe.ID)
	if err != nil {
		t.Fatalf("GetSharePermission failed: %v", err)
	}
	if perm != "editor" {
		t.Errorf("Expected 'editor' (note_share priority), got '%s'", perm)
	}
}

func TestGetSharedRecipesForUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner := 1
	user := 2
	createTestUser(t, db, owner)
	createTestUserWithName(t, db, user, "user2")

	recipe, err := db.CreateRecipeNote(owner, "Shared Recipe", "content", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	// Share recipe directly
	_, err = db.Exec(`
		INSERT INTO note_shares (note_id, owner_user_id, shared_with_user_id, role, created_at, updated_at)
		VALUES (?, ?, ?, 'editor', datetime('now'), datetime('now'))
	`, recipe.ID, owner, user)
	if err != nil {
		t.Fatalf("Share creation failed: %v", err)
	}

	shared, err := db.GetSharedRecipesForUser(user)
	if err != nil {
		t.Fatalf("GetSharedRecipesForUser failed: %v", err)
	}
	if len(shared) != 1 {
		t.Fatalf("Expected 1 shared recipe, got %d", len(shared))
	}
	if shared[0].Title != "Shared Recipe" {
		t.Errorf("Expected title 'Shared Recipe', got '%s'", shared[0].Title)
	}
	if shared[0].NoteType != "recipe" {
		t.Errorf("Expected note_type 'recipe', got '%s'", shared[0].NoteType)
	}
	if shared[0].ShareRole != "editor" {
		t.Errorf("Expected share_role 'editor', got '%s'", shared[0].ShareRole)
	}
}

func TestGetSharedRecipesForUser_ViaCollection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner := 1
	user := 2
	createTestUser(t, db, owner)
	createTestUserWithName(t, db, user, "user2")

	coll, err := db.CreateRecipeCollection(owner, "Shared Cookbook", nil, nil)
	if err != nil {
		t.Fatalf("CreateRecipeCollection failed: %v", err)
	}

	recipe, err := db.CreateRecipeNote(owner, "Collection Recipe", "content", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	err = db.AddRecipeToCollection(owner, coll.ID, recipe.ID)
	if err != nil {
		t.Fatalf("AddRecipeToCollection failed: %v", err)
	}

	_, err = db.CreateCollectionShare(owner, coll.ID, user, "viewer")
	if err != nil {
		t.Fatalf("CreateCollectionShare failed: %v", err)
	}

	shared, err := db.GetSharedRecipesForUser(user)
	if err != nil {
		t.Fatalf("GetSharedRecipesForUser failed: %v", err)
	}
	if len(shared) != 1 {
		t.Fatalf("Expected 1 shared recipe via collection, got %d", len(shared))
	}
	if shared[0].Title != "Collection Recipe" {
		t.Errorf("Expected title 'Collection Recipe', got '%s'", shared[0].Title)
	}
}

func TestGetSharedRecipesForUser_Dedup(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner := 1
	user := 2
	createTestUser(t, db, owner)
	createTestUserWithName(t, db, user, "user2")

	coll, err := db.CreateRecipeCollection(owner, "Cookbook", nil, nil)
	if err != nil {
		t.Fatalf("CreateRecipeCollection failed: %v", err)
	}

	recipe, err := db.CreateRecipeNote(owner, "Dedup Recipe", "content", "/Rezepte")
	if err != nil {
		t.Fatalf("CreateRecipeNote failed: %v", err)
	}

	err = db.AddRecipeToCollection(owner, coll.ID, recipe.ID)
	if err != nil {
		t.Fatalf("AddRecipeToCollection failed: %v", err)
	}

	// Share via both note_share AND collection_share
	_, err = db.Exec(`
		INSERT INTO note_shares (note_id, owner_user_id, shared_with_user_id, role, created_at, updated_at)
		VALUES (?, ?, ?, 'editor', datetime('now'), datetime('now'))
	`, recipe.ID, owner, user)
	if err != nil {
		t.Fatalf("Note share creation failed: %v", err)
	}

	_, err = db.CreateCollectionShare(owner, coll.ID, user, "viewer")
	if err != nil {
		t.Fatalf("CreateCollectionShare failed: %v", err)
	}

	// Should get exactly 1 entry (dedup), with the higher-priority role (editor from note_share)
	shared, err := db.GetSharedRecipesForUser(user)
	if err != nil {
		t.Fatalf("GetSharedRecipesForUser failed: %v", err)
	}
	if len(shared) != 1 {
		t.Fatalf("Expected 1 shared recipe (dedup), got %d", len(shared))
	}
	if shared[0].ShareRole != "editor" {
		t.Errorf("Expected 'editor' (highest priority wins), got '%s'", shared[0].ShareRole)
	}
}

func TestGetSharedRecipesForUser_ExcludesNonRecipes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner := 1
	user := 2
	createTestUser(t, db, owner)
	createTestUserWithName(t, db, user, "user2")

	// Create a regular note (not recipe)
	regularNote, err := db.CreateNote(owner, "Regular Note", "content", "/")
	if err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}

	// Share it
	_, err = db.Exec(`
		INSERT INTO note_shares (note_id, owner_user_id, shared_with_user_id, role, created_at, updated_at)
		VALUES (?, ?, ?, 'viewer', datetime('now'), datetime('now'))
	`, regularNote.ID, owner, user)
	if err != nil {
		t.Fatalf("Share creation failed: %v", err)
	}

	// GetSharedRecipesForUser should NOT include regular notes
	shared, err := db.GetSharedRecipesForUser(user)
	if err != nil {
		t.Fatalf("GetSharedRecipesForUser failed: %v", err)
	}
	if len(shared) != 0 {
		t.Errorf("Expected 0 shared recipes (only regular notes shared), got %d", len(shared))
	}
}

func TestCollectionShareCascadeOnDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner := 1
	viewer := 2
	createTestUser(t, db, owner)
	createTestUserWithName(t, db, viewer, "viewer")

	coll, err := db.CreateRecipeCollection(owner, "To Delete", nil, nil)
	if err != nil {
		t.Fatalf("CreateRecipeCollection failed: %v", err)
	}

	_, err = db.CreateCollectionShare(owner, coll.ID, viewer, "viewer")
	if err != nil {
		t.Fatalf("CreateCollectionShare failed: %v", err)
	}

	// Delete collection should cascade to shares
	err = db.DeleteRecipeCollection(owner, coll.ID)
	if err != nil {
		t.Fatalf("DeleteRecipeCollection failed: %v", err)
	}

	// Viewer should no longer see it
	shared, err := db.GetSharedCollectionsForUser(viewer)
	if err != nil {
		t.Fatalf("GetSharedCollectionsForUser failed: %v", err)
	}
	if len(shared) != 0 {
		t.Errorf("Expected 0 shared collections after cascade delete, got %d", len(shared))
	}
}

// Helper for creating a second test user with a unique username
func createTestUserWithName(t *testing.T, db *DB, userID int, username string) {
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (?, ?, ?, 'hash', datetime('now'))
	`, userID, username, username+"@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user %s: %v", username, err)
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

// --- Recipe Image Tests ---

func TestGetRecipeImages_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)
	note, _ := db.CreateRecipeNote(userID, "Test Recipe", "", "/Rezepte")

	images, err := db.GetRecipeImages(note.ID)
	if err != nil {
		t.Fatalf("GetRecipeImages failed: %v", err)
	}
	if len(images) != 0 {
		t.Errorf("Expected 0 images, got %d", len(images))
	}
}

func TestAddRecipeImage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)
	note, _ := db.CreateRecipeNote(userID, "Test Recipe", "", "/Rezepte")

	// Get initial metadata updated_at
	metaBefore, _ := db.GetRecipeMetadata(note.ID)

	caption := "My photo"
	img, err := db.AddRecipeImage(note.ID, userID, "/api/uploads/1/test.jpg", &caption)
	if err != nil {
		t.Fatalf("AddRecipeImage failed: %v", err)
	}
	if img.DisplayOrder != 0 {
		t.Errorf("Expected display_order 0, got %d", img.DisplayOrder)
	}
	if img.ImageURL != "/api/uploads/1/test.jpg" {
		t.Errorf("Expected image_url '/api/uploads/1/test.jpg', got '%s'", img.ImageURL)
	}
	if img.Caption == nil || *img.Caption != "My photo" {
		t.Errorf("Expected caption 'My photo', got %v", img.Caption)
	}

	// Second image should have display_order = 1
	img2, err := db.AddRecipeImage(note.ID, userID, "/api/uploads/1/test2.jpg", nil)
	if err != nil {
		t.Fatalf("AddRecipeImage second failed: %v", err)
	}
	if img2.DisplayOrder != 1 {
		t.Errorf("Expected display_order 1, got %d", img2.DisplayOrder)
	}

	// Verify metadata.updated_at was bumped
	metaAfter, _ := db.GetRecipeMetadata(note.ID)
	if metaBefore != nil && metaAfter != nil && metaBefore.UpdatedAt == metaAfter.UpdatedAt {
		t.Error("Expected metadata.updated_at to be bumped after adding image")
	}

	// Verify all images
	images, _ := db.GetRecipeImages(note.ID)
	if len(images) != 2 {
		t.Fatalf("Expected 2 images, got %d", len(images))
	}
}

func TestDeleteRecipeImage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)
	note, _ := db.CreateRecipeNote(userID, "Test Recipe", "", "/Rezepte")

	img, _ := db.AddRecipeImage(note.ID, userID, "/api/uploads/1/test.jpg", nil)

	err := db.DeleteRecipeImage(img.ID)
	if err != nil {
		t.Fatalf("DeleteRecipeImage failed: %v", err)
	}

	images, _ := db.GetRecipeImages(note.ID)
	if len(images) != 0 {
		t.Errorf("Expected 0 images after delete, got %d", len(images))
	}

	// Delete non-existent should return ErrNotFound
	err = db.DeleteRecipeImage(99999)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestUpdateRecipeImageCaption(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)
	note, _ := db.CreateRecipeNote(userID, "Test Recipe", "", "/Rezepte")

	img, _ := db.AddRecipeImage(note.ID, userID, "/api/uploads/1/test.jpg", nil)

	newCaption := "Updated caption"
	err := db.UpdateRecipeImageCaption(img.ID, &newCaption)
	if err != nil {
		t.Fatalf("UpdateRecipeImageCaption failed: %v", err)
	}

	updated, _ := db.GetRecipeImage(img.ID)
	if updated.Caption == nil || *updated.Caption != "Updated caption" {
		t.Errorf("Expected caption 'Updated caption', got %v", updated.Caption)
	}

	// Set caption to nil
	err = db.UpdateRecipeImageCaption(img.ID, nil)
	if err != nil {
		t.Fatalf("UpdateRecipeImageCaption to nil failed: %v", err)
	}
	updated, _ = db.GetRecipeImage(img.ID)
	if updated.Caption != nil {
		t.Errorf("Expected nil caption, got %v", updated.Caption)
	}
}

func TestReorderRecipeImages(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)
	note, _ := db.CreateRecipeNote(userID, "Test Recipe", "", "/Rezepte")

	img1, _ := db.AddRecipeImage(note.ID, userID, "/api/uploads/1/a.jpg", nil)
	img2, _ := db.AddRecipeImage(note.ID, userID, "/api/uploads/1/b.jpg", nil)
	img3, _ := db.AddRecipeImage(note.ID, userID, "/api/uploads/1/c.jpg", nil)

	// Reverse order: 3, 1, 2
	err := db.ReorderRecipeImages(note.ID, []int{img3.ID, img1.ID, img2.ID})
	if err != nil {
		t.Fatalf("ReorderRecipeImages failed: %v", err)
	}

	images, _ := db.GetRecipeImages(note.ID)
	if len(images) != 3 {
		t.Fatalf("Expected 3 images, got %d", len(images))
	}
	// Should be ordered: img3(0), img1(1), img2(2)
	if images[0].ID != img3.ID {
		t.Errorf("Expected first image to be img3 (id=%d), got id=%d", img3.ID, images[0].ID)
	}
	if images[1].ID != img1.ID {
		t.Errorf("Expected second image to be img1 (id=%d), got id=%d", img1.ID, images[1].ID)
	}
	if images[2].ID != img2.ID {
		t.Errorf("Expected third image to be img2 (id=%d), got id=%d", img2.ID, images[2].ID)
	}
	// Verify display_order is 0, 1, 2
	for i, img := range images {
		if img.DisplayOrder != i {
			t.Errorf("Image %d: expected display_order %d, got %d", img.ID, i, img.DisplayOrder)
		}
	}
}

func TestReorderRecipeImages_InvalidIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)
	note, _ := db.CreateRecipeNote(userID, "Test Recipe", "", "/Rezepte")

	img, _ := db.AddRecipeImage(note.ID, userID, "/api/uploads/1/a.jpg", nil)

	// Include an ID that doesn't belong to this recipe
	err := db.ReorderRecipeImages(note.ID, []int{img.ID, 99999})
	if err == nil {
		t.Error("Expected error for invalid IDs, got nil")
	}
}

func TestReorderRecipeImages_DuplicateIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)
	note, _ := db.CreateRecipeNote(userID, "Test Recipe", "", "/Rezepte")

	img, _ := db.AddRecipeImage(note.ID, userID, "/api/uploads/1/a.jpg", nil)

	err := db.ReorderRecipeImages(note.ID, []int{img.ID, img.ID})
	if err == nil {
		t.Error("Expected error for duplicate IDs, got nil")
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
