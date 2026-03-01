package htmlutil

import (
	"math"
	"testing"
)

func TestParseIngredientString(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
		wantAmt  *float64
		wantUnit *string
	}{
		// Metric
		{"500g Hackfleisch", "Hackfleisch", floatPtr(500), strPtr("g")},
		{"200 ml Milch", "Milch", floatPtr(200), strPtr("ml")},
		{"1.5 kg Mehl", "Mehl", floatPtr(1.5), strPtr("kg")},
		{"2 l Wasser", "Wasser", floatPtr(2), strPtr("l")},

		// Imperial
		{"2 cups flour", "flour", floatPtr(2), strPtr("cup")},
		{"1 tablespoon olive oil", "olive oil", floatPtr(1), strPtr("tbsp")},
		{"3 oz butter", "butter", floatPtr(3), strPtr("oz")},
		{"2 lbs ground beef", "ground beef", floatPtr(2), strPtr("lb")},

		// Fractions
		{"1/2 cup sugar", "sugar", floatPtr(0.5), strPtr("cup")},
		{"1 1/2 cups flour", "flour", floatPtr(1.5), strPtr("cup")},
		{"3/4 tsp salt", "salt", floatPtr(0.75), strPtr("tsp")},

		// Unicode fractions
		{"½ cup butter", "butter", floatPtr(0.5), strPtr("cup")},
		{"1½ cups milk", "milk", floatPtr(1.5), strPtr("cup")},
		{"¼ tsp pepper", "pepper", floatPtr(0.25), strPtr("tsp")},

		// German units
		{"2 EL Olivenöl", "Olivenöl", floatPtr(2), strPtr("EL")},
		{"1 TL Salz", "Salz", floatPtr(1), strPtr("TL")},
		{"1 Prise Pfeffer", "Pfeffer", floatPtr(1), strPtr("Prise")},
		{"1 Bund Petersilie", "Petersilie", floatPtr(1), strPtr("Bund")},

		// No amount
		{"Salz und Pfeffer", "Salz und Pfeffer", nil, nil},
		{"salt to taste", "salt to taste", nil, nil},

		// Amount but no known unit
		{"3 Eier", "Eier", floatPtr(3), nil},
		{"2 large onions", "large onions", floatPtr(2), nil},

		// Edge cases
		{"", "", nil, nil},
		{"  flour  ", "flour", nil, nil},

		// Decimal with comma (German)
		{"1,5 kg Kartoffeln", "Kartoffeln", floatPtr(1.5), strPtr("kg")},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			name, amt, unit := ParseIngredientString(tc.input)

			if name != tc.wantName {
				t.Errorf("name = %q, want %q", name, tc.wantName)
			}

			if tc.wantAmt == nil && amt != nil {
				t.Errorf("amount = %v, want nil", *amt)
			} else if tc.wantAmt != nil && amt == nil {
				t.Errorf("amount = nil, want %v", *tc.wantAmt)
			} else if tc.wantAmt != nil && amt != nil && math.Abs(*amt-*tc.wantAmt) > 0.01 {
				t.Errorf("amount = %v, want %v", *amt, *tc.wantAmt)
			}

			if tc.wantUnit == nil && unit != nil {
				t.Errorf("unit = %q, want nil", *unit)
			} else if tc.wantUnit != nil && unit == nil {
				t.Errorf("unit = nil, want %q", *tc.wantUnit)
			} else if tc.wantUnit != nil && unit != nil && *unit != *tc.wantUnit {
				t.Errorf("unit = %q, want %q", *unit, *tc.wantUnit)
			}
		})
	}
}

func TestParseFraction(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"1/2", 0.5},
		{"3/4", 0.75},
		{"1/3", 1.0 / 3.0},
		{"2/3", 2.0 / 3.0},
		{"1/4", 0.25},
		{"invalid", 0},
		{"1/0", 0},
		{"", 0},
	}

	for _, tc := range tests {
		got := parseFraction(tc.input)
		if math.Abs(got-tc.want) > 0.001 {
			t.Errorf("parseFraction(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func floatPtr(v float64) *float64 {
	return &v
}

func strPtr(v string) *string {
	return &v
}
