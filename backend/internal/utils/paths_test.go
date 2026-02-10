package utils

import "testing"

func TestValidateFolderPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"no-leading-slash", "foo/bar", true},
		{"root", "/", true},
		{"dotdot", "/foo/../bar", true},
		{"double-slash", "/foo//bar", true},
		{"trailing-slash", "/foo/", true},
		{"valid", "/foo/bar", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFolderPath(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for %q", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
		})
	}
}

func TestNormalizeFolderPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"root", "/", "/"},
		{"dot", ".", "/"},
		{"no-leading-slash", "foo/bar", "/foo/bar"},
		{"clean", "/foo/bar/..", "/foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeFolderPath(tt.input)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestGetParentPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"root", "/", ""},
		{"one-level", "/foo", "/"},
		{"two-level", "/foo/bar", "/foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetParentPath(tt.input)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestGetFolderName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"root", "/", "Root"},
		{"folder", "/foo", "foo"},
		{"nested", "/foo/bar", "bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFolderName(tt.input)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
