package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestVector represents a test case from JSON files
type TestVector struct {
	Name     string         `json:"name"`
	Input    string         `json:"input"`
	Expected []ExpectedLink `json:"expected"`
}

type ExpectedLink struct {
	TargetTitle string `json:"target_title"`
	Alias       string `json:"alias,omitempty"`
	SpanStart   int    `json:"span_start"`
	SpanEnd     int    `json:"span_end"`
}

func TestParse_BasicLinks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []WikiLink
	}{
		{
			name:  "simple link",
			input: "[[Note]]",
			expected: []WikiLink{
				{TargetRaw: "Note", TargetTitle: "Note", Alias: "", SpanStart: 0, SpanEnd: 8},
			},
		},
		{
			name:  "link with alias",
			input: "[[Note|Display Text]]",
			expected: []WikiLink{
				{TargetRaw: "Note|Display Text", TargetTitle: "Note", Alias: "Display Text", SpanStart: 0, SpanEnd: 21},
			},
		},
		{
			name:  "link in text",
			input: "Hello [[World]] there",
			expected: []WikiLink{
				{TargetRaw: "World", TargetTitle: "World", Alias: "", SpanStart: 6, SpanEnd: 15},
			},
		},
		{
			name:  "multiple links",
			input: "See [[Link1]] and [[Link2]]",
			expected: []WikiLink{
				{TargetRaw: "Link1", TargetTitle: "Link1", Alias: "", SpanStart: 4, SpanEnd: 13},
				{TargetRaw: "Link2", TargetTitle: "Link2", Alias: "", SpanStart: 18, SpanEnd: 27},
			},
		},
		{
			name:  "link with spaces in title",
			input: "[[My Long Note Title]]",
			expected: []WikiLink{
				{TargetRaw: "My Long Note Title", TargetTitle: "My Long Note Title", Alias: "", SpanStart: 0, SpanEnd: 22},
			},
		},
		{
			name:  "link with whitespace trimmed",
			input: "[[  Trimmed  ]]",
			expected: []WikiLink{
				{TargetRaw: "  Trimmed  ", TargetTitle: "Trimmed", Alias: "", SpanStart: 0, SpanEnd: 15},
			},
		},
		{
			name:     "empty link",
			input:    "[[]]",
			expected: nil,
		},
		{
			name:     "whitespace only link",
			input:    "[[   ]]",
			expected: nil,
		},
		{
			name:  "alias with whitespace trimmed",
			input: "[[Note |  Alias  ]]",
			expected: []WikiLink{
				{TargetRaw: "Note |  Alias  ", TargetTitle: "Note", Alias: "Alias", SpanStart: 0, SpanEnd: 19},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.input)
			if len(result.Links) != len(tt.expected) {
				t.Errorf("expected %d links, got %d", len(tt.expected), len(result.Links))
				return
			}
			for i, link := range result.Links {
				exp := tt.expected[i]
				if link.TargetRaw != exp.TargetRaw {
					t.Errorf("link %d: expected TargetRaw %q, got %q", i, exp.TargetRaw, link.TargetRaw)
				}
				if link.TargetTitle != exp.TargetTitle {
					t.Errorf("link %d: expected TargetTitle %q, got %q", i, exp.TargetTitle, link.TargetTitle)
				}
				if link.Alias != exp.Alias {
					t.Errorf("link %d: expected Alias %q, got %q", i, exp.Alias, link.Alias)
				}
				if link.SpanStart != exp.SpanStart {
					t.Errorf("link %d: expected SpanStart %d, got %d", i, exp.SpanStart, link.SpanStart)
				}
				if link.SpanEnd != exp.SpanEnd {
					t.Errorf("link %d: expected SpanEnd %d, got %d", i, exp.SpanEnd, link.SpanEnd)
				}
			}
		})
	}
}

func TestParse_CodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []WikiLink
	}{
		{
			name:     "link in code fence",
			input:    "```\n[[Link]]\n```",
			expected: nil,
		},
		{
			name:     "link in code fence with language",
			input:    "```go\n[[Link]]\n```",
			expected: nil,
		},
		{
			name:  "link before code fence",
			input: "[[Before]]\n```\n[[Inside]]\n```",
			expected: []WikiLink{
				{TargetRaw: "Before", TargetTitle: "Before", Alias: "", SpanStart: 0, SpanEnd: 10},
			},
		},
		{
			name:  "link after code fence",
			input: "```\n[[Inside]]\n```\n[[After]]",
			expected: []WikiLink{
				{TargetRaw: "After", TargetTitle: "After", Alias: "", SpanStart: 19, SpanEnd: 28},
			},
		},
		{
			name:     "link in inline code",
			input:    "Use `[[Link]]` here",
			expected: nil,
		},
		{
			name:  "link outside inline code",
			input: "Use `code` then [[Link]]",
			expected: []WikiLink{
				{TargetRaw: "Link", TargetTitle: "Link", Alias: "", SpanStart: 16, SpanEnd: 24},
			},
		},
		{
			name:     "link in double backtick inline code",
			input:    "Use ``[[Link]]`` here",
			expected: nil,
		},
		{
			name:  "mixed code and links",
			input: "[[First]] `code` [[Second]] ```\n[[Hidden]]\n``` [[Third]]",
			expected: []WikiLink{
				{TargetRaw: "First", TargetTitle: "First", Alias: "", SpanStart: 0, SpanEnd: 9},
				{TargetRaw: "Second", TargetTitle: "Second", Alias: "", SpanStart: 17, SpanEnd: 27},
				{TargetRaw: "Third", TargetTitle: "Third", Alias: "", SpanStart: 47, SpanEnd: 56},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.input)
			if len(result.Links) != len(tt.expected) {
				t.Errorf("expected %d links, got %d", len(tt.expected), len(result.Links))
				for i, l := range result.Links {
					t.Logf("  link %d: %+v", i, l)
				}
				return
			}
			for i, link := range result.Links {
				exp := tt.expected[i]
				if link.TargetTitle != exp.TargetTitle {
					t.Errorf("link %d: expected TargetTitle %q, got %q", i, exp.TargetTitle, link.TargetTitle)
				}
				if link.SpanStart != exp.SpanStart {
					t.Errorf("link %d: expected SpanStart %d, got %d", i, exp.SpanStart, link.SpanStart)
				}
				if link.SpanEnd != exp.SpanEnd {
					t.Errorf("link %d: expected SpanEnd %d, got %d", i, exp.SpanEnd, link.SpanEnd)
				}
			}
		})
	}
}

func TestParse_Escaping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []WikiLink
	}{
		{
			name:     "escaped opening bracket",
			input:    `\[[Not a link]]`,
			expected: nil,
		},
		{
			name:  "normal link after escaped",
			input: `\[[Escaped]] [[Real]]`,
			expected: []WikiLink{
				{TargetRaw: "Real", TargetTitle: "Real", Alias: "", SpanStart: 13, SpanEnd: 21},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.input)
			if len(result.Links) != len(tt.expected) {
				t.Errorf("expected %d links, got %d", len(tt.expected), len(result.Links))
				return
			}
			for i, link := range result.Links {
				exp := tt.expected[i]
				if link.TargetTitle != exp.TargetTitle {
					t.Errorf("link %d: expected TargetTitle %q, got %q", i, exp.TargetTitle, link.TargetTitle)
				}
			}
		})
	}
}

func TestParse_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []WikiLink
	}{
		{
			name:     "unclosed link",
			input:    "[[Unclosed",
			expected: nil,
		},
		{
			name:     "unclosed link at end",
			input:    "Some text [[Unclosed",
			expected: nil,
		},
		{
			name:     "link with newline",
			input:    "[[Multi\nLine]]",
			expected: nil,
		},
		{
			name:     "single bracket",
			input:    "[Not a link]",
			expected: nil,
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name:  "consecutive links",
			input: "[[A]][[B]][[C]]",
			expected: []WikiLink{
				{TargetRaw: "A", TargetTitle: "A", Alias: "", SpanStart: 0, SpanEnd: 5},
				{TargetRaw: "B", TargetTitle: "B", Alias: "", SpanStart: 5, SpanEnd: 10},
				{TargetRaw: "C", TargetTitle: "C", Alias: "", SpanStart: 10, SpanEnd: 15},
			},
		},
		{
			name:  "link with pipe but no alias",
			input: "[[Note|]]",
			expected: []WikiLink{
				{TargetRaw: "Note|", TargetTitle: "Note", Alias: "", SpanStart: 0, SpanEnd: 9},
			},
		},
		{
			name:     "pipe only",
			input:    "[[|Alias]]",
			expected: nil, // Empty title should be skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.input)
			if len(result.Links) != len(tt.expected) {
				t.Errorf("expected %d links, got %d", len(tt.expected), len(result.Links))
				for i, l := range result.Links {
					t.Logf("  link %d: %+v", i, l)
				}
				return
			}
			for i, link := range result.Links {
				exp := tt.expected[i]
				if link.TargetTitle != exp.TargetTitle {
					t.Errorf("link %d: expected TargetTitle %q, got %q", i, exp.TargetTitle, link.TargetTitle)
				}
				if link.SpanStart != exp.SpanStart {
					t.Errorf("link %d: expected SpanStart %d, got %d", i, exp.SpanStart, link.SpanStart)
				}
				if link.SpanEnd != exp.SpanEnd {
					t.Errorf("link %d: expected SpanEnd %d, got %d", i, exp.SpanEnd, link.SpanEnd)
				}
			}
		})
	}
}

func TestParse_UTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []WikiLink
	}{
		{
			name:  "german umlauts",
			input: "[[Übersicht]]",
			expected: []WikiLink{
				{TargetRaw: "Übersicht", TargetTitle: "Übersicht", Alias: "", SpanStart: 0, SpanEnd: 14},
			},
		},
		{
			name:  "emoji in title",
			input: "[[📝 Notes]]",
			expected: []WikiLink{
				{TargetRaw: "📝 Notes", TargetTitle: "📝 Notes", Alias: "", SpanStart: 0, SpanEnd: 14},
			},
		},
		{
			name:  "chinese characters",
			input: "[[笔记]]",
			expected: []WikiLink{
				{TargetRaw: "笔记", TargetTitle: "笔记", Alias: "", SpanStart: 0, SpanEnd: 10},
			},
		},
		{
			name:  "mixed utf8 with alias",
			input: "[[日本語|Japanese]]",
			expected: []WikiLink{
				{TargetRaw: "日本語|Japanese", TargetTitle: "日本語", Alias: "Japanese", SpanStart: 0, SpanEnd: 22},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.input)
			if len(result.Links) != len(tt.expected) {
				t.Errorf("expected %d links, got %d", len(tt.expected), len(result.Links))
				return
			}
			for i, link := range result.Links {
				exp := tt.expected[i]
				if link.TargetTitle != exp.TargetTitle {
					t.Errorf("link %d: expected TargetTitle %q, got %q", i, exp.TargetTitle, link.TargetTitle)
				}
				if link.SpanStart != exp.SpanStart {
					t.Errorf("link %d: expected SpanStart %d, got %d", i, exp.SpanStart, link.SpanStart)
				}
				if link.SpanEnd != exp.SpanEnd {
					t.Errorf("link %d: expected SpanEnd %d, got %d", i, exp.SpanEnd, link.SpanEnd)
				}
				// Verify span extraction is correct
				extracted := tt.input[link.SpanStart:link.SpanEnd]
				expectedRaw := "[[" + link.TargetRaw + "]]"
				if extracted != expectedRaw {
					t.Errorf("link %d: span extraction mismatch: got %q, expected %q", i, extracted, expectedRaw)
				}
			}
		})
	}
}

// TestFromVectors runs tests from JSON vector files in testdata
func TestFromVectors(t *testing.T) {
	vectorDir := "../../../testdata/parser"
	files, err := filepath.Glob(filepath.Join(vectorDir, "*.json"))
	if err != nil {
		t.Fatalf("failed to glob test vectors: %v", err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Errorf("failed to read %s: %v", file, err)
			continue
		}

		var vectors []TestVector
		if err := json.Unmarshal(data, &vectors); err != nil {
			t.Errorf("failed to parse %s: %v", file, err)
			continue
		}

		for _, v := range vectors {
			t.Run(v.Name, func(t *testing.T) {
				result := Parse(v.Input)
				if len(result.Links) != len(v.Expected) {
					t.Errorf("expected %d links, got %d", len(v.Expected), len(result.Links))
					return
				}
				for i, link := range result.Links {
					exp := v.Expected[i]
					if link.TargetTitle != exp.TargetTitle {
						t.Errorf("link %d: expected title %q, got %q", i, exp.TargetTitle, link.TargetTitle)
					}
					if link.Alias != exp.Alias {
						t.Errorf("link %d: expected alias %q, got %q", i, exp.Alias, link.Alias)
					}
					if link.SpanStart != exp.SpanStart {
						t.Errorf("link %d: expected start %d, got %d", i, exp.SpanStart, link.SpanStart)
					}
					if link.SpanEnd != exp.SpanEnd {
						t.Errorf("link %d: expected end %d, got %d", i, exp.SpanEnd, link.SpanEnd)
					}
				}
			})
		}
	}
}

func BenchmarkParse(b *testing.B) {
	// Simulate a typical note with multiple links
	content := `# Test Note

This is a [[simple link]] and here's [[another one|with alias]].

` + "```" + `go
// This [[should be ignored]]
func main() {}
` + "```" + `

More text with [[Link1]], [[Link2]], and [[Link3]].

Some inline code: ` + "`[[ignored]]`" + ` but [[this works]].
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Parse(content)
	}
}

func BenchmarkParseLargeNote(b *testing.B) {
	// Build a large note with many links
	var builder string
	for i := 0; i < 100; i++ {
		builder += "Some text [[Link" + string(rune('A'+i%26)) + "]] more text.\n"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Parse(builder)
	}
}
