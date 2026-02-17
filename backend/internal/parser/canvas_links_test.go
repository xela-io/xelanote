package parser

import "testing"

func TestExtractCanvasFileRefs_Empty(t *testing.T) {
	refs, err := ExtractCanvasFileRefs(`{"nodes":[],"edges":[]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("expected 0 refs, got %d", len(refs))
	}
}

func TestExtractCanvasFileRefs_FileNodes(t *testing.T) {
	canvas := `{
		"nodes": [
			{"id": "1", "type": "text", "x": 0, "y": 0, "width": 100, "height": 100, "text": "hello"},
			{"id": "2", "type": "file", "x": 200, "y": 0, "width": 100, "height": 100, "file": "My Note"},
			{"id": "3", "type": "file", "x": 400, "y": 0, "width": 100, "height": 100, "file": "Another Note"},
			{"id": "4", "type": "link", "x": 600, "y": 0, "width": 100, "height": 100, "url": "https://example.com"},
			{"id": "5", "type": "file", "x": 800, "y": 0, "width": 100, "height": 100, "file": "uploads/image.png"}
		],
		"edges": []
	}`

	refs, err := ExtractCanvasFileRefs(canvas)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include "My Note" and "Another Note" but NOT "uploads/image.png" (file path)
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d: %v", len(refs), refs)
	}
	if refs[0] != "My Note" {
		t.Errorf("expected refs[0]='My Note', got %q", refs[0])
	}
	if refs[1] != "Another Note" {
		t.Errorf("expected refs[1]='Another Note', got %q", refs[1])
	}
}

func TestExtractCanvasFileRefs_InvalidJSON(t *testing.T) {
	_, err := ExtractCanvasFileRefs(`not json`)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestExtractCanvasFileRefs_NoNodes(t *testing.T) {
	refs, err := ExtractCanvasFileRefs(`{}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("expected 0 refs, got %d", len(refs))
	}
}

func TestExtractCanvasFileRefs_DuplicateFilenames(t *testing.T) {
	canvas := `{
		"nodes": [
			{"id": "1", "type": "file", "x": 0, "y": 0, "width": 100, "height": 100, "file": "Same Note"},
			{"id": "2", "type": "file", "x": 200, "y": 0, "width": 100, "height": 100, "file": "Same Note"}
		],
		"edges": []
	}`

	refs, err := ExtractCanvasFileRefs(canvas)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parser deduplicates, so only 1 unique ref should be returned
	if len(refs) != 1 {
		t.Fatalf("expected 1 deduplicated ref, got %d", len(refs))
	}
	if refs[0] != "Same Note" {
		t.Errorf("expected 'Same Note', got %q", refs[0])
	}
}
