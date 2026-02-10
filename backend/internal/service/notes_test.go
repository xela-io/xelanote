package service

import (
	"testing"
	"time"
)

func TestNoteService_UpdateNote_RejectsInvalidFolderPath(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)

	user := createTestUser(t, database, "folderuser")

	note, err := service.CreateNote(user.ID, "Title", "content", "/")
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	_, err = service.UpdateNote(user.ID, note.ID, "Title", "content", "invalid", note.Version)
	if err == nil {
		t.Fatalf("expected error for invalid folder path")
	}
}

func TestNoteService_UpdateNote_SnapshotThreshold(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)

	user := createTestUser(t, database, "snapshotuser")

	note, err := service.CreateNote(user.ID, "Title", "content", "/")
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	updated, err := service.UpdateNote(user.ID, note.ID, "Title", "content v2", "", note.Version)
	if err != nil {
		t.Fatalf("failed to update note: %v", err)
	}

	versions, _, total, err := service.GetNoteVersions(user.ID, note.ID, 10, "")
	if err != nil {
		t.Fatalf("failed to get versions: %v", err)
	}
	if total != 1 || len(versions) != 1 {
		t.Fatalf("expected 1 version snapshot, got %d", total)
	}

	// Second update within threshold should not create another snapshot
	if _, err := service.UpdateNote(user.ID, note.ID, "Title", "content v3", "", updated.Version); err != nil {
		t.Fatalf("failed to update note: %v", err)
	}

	versions, _, total, err = service.GetNoteVersions(user.ID, note.ID, 10, "")
	if err != nil {
		t.Fatalf("failed to get versions: %v", err)
	}
	if total != 1 || len(versions) != 1 {
		t.Fatalf("expected still 1 version snapshot, got %d", total)
	}
}

func TestNoteService_UpdateNote_SnapshotAfterThreshold(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)

	user := createTestUser(t, database, "thresholduser")

	note, err := service.CreateNote(user.ID, "Title", "content", "/")
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	updated, err := service.UpdateNote(user.ID, note.ID, "Title", "content v2", "", note.Version)
	if err != nil {
		t.Fatalf("failed to update note: %v", err)
	}

	// Make the last snapshot old enough to trigger a new one
	if _, err := database.Exec(`UPDATE note_versions SET snapshot_at = ? WHERE note_id = ?`,
		time.Now().Add(-snapshotThreshold*2).Format(time.RFC3339), note.ID); err != nil {
		t.Fatalf("failed to backdate snapshot: %v", err)
	}

	if _, err := service.UpdateNote(user.ID, note.ID, "Title", "content v3", "", updated.Version); err != nil {
		t.Fatalf("failed to update note: %v", err)
	}

	versions, _, total, err := service.GetNoteVersions(user.ID, note.ID, 10, "")
	if err != nil {
		t.Fatalf("failed to get versions: %v", err)
	}
	if total < 2 || len(versions) < 2 {
		t.Fatalf("expected 2+ version snapshots, got %d", total)
	}
}
