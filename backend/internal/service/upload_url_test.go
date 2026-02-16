package service

import "testing"

func TestParseUploadURL(t *testing.T) {
	tests := []struct {
		name       string
		rawURL     string
		wantUserID int
		wantFile   string
		wantErr    bool
	}{
		{
			name:       "valid URL",
			rawURL:     "/api/uploads/42/photo.jpg",
			wantUserID: 42,
			wantFile:   "photo.jpg",
		},
		{
			name:       "valid URL with nested path",
			rawURL:     "/api/uploads/1/subdir/image.png",
			wantUserID: 1,
			wantFile:   "image.png",
		},
		{
			name:    "missing prefix",
			rawURL:  "/uploads/42/photo.jpg",
			wantErr: true,
		},
		{
			name:    "no user ID",
			rawURL:  "/api/uploads//photo.jpg",
			wantErr: true,
		},
		{
			name:    "no filename",
			rawURL:  "/api/uploads/42/",
			wantErr: true,
		},
		{
			name:    "non-numeric user ID",
			rawURL:  "/api/uploads/abc/photo.jpg",
			wantErr: true,
		},
		{
			name:    "empty string",
			rawURL:  "",
			wantErr: true,
		},
		{
			name:    "only prefix",
			rawURL:  "/api/uploads/",
			wantErr: true,
		},
		{
			name:    "external URL",
			rawURL:  "https://evil.com/api/uploads/42/photo.jpg",
			wantErr: true,
		},
		{
			name:    "dot filename",
			rawURL:  "/api/uploads/42/..",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, filename, err := ParseUploadURL(tt.rawURL)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseUploadURL(%q) expected error, got userID=%d filename=%q", tt.rawURL, userID, filename)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseUploadURL(%q) unexpected error: %v", tt.rawURL, err)
			}
			if userID != tt.wantUserID {
				t.Errorf("userID = %d, want %d", userID, tt.wantUserID)
			}
			if filename != tt.wantFile {
				t.Errorf("filename = %q, want %q", filename, tt.wantFile)
			}
		})
	}
}
