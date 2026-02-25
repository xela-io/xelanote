package db

import "testing"

func TestNoteUserStateUpsertAndGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createTestNote(t, db, "note1", 1, "Test Note")

	// Initially no state
	state, err := db.GetNoteUserState(1, "note1")
	if err != nil {
		t.Fatalf("GetNoteUserState failed: %v", err)
	}
	if state != nil {
		t.Fatalf("Expected nil state, got %v", *state)
	}

	// Upsert state
	err = db.UpsertNoteUserState(1, "note1", `{"collapse_state":{"abc":true}}`)
	if err != nil {
		t.Fatalf("UpsertNoteUserState failed: %v", err)
	}

	// Read back
	state, err = db.GetNoteUserState(1, "note1")
	if err != nil {
		t.Fatalf("GetNoteUserState after upsert failed: %v", err)
	}
	if state == nil {
		t.Fatal("Expected non-nil state after upsert")
	}
	if *state != `{"collapse_state":{"abc":true}}` {
		t.Fatalf("Unexpected state data: %s", *state)
	}

	// Update existing
	err = db.UpsertNoteUserState(1, "note1", `{"collapse_state":{"abc":false,"def":true}}`)
	if err != nil {
		t.Fatalf("UpsertNoteUserState update failed: %v", err)
	}

	state, err = db.GetNoteUserState(1, "note1")
	if err != nil {
		t.Fatalf("GetNoteUserState after update failed: %v", err)
	}
	if state == nil || *state != `{"collapse_state":{"abc":false,"def":true}}` {
		t.Fatalf("Unexpected state data after update: %v", state)
	}
}

func TestNoteUserStateNotFoundReturnsNil(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")

	state, err := db.GetNoteUserState(1, "nonexistent")
	if err != nil {
		t.Fatalf("GetNoteUserState failed: %v", err)
	}
	if state != nil {
		t.Fatalf("Expected nil for nonexistent note, got %v", *state)
	}
}

func TestNoteUserStateCascadeOnNoteDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createTestNote(t, db, "note1", 1, "Test Note")

	err := db.UpsertNoteUserState(1, "note1", `{"collapse_state":{"x":true}}`)
	if err != nil {
		t.Fatalf("UpsertNoteUserState failed: %v", err)
	}

	// Delete the note (hard delete for cascade test)
	_, err = db.Exec("DELETE FROM notes WHERE id = ?", "note1")
	if err != nil {
		t.Fatalf("Delete note failed: %v", err)
	}

	// State should be gone
	state, err := db.GetNoteUserState(1, "note1")
	if err != nil {
		t.Fatalf("GetNoteUserState after delete failed: %v", err)
	}
	if state != nil {
		t.Fatalf("Expected nil after cascade delete, got %v", *state)
	}
}

func TestNoteUserStateUserIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createNamedTestUser(t, db, 2, "bob", "bob@test.com")
	createTestNote(t, db, "note1", 1, "Test Note")

	// Alice sets state
	err := db.UpsertNoteUserState(1, "note1", `{"collapse_state":{"a":true}}`)
	if err != nil {
		t.Fatalf("UpsertNoteUserState for alice failed: %v", err)
	}

	// Bob sets different state
	err = db.UpsertNoteUserState(2, "note1", `{"collapse_state":{"b":false}}`)
	if err != nil {
		t.Fatalf("UpsertNoteUserState for bob failed: %v", err)
	}

	// Alice's state is intact
	aliceState, err := db.GetNoteUserState(1, "note1")
	if err != nil {
		t.Fatalf("GetNoteUserState for alice failed: %v", err)
	}
	if aliceState == nil || *aliceState != `{"collapse_state":{"a":true}}` {
		t.Fatalf("Alice state mismatch: %v", aliceState)
	}

	// Bob's state is intact
	bobState, err := db.GetNoteUserState(2, "note1")
	if err != nil {
		t.Fatalf("GetNoteUserState for bob failed: %v", err)
	}
	if bobState == nil || *bobState != `{"collapse_state":{"b":false}}` {
		t.Fatalf("Bob state mismatch: %v", bobState)
	}
}

func TestNoteUserStateDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createNamedTestUser(t, db, 1, "alice", "alice@test.com")
	createTestNote(t, db, "note1", 1, "Test Note")

	err := db.UpsertNoteUserState(1, "note1", `{"collapse_state":{"x":true}}`)
	if err != nil {
		t.Fatalf("UpsertNoteUserState failed: %v", err)
	}

	err = db.DeleteNoteUserState(1, "note1")
	if err != nil {
		t.Fatalf("DeleteNoteUserState failed: %v", err)
	}

	state, err := db.GetNoteUserState(1, "note1")
	if err != nil {
		t.Fatalf("GetNoteUserState after delete failed: %v", err)
	}
	if state != nil {
		t.Fatalf("Expected nil after explicit delete, got %v", *state)
	}
}
