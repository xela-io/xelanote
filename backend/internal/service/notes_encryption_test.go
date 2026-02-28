package service

import (
	"testing"
)

func TestNoteService_CreateEncryptedNote(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)

	user := createTestUser(t, database, "encuser")

	t.Run("creates encrypted note with valid fields", func(t *testing.T) {
		note, err := service.CreateEncryptedNote(
			user.ID,
			"Encrypted Title",
			nil,   // encryptedTitle
			false, // titleEncrypted
			[]byte("encrypted-content-bytes"),
			"wrapped-dek-value",
			"v2",
			nil, // no keywords
			"/",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if note == nil {
			t.Fatal("expected non-nil note")
		}
		if note.Title != "Encrypted Title" {
			t.Errorf("expected title 'Encrypted Title', got %q", note.Title)
		}
		if note.WrappedDEK != "wrapped-dek-value" {
			t.Errorf("expected wrapped DEK 'wrapped-dek-value', got %q", note.WrappedDEK)
		}
	})

	t.Run("rejects empty encrypted content", func(t *testing.T) {
		_, err := service.CreateEncryptedNote(
			user.ID,
			"Title",
			nil, false,
			nil, // empty content
			"wrapped-dek",
			"v2",
			nil,
			"/",
		)
		if err == nil {
			t.Fatal("expected error for empty encrypted content")
		}
	})

	t.Run("rejects empty wrapped DEK", func(t *testing.T) {
		_, err := service.CreateEncryptedNote(
			user.ID,
			"Title",
			nil, false,
			[]byte("content"),
			"", // empty DEK
			"v2",
			nil,
			"/",
		)
		if err == nil {
			t.Fatal("expected error for empty wrapped DEK")
		}
	})

	t.Run("creates with encrypted title", func(t *testing.T) {
		encTitle := "encrypted-title-data"
		note, err := service.CreateEncryptedNote(
			user.ID,
			"Plaintext Fallback",
			&encTitle,
			true,
			[]byte("encrypted-content"),
			"wrapped-dek",
			"v2",
			nil,
			"/",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !note.TitleEncrypted {
			t.Error("expected TitleEncrypted to be true")
		}
	})
}

func TestNoteService_UpdateEncryptedNote_ClearsPlaintextMetadata(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)
	user := createTestUser(t, database, "encmetauser")

	target, err := service.CreateNote(user.ID, "Target", "target content", "/")
	if err != nil {
		t.Fatalf("failed to create target note: %v", err)
	}

	sourceContent := "[[Target]]\n- [ ] hidden @due(2026-03-10)"
	source, err := service.CreateNote(user.ID, "Source", sourceContent, "/")
	if err != nil {
		t.Fatalf("failed to create source note: %v", err)
	}

	_, err = database.Exec(`INSERT INTO note_keywords (note_id, keyword) VALUES (?, ?)`, source.ID, "legacy")
	if err != nil {
		t.Fatalf("failed to seed legacy keyword: %v", err)
	}

	var linksBefore, unresolvedBefore, dueDatesBefore, keywordsBefore int
	if err := database.QueryRow(`SELECT COUNT(*) FROM links WHERE source_id = ?`, source.ID).Scan(&linksBefore); err != nil {
		t.Fatalf("count links before: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM unresolved_links WHERE source_id = ?`, source.ID).Scan(&unresolvedBefore); err != nil {
		t.Fatalf("count unresolved before: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM note_due_dates WHERE note_id = ?`, source.ID).Scan(&dueDatesBefore); err != nil {
		t.Fatalf("count due dates before: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM note_keywords WHERE note_id = ?`, source.ID).Scan(&keywordsBefore); err != nil {
		t.Fatalf("count keywords before: %v", err)
	}

	if linksBefore == 0 {
		t.Fatal("expected resolved links before encryption")
	}
	if dueDatesBefore == 0 {
		t.Fatal("expected due dates before encryption")
	}
	if keywordsBefore == 0 {
		t.Fatal("expected keywords before encryption")
	}

	_, err = service.UpdateEncryptedNote(
		user.ID,
		source.ID,
		"Encrypted Source",
		nil,
		false,
		[]byte("encrypted-content"),
		"wrapped-dek",
		`{"algorithm":"XChaCha20-Poly1305","version":3}`,
		"/",
		nil,
		source.Version,
	)
	if err != nil {
		t.Fatalf("update encrypted note failed: %v", err)
	}

	var linksAfter, unresolvedAfter, dueDatesAfter, keywordsAfter int
	if err := database.QueryRow(`SELECT COUNT(*) FROM links WHERE source_id = ?`, source.ID).Scan(&linksAfter); err != nil {
		t.Fatalf("count links after: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM unresolved_links WHERE source_id = ?`, source.ID).Scan(&unresolvedAfter); err != nil {
		t.Fatalf("count unresolved after: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM note_due_dates WHERE note_id = ?`, source.ID).Scan(&dueDatesAfter); err != nil {
		t.Fatalf("count due dates after: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM note_keywords WHERE note_id = ?`, source.ID).Scan(&keywordsAfter); err != nil {
		t.Fatalf("count keywords after: %v", err)
	}

	if linksAfter != 0 {
		t.Fatalf("expected links cleared, got %d", linksAfter)
	}
	if unresolvedAfter != 0 {
		t.Fatalf("expected unresolved links cleared, got %d", unresolvedAfter)
	}
	if dueDatesAfter != 0 {
		t.Fatalf("expected due dates cleared, got %d", dueDatesAfter)
	}
	if keywordsAfter != 0 {
		t.Fatalf("expected keywords cleared, got %d", keywordsAfter)
	}

	// Keep target used to avoid accidental linter optimization assumptions.
	if target.ID == "" {
		t.Fatal("target note id must not be empty")
	}
}

func TestNoteService_DecryptNote(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)

	user := createTestUser(t, database, "decryptuser")

	t.Run("rejects decryption of non-encrypted note", func(t *testing.T) {
		// Create a plaintext note
		note, err := service.CreateNote(user.ID, "Plain", "plaintext content", "/")
		if err != nil {
			t.Fatalf("failed to create note: %v", err)
		}

		_, err = service.DecryptNote(user.ID, note.ID, "Plain", "plaintext content", note.Version)
		if err == nil {
			t.Fatal("expected error when decrypting non-encrypted note")
		}
	})

	t.Run("decrypts encrypted note successfully", func(t *testing.T) {
		// Create an encrypted note
		encNote, err := service.CreateEncryptedNote(
			user.ID,
			"Encrypted",
			nil, false,
			[]byte("encrypted-bytes"),
			"wrapped-dek",
			"v2",
			nil,
			"/",
		)
		if err != nil {
			t.Fatalf("failed to create encrypted note: %v", err)
		}

		decrypted, err := service.DecryptNote(
			user.ID,
			encNote.ID,
			"Decrypted Title",
			"Decrypted plaintext content",
			encNote.Version,
		)
		if err != nil {
			t.Fatalf("unexpected error decrypting note: %v", err)
		}
		if decrypted.Content != "Decrypted plaintext content" {
			t.Errorf("expected decrypted content, got %q", decrypted.Content)
		}
		if decrypted.ContentEncrypted {
			t.Error("expected ContentEncrypted to be false after decryption")
		}
	})

	t.Run("returns error for nonexistent note", func(t *testing.T) {
		_, err := service.DecryptNote(user.ID, "nonexistent-id", "Title", "Content", 1)
		if err == nil {
			t.Fatal("expected error for nonexistent note")
		}
	})
}

func TestNoteService_BatchUpdateWrappedDEKs(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)

	user := createTestUser(t, database, "batchuser")

	t.Run("returns zero for empty updates", func(t *testing.T) {
		count, err := service.BatchUpdateWrappedDEKs(user.ID, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 updated, got %d", count)
		}
	})

	t.Run("updates wrapped DEKs for encrypted notes", func(t *testing.T) {
		note1, err := service.CreateEncryptedNote(
			user.ID, "BatchNote1", nil, false,
			[]byte("content1"), "old-dek-1", "v2", nil, "/",
		)
		if err != nil {
			t.Fatalf("failed to create note1: %v", err)
		}
		note2, err := service.CreateEncryptedNote(
			user.ID, "BatchNote2", nil, false,
			[]byte("content2"), "old-dek-2", "v2", nil, "/",
		)
		if err != nil {
			t.Fatalf("failed to create note2: %v", err)
		}

		updates := []struct {
			NoteID     string `json:"note_id"`
			WrappedDEK string `json:"wrapped_dek"`
		}{
			{NoteID: note1.ID, WrappedDEK: "new-dek-1"},
			{NoteID: note2.ID, WrappedDEK: "new-dek-2"},
		}

		count, err := service.BatchUpdateWrappedDEKs(user.ID, updates)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 updated, got %d", count)
		}

		var dek string
		if err := database.QueryRow(`SELECT wrapped_dek FROM notes WHERE id = ?`, note1.ID).Scan(&dek); err != nil {
			t.Fatalf("failed to query note1: %v", err)
		}
		if dek != "new-dek-1" {
			t.Errorf("expected 'new-dek-1', got %q", dek)
		}
	})

	t.Run("skips non-encrypted notes", func(t *testing.T) {
		plainNote, err := service.CreateNote(user.ID, "PlainBatch", "content", "/")
		if err != nil {
			t.Fatalf("failed to create plaintext note: %v", err)
		}

		updates := []struct {
			NoteID     string `json:"note_id"`
			WrappedDEK string `json:"wrapped_dek"`
		}{
			{NoteID: plainNote.ID, WrappedDEK: "should-not-apply"},
		}

		count, err := service.BatchUpdateWrappedDEKs(user.ID, updates)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 updated for plaintext note, got %d", count)
		}
	})

	t.Run("rejects nonexistent note", func(t *testing.T) {
		updates := []struct {
			NoteID     string `json:"note_id"`
			WrappedDEK string `json:"wrapped_dek"`
		}{
			{NoteID: "nonexistent-id", WrappedDEK: "dek"},
		}

		_, err := service.BatchUpdateWrappedDEKs(user.ID, updates)
		if err == nil {
			t.Fatal("expected error for nonexistent note")
		}
	})
}

func TestNoteService_UserHasEncryptedNotes(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)

	user := createTestUser(t, database, "hasencuser")

	t.Run("returns false for user with no notes", func(t *testing.T) {
		has, err := service.UserHasEncryptedNotes(user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if has {
			t.Error("expected false for user with no notes")
		}
	})

	t.Run("returns false for user with only plaintext notes", func(t *testing.T) {
		_, err := service.CreateNote(user.ID, "Plain", "content", "/")
		if err != nil {
			t.Fatalf("failed to create note: %v", err)
		}

		has, err := service.UserHasEncryptedNotes(user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if has {
			t.Error("expected false for user with only plaintext notes")
		}
	})

	t.Run("returns true for user with encrypted notes", func(t *testing.T) {
		_, err := service.CreateEncryptedNote(
			user.ID, "Enc", nil, false,
			[]byte("encrypted"), "dek", "v2", nil, "/",
		)
		if err != nil {
			t.Fatalf("failed to create encrypted note: %v", err)
		}

		has, err := service.UserHasEncryptedNotes(user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !has {
			t.Error("expected true for user with encrypted notes")
		}
	})
}
