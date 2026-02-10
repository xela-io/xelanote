package api

import (
	"testing"
)

func TestValidateNoteFields(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		content    string
		folderPath string
		wantErr    string
	}{
		{"valid", "My Note", "content", "/folder", ""},
		{"empty title", "", "content", "/", "title is required"},
		{"title too long", string(make([]byte, MaxNoteTitleLength+1)), "", "/", "title too long"},
		{"content too long", "title", string(make([]byte, MaxNoteContentLength+1)), "/", "content too long"},
		{"folder too long", "title", "", string(make([]byte, MaxFolderPathLength+1)), "folder path too long"},
		{"empty folder allowed", "title", "content", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNoteFields(tt.title, tt.content, tt.folderPath)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("expected error %q, got %q", tt.wantErr, err.Error())
				}
			}
		})
	}
}

func TestParseETag(t *testing.T) {
	tests := []struct {
		name           string
		etag           string
		noteID         string
		currentVersion int
		wantVersion    int
		wantErr        bool
	}{
		{
			name:        "old-style integer ETag",
			etag:        "5",
			noteID:      "note-1",
			wantVersion: 5,
		},
		{
			name:        "quoted integer ETag",
			etag:        `"5"`,
			noteID:      "note-1",
			wantVersion: 5,
		},
		{
			name:           "hashed ETag matches current version",
			etag:           generateETag("note-1", 3),
			noteID:         "note-1",
			currentVersion: 3,
			wantVersion:    3,
		},
		{
			name:           "hashed ETag mismatch",
			etag:           generateETag("note-1", 2),
			noteID:         "note-1",
			currentVersion: 3,
			wantErr:        true,
		},
		{
			name:           "hashed ETag wrong note ID",
			etag:           generateETag("note-2", 3),
			noteID:         "note-1",
			currentVersion: 3,
			wantErr:        true,
		},
		{
			name:    "garbage ETag",
			etag:    "not-a-valid-etag",
			noteID:  "note-1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := parseETag(tt.etag, tt.noteID, tt.currentVersion)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got version %d", version)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if version != tt.wantVersion {
					t.Errorf("expected version %d, got %d", tt.wantVersion, version)
				}
			}
		})
	}
}

func TestGenerateETag(t *testing.T) {
	// ETag should be deterministic
	etag1 := generateETag("note-1", 1)
	etag2 := generateETag("note-1", 1)
	if etag1 != etag2 {
		t.Errorf("ETags should be deterministic: %q != %q", etag1, etag2)
	}

	// Different versions should produce different ETags
	etag3 := generateETag("note-1", 2)
	if etag1 == etag3 {
		t.Errorf("different versions should produce different ETags")
	}

	// Different note IDs should produce different ETags
	etag4 := generateETag("note-2", 1)
	if etag1 == etag4 {
		t.Errorf("different note IDs should produce different ETags")
	}

	// Should be quoted
	if etag1[0] != '"' || etag1[len(etag1)-1] != '"' {
		t.Errorf("ETag should be quoted, got %q", etag1)
	}
}
