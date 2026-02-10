package parser

import (
	"testing"
)

func TestStripColorTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic named color",
			input:    "{color:primary}hello{/color}",
			expected: "hello",
		},
		{
			name:     "hex color",
			input:    "{color:#ff0000}red text{/color}",
			expected: "red text",
		},
		{
			name:     "rgb color",
			input:    "{color:rgb(255, 0, 0)}red text{/color}",
			expected: "red text",
		},
		{
			name:     "rgba color",
			input:    "{color:rgba(255, 0, 0, 0.5)}semi-red{/color}",
			expected: "semi-red",
		},
		{
			name:     "no color tags",
			input:    "normal text without colors",
			expected: "normal text without colors",
		},
		{
			name:     "multiple color blocks",
			input:    "{color:primary}one{/color} and {color:destructive}two{/color}",
			expected: "one and two",
		},
		{
			name:     "nested markdown inside color",
			input:    "{color:accent}**bold** and *italic*{/color}",
			expected: "**bold** and *italic*",
		},
		{
			name:     "color in sentence",
			input:    "This is {color:muted}muted text{/color} in a sentence.",
			expected: "This is muted text in a sentence.",
		},
		{
			name:     "empty color content",
			input:    "{color:primary}{/color}",
			expected: "",
		},
		{
			name:     "unclosed color tag preserved",
			input:    "{color:primary}unclosed",
			expected: "unclosed",
		},
		{
			name:     "only closing tag",
			input:    "text{/color}",
			expected: "text",
		},
		{
			name:     "complex mixed content",
			input:    "# Heading\n\n{color:primary}Colored paragraph{/color}\n\n- {color:destructive}Red item{/color}\n- Normal item",
			expected: "# Heading\n\nColored paragraph\n\n- Red item\n- Normal item",
		},
		{
			name:     "short hex",
			input:    "{color:#fff}white{/color}",
			expected: "white",
		},
		{
			name:     "all named colors",
			input:    "{color:primary}a{/color}{color:destructive}b{/color}{color:accent}c{/color}{color:muted}d{/color}{color:secondary}e{/color}",
			expected: "abcde",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripColorTags(tt.input)
			if result != tt.expected {
				t.Errorf("StripColorTags(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func BenchmarkStripColorTags(b *testing.B) {
	input := "{color:primary}This is some colored text{/color} mixed with {color:#ff0000}hex colors{/color} and {color:rgb(0, 255, 0)}rgb colors{/color}."

	for i := 0; i < b.N; i++ {
		StripColorTags(input)
	}
}
