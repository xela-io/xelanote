package db

import "testing"

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
