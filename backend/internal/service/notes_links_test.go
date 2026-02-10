package service

import (
	"testing"
)

func TestUpdateLinksFromClient_ResolvedLinks(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)
	user := createTestUser(t, database, "user")

	// Create target notes
	target1, err := service.CreateNote(user.ID, "Target Note One", "content1", "/")
	if err != nil {
		t.Fatalf("failed to create target1: %v", err)
	}
	target2, err := service.CreateNote(user.ID, "Target Note Two", "content2", "/")
	if err != nil {
		t.Fatalf("failed to create target2: %v", err)
	}

	// Create source note (no links yet)
	source, err := service.CreateNote(user.ID, "Source Note", "content", "/")
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Add links via client (simulating E2E encrypted note)
	linkTitles := []string{"Target Note One", "Target Note Two"}
	if err := service.UpdateLinksFromClient(user.ID, source.ID, linkTitles); err != nil {
		t.Fatalf("UpdateLinksFromClient failed: %v", err)
	}

	// Verify backlinks
	backlinks1, err := service.GetBacklinks(user.ID, target1.ID)
	if err != nil {
		t.Fatalf("GetBacklinks failed for target1: %v", err)
	}
	if len(backlinks1) != 1 {
		t.Errorf("expected 1 backlink to target1, got %d", len(backlinks1))
	}
	if len(backlinks1) > 0 && backlinks1[0].ID != source.ID {
		t.Errorf("expected backlink from source, got %s", backlinks1[0].ID)
	}

	backlinks2, err := service.GetBacklinks(user.ID, target2.ID)
	if err != nil {
		t.Fatalf("GetBacklinks failed for target2: %v", err)
	}
	if len(backlinks2) != 1 {
		t.Errorf("expected 1 backlink to target2, got %d", len(backlinks2))
	}
}

func TestUpdateLinksFromClient_UnresolvedLinks(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)
	user := createTestUser(t, database, "user")

	// Create source note with empty content (simulating E2E encrypted note)
	// The content is empty because the server can't decrypt it
	source, err := service.CreateNote(user.ID, "Source Note", "", "/")
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Add links to non-existent notes (client-side extracted from encrypted content)
	linkTitles := []string{"Non Existent Note", "Another Missing Note"}
	if err := service.UpdateLinksFromClient(user.ID, source.ID, linkTitles); err != nil {
		t.Fatalf("UpdateLinksFromClient failed: %v", err)
	}

	// Now create one of the target notes
	target, err := service.CreateNote(user.ID, "Non Existent Note", "now exists", "/")
	if err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// The unresolved link should have been resolved when target was created
	// Because source note has empty content, resolveUnresolvedLinks should use
	// the direct resolution method (ResolveUnresolvedLink)
	backlinks, err := service.GetBacklinks(user.ID, target.ID)
	if err != nil {
		t.Fatalf("GetBacklinks failed: %v", err)
	}
	if len(backlinks) != 1 {
		t.Errorf("expected 1 backlink after resolving, got %d", len(backlinks))
	}
}

func TestUpdateLinksFromClient_Deduplication(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)
	user := createTestUser(t, database, "user")

	// Create target note
	target, err := service.CreateNote(user.ID, "Target", "content", "/")
	if err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Create source note
	source, err := service.CreateNote(user.ID, "Source", "content", "/")
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Add duplicate links (different cases, spaces)
	linkTitles := []string{"Target", "target", "TARGET", " Target ", "target"}
	if err := service.UpdateLinksFromClient(user.ID, source.ID, linkTitles); err != nil {
		t.Fatalf("UpdateLinksFromClient failed: %v", err)
	}

	// Should only have one backlink despite duplicates
	backlinks, err := service.GetBacklinks(user.ID, target.ID)
	if err != nil {
		t.Fatalf("GetBacklinks failed: %v", err)
	}
	if len(backlinks) != 1 {
		t.Errorf("expected 1 backlink after deduplication, got %d", len(backlinks))
	}
}

func TestUpdateLinksFromClient_UnresolvedCleanup(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)
	user := createTestUser(t, database, "user")

	// Create source note
	source, err := service.CreateNote(user.ID, "Source", "content", "/")
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Add unresolved links
	linkTitles := []string{"Ghost Note", "Another Ghost"}
	if err := service.UpdateLinksFromClient(user.ID, source.ID, linkTitles); err != nil {
		t.Fatalf("UpdateLinksFromClient failed: %v", err)
	}

	// Remove all links from client
	if err := service.UpdateLinksFromClient(user.ID, source.ID, []string{}); err != nil {
		t.Fatalf("UpdateLinksFromClient cleanup failed: %v", err)
	}

	// Create target note; should not resolve since unresolved links were cleared
	target, err := service.CreateNote(user.ID, "Ghost Note", "now exists", "/")
	if err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	backlinks, err := service.GetBacklinks(user.ID, target.ID)
	if err != nil {
		t.Fatalf("GetBacklinks failed: %v", err)
	}
	if len(backlinks) != 0 {
		t.Fatalf("expected 0 backlinks after unresolved cleanup, got %d", len(backlinks))
	}
}

func TestGetBacklinks_ExcludesDeletedSource(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)
	user := createTestUser(t, database, "user")

	target, err := service.CreateNote(user.ID, "Target", "content", "/")
	if err != nil {
		t.Fatalf("failed to create target: %v", err)
	}
	source, err := service.CreateNote(user.ID, "Source", "content", "/")
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	if err := service.UpdateLinksFromClient(user.ID, source.ID, []string{"Target"}); err != nil {
		t.Fatalf("UpdateLinksFromClient failed: %v", err)
	}

	backlinks, err := service.GetBacklinks(user.ID, target.ID)
	if err != nil {
		t.Fatalf("GetBacklinks failed: %v", err)
	}
	if len(backlinks) != 1 {
		t.Fatalf("expected 1 backlink, got %d", len(backlinks))
	}

	if err := service.DeleteNote(user.ID, source.ID); err != nil {
		t.Fatalf("failed to delete source: %v", err)
	}

	backlinks, err = service.GetBacklinks(user.ID, target.ID)
	if err != nil {
		t.Fatalf("GetBacklinks failed: %v", err)
	}
	if len(backlinks) != 0 {
		t.Fatalf("expected 0 backlinks after deleting source, got %d", len(backlinks))
	}
}

func TestGetBacklinks_IsUserScoped(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)
	user1 := createTestUser(t, database, "user1")
	user2 := createTestUser(t, database, "user2")

	target, err := service.CreateNote(user1.ID, "SharedTitle", "content", "/")
	if err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	source, err := service.CreateNote(user2.ID, "Source", "content", "/")
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	if err := service.UpdateLinksFromClient(user2.ID, source.ID, []string{"SharedTitle"}); err != nil {
		t.Fatalf("UpdateLinksFromClient failed: %v", err)
	}

	backlinks, err := service.GetBacklinks(user1.ID, target.ID)
	if err != nil {
		t.Fatalf("GetBacklinks failed: %v", err)
	}
	if len(backlinks) != 0 {
		t.Fatalf("expected 0 backlinks across users, got %d", len(backlinks))
	}
}

func TestUpdateLinksFromClient_EmptyTitlesSkipped(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)
	user := createTestUser(t, database, "user")

	// Create target note
	target, err := service.CreateNote(user.ID, "Target", "content", "/")
	if err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Create source note
	source, err := service.CreateNote(user.ID, "Source", "content", "/")
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Add links with empty titles
	linkTitles := []string{"", "Target", "", "   ", ""}
	if err := service.UpdateLinksFromClient(user.ID, source.ID, linkTitles); err != nil {
		t.Fatalf("UpdateLinksFromClient failed: %v", err)
	}

	// Should only have one valid backlink
	backlinks, err := service.GetBacklinks(user.ID, target.ID)
	if err != nil {
		t.Fatalf("GetBacklinks failed: %v", err)
	}
	if len(backlinks) != 1 {
		t.Errorf("expected 1 backlink after skipping empty titles, got %d", len(backlinks))
	}
}

func TestUpdateLinksFromClient_TooManyLinks(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)
	user := createTestUser(t, database, "user")

	// Create source note
	source, err := service.CreateNote(user.ID, "Source", "content", "/")
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Create 501 links (exceeds limit of 500)
	linkTitles := make([]string, 501)
	for i := range linkTitles {
		linkTitles[i] = "Link " + string(rune('A'+i%26))
	}

	err = service.UpdateLinksFromClient(user.ID, source.ID, linkTitles)
	if err != ErrTooManyLinks {
		t.Errorf("expected ErrTooManyLinks, got %v", err)
	}
}

func TestUpdateLinksFromClient_LongTitlesSkipped(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)
	user := createTestUser(t, database, "user")

	// Create target note with valid name
	target, err := service.CreateNote(user.ID, "Valid", "content", "/")
	if err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Create source note
	source, err := service.CreateNote(user.ID, "Source", "content", "/")
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Create link with very long title (> 200 chars)
	longTitle := make([]byte, 250)
	for i := range longTitle {
		longTitle[i] = 'a'
	}

	linkTitles := []string{string(longTitle), "Valid"}
	if err := service.UpdateLinksFromClient(user.ID, source.ID, linkTitles); err != nil {
		t.Fatalf("UpdateLinksFromClient failed: %v", err)
	}

	// Should only have one valid backlink (long title skipped silently)
	backlinks, err := service.GetBacklinks(user.ID, target.ID)
	if err != nil {
		t.Fatalf("GetBacklinks failed: %v", err)
	}
	if len(backlinks) != 1 {
		t.Errorf("expected 1 backlink after skipping long title, got %d", len(backlinks))
	}
}

func TestUpdateLinksFromClient_ReplacesExistingLinks(t *testing.T) {
	database := setupTestDB(t)
	service := NewNoteService(database)
	user := createTestUser(t, database, "user")

	// Create target notes
	target1, err := service.CreateNote(user.ID, "Target1", "content1", "/")
	if err != nil {
		t.Fatalf("failed to create target1: %v", err)
	}
	target2, err := service.CreateNote(user.ID, "Target2", "content2", "/")
	if err != nil {
		t.Fatalf("failed to create target2: %v", err)
	}

	// Create source note
	source, err := service.CreateNote(user.ID, "Source", "content", "/")
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Add link to target1
	if err := service.UpdateLinksFromClient(user.ID, source.ID, []string{"Target1"}); err != nil {
		t.Fatalf("UpdateLinksFromClient failed: %v", err)
	}

	// Verify backlink to target1
	backlinks1, err := service.GetBacklinks(user.ID, target1.ID)
	if err != nil {
		t.Fatalf("GetBacklinks failed: %v", err)
	}
	if len(backlinks1) != 1 {
		t.Errorf("expected 1 backlink to target1, got %d", len(backlinks1))
	}

	// Now update links to only target2 (removing target1)
	if err := service.UpdateLinksFromClient(user.ID, source.ID, []string{"Target2"}); err != nil {
		t.Fatalf("UpdateLinksFromClient failed: %v", err)
	}

	// Verify backlink to target1 is gone
	backlinks1After, err := service.GetBacklinks(user.ID, target1.ID)
	if err != nil {
		t.Fatalf("GetBacklinks failed: %v", err)
	}
	if len(backlinks1After) != 0 {
		t.Errorf("expected 0 backlinks to target1 after update, got %d", len(backlinks1After))
	}

	// Verify backlink to target2 exists
	backlinks2, err := service.GetBacklinks(user.ID, target2.ID)
	if err != nil {
		t.Fatalf("GetBacklinks failed: %v", err)
	}
	if len(backlinks2) != 1 {
		t.Errorf("expected 1 backlink to target2, got %d", len(backlinks2))
	}
}
