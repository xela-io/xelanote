package service

import (
	"testing"
)

func TestNoteService_DeleteNote(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)

	user := createTestUser(t, database, "trashuser")

	t.Run("soft-deletes a note", func(t *testing.T) {
		note, err := service.CreateNote(user.ID, "ToDelete", "content", "/")
		if err != nil {
			t.Fatalf("failed to create note: %v", err)
		}

		err = service.DeleteNote(user.ID, note.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Note should appear in deleted list
		deleted, _, err := service.ListDeletedNotes(user.ID, 10, "")
		if err != nil {
			t.Fatalf("failed to list deleted: %v", err)
		}
		found := false
		for _, d := range deleted {
			if d.ID == note.ID {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected deleted note in trash list")
		}
	})

	t.Run("returns error for nonexistent note", func(t *testing.T) {
		err := service.DeleteNote(user.ID, "nonexistent-id")
		if err == nil {
			t.Fatal("expected error for nonexistent note")
		}
	})
}

func TestNoteService_ListDeletedNotes(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)

	user := createTestUser(t, database, "listtrashuser")

	t.Run("returns empty list when no deleted notes", func(t *testing.T) {
		deleted, _, err := service.ListDeletedNotes(user.ID, 10, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(deleted) != 0 {
			t.Errorf("expected 0 deleted notes, got %d", len(deleted))
		}
	})

	t.Run("returns deleted notes", func(t *testing.T) {
		note, err := service.CreateNote(user.ID, "WillDelete", "content", "/")
		if err != nil {
			t.Fatalf("failed to create note: %v", err)
		}
		if err := service.DeleteNote(user.ID, note.ID); err != nil {
			t.Fatalf("failed to delete: %v", err)
		}

		deleted, _, err := service.ListDeletedNotes(user.ID, 10, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(deleted) == 0 {
			t.Error("expected at least 1 deleted note")
		}
	})
}

func TestNoteService_RestoreNote(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)

	user := createTestUser(t, database, "restoreuser")

	t.Run("restores a soft-deleted note", func(t *testing.T) {
		note, err := service.CreateNote(user.ID, "Restore Me", "content", "/")
		if err != nil {
			t.Fatalf("failed to create note: %v", err)
		}

		if err := service.DeleteNote(user.ID, note.ID); err != nil {
			t.Fatalf("failed to delete: %v", err)
		}

		restored, err := service.RestoreNote(user.ID, note.ID)
		if err != nil {
			t.Fatalf("unexpected error restoring: %v", err)
		}
		if restored.ID != note.ID {
			t.Errorf("expected restored note ID %s, got %s", note.ID, restored.ID)
		}
		if restored.Title != "Restore Me" {
			t.Errorf("expected title 'Restore Me', got %q", restored.Title)
		}

		// Should no longer be in deleted list
		deleted, _, err := service.ListDeletedNotes(user.ID, 10, "")
		if err != nil {
			t.Fatalf("failed to list deleted: %v", err)
		}
		for _, d := range deleted {
			if d.ID == note.ID {
				t.Error("restored note should not be in deleted list")
			}
		}
	})
}

func TestNoteService_PermanentlyDeleteNote(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)

	user := createTestUser(t, database, "permdeluser")

	t.Run("permanently deletes a note", func(t *testing.T) {
		note, err := service.CreateNote(user.ID, "PermDelete", "content", "/")
		if err != nil {
			t.Fatalf("failed to create note: %v", err)
		}

		// Soft-delete first
		if err := service.DeleteNote(user.ID, note.ID); err != nil {
			t.Fatalf("failed to soft-delete: %v", err)
		}

		// Permanently delete
		err = service.PermanentlyDeleteNote(user.ID, note.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should not appear in deleted list
		deleted, _, err := service.ListDeletedNotes(user.ID, 10, "")
		if err != nil {
			t.Fatalf("failed to list deleted: %v", err)
		}
		for _, d := range deleted {
			if d.ID == note.ID {
				t.Error("permanently deleted note should not be in trash")
			}
		}
	})
}

func TestNoteService_GetDeletedNotesCount(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)

	user := createTestUser(t, database, "countuser")

	t.Run("returns 0 for new user", func(t *testing.T) {
		count, err := service.GetDeletedNotesCount(user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0, got %d", count)
		}
	})

	t.Run("counts deleted notes", func(t *testing.T) {
		note1, _ := service.CreateNote(user.ID, "Count1", "c", "/")
		note2, _ := service.CreateNote(user.ID, "Count2", "c", "/")
		service.DeleteNote(user.ID, note1.ID)
		service.DeleteNote(user.ID, note2.ID)

		count, err := service.GetDeletedNotesCount(user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 deleted notes, got %d", count)
		}
	})
}

func TestNoteService_EmptyTrash(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)

	user := createTestUser(t, database, "emptytrashuser")

	t.Run("empties trash and returns count", func(t *testing.T) {
		// Create and delete 3 notes
		for i := 0; i < 3; i++ {
			note, err := service.CreateNote(user.ID, "Trash"+string(rune('A'+i)), "c", "/")
			if err != nil {
				t.Fatalf("failed to create note: %v", err)
			}
			if err := service.DeleteNote(user.ID, note.ID); err != nil {
				t.Fatalf("failed to delete note: %v", err)
			}
		}

		count, err := service.EmptyTrash(user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 3 {
			t.Errorf("expected 3 permanently deleted, got %d", count)
		}

		// Trash should be empty now
		remaining, err := service.GetDeletedNotesCount(user.ID)
		if err != nil {
			t.Fatalf("failed to get count: %v", err)
		}
		if remaining != 0 {
			t.Errorf("expected 0 remaining in trash, got %d", remaining)
		}
	})

	t.Run("returns 0 for empty trash", func(t *testing.T) {
		count, err := service.EmptyTrash(user.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 for empty trash, got %d", count)
		}
	})
}
