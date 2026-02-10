package db

import (
	"encoding/json"
	"testing"
)

func TestGetUserFeature_NotSet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Test: journal and recipe are enabled by default
	for _, name := range []string{"journal", "recipe"} {
		feature, err := db.GetUserFeature(userID, name)
		if err != nil {
			t.Fatalf("GetUserFeature(%s) failed: %v", name, err)
		}
		if !feature.Enabled {
			t.Errorf("Expected %s to be enabled by default", name)
		}
		if feature.UserID != userID {
			t.Errorf("Expected UserID %d, got %d", userID, feature.UserID)
		}
	}

	// Test: other features are disabled by default
	feature, err := db.GetUserFeature(userID, "some_other_feature")
	if err != nil {
		t.Fatalf("GetUserFeature failed: %v", err)
	}
	if feature.Enabled {
		t.Error("Expected unknown feature to be disabled by default")
	}
}

func TestSetUserFeature_Enable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Test: Enable feature
	err := db.SetUserFeature(userID, "journal", true, nil)
	if err != nil {
		t.Fatalf("SetUserFeature failed: %v", err)
	}

	// Verify
	feature, err := db.GetUserFeature(userID, "journal")
	if err != nil {
		t.Fatalf("GetUserFeature failed: %v", err)
	}

	if !feature.Enabled {
		t.Error("Expected feature to be enabled")
	}
}

func TestSetUserFeature_Disable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Enable first
	err := db.SetUserFeature(userID, "journal", true, nil)
	if err != nil {
		t.Fatalf("SetUserFeature (enable) failed: %v", err)
	}

	// Then disable
	err = db.SetUserFeature(userID, "journal", false, nil)
	if err != nil {
		t.Fatalf("SetUserFeature (disable) failed: %v", err)
	}

	// Verify
	feature, err := db.GetUserFeature(userID, "journal")
	if err != nil {
		t.Fatalf("GetUserFeature failed: %v", err)
	}

	if feature.Enabled {
		t.Error("Expected feature to be disabled")
	}
}

func TestSetUserFeature_WithSettings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Test: Set feature with custom settings
	settings := json.RawMessage(`{"language": "de", "theme": "dark"}`)
	err := db.SetUserFeature(userID, "journal", true, settings)
	if err != nil {
		t.Fatalf("SetUserFeature failed: %v", err)
	}

	// Verify
	feature, err := db.GetUserFeature(userID, "journal")
	if err != nil {
		t.Fatalf("GetUserFeature failed: %v", err)
	}

	if !feature.Enabled {
		t.Error("Expected feature to be enabled")
	}

	var parsedSettings map[string]string
	if err := json.Unmarshal(feature.Settings, &parsedSettings); err != nil {
		t.Fatalf("Failed to parse settings: %v", err)
	}

	if parsedSettings["language"] != "de" {
		t.Errorf("Expected language 'de', got '%s'", parsedSettings["language"])
	}
}

func TestSetUserFeature_Upsert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// First insert
	settings1 := json.RawMessage(`{"version": 1}`)
	err := db.SetUserFeature(userID, "journal", true, settings1)
	if err != nil {
		t.Fatalf("SetUserFeature (first) failed: %v", err)
	}

	// Update (upsert)
	settings2 := json.RawMessage(`{"version": 2}`)
	err = db.SetUserFeature(userID, "journal", true, settings2)
	if err != nil {
		t.Fatalf("SetUserFeature (upsert) failed: %v", err)
	}

	// Verify: Should update, not create duplicate
	feature, err := db.GetUserFeature(userID, "journal")
	if err != nil {
		t.Fatalf("GetUserFeature failed: %v", err)
	}

	var parsedSettings map[string]int
	if err := json.Unmarshal(feature.Settings, &parsedSettings); err != nil {
		t.Fatalf("Failed to parse settings: %v", err)
	}

	if parsedSettings["version"] != 2 {
		t.Errorf("Expected version 2, got %d", parsedSettings["version"])
	}
}

func TestListUserFeatures(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Set multiple features
	err := db.SetUserFeature(userID, "journal", true, nil)
	if err != nil {
		t.Fatalf("SetUserFeature (journal) failed: %v", err)
	}

	err = db.SetUserFeature(userID, "ai_summary", false, nil)
	if err != nil {
		t.Fatalf("SetUserFeature (ai_summary) failed: %v", err)
	}

	// Test: List all features
	features, err := db.ListUserFeatures(userID)
	if err != nil {
		t.Fatalf("ListUserFeatures failed: %v", err)
	}

	if len(features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(features))
	}

	// Check that journal is enabled and ai_summary is disabled
	journalFound := false
	aiSummaryFound := false
	for _, f := range features {
		if f.Feature == "journal" {
			journalFound = true
			if !f.Enabled {
				t.Error("Expected journal to be enabled")
			}
		}
		if f.Feature == "ai_summary" {
			aiSummaryFound = true
			if f.Enabled {
				t.Error("Expected ai_summary to be disabled")
			}
		}
	}

	if !journalFound {
		t.Error("journal feature not found in list")
	}
	if !aiSummaryFound {
		t.Error("ai_summary feature not found in list")
	}
}

func TestListUserFeatures_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Test: No features set
	features, err := db.ListUserFeatures(userID)
	if err != nil {
		t.Fatalf("ListUserFeatures failed: %v", err)
	}

	if len(features) != 0 {
		t.Errorf("Expected 0 features, got %d", len(features))
	}
}

func TestUserFeatures_Isolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create two users
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (1, 'user1', 'user1@example.com', 'hash1', datetime('now')),
		       (2, 'user2', 'user2@example.com', 'hash2', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to create users: %v", err)
	}

	// User 1 enables ai_summary
	err = db.SetUserFeature(1, "ai_summary", true, nil)
	if err != nil {
		t.Fatalf("SetUserFeature (user1) failed: %v", err)
	}

	// User 2 should not have feature in DB (only default applies)
	feature, err := db.GetUserFeature(2, "ai_summary")
	if err != nil {
		t.Fatalf("GetUserFeature (user2) failed: %v", err)
	}

	if feature.Enabled {
		t.Error("Expected user2's ai_summary feature to be disabled (isolated from user1)")
	}
}
