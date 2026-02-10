// Package parser provides a wikilink scanner for extracting [[Title]] and [[Title|Alias]] links
// from markdown content while properly handling code blocks and escape sequences.
package parser

import (
	"strings"

	"github.com/xela-io/xelanote/internal/utils"
)

// WikiLink represents a parsed wikilink with position information.
type WikiLink struct {
	TargetRaw   string // Original content: "Title|Alias" or "Title"
	TargetTitle string // Extracted title (normalized, trimmed)
	Alias       string // Alias if present, otherwise empty
	SpanStart   int    // Byte offset of opening [[
	SpanEnd     int    // Byte offset after closing ]]
}

// ParseResult contains all extracted wikilinks and parsing metadata.
type ParseResult struct {
	Links []WikiLink
}

// Parse extracts all wikilinks from the given content.
// It properly handles:
// - [[Title]] basic links
// - [[Title|Alias]] aliased links
// - Code fences (```) - links inside are ignored
// - Inline code (`) - links inside are ignored
// - Escape sequences (\[\[) - not treated as links
func Parse(content string) ParseResult {
	p := &parser{
		content: content,
		pos:     0,
		result:  ParseResult{},
	}
	p.parse()
	return p.result
}

type parser struct {
	content string
	pos     int
	result  ParseResult
}

func (p *parser) parse() {
	for p.pos < len(p.content) {
		switch {
		case p.matchCodeFence():
			p.skipCodeFence()
		case p.matchInlineCode():
			p.skipInlineCode()
		case p.matchEscapedBracket():
			p.pos += 2 // Skip \[
		case p.matchWikiLink():
			p.parseWikiLink()
		default:
			p.pos++
		}
	}
}

// matchCodeFence checks for ``` at current position
func (p *parser) matchCodeFence() bool {
	return p.pos+2 < len(p.content) &&
		p.content[p.pos] == '`' &&
		p.content[p.pos+1] == '`' &&
		p.content[p.pos+2] == '`'
}

// skipCodeFence skips over a code fence block
func (p *parser) skipCodeFence() {
	p.pos += 3 // Skip opening ```

	// Skip to end of opening line (may have language identifier)
	for p.pos < len(p.content) && p.content[p.pos] != '\n' {
		p.pos++
	}
	if p.pos < len(p.content) {
		p.pos++ // Skip newline
	}

	// Find closing ```
	for p.pos < len(p.content) {
		if p.matchCodeFence() {
			p.pos += 3 // Skip closing ```
			return
		}
		p.pos++
	}
}

// matchInlineCode checks for ` at current position (not part of ```)
func (p *parser) matchInlineCode() bool {
	if p.content[p.pos] != '`' {
		return false
	}
	// Make sure it's not a code fence
	if p.pos+2 < len(p.content) &&
		p.content[p.pos+1] == '`' &&
		p.content[p.pos+2] == '`' {
		return false
	}
	return true
}

// skipInlineCode skips over inline code
func (p *parser) skipInlineCode() {
	// Count opening backticks
	numBackticks := 0
	start := p.pos
	for p.pos < len(p.content) && p.content[p.pos] == '`' {
		numBackticks++
		p.pos++
	}

	// Find matching closing backticks
	for p.pos < len(p.content) {
		if p.content[p.pos] == '`' {
			// Count consecutive backticks
			count := 0
			checkPos := p.pos
			for checkPos < len(p.content) && p.content[checkPos] == '`' {
				count++
				checkPos++
			}
			if count == numBackticks {
				p.pos = checkPos
				return
			}
		}
		p.pos++
	}

	// No closing found, reset to just after opening
	p.pos = start + numBackticks
}

// matchEscapedBracket checks for \[ at current position
func (p *parser) matchEscapedBracket() bool {
	return p.pos+1 < len(p.content) &&
		p.content[p.pos] == '\\' &&
		p.content[p.pos+1] == '['
}

// matchWikiLink checks for [[ at current position
func (p *parser) matchWikiLink() bool {
	return p.pos+1 < len(p.content) &&
		p.content[p.pos] == '[' &&
		p.content[p.pos+1] == '['
}

// parseWikiLink extracts a wikilink starting at current position
func (p *parser) parseWikiLink() {
	start := p.pos
	p.pos += 2 // Skip [[

	// Find closing ]]
	contentStart := p.pos
	depth := 1
	for p.pos < len(p.content) {
		// Check for nested [[ (invalid but handle gracefully)
		if p.pos+1 < len(p.content) &&
			p.content[p.pos] == '[' &&
			p.content[p.pos+1] == '[' {
			depth++
			p.pos += 2
			continue
		}

		// Check for closing ]]
		if p.pos+1 < len(p.content) &&
			p.content[p.pos] == ']' &&
			p.content[p.pos+1] == ']' {
			depth--
			if depth == 0 {
				break
			}
			p.pos += 2
			continue
		}

		// Don't allow newlines in wikilinks
		if p.content[p.pos] == '\n' {
			// Invalid wikilink, skip the opening [[
			p.pos = start + 2
			return
		}

		p.pos++
	}

	// Check if we found valid closing
	if p.pos+1 >= len(p.content) ||
		p.content[p.pos] != ']' ||
		p.content[p.pos+1] != ']' {
		// Invalid wikilink, skip opening [[
		p.pos = start + 2
		return
	}

	// Extract content between [[ and ]]
	rawContent := p.content[contentStart:p.pos]

	// Skip closing ]]
	p.pos += 2

	// Skip empty links
	if strings.TrimSpace(rawContent) == "" {
		return
	}

	// Parse title and alias
	link := WikiLink{
		TargetRaw: rawContent,
		SpanStart: start,
		SpanEnd:   p.pos,
	}

	// Check for alias separator |
	if idx := strings.Index(rawContent, "|"); idx != -1 {
		link.TargetTitle = strings.TrimSpace(rawContent[:idx])
		link.Alias = strings.TrimSpace(rawContent[idx+1:])
	} else {
		link.TargetTitle = strings.TrimSpace(rawContent)
	}

	// Skip links with empty title
	if link.TargetTitle == "" {
		return
	}

	p.result.Links = append(p.result.Links, link)
}

// NormalizeTitle returns a normalized version of a title for matching.
// Deprecated: Use utils.NormalizeTitle instead. This is kept for backwards compatibility.
func NormalizeTitle(title string) string {
	return utils.NormalizeTitle(title)
}
