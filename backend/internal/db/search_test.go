package db

import (
	"context"
	"testing"
)

func TestEscapeSnippetHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "preserves mark tags",
			input:    "This is a <mark>test</mark> snippet",
			expected: "This is a <mark>test</mark> snippet",
		},
		{
			name:     "escapes script tag",
			input:    "This is <script>alert(1)</script> malicious",
			expected: "This is &lt;script&gt;alert(1)&lt;/script&gt; malicious",
		},
		{
			name:     "escapes XSS in marked content",
			input:    "Found: <mark><script>alert(1)</script></mark>",
			expected: "Found: <mark>&lt;script&gt;alert(1)&lt;/script&gt;</mark>",
		},
		{
			name:     "escapes HTML entities",
			input:    "Test <b>bold</b> & \"quotes\" <mark>highlighted</mark>",
			expected: "Test &lt;b&gt;bold&lt;/b&gt; &amp; &#34;quotes&#34; <mark>highlighted</mark>",
		},
		{
			name:     "multiple mark tags preserved",
			input:    "<mark>one</mark> and <mark>two</mark>",
			expected: "<mark>one</mark> and <mark>two</mark>",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only mark tags",
			input:    "<mark>test</mark>",
			expected: "<mark>test</mark>",
		},
		{
			name:     "nested HTML in mark",
			input:    "<mark><img src=x onerror=alert(1)></mark>",
			expected: "<mark>&lt;img src=x onerror=alert(1)&gt;</mark>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeSnippetHTML(tt.input)
			if result != tt.expected {
				t.Errorf("escapeSnippetHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildFTSQuery(t *testing.T) {
	tests := []struct {
		name     string
		terms    []string
		expected string
	}{
		{
			name:     "single term",
			terms:    []string{"hello"},
			expected: `"hello"*`,
		},
		{
			name:     "multiple terms",
			terms:    []string{"hello", "world"},
			expected: `"hello"* "world"*`,
		},
		{
			name:     "term with quotes",
			terms:    []string{`say "hello"`},
			expected: `"say ""hello"""*`,
		},
		{
			name:     "empty terms",
			terms:    []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFTSQuery(tt.terms)
			if result != tt.expected {
				t.Errorf("buildFTSQuery(%v) = %q, want %q", tt.terms, result, tt.expected)
			}
		})
	}
}

func TestMergeSearchResults(t *testing.T) {
	t.Run("plaintext first then keywords", func(t *testing.T) {
		plaintext := []SearchResult{
			{ID: "p1", Title: "Plain 1", Snippet: "snippet 1", Rank: -1.0},
			{ID: "p2", Title: "Plain 2", Snippet: "snippet 2", Rank: -0.5},
		}
		keywords := []SearchResult{
			{ID: "k1", Title: "", Encrypted: true, MatchedKeywords: []string{"bitcoin", "crypto"}, Rank: -0.8},
		}

		merged := mergeSearchResults(plaintext, keywords, 10)

		if len(merged) != 3 {
			t.Fatalf("expected 3 results, got %d", len(merged))
		}
		if merged[0].ID != "p1" {
			t.Errorf("expected first result to be p1, got %s", merged[0].ID)
		}
		if merged[1].ID != "p2" {
			t.Errorf("expected second result to be p2, got %s", merged[1].ID)
		}
		if merged[2].ID != "k1" {
			t.Errorf("expected third result to be k1, got %s", merged[2].ID)
		}
		if !merged[2].Encrypted {
			t.Error("expected k1 to be encrypted")
		}
	})

	t.Run("limit applied", func(t *testing.T) {
		plaintext := []SearchResult{
			{ID: "p1", Title: "Plain 1"},
			{ID: "p2", Title: "Plain 2"},
		}
		keywords := []SearchResult{
			{ID: "k1", Encrypted: true, MatchedKeywords: []string{"test"}},
			{ID: "k2", Encrypted: true, MatchedKeywords: []string{"test2"}},
		}

		merged := mergeSearchResults(plaintext, keywords, 3)

		if len(merged) != 3 {
			t.Fatalf("expected 3 results (limited), got %d", len(merged))
		}
	})

	t.Run("empty inputs", func(t *testing.T) {
		merged := mergeSearchResults(nil, nil, 10)
		if len(merged) != 0 {
			t.Fatalf("expected 0 results, got %d", len(merged))
		}
	})
}

func TestMergeSearchResultsDeduplicate(t *testing.T) {
	plaintext := []SearchResult{
		{ID: "shared", Title: "Shared Note", Snippet: "rich snippet", Rank: -1.0},
		{ID: "p1", Title: "Plain Only", Snippet: "plain snippet", Rank: -0.5},
	}
	keywords := []SearchResult{
		{ID: "shared", Encrypted: true, MatchedKeywords: []string{"keyword1", "keyword2"}, Rank: -0.8},
		{ID: "k1", Encrypted: true, MatchedKeywords: []string{"only_keyword"}, Rank: -0.3},
	}

	merged := mergeSearchResults(plaintext, keywords, 10)

	if len(merged) != 3 {
		t.Fatalf("expected 3 results (shared deduplicated), got %d", len(merged))
	}

	// First result should be the shared note with plaintext snippet preserved
	if merged[0].ID != "shared" {
		t.Errorf("expected first result to be shared, got %s", merged[0].ID)
	}
	if merged[0].Snippet != "rich snippet" {
		t.Errorf("expected plaintext snippet preserved, got %q", merged[0].Snippet)
	}
	if len(merged[0].MatchedKeywords) != 2 {
		t.Errorf("expected 2 matched keywords on deduplicated result, got %d", len(merged[0].MatchedKeywords))
	}
	// Encrypted flag should NOT be overwritten (plaintext result stays non-encrypted)
	if merged[0].Encrypted {
		t.Error("expected shared result to keep plaintext encrypted=false")
	}
}

func TestSearchKeywords(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create an encrypted note
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, content_encrypted, created_at, updated_at)
		VALUES ('enc1', '', '', '', '/', ?, 1, datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create encrypted note: %v", err)
	}

	// Insert keywords for the encrypted note
	if err := db.InsertNoteKeyword("enc1", "bitcoin"); err != nil {
		t.Fatalf("Failed to insert keyword: %v", err)
	}
	if err := db.InsertNoteKeyword("enc1", "investment"); err != nil {
		t.Fatalf("Failed to insert keyword: %v", err)
	}
	if err := db.InsertNoteKeyword("enc1", "strategy"); err != nil {
		t.Fatalf("Failed to insert keyword: %v", err)
	}

	ctx := context.Background()

	// Search for a keyword that exists
	ftsQ := buildFTSQuery([]string{"bitcoin"})
	results, err := db.searchKeywords(ctx, userID, ftsQ, 20)
	if err != nil {
		t.Fatalf("searchKeywords failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "enc1" {
		t.Errorf("expected note enc1, got %s", results[0].ID)
	}
	if !results[0].Encrypted {
		t.Error("expected result to be marked as encrypted")
	}
	if len(results[0].MatchedKeywords) == 0 {
		t.Error("expected matched keywords to be populated")
	}
}

func TestSearchKeywordsNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	ctx := context.Background()

	ftsQ := buildFTSQuery([]string{"nonexistent"})
	results, err := db.searchKeywords(ctx, userID, ftsQ, 20)
	if err != nil {
		t.Fatalf("searchKeywords failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestSearchIntegration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userID := 1
	createTestUser(t, db, userID)

	// Create a plaintext note (will be indexed in notes_fts)
	_, err := db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, created_at, updated_at)
		VALUES ('plain1', 'Bitcoin Guide', 'bitcoin guide', 'A guide about bitcoin trading and investment', '/', ?, datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create plaintext note: %v", err)
	}

	// Create an encrypted note with keywords
	_, err = db.Exec(`
		INSERT INTO notes (id, title, title_norm, content, folder_path, user_id, content_encrypted, created_at, updated_at)
		VALUES ('enc1', '', '', '', '/', ?, 1, datetime('now'), datetime('now'))
	`, userID)
	if err != nil {
		t.Fatalf("Failed to create encrypted note: %v", err)
	}
	if err := db.InsertNoteKeyword("enc1", "bitcoin"); err != nil {
		t.Fatalf("Failed to insert keyword: %v", err)
	}
	if err := db.InsertNoteKeyword("enc1", "crypto"); err != nil {
		t.Fatalf("Failed to insert keyword: %v", err)
	}

	ctx := context.Background()

	// Search should return both notes
	results, err := db.Search(ctx, userID, "bitcoin", 20)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (plaintext + keyword), got %d", len(results))
	}

	// First result should be the plaintext note (plaintext results come first)
	if results[0].ID != "plain1" {
		t.Errorf("expected first result to be plain1, got %s", results[0].ID)
	}
	if results[0].Encrypted {
		t.Error("expected first result to not be encrypted")
	}

	// Second result should be the encrypted note
	if results[1].ID != "enc1" {
		t.Errorf("expected second result to be enc1, got %s", results[1].ID)
	}
	if !results[1].Encrypted {
		t.Error("expected second result to be encrypted")
	}
	if len(results[1].MatchedKeywords) == 0 {
		t.Error("expected matched keywords on encrypted result")
	}
}
