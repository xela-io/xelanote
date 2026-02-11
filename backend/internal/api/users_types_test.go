package api

import (
	"testing"

	"github.com/xela-io/xelanote/internal/service"
)

func TestMapClaudeAPIKeyStatus_shouldReturnDefault_whenInputIsNil(t *testing.T) {
	got := mapClaudeAPIKeyStatus(nil)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.HasKey {
		t.Fatal("expected HasKey=false for nil input")
	}
	if got.UpdatedAt != nil {
		t.Fatal("expected UpdatedAt=nil for nil input")
	}
	if got.MaskedKey != nil {
		t.Fatal("expected MaskedKey=nil for nil input")
	}
}

func TestMapClaudeAPIKeyStatus_shouldMapAllFields_whenInputHasValues(t *testing.T) {
	updatedAt := "2026-02-11T10:11:12Z"
	masked := "sk-ant-api0...1234"
	in := &service.ClaudeAPIKeyStatus{
		HasKey:    true,
		UpdatedAt: &updatedAt,
		MaskedKey: &masked,
	}

	got := mapClaudeAPIKeyStatus(in)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if !got.HasKey {
		t.Fatal("expected HasKey=true")
	}
	if got.UpdatedAt == nil || *got.UpdatedAt != updatedAt {
		t.Fatalf("expected UpdatedAt=%q, got %v", updatedAt, got.UpdatedAt)
	}
	if got.MaskedKey == nil || *got.MaskedKey != masked {
		t.Fatalf("expected MaskedKey=%q, got %v", masked, got.MaskedKey)
	}
}

func TestMapGeminiAPIKeyStatus_shouldReturnDefault_whenInputIsNil(t *testing.T) {
	got := mapGeminiAPIKeyStatus(nil)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.HasKey {
		t.Fatal("expected HasKey=false for nil input")
	}
	if got.UpdatedAt != nil {
		t.Fatal("expected UpdatedAt=nil for nil input")
	}
	if got.MaskedKey != nil {
		t.Fatal("expected MaskedKey=nil for nil input")
	}
}

func TestMapGeminiAPIKeyStatus_shouldMapAllFields_whenInputHasValues(t *testing.T) {
	updatedAt := "2026-02-11T10:11:12Z"
	masked := "AIzaSy...abcd"
	in := &service.GeminiAPIKeyStatus{
		HasKey:    true,
		UpdatedAt: &updatedAt,
		MaskedKey: &masked,
	}

	got := mapGeminiAPIKeyStatus(in)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if !got.HasKey {
		t.Fatal("expected HasKey=true")
	}
	if got.UpdatedAt == nil || *got.UpdatedAt != updatedAt {
		t.Fatalf("expected UpdatedAt=%q, got %v", updatedAt, got.UpdatedAt)
	}
	if got.MaskedKey == nil || *got.MaskedKey != masked {
		t.Fatalf("expected MaskedKey=%q, got %v", masked, got.MaskedKey)
	}
}
