package db

import "testing"

func TestGetDietaryPreference_DefaultNone(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	pref, err := db.GetDietaryPreference(1)
	if err != nil {
		t.Fatalf("GetDietaryPreference failed: %v", err)
	}
	if pref != "none" {
		t.Errorf("expected 'none', got %q", pref)
	}
}

func TestSetAndGetDietaryPreference(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	// Set preference
	if err := db.SetDietaryPreference(1, "vegan"); err != nil {
		t.Fatalf("SetDietaryPreference failed: %v", err)
	}

	// Read back
	pref, err := db.GetDietaryPreference(1)
	if err != nil {
		t.Fatalf("GetDietaryPreference failed: %v", err)
	}
	if pref != "vegan" {
		t.Errorf("expected 'vegan', got %q", pref)
	}
}

func TestSetDietaryPreference_UpsertWithoutExistingRow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	// No preferences row exists yet – SetDietaryPreference should insert
	if err := db.SetDietaryPreference(1, "pescatarian"); err != nil {
		t.Fatalf("SetDietaryPreference (upsert) failed: %v", err)
	}

	pref, err := db.GetDietaryPreference(1)
	if err != nil {
		t.Fatalf("GetDietaryPreference failed: %v", err)
	}
	if pref != "pescatarian" {
		t.Errorf("expected 'pescatarian', got %q", pref)
	}
}

func TestSetDietaryPreference_Overwrite(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	// Set initial value
	if err := db.SetDietaryPreference(1, "vegetarian"); err != nil {
		t.Fatalf("SetDietaryPreference failed: %v", err)
	}

	// Overwrite
	if err := db.SetDietaryPreference(1, "flexitarian"); err != nil {
		t.Fatalf("SetDietaryPreference (overwrite) failed: %v", err)
	}

	pref, err := db.GetDietaryPreference(1)
	if err != nil {
		t.Fatalf("GetDietaryPreference failed: %v", err)
	}
	if pref != "flexitarian" {
		t.Errorf("expected 'flexitarian', got %q", pref)
	}
}

func TestGetDietaryPreference_ExistingRowWithEmptyValue(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	// Create a preferences row with empty dietary_preference
	_, _, err := db.GetOrCreateUserPreferences(1)
	if err != nil {
		t.Fatalf("GetOrCreateUserPreferences failed: %v", err)
	}

	pref, err := db.GetDietaryPreference(1)
	if err != nil {
		t.Fatalf("GetDietaryPreference failed: %v", err)
	}
	if pref != "none" {
		t.Errorf("expected 'none' for existing row with default, got %q", pref)
	}
}
