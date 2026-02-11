package db

import "testing"

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
