//go:build fts5

package db

import (
	"fmt"
	"testing"
)

// setupGraphTestDB creates an in-memory database with test data for graph tests
func setupGraphTestDB(tb testing.TB, numNotes, numLinks, numUnresolvedLinks int) *DB {
	tb.Helper()

	db, err := Open(":memory:", "")
	if err != nil {
		tb.Fatalf("failed to open in-memory db: %v", err)
	}
	if err := db.Migrate(); err != nil {
		tb.Fatalf("failed to migrate db: %v", err)
	}

	// Create test user
	_, err = db.Exec(`INSERT INTO users (id, username, email, password_hash) VALUES (1, 'testuser', 'test@example.com', 'hash')`)
	if err != nil {
		tb.Fatalf("failed to create test user: %v", err)
	}

	// Create notes (resolved nodes)
	for i := 1; i <= numNotes; i++ {
		noteID := fmt.Sprintf("note-%d", i)
		title := fmt.Sprintf("Note %03d", i) // Zero-padded for alphabetical sort
		folderPath := fmt.Sprintf("/folder%d/", i%3)
		_, err = db.Exec(`
			INSERT INTO notes (id, user_id, title, title_norm, content, folder_path, is_deleted)
			VALUES (?, 1, ?, ?, 'Content', ?, 0)
		`, noteID, title, title, folderPath)
		if err != nil {
			tb.Fatalf("failed to create note %d: %v", i, err)
		}
	}

	// Create links (resolved edges)
	for i := 1; i <= numLinks && i < numNotes; i++ {
		sourceID := fmt.Sprintf("note-%d", i)
		targetID := fmt.Sprintf("note-%d", i+1)
		_, err = db.Exec(`
			INSERT INTO links (source_id, target_id)
			VALUES (?, ?)
		`, sourceID, targetID)
		if err != nil {
			tb.Fatalf("failed to create link %d: %v", i, err)
		}
	}

	// Create unresolved links
	for i := 1; i <= numUnresolvedLinks; i++ {
		sourceID := fmt.Sprintf("note-%d", i)
		targetRef := fmt.Sprintf("Missing Note %d", i)
		targetRefNorm := fmt.Sprintf("missing note %d", i)
		_, err = db.Exec(`
			INSERT INTO unresolved_links (source_id, target_ref, target_ref_norm)
			VALUES (?, ?, ?)
		`, sourceID, targetRef, targetRefNorm)
		if err != nil {
			tb.Fatalf("failed to create unresolved link %d: %v", i, err)
		}
	}

	return db
}

// BenchmarkGetGlobalGraph_Current benchmarks the CURRENT implementation (4 queries)
func BenchmarkGetGlobalGraph_Current(b *testing.B) {
	benchmarks := []struct {
		name            string
		notes           int
		links           int
		unresolvedLinks int
		maxNodes        int
	}{
		{"Small_100notes", 100, 99, 20, 1000},
		{"Medium_500notes", 500, 499, 100, 1000},
		{"Large_1000notes", 1000, 999, 200, 1000},
		{"Truncated_2000notes_limit1000", 2000, 1999, 400, 1000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			db := setupGraphTestDB(b, bm.notes, bm.links, bm.unresolvedLinks)
			defer db.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				graph, err := db.GetGlobalGraph(1, bm.maxNodes)
				if err != nil {
					b.Fatalf("GetGlobalGraph failed: %v", err)
				}
				// Verify we got data
				if len(graph.Nodes) == 0 {
					b.Fatal("expected nodes, got none")
				}
			}
		})
	}
}

// BenchmarkGetFilteredGraph_Current benchmarks the CURRENT implementation (4 queries)
func BenchmarkGetFilteredGraph_Current(b *testing.B) {
	benchmarks := []struct {
		name            string
		notes           int
		links           int
		unresolvedLinks int
		folderPath      string
		maxNodes        int
	}{
		{"Small_100notes_folder0", 100, 99, 20, "/folder0/", 1000},
		{"Medium_500notes_folder1", 500, 499, 100, "/folder1/", 1000},
		{"Large_1000notes_folder2", 1000, 999, 200, "/folder2/", 1000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			db := setupGraphTestDB(b, bm.notes, bm.links, bm.unresolvedLinks)
			defer db.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				graph, err := db.GetFilteredGraph(1, bm.folderPath, bm.maxNodes)
				if err != nil {
					b.Fatalf("GetFilteredGraph failed: %v", err)
				}
				// Verify we got data
				if len(graph.Nodes) == 0 {
					b.Fatal("expected nodes, got none")
				}
			}
		})
	}
}

// Tests - to be implemented after benchmarking

func TestGetGlobalGraph_EmptyDatabase(t *testing.T) {
	db := setupGraphTestDB(t, 0, 0, 0)
	defer db.Close()

	graph, err := db.GetGlobalGraph(1, 1000)
	if err != nil {
		t.Fatalf("GetGlobalGraph failed: %v", err)
	}

	if len(graph.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(graph.Edges))
	}
	if graph.Metadata.Truncated {
		t.Error("expected truncated=false for empty database")
	}
}

func TestGetGlobalGraph_ResolvedNodesOnly(t *testing.T) {
	db := setupGraphTestDB(t, 5, 0, 0)
	defer db.Close()

	graph, err := db.GetGlobalGraph(1, 1000)
	if err != nil {
		t.Fatalf("GetGlobalGraph failed: %v", err)
	}

	if len(graph.Nodes) != 5 {
		t.Errorf("expected 5 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(graph.Edges))
	}

	// All nodes should be resolved
	for _, node := range graph.Nodes {
		if !node.IsResolved {
			t.Errorf("expected node %s to be resolved", node.ID)
		}
	}
}

func TestGetGlobalGraph_UnresolvedLinks(t *testing.T) {
	db := setupGraphTestDB(t, 3, 0, 2)
	defer db.Close()

	graph, err := db.GetGlobalGraph(1, 1000)
	if err != nil {
		t.Fatalf("GetGlobalGraph failed: %v", err)
	}

	// 3 resolved + 2 unresolved = 5 nodes
	if len(graph.Nodes) != 5 {
		t.Errorf("expected 5 nodes, got %d", len(graph.Nodes))
	}

	// Count node types
	resolvedCount := 0
	unresolvedCount := 0
	for _, node := range graph.Nodes {
		if node.IsResolved {
			resolvedCount++
		} else {
			unresolvedCount++
		}
	}

	if resolvedCount != 3 {
		t.Errorf("expected 3 resolved nodes, got %d", resolvedCount)
	}
	if unresolvedCount != 2 {
		t.Errorf("expected 2 unresolved nodes, got %d", unresolvedCount)
	}

	// Check unresolved edges
	unresolvedEdges := 0
	for _, edge := range graph.Edges {
		if edge.Type == "unresolved" {
			unresolvedEdges++
		}
	}
	if unresolvedEdges != 2 {
		t.Errorf("expected 2 unresolved edges, got %d", unresolvedEdges)
	}
}

func TestGetGlobalGraph_TruncationFiltersEdges(t *testing.T) {
	// Create 5 notes with links, but LIMIT to 3
	db := setupGraphTestDB(t, 5, 4, 0)
	defer db.Close()

	graph, err := db.GetGlobalGraph(1, 3)
	if err != nil {
		t.Fatalf("GetGlobalGraph failed: %v", err)
	}

	// Should get exactly 3 nodes (truncated)
	if len(graph.Nodes) != 3 {
		t.Errorf("expected 3 nodes (truncated), got %d", len(graph.Nodes))
	}

	if !graph.Metadata.Truncated {
		t.Error("expected truncated=true when LIMIT reached")
	}

	// Build nodeID set from returned nodes
	nodeIDSet := make(map[string]bool)
	for _, node := range graph.Nodes {
		nodeIDSet[node.ID] = true
	}

	// All edges must have both endpoints in nodeIDSet
	for _, edge := range graph.Edges {
		if !nodeIDSet[edge.SourceID] {
			t.Errorf("edge source %s not in node set", edge.SourceID)
		}
		if !nodeIDSet[edge.TargetID] {
			t.Errorf("edge target %s not in node set", edge.TargetID)
		}
	}
}

func TestGetGlobalGraph_UserIsolation(t *testing.T) {
	db, err := Open(":memory:", "")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create two users
	_, err = db.Exec(`INSERT INTO users (id, username, email, password_hash) VALUES (1, 'user1', 'u1@example.com', 'hash')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO users (id, username, email, password_hash) VALUES (2, 'user2', 'u2@example.com', 'hash')`)
	if err != nil {
		t.Fatal(err)
	}

	// User 1 has a note
	_, err = db.Exec(`INSERT INTO notes (id, user_id, title, title_norm, content) VALUES ('note1', 1, 'User1 Note', 'user1 note', 'content')`)
	if err != nil {
		t.Fatal(err)
	}

	// User 2 has a note
	_, err = db.Exec(`INSERT INTO notes (id, user_id, title, title_norm, content) VALUES ('note2', 2, 'User2 Note', 'user2 note', 'content')`)
	if err != nil {
		t.Fatal(err)
	}

	// User 1's graph should only contain their note
	graph1, err := db.GetGlobalGraph(1, 1000)
	if err != nil {
		t.Fatalf("GetGlobalGraph user1 failed: %v", err)
	}
	if len(graph1.Nodes) != 1 {
		t.Errorf("user 1 expected 1 node, got %d", len(graph1.Nodes))
	}
	if graph1.Nodes[0].ID != "note1" {
		t.Errorf("user 1 expected note1, got %s", graph1.Nodes[0].ID)
	}

	// User 2's graph should only contain their note
	graph2, err := db.GetGlobalGraph(2, 1000)
	if err != nil {
		t.Fatalf("GetGlobalGraph user2 failed: %v", err)
	}
	if len(graph2.Nodes) != 1 {
		t.Errorf("user 2 expected 1 node, got %d", len(graph2.Nodes))
	}
	if graph2.Nodes[0].ID != "note2" {
		t.Errorf("user 2 expected note2, got %s", graph2.Nodes[0].ID)
	}
}

func TestGetFilteredGraph_FolderFilter(t *testing.T) {
	db := setupGraphTestDB(t, 9, 0, 0) // 3 notes per folder (0, 1, 2)
	defer db.Close()

	graph, err := db.GetFilteredGraph(1, "/folder0/", 1000)
	if err != nil {
		t.Fatalf("GetFilteredGraph failed: %v", err)
	}

	// Should get 3 notes from folder0 (notes 3, 6, 9 because i%3==0)
	if len(graph.Nodes) != 3 {
		t.Errorf("expected 3 nodes in folder0, got %d", len(graph.Nodes))
	}

	// All nodes should be in folder0
	for _, node := range graph.Nodes {
		if node.FolderPath != "/folder0/" {
			t.Errorf("expected node in /folder0/, got %s", node.FolderPath)
		}
	}
}

// TestGetGlobalGraph_TruncationWithMixedNodes tests the INNER JOIN mechanism
// This verifies that unresolved nodes are ONLY included if their source is in the truncated set
func TestGetGlobalGraph_TruncationWithMixedNodes(t *testing.T) {
	db, err := Open(":memory:", "")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create user
	_, err = db.Exec(`INSERT INTO users (id, username, email, password_hash) VALUES (1, 'testuser', 'test@example.com', 'hash')`)
	if err != nil {
		t.Fatal(err)
	}

	// Create 3 notes alphabetically: Note A, Note B, Note C
	_, err = db.Exec(`INSERT INTO notes (id, user_id, title, title_norm, content) VALUES ('note-a', 1, 'Note A', 'note a', 'content')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO notes (id, user_id, title, title_norm, content) VALUES ('note-b', 1, 'Note B', 'note b', 'content')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO notes (id, user_id, title, title_norm, content) VALUES ('note-c', 1, 'Note C', 'note c', 'content')`)
	if err != nil {
		t.Fatal(err)
	}

	// Note A has unresolved link to "Missing"
	_, err = db.Exec(`INSERT INTO unresolved_links (source_id, target_ref, target_ref_norm) VALUES ('note-a', 'Missing', 'missing')`)
	if err != nil {
		t.Fatal(err)
	}

	// Note C has unresolved link to "Also Missing"
	_, err = db.Exec(`INSERT INTO unresolved_links (source_id, target_ref, target_ref_norm) VALUES ('note-c', 'Also Missing', 'also missing')`)
	if err != nil {
		t.Fatal(err)
	}

	// LIMIT to 2 nodes → only Note A and Note B loaded
	graph, err := db.GetGlobalGraph(1, 2)
	if err != nil {
		t.Fatalf("GetGlobalGraph failed: %v", err)
	}

	if !graph.Metadata.Truncated {
		t.Error("expected truncated=true")
	}

	// Should get: Note A, Note B (resolved), Missing (unresolved from Note A)
	// Should NOT get: Note C (truncated), Also Missing (source Note C not loaded)
	if len(graph.Nodes) != 3 {
		t.Errorf("expected 3 nodes (A, B, Missing), got %d", len(graph.Nodes))
	}

	// Check that "Missing" is present but "Also Missing" is not
	hasMissing := false
	hasAlsoMissing := false
	for _, node := range graph.Nodes {
		if node.ID == "unresolved:missing" {
			hasMissing = true
		}
		if node.ID == "unresolved:also missing" {
			hasAlsoMissing = true
		}
	}

	if !hasMissing {
		t.Error("expected 'Missing' node (from loaded Note A)")
	}
	if hasAlsoMissing {
		t.Error("should NOT have 'Also Missing' node (from truncated Note C)")
	}

	// Check edge exists from Note A to Missing
	hasEdgeToMissing := false
	for _, edge := range graph.Edges {
		if edge.SourceID == "note-a" && edge.TargetID == "unresolved:missing" && edge.Type == "unresolved" {
			hasEdgeToMissing = true
		}
	}
	if !hasEdgeToMissing {
		t.Error("expected edge from note-a to unresolved:missing")
	}
}

// TestGetFilteredGraph_TruncationWithMixedNodes tests folder filtering with INNER JOIN
func TestGetFilteredGraph_TruncationWithMixedNodes(t *testing.T) {
	db, err := Open(":memory:", "")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create user
	_, err = db.Exec(`INSERT INTO users (id, username, email, password_hash) VALUES (1, 'testuser', 'test@example.com', 'hash')`)
	if err != nil {
		t.Fatal(err)
	}

	// Create notes in different folders
	_, err = db.Exec(`INSERT INTO notes (id, user_id, title, title_norm, content, folder_path) VALUES ('note-a', 1, 'Note A', 'note a', 'content', '/work/')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO notes (id, user_id, title, title_norm, content, folder_path) VALUES ('note-b', 1, 'Note B', 'note b', 'content', '/work/')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO notes (id, user_id, title, title_norm, content, folder_path) VALUES ('note-c', 1, 'Note C', 'note c', 'content', '/personal/')`)
	if err != nil {
		t.Fatal(err)
	}

	// Note A (in /work/) has unresolved link
	_, err = db.Exec(`INSERT INTO unresolved_links (source_id, target_ref, target_ref_norm) VALUES ('note-a', 'Work Missing', 'work missing')`)
	if err != nil {
		t.Fatal(err)
	}

	// Note C (in /personal/) has unresolved link
	_, err = db.Exec(`INSERT INTO unresolved_links (source_id, target_ref, target_ref_norm) VALUES ('note-c', 'Personal Missing', 'personal missing')`)
	if err != nil {
		t.Fatal(err)
	}

	// Filter to /work/ folder
	graph, err := db.GetFilteredGraph(1, "/work/", 1000)
	if err != nil {
		t.Fatalf("GetFilteredGraph failed: %v", err)
	}

	// Should get: Note A, Note B (in /work/), Work Missing (from Note A)
	// Should NOT get: Note C (/personal/), Personal Missing (from Note C in different folder)
	resolvedCount := 0
	unresolvedCount := 0
	hasWorkMissing := false
	hasPersonalMissing := false

	for _, node := range graph.Nodes {
		if node.IsResolved {
			resolvedCount++
			if node.FolderPath != "/work/" {
				t.Errorf("resolved node %s not in /work/: %s", node.ID, node.FolderPath)
			}
		} else {
			unresolvedCount++
			if node.ID == "unresolved:work missing" {
				hasWorkMissing = true
			}
			if node.ID == "unresolved:personal missing" {
				hasPersonalMissing = true
			}
		}
	}

	if resolvedCount != 2 {
		t.Errorf("expected 2 resolved nodes in /work/, got %d", resolvedCount)
	}
	if unresolvedCount != 1 {
		t.Errorf("expected 1 unresolved node from /work/, got %d", unresolvedCount)
	}
	if !hasWorkMissing {
		t.Error("expected 'Work Missing' from Note A in /work/")
	}
	if hasPersonalMissing {
		t.Error("should NOT have 'Personal Missing' from Note C in /personal/")
	}
}

// TestGetGlobalGraph_DeterministicOrder verifies that resolved nodes come before unresolved nodes
func TestGetGlobalGraph_DeterministicOrder(t *testing.T) {
	db := setupGraphTestDB(t, 3, 0, 2)
	defer db.Close()

	graph, err := db.GetGlobalGraph(1, 1000)
	if err != nil {
		t.Fatalf("GetGlobalGraph failed: %v", err)
	}

	// All resolved nodes should come before all unresolved nodes
	seenUnresolved := false
	for _, node := range graph.Nodes {
		if !node.IsResolved {
			seenUnresolved = true
		}
		if seenUnresolved && node.IsResolved {
			t.Error("resolved node found after unresolved node - ORDER BY is_resolved DESC not working")
			break
		}
	}
}
