package service

import (
	"errors"
	"fmt"
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

// --- Happy-path note sharing ---

func TestSharingService_ShareAndGetNoteShares(t *testing.T) {
	database := setupTestDB(t)
	svc := NewSharingService(database)
	noteService := NewNoteService(database)

	owner := createTestUser(t, database, "shareowner")
	viewer := createTestUser(t, database, "viewer")

	note, err := noteService.CreateNote(owner.ID, "Shared Note", "content", "/")
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	// Share note
	share, err := svc.ShareNote(owner.ID, note.ID, viewer.Username, "viewer")
	if err != nil {
		t.Fatalf("failed to share note: %v", err)
	}
	if share == nil {
		t.Fatal("expected non-nil share")
	}

	// Get shares
	shares, err := svc.GetNoteShares(owner.ID, note.ID)
	if err != nil {
		t.Fatalf("failed to get shares: %v", err)
	}
	if len(shares) != 1 {
		t.Fatalf("expected 1 share, got %d", len(shares))
	}
}

func TestSharingService_UnshareNote(t *testing.T) {
	database := setupTestDB(t)
	svc := NewSharingService(database)
	noteService := NewNoteService(database)

	owner := createTestUser(t, database, "unshareowner")
	target := createTestUser(t, database, "unsharetgt")

	note, err := noteService.CreateNote(owner.ID, "Unshare Note", "c", "/")
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	// Share, then unshare
	if _, err := svc.ShareNote(owner.ID, note.ID, target.Username, "viewer"); err != nil {
		t.Fatalf("failed to share: %v", err)
	}
	if err := svc.UnshareNote(owner.ID, note.ID, target.ID); err != nil {
		t.Fatalf("failed to unshare: %v", err)
	}

	// Verify gone
	shares, err := svc.GetNoteShares(owner.ID, note.ID)
	if err != nil {
		t.Fatalf("failed to get shares: %v", err)
	}
	if len(shares) != 0 {
		t.Fatalf("expected 0 shares after unshare, got %d", len(shares))
	}
}

func TestSharingService_UpdateShareRole(t *testing.T) {
	database := setupTestDB(t)
	svc := NewSharingService(database)
	noteService := NewNoteService(database)

	owner := createTestUser(t, database, "roleowner")
	target := createTestUser(t, database, "roletgt")

	note, err := noteService.CreateNote(owner.ID, "Role Note", "c", "/")
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	// Share as viewer
	if _, err := svc.ShareNote(owner.ID, note.ID, target.Username, "viewer"); err != nil {
		t.Fatalf("failed to share: %v", err)
	}

	// Update to editor
	if err := svc.UpdateShareRole(owner.ID, note.ID, target.ID, "editor"); err != nil {
		t.Fatalf("failed to update role: %v", err)
	}

	// Verify
	role, err := svc.CanAccessSharedNote(target.ID, note.ID)
	if err != nil {
		t.Fatalf("failed to check access: %v", err)
	}
	if role != "editor" {
		t.Errorf("expected role 'editor', got %q", role)
	}
}

func TestSharingService_GetSharedNotesForUser(t *testing.T) {
	database := setupTestDB(t)
	svc := NewSharingService(database)
	noteService := NewNoteService(database)

	owner := createTestUser(t, database, "listowner")
	target := createTestUser(t, database, "listtgt")

	// Create and share two notes
	for i := 0; i < 2; i++ {
		note, err := noteService.CreateNote(owner.ID, fmt.Sprintf("Note %d", i), "c", "/")
		if err != nil {
			t.Fatalf("failed to create note %d: %v", i, err)
		}
		if _, err := svc.ShareNote(owner.ID, note.ID, target.Username, "viewer"); err != nil {
			t.Fatalf("failed to share note %d: %v", i, err)
		}
	}

	shared, err := svc.GetSharedNotesForUser(target.ID)
	if err != nil {
		t.Fatalf("failed to get shared notes: %v", err)
	}
	if len(shared) != 2 {
		t.Errorf("expected 2 shared notes, got %d", len(shared))
	}
}

func TestSharingService_GetSharedNote(t *testing.T) {
	database := setupTestDB(t)
	svc := NewSharingService(database)
	noteService := NewNoteService(database)

	owner := createTestUser(t, database, "getshowner")
	target := createTestUser(t, database, "getshtgt")

	note, err := noteService.CreateNote(owner.ID, "Get Shared", "shared content", "/")
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	if _, err := svc.ShareNote(owner.ID, note.ID, target.Username, "viewer"); err != nil {
		t.Fatalf("failed to share: %v", err)
	}

	t.Run("returns shared note", func(t *testing.T) {
		sn, err := svc.GetSharedNote(target.ID, note.ID)
		if err != nil {
			t.Fatalf("failed to get shared note: %v", err)
		}
		if sn.Title != "Get Shared" {
			t.Errorf("expected title 'Get Shared', got %q", sn.Title)
		}
	})

	t.Run("returns not found for non-shared user", func(t *testing.T) {
		other := createTestUser(t, database, "noaccess")
		_, err := svc.GetSharedNote(other.ID, note.ID)
		if !errors.Is(err, db.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestSharingService_UpdateSharedNote(t *testing.T) {
	database := setupTestDB(t)
	svc := NewSharingService(database)
	noteService := NewNoteService(database)

	owner := createTestUser(t, database, "updshowner")
	editor := createTestUser(t, database, "upsheditor")
	viewer := createTestUser(t, database, "updshviewer")

	note, err := noteService.CreateNote(owner.ID, "Edit Shared", "original", "/")
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	// Share with editor role
	if _, err := svc.ShareNote(owner.ID, note.ID, editor.Username, "editor"); err != nil {
		t.Fatalf("failed to share as editor: %v", err)
	}
	// Share with viewer role
	if _, err := svc.ShareNote(owner.ID, note.ID, viewer.Username, "viewer"); err != nil {
		t.Fatalf("failed to share as viewer: %v", err)
	}

	t.Run("editor can update", func(t *testing.T) {
		updated, err := svc.UpdateSharedNote(editor.ID, note.ID, "Edit Shared", "edited by editor", note.Version)
		if err != nil {
			t.Fatalf("editor update failed: %v", err)
		}
		if updated.Content != "edited by editor" {
			t.Errorf("expected updated content, got %q", updated.Content)
		}
	})

	t.Run("viewer cannot update", func(t *testing.T) {
		_, err := svc.UpdateSharedNote(viewer.ID, note.ID, "Edit Shared", "viewer edit", 1)
		if err == nil || !strings.Contains(err.Error(), "insufficient permissions") {
			t.Errorf("expected permission error, got %v", err)
		}
	})
}

func TestSharingService_CanAccessSharedNote(t *testing.T) {
	database := setupTestDB(t)
	svc := NewSharingService(database)
	noteService := NewNoteService(database)

	owner := createTestUser(t, database, "accowner")
	shared := createTestUser(t, database, "accshared")
	unshared := createTestUser(t, database, "accunshared")

	note, err := noteService.CreateNote(owner.ID, "Access Note", "c", "/")
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	if _, err := svc.ShareNote(owner.ID, note.ID, shared.Username, "viewer"); err != nil {
		t.Fatalf("failed to share: %v", err)
	}

	t.Run("shared user has access", func(t *testing.T) {
		role, err := svc.CanAccessSharedNote(shared.ID, note.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if role == "" {
			t.Error("expected non-empty role for shared user")
		}
	})

	t.Run("unshared user has no access", func(t *testing.T) {
		role, err := svc.CanAccessSharedNote(unshared.ID, note.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if role != "" {
			t.Errorf("expected empty role, got %q", role)
		}
	})
}

// --- Folder sharing happy path ---

func TestSharingService_ShareAndGetFolderShares(t *testing.T) {
	database := setupTestDB(t)
	svc := NewSharingService(database)

	owner := createTestUser(t, database, "fshowner")
	target := createTestUser(t, database, "fshtgt")

	folder, err := database.CreateFolder(owner.ID, "/Shared", nil)
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	if err := database.UpdateFolderEncryptionDefault(owner.ID, folder.ID, false); err != nil {
		t.Fatalf("failed to set encryption default: %v", err)
	}

	// Share folder
	share, err := svc.ShareFolder(owner.ID, folder.ID, target.Username, "viewer")
	if err != nil {
		t.Fatalf("failed to share folder: %v", err)
	}
	if share == nil {
		t.Fatal("expected non-nil share")
	}

	// Get shares
	shares, err := svc.GetFolderShares(owner.ID, folder.ID)
	if err != nil {
		t.Fatalf("failed to get folder shares: %v", err)
	}
	if len(shares) != 1 {
		t.Fatalf("expected 1 folder share, got %d", len(shares))
	}
}

func TestSharingService_UnshareFolderAndRoleUpdate(t *testing.T) {
	database := setupTestDB(t)
	svc := NewSharingService(database)

	owner := createTestUser(t, database, "frolowner")
	target := createTestUser(t, database, "froltgt")

	folder, err := database.CreateFolder(owner.ID, "/FolderRole", nil)
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	if err := database.UpdateFolderEncryptionDefault(owner.ID, folder.ID, false); err != nil {
		t.Fatalf("failed to set encryption default: %v", err)
	}

	// Share
	if _, err := svc.ShareFolder(owner.ID, folder.ID, target.Username, "viewer"); err != nil {
		t.Fatalf("failed to share: %v", err)
	}

	// Update role
	if err := svc.UpdateFolderShareRole(owner.ID, folder.ID, target.ID, "editor"); err != nil {
		t.Fatalf("failed to update role: %v", err)
	}

	// Unshare
	if err := svc.UnshareFolder(owner.ID, folder.ID, target.ID); err != nil {
		t.Fatalf("failed to unshare: %v", err)
	}

	// Verify gone
	shares, err := svc.GetFolderShares(owner.ID, folder.ID)
	if err != nil {
		t.Fatalf("failed to get shares: %v", err)
	}
	if len(shares) != 0 {
		t.Errorf("expected 0 shares, got %d", len(shares))
	}
}

func TestSharingService_GetSharedFoldersForUser(t *testing.T) {
	database := setupTestDB(t)
	svc := NewSharingService(database)

	owner := createTestUser(t, database, "gsfowner")
	target := createTestUser(t, database, "gsftgt")

	folder, err := database.CreateFolder(owner.ID, "/ListShared", nil)
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}
	if err := database.UpdateFolderEncryptionDefault(owner.ID, folder.ID, false); err != nil {
		t.Fatalf("failed to set encryption default: %v", err)
	}

	if _, err := svc.ShareFolder(owner.ID, folder.ID, target.Username, "viewer"); err != nil {
		t.Fatalf("failed to share: %v", err)
	}

	folders, err := svc.GetSharedFoldersForUser(target.ID)
	if err != nil {
		t.Fatalf("failed to get shared folders: %v", err)
	}
	if len(folders) != 1 {
		t.Errorf("expected 1 shared folder, got %d", len(folders))
	}
}

func TestSharingService_SearchUsers(t *testing.T) {
	database := setupTestDB(t)
	svc := NewSharingService(database)

	user1 := createTestUser(t, database, "searchuser1")
	createTestUser(t, database, "searchuser2")

	results, err := svc.SearchUsers("searchuser", user1.ID)
	if err != nil {
		t.Fatalf("failed to search users: %v", err)
	}
	// Should find searchuser2 but not searchuser1 (self)
	if len(results) != 1 {
		t.Errorf("expected 1 search result (excluding self), got %d", len(results))
	}
}
