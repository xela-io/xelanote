package parser

import (
	"testing"
)

func TestParseDueDates_Basic(t *testing.T) {
	content := "Task @due(2026-02-10)"
	results := ParseDueDates(content)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Date != "2026-02-10" {
		t.Errorf("expected date 2026-02-10, got %s", results[0].Date)
	}
	if results[0].LineText != "Task" {
		t.Errorf("expected line text 'Task', got '%s'", results[0].LineText)
	}
	if results[0].IsTaskItem {
		t.Error("expected IsTaskItem=false")
	}
}

func TestParseDueDates_TaskItem(t *testing.T) {
	content := "- [ ] Einkaufen @due(2026-02-10)"
	results := ParseDueDates(content)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].IsTaskItem {
		t.Error("expected IsTaskItem=true")
	}
	if results[0].IsCompleted {
		t.Error("expected IsCompleted=false")
	}
	if results[0].LineText != "Einkaufen" {
		t.Errorf("expected 'Einkaufen', got '%s'", results[0].LineText)
	}
}

func TestParseDueDates_CompletedTask(t *testing.T) {
	content := "- [x] Done task @due(2026-02-10)"
	results := ParseDueDates(content)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].IsCompleted {
		t.Error("expected IsCompleted=true")
	}
}

func TestParseDueDates_CompletedTaskUpperX(t *testing.T) {
	content := "- [X] Done task @due(2026-02-10)"
	results := ParseDueDates(content)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].IsCompleted {
		t.Error("expected IsCompleted=true for [X]")
	}
}

func TestParseDueDates_MultipleDates(t *testing.T) {
	content := "First @due(2026-02-10)\nSecond @due(2026-03-15)"
	results := ParseDueDates(content)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Date != "2026-02-10" {
		t.Errorf("expected 2026-02-10, got %s", results[0].Date)
	}
	if results[1].Date != "2026-03-15" {
		t.Errorf("expected 2026-03-15, got %s", results[1].Date)
	}
	if results[0].LineIndex != 0 {
		t.Errorf("expected LineIndex 0, got %d", results[0].LineIndex)
	}
	if results[1].LineIndex != 1 {
		t.Errorf("expected LineIndex 1, got %d", results[1].LineIndex)
	}
}

func TestParseDueDates_CodeBlock(t *testing.T) {
	content := "```\n@due(2026-02-10)\n```"
	results := ParseDueDates(content)
	if len(results) != 0 {
		t.Fatalf("expected 0 results in code block, got %d", len(results))
	}
}

func TestParseDueDates_InvalidDate(t *testing.T) {
	tests := []string{
		"@due(2026-02-30)", // Feb 30
		"@due(2026-13-01)", // Month 13
		"@due(2026-04-31)", // Apr 31
		"@due(tomorrow)",   // Not a date
		"@due(2026-02-29)", // 2026 not leap year
	}
	for _, content := range tests {
		results := ParseDueDates(content)
		if len(results) != 0 {
			t.Errorf("expected 0 results for %q, got %d", content, len(results))
		}
	}
}

func TestParseDueDates_EmptyContent(t *testing.T) {
	results := ParseDueDates("")
	if len(results) != 0 {
		t.Fatalf("expected 0 results for empty content, got %d", len(results))
	}
}

func TestParseDueDates_NoDueDates(t *testing.T) {
	results := ParseDueDates("Just a regular note without due dates.")
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestParseDueDates_MultipleDatesOnSameLine(t *testing.T) {
	content := "Start @due(2026-02-10) End @due(2026-03-15)"
	results := ParseDueDates(content)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Date != "2026-02-10" {
		t.Errorf("expected 2026-02-10, got %s", results[0].Date)
	}
	if results[1].Date != "2026-03-15" {
		t.Errorf("expected 2026-03-15, got %s", results[1].Date)
	}
	// Both should have same line text (cleaned from @due())
	if results[0].LineText != "Start  End" {
		t.Errorf("expected 'Start  End', got '%s'", results[0].LineText)
	}
}

func TestParseDueDates_TaskWithStarMarker(t *testing.T) {
	content := "* [ ] Task @due(2026-02-10)"
	results := ParseDueDates(content)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].IsTaskItem {
		t.Error("expected IsTaskItem=true for * marker")
	}
}

func TestParseDueDates_LeapYear(t *testing.T) {
	content := "@due(2024-02-29)"
	results := ParseDueDates(content)
	if len(results) != 1 {
		t.Fatalf("expected 1 result for leap year date, got %d", len(results))
	}
}
