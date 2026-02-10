package utils

import "testing"

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"trim-and-lower", "  Hello World  ", "hello world"},
		{"already-normalized", "foo", "foo"},
		{"mixed-case", "MiXeD", "mixed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeTitle(tt.input)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
