package db

import "testing"

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
		[]byte("enc"), "dek", "{}", "", nil, note.Version,
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
		[]byte("enc"), "dek", "{}", "", nil, recipe.Version,
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
