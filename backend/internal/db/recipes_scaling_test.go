package db

import (
	"math"
	"testing"
)

// --- Scaling Tests ---

func TestScaleIngredients(t *testing.T) {
	amt := 500.0
	ingredients := []RecipeIngredient{
		{Name: "Mehl", Amount: &amt, Scalable: true},
		{Name: "Salz", Scalable: false},
		{Name: "Wasser", Amount: nil, Scalable: true}, // nil amount
	}

	result := ScaleIngredients(ingredients, 4, 8)
	if len(result) != 3 {
		t.Fatalf("Expected 3 scaled ingredients, got %d", len(result))
	}

	// Scalable with amount
	if result[0].ScaledAmount == nil || *result[0].ScaledAmount != 1000.0 {
		t.Errorf("Expected scaled amount 1000, got %v", result[0].ScaledAmount)
	}
	if result[0].DisplayAmount != "1000" {
		t.Errorf("Expected display '1000', got '%s'", result[0].DisplayAmount)
	}

	// Not scalable
	if result[1].ScaledAmount != nil {
		t.Errorf("Expected nil scaled amount for non-scalable, got %v", result[1].ScaledAmount)
	}

	// Nil amount (scalable but no amount)
	if result[2].ScaledAmount != nil {
		t.Errorf("Expected nil scaled amount for nil amount, got %v", result[2].ScaledAmount)
	}
	if result[2].DisplayAmount != "" {
		t.Errorf("Expected empty display for nil amount, got '%s'", result[2].DisplayAmount)
	}
}

func TestScaleIngredients_Rounding(t *testing.T) {
	amt := 100.0
	ingredients := []RecipeIngredient{
		{Name: "Test", Amount: &amt, Scalable: true},
	}

	// 100 * 3/4 = 75 (exact)
	result := ScaleIngredients(ingredients, 4, 3)
	if *result[0].ScaledAmount != 75.0 {
		t.Errorf("Expected 75, got %v", *result[0].ScaledAmount)
	}

	// 100 * 1/3 = 33.33 (rounded to 2 decimals)
	result = ScaleIngredients(ingredients, 3, 1)
	expected := math.Round(100.0/3.0*100) / 100
	if *result[0].ScaledAmount != expected {
		t.Errorf("Expected %v, got %v", expected, *result[0].ScaledAmount)
	}
}

func TestScaleIngredients_DisplayAmount(t *testing.T) {
	tests := []struct {
		value    float64
		expected string
	}{
		{500.0, "500"},
		{2.5, "2.5"},
		{1.33, "1.33"},
		{0.0, "0"},
		{10.0, "10"},
	}

	for _, tt := range tests {
		result := FormatDisplayAmount(tt.value)
		if result != tt.expected {
			t.Errorf("FormatDisplayAmount(%v) = '%s', expected '%s'", tt.value, result, tt.expected)
		}
	}
}

func TestScaleIngredients_AmountTextIgnored(t *testing.T) {
	amt := 100.0
	text := "ca. 100"
	ingredients := []RecipeIngredient{
		{Name: "Mehl", Amount: &amt, AmountText: &text, Scalable: true},
	}

	// When scaling, amount_text should be ignored
	result := ScaleIngredients(ingredients, 4, 8)
	if result[0].DisplayAmount != "200" {
		t.Errorf("Expected '200' (amount_text ignored during scaling), got '%s'", result[0].DisplayAmount)
	}
}

func TestScaleIngredients_AmountNilAmountTextSet(t *testing.T) {
	text := "etwas"
	ingredients := []RecipeIngredient{
		{Name: "Salz", Amount: nil, AmountText: &text, Scalable: true},
	}

	result := ScaleIngredients(ingredients, 4, 8)
	if result[0].DisplayAmount != "" {
		t.Errorf("Expected empty display for nil amount, got '%s'", result[0].DisplayAmount)
	}
}

func TestFormatAmount_OriginalServings(t *testing.T) {
	amt := 100.0
	text := "ca. 100"

	// When not scaling (original servings), amount_text is used
	result := FormatAmount(&amt, &text)
	if result != "ca. 100" {
		t.Errorf("Expected 'ca. 100', got '%s'", result)
	}

	// Without amount_text, numeric format
	result = FormatAmount(&amt, nil)
	if result != "100" {
		t.Errorf("Expected '100', got '%s'", result)
	}

	// Nil amount
	result = FormatAmount(nil, &text)
	if result != "" {
		t.Errorf("Expected empty string for nil amount, got '%s'", result)
	}
}
