package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/xela-io/xelanote/internal/db"
)

func setNoteEncrypted(t *testing.T, database *db.DB, noteID string, encrypted bool) {
	t.Helper()

	value := 0
	if encrypted {
		value = 1
	}

	if _, err := database.Exec(`UPDATE notes SET content_encrypted = ? WHERE id = ?`, value, noteID); err != nil {
		t.Fatalf("failed to mark note encrypted: %v", err)
	}
}

func TestSharingService_ShareNoteValidation(t *testing.T) {
	database := setupTestDB(t)
	service := NewSharingService(database)
	noteService := NewNoteService(database)

	owner := createTestUser(t, database, "owner")
	other := createTestUser(t, database, "other")

	note, err := noteService.CreateNote(owner.ID, "Title", "content", "/")
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	t.Run("should reject self-share", func(t *testing.T) {
		_, err := service.ShareNote(owner.ID, note.ID, owner.Username, "viewer")
		if !errors.Is(err, ErrCannotShareWithSelf) {
			t.Fatalf("expected ErrCannotShareWithSelf, got %v", err)
		}
	})

	t.Run("should reject non-owner", func(t *testing.T) {
		_, err := service.ShareNote(other.ID, note.ID, owner.Username, "viewer")
		if !errors.Is(err, ErrNotNoteOwner) {
			t.Fatalf("expected ErrNotNoteOwner, got %v", err)
		}
	})

	t.Run("should reject encrypted note", func(t *testing.T) {
		setNoteEncrypted(t, database, note.ID, true)
		_, err := service.ShareNote(owner.ID, note.ID, other.Username, "viewer")
		if !errors.Is(err, ErrCannotShareEncrypted) {
			t.Fatalf("expected ErrCannotShareEncrypted, got %v", err)
		}
		setNoteEncrypted(t, database, note.ID, false)
	})

	t.Run("should reject duplicate share", func(t *testing.T) {
		if _, err := service.ShareNote(owner.ID, note.ID, other.Username, "viewer"); err != nil {
			t.Fatalf("initial share failed: %v", err)
		}
		_, err := service.ShareNote(owner.ID, note.ID, other.Username, "viewer")
		if err == nil || !strings.Contains(err.Error(), "already shared") {
			t.Fatalf("expected duplicate share error, got %v", err)
		}
	})
}

func TestSharingService_ShareFolderValidation(t *testing.T) {
	database := setupTestDB(t)
	service := NewSharingService(database)
	noteService := NewNoteService(database)

	owner := createTestUser(t, database, "owner2")
	other := createTestUser(t, database, "other2")

	folder, err := database.CreateFolder(owner.ID, "/Projects", nil)
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	if err := database.UpdateFolderEncryptionDefault(owner.ID, folder.ID, false); err != nil {
		t.Fatalf("failed to ensure folder encryption default is false: %v", err)
	}

	t.Run("should reject self-share", func(t *testing.T) {
		_, err := service.ShareFolder(owner.ID, folder.ID, owner.Username, "viewer")
		if !errors.Is(err, ErrCannotShareWithSelf) {
			t.Fatalf("expected ErrCannotShareWithSelf, got %v", err)
		}
	})

	t.Run("should reject non-owner", func(t *testing.T) {
		_, err := service.ShareFolder(other.ID, folder.ID, owner.Username, "viewer")
		if !errors.Is(err, ErrNotFolderOwner) {
			t.Fatalf("expected ErrNotFolderOwner, got %v", err)
		}
	})

	t.Run("should reject encrypted folder default", func(t *testing.T) {
		if err := database.UpdateFolderEncryptionDefault(owner.ID, folder.ID, true); err != nil {
			t.Fatalf("failed to set folder encryption default: %v", err)
		}
		_, err := service.ShareFolder(owner.ID, folder.ID, other.Username, "viewer")
		if !errors.Is(err, ErrCannotShareEncryptedFolder) {
			t.Fatalf("expected ErrCannotShareEncryptedFolder, got %v", err)
		}
		if err := database.UpdateFolderEncryptionDefault(owner.ID, folder.ID, false); err != nil {
			t.Fatalf("failed to reset folder encryption default: %v", err)
		}
	})

	t.Run("should reject folder with encrypted notes", func(t *testing.T) {
		note, err := noteService.CreateNote(owner.ID, "Encrypted", "content", folder.Path)
		if err != nil {
			t.Fatalf("failed to create note: %v", err)
		}
		setNoteEncrypted(t, database, note.ID, true)

		_, err = service.ShareFolder(owner.ID, folder.ID, other.Username, "viewer")
		if !errors.Is(err, ErrFolderHasEncryptedNotes) {
			t.Fatalf("expected ErrFolderHasEncryptedNotes, got %v", err)
		}
	})
}
