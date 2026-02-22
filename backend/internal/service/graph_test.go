//go:build fts5

package service

import (
	"testing"

	"github.com/xela-io/xelanote/internal/cache"
)

func setupGraphTest(t *testing.T) (*GraphService, *NoteService, int) {
	t.Helper()

	database := setupTestDB(t)
	c := cache.New(0)
	t.Cleanup(func() { c.Close() })

	graphSvc := NewGraphService(database, c)
	noteSvc := NewNoteService(database)
	user := createTestUser(t, database, "graphuser")

	return graphSvc, noteSvc, user.ID
}

func TestGraphService_GetGlobalGraph_Empty(t *testing.T) {
	graphSvc, _, userID := setupGraphTest(t)

	graph, err := graphSvc.GetGlobalGraph(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if graph == nil {
		t.Fatal("expected non-nil graph")
	}
	if len(graph.Nodes) != 0 {
		t.Errorf("expected 0 nodes for empty graph, got %d", len(graph.Nodes))
	}
}

func TestGraphService_GetGlobalGraph_WithNotes(t *testing.T) {
	graphSvc, noteSvc, userID := setupGraphTest(t)

	// Create notes with wikilinks
	if _, err := noteSvc.CreateNote(userID, "Note A", "Link to [[Note B]]", "/"); err != nil {
		t.Fatalf("failed to create note A: %v", err)
	}
	if _, err := noteSvc.CreateNote(userID, "Note B", "Link to [[Note A]]", "/"); err != nil {
		t.Fatalf("failed to create note B: %v", err)
	}

	graph, err := graphSvc.GetGlobalGraph(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(graph.Nodes) < 2 {
		t.Errorf("expected at least 2 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) < 1 {
		t.Errorf("expected at least 1 edge, got %d", len(graph.Edges))
	}
}

func TestGraphService_GetGlobalGraph_Caching(t *testing.T) {
	graphSvc, noteSvc, userID := setupGraphTest(t)

	if _, err := noteSvc.CreateNote(userID, "Cached Note", "content", "/"); err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	// First call (cache miss)
	graph1, err := graphSvc.GetGlobalGraph(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call (cache hit - should return same data)
	graph2, err := graphSvc.GetGlobalGraph(userID)
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}

	if len(graph1.Nodes) != len(graph2.Nodes) {
		t.Errorf("cached response differs: %d vs %d nodes", len(graph1.Nodes), len(graph2.Nodes))
	}
}

func TestGraphService_InvalidateGraphCache(t *testing.T) {
	graphSvc, noteSvc, userID := setupGraphTest(t)

	if _, err := noteSvc.CreateNote(userID, "Invalidated Note", "content", "/"); err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	// Populate cache
	_, err := graphSvc.GetGlobalGraph(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invalidate
	graphSvc.InvalidateGraphCache(userID)

	// Next call should be a cache miss (no error expected)
	graph, err := graphSvc.GetGlobalGraph(userID)
	if err != nil {
		t.Fatalf("unexpected error after invalidation: %v", err)
	}
	if graph == nil {
		t.Fatal("expected non-nil graph after cache invalidation")
	}
}

func TestGraphService_GetFilteredGraph(t *testing.T) {
	graphSvc, noteSvc, userID := setupGraphTest(t)

	// Create notes in different folders
	if _, err := noteSvc.CreateNote(userID, "Project Note", "Link to [[Other]]", "/Projects"); err != nil {
		t.Fatalf("failed to create note: %v", err)
	}
	if _, err := noteSvc.CreateNote(userID, "Other", "content", "/Other"); err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	graph, err := graphSvc.GetFilteredGraph(userID, "/Projects")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if graph == nil {
		t.Fatal("expected non-nil filtered graph")
	}
}

func TestGraphService_CrossUserIsolation(t *testing.T) {
	database := setupTestDB(t)
	c := cache.New(0)
	t.Cleanup(func() { c.Close() })

	graphSvc := NewGraphService(database, c)
	noteSvc := NewNoteService(database)

	user1 := createTestUser(t, database, "graphuser1")
	user2 := createTestUser(t, database, "graphuser2")

	// User1 creates a note
	if _, err := noteSvc.CreateNote(user1.ID, "Private Note", "content", "/"); err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	// User2's graph should be empty
	graph, err := graphSvc.GetGlobalGraph(user2.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(graph.Nodes) != 0 {
		t.Errorf("expected 0 nodes for user2, got %d", len(graph.Nodes))
	}
}
