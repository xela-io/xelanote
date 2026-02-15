package db

import (
	"testing"
)

func TestGetJournalByDate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create a journal entry
	journalDate := "2026-02-04"
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, created_at, updated_at)
		VALUES ('journal1', 'Dienstag, 4. Februar 2026', 'dienstag, 4. februar 2026', 'Journal content', '/Journal', ?, 'journal', ?, datetime('now'), datetime('now'))
	`, userID, journalDate)
	if err != nil {
		t.Fatalf("Failed to create journal entry: %v", err)
	}

	// Test: Find journal by date
	note, err := db.GetJournalByDate(userID, journalDate)
	if err != nil {
		t.Fatalf("GetJournalByDate failed: %v", err)
	}

	if note.ID != "journal1" {
		t.Errorf("Expected journal ID 'journal1', got '%s'", note.ID)
	}
	if note.NoteType != NoteTypeJournal {
		t.Errorf("Expected NoteType 'journal', got '%s'", note.NoteType)
	}
	if note.JournalDate == nil || *note.JournalDate != journalDate {
		t.Errorf("Expected JournalDate '%s', got '%v'", journalDate, note.JournalDate)
	}
}

func TestGetJournalByDate_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Test: Journal doesn't exist
	_, err := db.GetJournalByDate(userID, "2026-02-04")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestGetJournalByDate_ExcludesDeleted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create a deleted journal entry
	journalDate := "2026-02-04"
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, is_deleted, deleted_at, created_at, updated_at)
		VALUES ('journal1', 'Deleted Journal', 'deleted journal', 'Content', '/Journal', ?, 'journal', ?, 1, datetime('now'), datetime('now'), datetime('now'))
	`, userID, journalDate)
	if err != nil {
		t.Fatalf("Failed to create deleted journal: %v", err)
	}

	// Test: Deleted journal should not be found
	_, err = db.GetJournalByDate(userID, journalDate)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound for deleted journal, got %v", err)
	}
}

func TestListJournalDates(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create journal entries for February 2026
	dates := []string{"2026-02-01", "2026-02-15", "2026-02-28"}
	for i, date := range dates {
		_, err := db.Exec(`
			INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, created_at, updated_at)
			VALUES (?, ?, ?, 'Content', '/Journal', ?, 'journal', ?, datetime('now'), datetime('now'))
		`, "journal"+string(rune('1'+i)), "Journal "+date, "journal "+date, userID, date)
		if err != nil {
			t.Fatalf("Failed to create journal for %s: %v", date, err)
		}
	}

	// Create a journal entry for January (should not be included)
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, created_at, updated_at)
		VALUES ('jan_journal', 'January Journal', 'january journal', 'Content', '/Journal', ?, 'journal', '2026-01-15', datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create January journal: %v", err)
	}

	// Test: List February 2026 journals
	result, err := db.ListJournalDates(userID, 2026, 2)
	if err != nil {
		t.Fatalf("ListJournalDates failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 dates, got %d", len(result))
	}

	for i, expected := range dates {
		if i >= len(result) {
			break
		}
		if result[i] != expected {
			t.Errorf("Expected date %s at index %d, got %s", expected, i, result[i])
		}
	}
}

func TestListJournalDates_ExcludesDeleted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create active journal
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, created_at, updated_at)
		VALUES ('journal_active', 'Active', 'active', 'Content', '/Journal', ?, 'journal', '2026-02-01', datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create active journal: %v", err)
	}

	// Create deleted journal
	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, is_deleted, deleted_at, created_at, updated_at)
		VALUES ('journal_deleted', 'Deleted', 'deleted', 'Content', '/Journal', ?, 'journal', '2026-02-15', 1, datetime('now'), datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create deleted journal: %v", err)
	}

	// Test: Only active journal should be returned
	result, err := db.ListJournalDates(userID, 2026, 2)
	if err != nil {
		t.Fatalf("ListJournalDates failed: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 date (only active), got %d", len(result))
	}
	if len(result) > 0 && result[0] != "2026-02-01" {
		t.Errorf("Expected '2026-02-01', got '%s'", result[0])
	}
}

func TestJournalExistsForDate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create a journal entry
	journalDate := "2026-02-04"
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, created_at, updated_at)
		VALUES ('journal1', 'Journal', 'journal', 'Content', '/Journal', ?, 'journal', ?, datetime('now'), datetime('now'))
	`, userID, journalDate)
	if err != nil {
		t.Fatalf("Failed to create journal: %v", err)
	}

	// Test: Journal exists
	exists, noteID, err := db.JournalExistsForDate(userID, journalDate)
	if err != nil {
		t.Fatalf("JournalExistsForDate failed: %v", err)
	}
	if !exists {
		t.Error("Expected journal to exist")
	}
	if noteID != "journal1" {
		t.Errorf("Expected noteID 'journal1', got '%s'", noteID)
	}

	// Test: Journal doesn't exist
	exists, noteID, err = db.JournalExistsForDate(userID, "2026-02-05")
	if err != nil {
		t.Fatalf("JournalExistsForDate failed: %v", err)
	}
	if exists {
		t.Error("Expected journal to not exist")
	}
	if noteID != "" {
		t.Errorf("Expected empty noteID, got '%s'", noteID)
	}
}

func TestListJournalEntries_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	entries, err := db.ListJournalEntries(userID)
	if err != nil {
		t.Fatalf("ListJournalEntries failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

func TestListJournalEntries_OrderByDate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create entries in non-chronological order
	dates := []string{"2026-02-01", "2026-02-15", "2026-01-10"}
	for i, date := range dates {
		_, err := db.Exec(`
			INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, created_at, updated_at)
			VALUES (?, ?, ?, 'Content', '/Journal', ?, 'journal', ?, datetime('now'), datetime('now'))
		`, "journal"+string(rune('a'+i)), "Journal "+date, "journal "+date, userID, date)
		if err != nil {
			t.Fatalf("Failed to create journal for %s: %v", date, err)
		}
	}

	entries, err := db.ListJournalEntries(userID)
	if err != nil {
		t.Fatalf("ListJournalEntries failed: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(entries))
	}

	// Should be ordered by journal_date DESC
	expectedOrder := []string{"2026-02-15", "2026-02-01", "2026-01-10"}
	for i, expected := range expectedOrder {
		if entries[i].JournalDate == nil || *entries[i].JournalDate != expected {
			t.Errorf("Entry %d: expected date %s, got %v", i, expected, entries[i].JournalDate)
		}
	}
}

func TestListJournalEntries_ExcludesDeleted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create active journal
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, created_at, updated_at)
		VALUES ('journal_active', 'Active', 'active', 'Content', '/Journal', ?, 'journal', '2026-02-01', datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create active journal: %v", err)
	}

	// Create deleted journal
	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, is_deleted, deleted_at, created_at, updated_at)
		VALUES ('journal_deleted', 'Deleted', 'deleted', 'Content', '/Journal', ?, 'journal', '2026-02-15', 1, datetime('now'), datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create deleted journal: %v", err)
	}

	entries, err := db.ListJournalEntries(userID)
	if err != nil {
		t.Fatalf("ListJournalEntries failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry (only active), got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].ID != "journal_active" {
		t.Errorf("Expected 'journal_active', got '%s'", entries[0].ID)
	}
}

func TestListJournalEntries_ExcludesOtherTypes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create a regular note
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, created_at, updated_at)
		VALUES ('note1', 'Regular Note', 'regular note', 'Content', '/', ?, 'note', datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create regular note: %v", err)
	}

	// Create a recipe note
	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, created_at, updated_at)
		VALUES ('recipe1', 'Recipe', 'recipe', 'Content', '/Recipes', ?, 'recipe', datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create recipe note: %v", err)
	}

	// Create a journal note
	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, created_at, updated_at)
		VALUES ('journal1', 'Journal', 'journal', 'Content', '/Journal', ?, 'journal', '2026-02-01', datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create journal note: %v", err)
	}

	entries, err := db.ListJournalEntries(userID)
	if err != nil {
		t.Fatalf("ListJournalEntries failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry (only journal), got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].ID != "journal1" {
		t.Errorf("Expected 'journal1', got '%s'", entries[0].ID)
	}
}

func TestListJournalEntries_UserIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create two users
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (1, 'user1', 'user1@example.com', 'hash', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (2, 'user2', 'user2@example.com', 'hash', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}

	// Create journals for both users
	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, created_at, updated_at)
		VALUES ('j_user1', 'User1 Journal', 'user1 journal', 'Content', '/Journal', 1, 'journal', '2026-02-01', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to create user1 journal: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, created_at, updated_at)
		VALUES ('j_user2', 'User2 Journal', 'user2 journal', 'Content', '/Journal', 2, 'journal', '2026-02-01', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to create user2 journal: %v", err)
	}

	// User 1 should only see their own journals
	entries, err := db.ListJournalEntries(1)
	if err != nil {
		t.Fatalf("ListJournalEntries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("User 1: Expected 1 entry, got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].ID != "j_user1" {
		t.Errorf("User 1: Expected 'j_user1', got '%s'", entries[0].ID)
	}

	// User 2 should only see their own journals
	entries, err = db.ListJournalEntries(2)
	if err != nil {
		t.Fatalf("ListJournalEntries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("User 2: Expected 1 entry, got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].ID != "j_user2" {
		t.Errorf("User 2: Expected 'j_user2', got '%s'", entries[0].ID)
	}
}

func TestListJournalEntries_EncryptedFlag(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create unencrypted journal
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, content_encrypted, created_at, updated_at)
		VALUES ('j_plain', 'Plain Journal', 'plain journal', 'Content', '/Journal', ?, 'journal', '2026-02-01', 0, datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create plain journal: %v", err)
	}

	// Create encrypted journal
	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, content_encrypted, encrypted_content, wrapped_dek, encryption_version, created_at, updated_at)
		VALUES ('j_enc', 'Encrypted Journal', 'encrypted journal', '', '/Journal', ?, 'journal', '2026-02-02', 1, x'00', 'wrapped_key', 1, datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create encrypted journal: %v", err)
	}

	entries, err := db.ListJournalEntries(userID)
	if err != nil {
		t.Fatalf("ListJournalEntries failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	// Ordered by date DESC: encrypted first (2026-02-02), then plain (2026-02-01)
	if entries[0].ContentEncrypted != true {
		t.Error("Expected first entry (encrypted) to have ContentEncrypted=true")
	}
	if entries[1].ContentEncrypted != false {
		t.Error("Expected second entry (plain) to have ContentEncrypted=false")
	}
}

func TestListJournalEntries_TitleAlwaysPlaintext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Use CreateEncryptedJournalNote to simulate the real E2EE path
	encryptedTitle := `{"ct":"encrypted_title_data"}`
	note, err := db.CreateEncryptedJournalNote(
		userID,
		"Montag, 2. Februar 2026", // plaintext title
		&encryptedTitle,           // encrypted title (stored separately)
		true,                      // title_encrypted
		[]byte("encrypted_content"),
		"wrapped_dek_data",
		`{"alg":"AES-256-GCM"}`,
		"/Journal",
		"2026-02-02",
	)
	if err != nil {
		t.Fatalf("CreateEncryptedJournalNote failed: %v", err)
	}

	entries, err := db.ListJournalEntries(userID)
	if err != nil {
		t.Fatalf("ListJournalEntries failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	// Title should always be plaintext (the `title` column, not `encrypted_title`)
	if entries[0].Title != "Montag, 2. Februar 2026" {
		t.Errorf("Expected plaintext title 'Montag, 2. Februar 2026', got '%s'", entries[0].Title)
	}
	if entries[0].ID != note.ID {
		t.Errorf("Expected note ID '%s', got '%s'", note.ID, entries[0].ID)
	}
	if !entries[0].ContentEncrypted {
		t.Error("Expected ContentEncrypted=true for E2EE journal")
	}
}

func TestJournalNotIncludedInListNotes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create a regular note
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, created_at, updated_at)
		VALUES ('note1', 'Regular Note', 'regular note', 'Content', '/', ?, 'note', datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create regular note: %v", err)
	}

	// Create a journal note
	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, journal_date, created_at, updated_at)
		VALUES ('journal1', 'Journal', 'journal', 'Content', '/Journal', ?, 'journal', '2026-02-04', datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create journal note: %v", err)
	}

	// Test: ListNotes should only return regular notes
	notes, _, err := db.ListNotes(userID, 100, "", ListNotesOptions{})
	if err != nil {
		t.Fatalf("ListNotes failed: %v", err)
	}

	if len(notes) != 1 {
		t.Errorf("Expected 1 note (journal excluded), got %d", len(notes))
	}
	if len(notes) > 0 && notes[0].ID != "note1" {
		t.Errorf("Expected note1, got %s", notes[0].ID)
	}
}
