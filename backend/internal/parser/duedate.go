package parser

import (
	"regexp"
	"strings"
	"time"
)

// DueDate represents an extracted @due() date from note content.
type DueDate struct {
	Date        string // "YYYY-MM-DD"
	LineText    string // Cleaned line text (without @due() and checkbox prefix)
	LineIndex   int    // 0-based line index
	IsTaskItem  bool   // Line has a checkbox
	IsCompleted bool   // [x] or [X]
}

var (
	dueDateRegex    = regexp.MustCompile(`@due\((\d{4}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01]))\)`)
	dueDateCleanup  = regexp.MustCompile(`@due\([^)]*\)`)
	checkboxRegex   = regexp.MustCompile(`^\s*[-*+]\s*\[([xX ])\]\s*`)
	listPrefixRegex = regexp.MustCompile(`^\s*[-*+]\s*(?:\[[xX ]\]\s*)?`)
)

// ParseDueDates extracts all @due(YYYY-MM-DD) dates from content.
// It handles code blocks (```) and validates dates via time.Parse.
func ParseDueDates(content string) []DueDate {
	var results []DueDate
	lines := strings.Split(content, "\n")
	inCodeBlock := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		matches := dueDateRegex.FindAllStringSubmatch(line, -1)
		if len(matches) == 0 {
			continue
		}

		// Check for checkbox
		isTask := false
		isCompleted := false
		cbMatch := checkboxRegex.FindStringSubmatch(line)
		if cbMatch != nil {
			isTask = true
			isCompleted = cbMatch[1] == "x" || cbMatch[1] == "X"
		}

		for _, match := range matches {
			dateStr := match[1]

			// Validate date overflow via time.Parse
			_, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				continue
			}

			// Clean line text: remove all @due() matches
			cleanText := dueDateCleanup.ReplaceAllString(line, "")
			// For task items, also remove the checkbox prefix
			if isTask {
				cleanText = listPrefixRegex.ReplaceAllString(cleanText, "")
			}
			cleanText = strings.TrimSpace(cleanText)

			results = append(results, DueDate{
				Date:        dateStr,
				LineText:    cleanText,
				LineIndex:   i,
				IsTaskItem:  isTask,
				IsCompleted: isCompleted,
			})
		}
	}

	return results
}
