package db

import (
	"fmt"
	"math"
	"strconv"
)

// ScaleIngredients scales ingredients based on target servings.
// This is the canonical server-side implementation (I4).
func ScaleIngredients(ingredients []RecipeIngredient, baseServings, targetServings int) []ScaledIngredient {
	factor := float64(targetServings) / float64(baseServings)
	result := make([]ScaledIngredient, len(ingredients))

	for i, ing := range ingredients {
		result[i].RecipeIngredient = ing

		if !ing.Scalable || ing.Amount == nil {
			result[i].ScaledAmount = ing.Amount
			result[i].DisplayAmount = FormatAmount(ing.Amount, ing.AmountText)
			continue
		}

		scaled := math.Round(*ing.Amount*factor*100) / 100
		result[i].ScaledAmount = &scaled
		result[i].DisplayAmount = FormatDisplayAmount(scaled)
	}
	return result
}

// FormatDisplayAmount formats a float for display.
func FormatDisplayAmount(v float64) string {
	if v == math.Trunc(v) {
		return strconv.Itoa(int(v))
	}
	if v*10 == math.Trunc(v*10) {
		return fmt.Sprintf("%.1f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

// FormatAmount formats the original amount with optional text hint.
func FormatAmount(amount *float64, amountText *string) string {
	if amount == nil {
		return ""
	}
	if amountText != nil {
		return *amountText
	}
	return FormatDisplayAmount(*amount)
}
