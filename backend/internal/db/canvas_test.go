package db

import (
	"testing"
)

func TestCreateCanvasNote(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	note, err := db.CreateCanvasNote(userID, "My Canvas", `{"nodes":[],"edges":[]}`, "/")
	if err != nil {
		t.Fatalf("failed to create canvas note: %v", err)
	}

	if note.ID == "" {
		t.Error("expected non-empty note ID")
	}
	if note.Title != "My Canvas" {
		t.Errorf("expected title 'My Canvas', got %q", note.Title)
	}
	if note.NoteType != NoteTypeCanvas {
		t.Errorf("expected note_type 'canvas', got %q", note.NoteType)
	}
	if note.FolderPath != "/" {
		t.Errorf("expected folder_path '/', got %q", note.FolderPath)
	}
	if note.Content != `{"nodes":[],"edges":[]}` {
		t.Errorf("expected canvas JSON content, got %q", note.Content)
	}
}

func TestCreateCanvasNote_CustomFolder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create folder first
	_, err := db.Exec(`INSERT INTO folders (path, name, user_id, created_at) VALUES ('/projects', 'projects', ?, datetime('now'))`, userID)
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}

	note, err := db.CreateCanvasNote(userID, "Project Canvas", `{"nodes":[],"edges":[]}`, "/projects")
	if err != nil {
		t.Fatalf("failed to create canvas note: %v", err)
	}

	if note.FolderPath != "/projects" {
		t.Errorf("expected folder_path '/projects', got %q", note.FolderPath)
	}
}

func TestListCanvasNotes_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	notes, err := db.ListCanvasNotes(userID)
	if err != nil {
		t.Fatalf("failed to list canvas notes: %v", err)
	}

	if len(notes) != 0 {
		t.Errorf("expected 0 canvas notes, got %d", len(notes))
	}
}

func TestListCanvasNotes_OnlyCanvasType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create a regular note
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, note_type, version, created_at, updated_at)
		VALUES ('note1', 'Regular Note', 'regular note', 'content', '/', ?, 'note', 1, datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("failed to create regular note: %v", err)
	}

	// Create a canvas note
	_, err = db.CreateCanvasNote(userID, "Canvas 1", `{"nodes":[],"edges":[]}`, "/")
	if err != nil {
		t.Fatalf("failed to create canvas note: %v", err)
	}

	// Create another canvas note
	_, err = db.CreateCanvasNote(userID, "Canvas 2", `{"nodes":[],"edges":[]}`, "/")
	if err != nil {
		t.Fatalf("failed to create canvas note: %v", err)
	}

	canvasNotes, err := db.ListCanvasNotes(userID)
	if err != nil {
		t.Fatalf("failed to list canvas notes: %v", err)
	}

	if len(canvasNotes) != 2 {
		t.Errorf("expected 2 canvas notes, got %d", len(canvasNotes))
	}

	for _, note := range canvasNotes {
		if note.NoteType != NoteTypeCanvas {
			t.Errorf("expected note_type 'canvas', got %q for note %s", note.NoteType, note.ID)
		}
	}
}

func TestListCanvasNotes_ExcludesDeletedNotes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	note, err := db.CreateCanvasNote(userID, "Canvas To Delete", `{"nodes":[],"edges":[]}`, "/")
	if err != nil {
		t.Fatalf("failed to create canvas note: %v", err)
	}

	// Soft-delete the note
	_, err = db.Exec(`UPDATE notes SET is_deleted = 1 WHERE id = ?`, note.ID)
	if err != nil {
		t.Fatalf("failed to soft-delete note: %v", err)
	}

	canvasNotes, err := db.ListCanvasNotes(userID)
	if err != nil {
		t.Fatalf("failed to list canvas notes: %v", err)
	}

	if len(canvasNotes) != 0 {
		t.Errorf("expected 0 canvas notes after deletion, got %d", len(canvasNotes))
	}
}

func TestListCanvasNotes_DoesNotReturnOtherUsersNotes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create two users
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (1, 'user1', 'user1@test.com', 'hash', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("failed to create user 1: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at)
		VALUES (2, 'user2', 'user2@test.com', 'hash', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("failed to create user 2: %v", err)
	}

	// Each user creates a canvas note
	_, err = db.CreateCanvasNote(1, "User1 Canvas", `{"nodes":[],"edges":[]}`, "/")
	if err != nil {
		t.Fatalf("failed to create canvas note for user 1: %v", err)
	}

	_, err = db.CreateCanvasNote(2, "User2 Canvas", `{"nodes":[],"edges":[]}`, "/")
	if err != nil {
		t.Fatalf("failed to create canvas note for user 2: %v", err)
	}

	// User 1 should only see their canvas
	user1Notes, err := db.ListCanvasNotes(1)
	if err != nil {
		t.Fatalf("failed to list canvas notes for user 1: %v", err)
	}
	if len(user1Notes) != 1 {
		t.Errorf("expected 1 canvas note for user 1, got %d", len(user1Notes))
	}
	if user1Notes[0].Title != "User1 Canvas" {
		t.Errorf("expected title 'User1 Canvas', got %q", user1Notes[0].Title)
	}
}

func TestCanvasNoteAppearsInListNotesByFolder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create a canvas note in root folder
	_, err := db.CreateCanvasNote(userID, "My Canvas", `{"nodes":[],"edges":[]}`, "/")
	if err != nil {
		t.Fatalf("failed to create canvas note: %v", err)
	}

	// ListNotesByFolder should include canvas notes
	notesList, err := db.ListNotesByFolder(userID, "/", "id, title, note_type")
	if err != nil {
		t.Fatalf("failed to list notes by folder: %v", err)
	}

	found := false
	for _, n := range notesList {
		if n.Title == "My Canvas" {
			found = true
			if n.NoteType != NoteTypeCanvas {
				t.Errorf("expected note_type 'canvas', got %q", n.NoteType)
			}
			break
		}
	}
	if !found {
		t.Error("canvas note was not found in ListNotesByFolder results")
	}
}
