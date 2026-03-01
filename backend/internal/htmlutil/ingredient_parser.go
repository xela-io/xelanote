package htmlutil

import (
	"regexp"
	"strconv"
	"strings"
)

// unicodeFractions maps Unicode fraction characters to their float values.
var unicodeFractions = map[rune]float64{
	'½': 0.5,
	'⅓': 1.0 / 3.0,
	'⅔': 2.0 / 3.0,
	'¼': 0.25,
	'¾': 0.75,
	'⅕': 0.2,
	'⅖': 0.4,
	'⅗': 0.6,
	'⅘': 0.8,
	'⅙': 1.0 / 6.0,
	'⅚': 5.0 / 6.0,
	'⅛': 0.125,
	'⅜': 0.375,
	'⅝': 0.625,
	'⅞': 0.875,
}

// knownUnits maps lowercase unit strings (including German) to canonical forms.
var knownUnits = map[string]string{
	// Metric
	"g":     "g",
	"gr":    "g",
	"gram":  "g",
	"grams": "g",
	"gramm": "g",
	"kg":    "kg",
	"ml":    "ml",
	"l":     "l",
	"liter": "l",
	"litre": "l",
	// Imperial volume
	"cup":          "cup",
	"cups":         "cup",
	"tbsp":         "tbsp",
	"tablespoon":   "tbsp",
	"tablespoons":  "tbsp",
	"tsp":          "tsp",
	"teaspoon":     "tsp",
	"teaspoons":    "tsp",
	"fl oz":        "fl oz",
	"fluid ounce":  "fl oz",
	"fluid ounces": "fl oz",
	"oz":           "oz",
	"ounce":        "oz",
	"ounces":       "oz",
	"lb":           "lb",
	"lbs":          "lb",
	"pound":        "lb",
	"pounds":       "lb",
	"pint":         "pint",
	"pints":        "pint",
	"quart":        "quart",
	"quarts":       "quart",
	"gallon":       "gallon",
	"gallons":      "gallon",
	// German
	"el":       "EL",
	"tl":       "TL",
	"prise":    "Prise",
	"prisen":   "Prise",
	"bund":     "Bund",
	"stück":    "Stück",
	"stk":      "Stück",
	"scheibe":  "Scheibe",
	"scheiben": "Scheibe",
	"zehe":     "Zehe",
	"zehen":    "Zehe",
	"dose":     "Dose",
	"dosen":    "Dose",
	"becher":   "Becher",
	"packung":  "Packung",
	"päckchen": "Päckchen",
	"pkg":      "Packung",
	"msp":      "Msp",
}

// ingredientRe matches an amount (with optional fraction) and optional unit at the start of a string.
// Groups: (1) whole number, (2) fraction like 1/2, (3) rest of the string.
var ingredientRe = regexp.MustCompile(
	`^(\d+(?:[.,]\d+)?)\s*` + // whole number or decimal
		`(?:(\d+/\d+)\s*)?` + // optional fraction part (e.g. "1/2")
		`(.*)$`,
)

// fractionOnlyRe matches a string starting with only a fraction like "1/2 cup flour".
var fractionOnlyRe = regexp.MustCompile(`^(\d+/\d+)\s+(.*)$`)

// ParseIngredientString parses a raw ingredient string like "500g Hackfleisch" into components.
// Returns name, amount, unit. If no pattern matches, the whole string becomes the name.
func ParseIngredientString(s string) (name string, amount *float64, unit *string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil, nil
	}

	// Step 1: Replace unicode fractions with decimal values.
	s, unicodeVal := replaceUnicodeFractions(s)

	// Step 2: Try fraction-only pattern (e.g. "1/2 cup flour").
	if unicodeVal == 0 {
		if m := fractionOnlyRe.FindStringSubmatch(s); m != nil {
			frac := parseFraction(m[1])
			if frac > 0 {
				amt := frac
				amount = &amt
				s = strings.TrimSpace(m[2])
				return extractUnitAndName(s, amount)
			}
		}
	}

	// Step 3: Try the main pattern with whole number.
	m := ingredientRe.FindStringSubmatch(s)
	if m == nil || m[1] == "" {
		if unicodeVal > 0 {
			// We had a unicode fraction but no number prefix.
			amount = &unicodeVal
			return extractUnitAndName(s, amount)
		}
		return s, nil, nil
	}

	// Parse whole number.
	numStr := strings.Replace(m[1], ",", ".", 1)
	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return s, nil, nil
	}

	// Add fraction part if present.
	if m[2] != "" {
		val += parseFraction(m[2])
	}

	// Add unicode fraction if it was replaced.
	val += unicodeVal

	amount = &val
	rest := strings.TrimSpace(m[3])
	return extractUnitAndName(rest, amount)
}

// extractUnitAndName tries to match the leading word(s) of rest as a known unit.
func extractUnitAndName(rest string, amount *float64) (string, *float64, *string) {
	if rest == "" {
		return "", amount, nil
	}

	lower := strings.ToLower(rest)

	// Try two-word units first (e.g. "fl oz", "fluid ounces").
	words := strings.Fields(rest)
	if len(words) >= 2 {
		twoWord := strings.ToLower(words[0] + " " + words[1])
		if canonical, ok := knownUnits[twoWord]; ok {
			name := strings.TrimSpace(strings.Join(words[2:], " "))
			return name, amount, &canonical
		}
	}

	// Try single-word unit.
	if len(words) >= 1 {
		oneWord := strings.ToLower(words[0])
		// Also handle "500g" where unit is glued to the number — this is handled
		// by the regex already giving us the rest after the number.
		if canonical, ok := knownUnits[oneWord]; ok {
			name := strings.TrimSpace(strings.Join(words[1:], " "))
			return name, amount, &canonical
		}
	}

	// Check if rest starts with a period-suffixed abbreviation (e.g. "EL." → "EL").
	if len(words) >= 1 && strings.HasSuffix(words[0], ".") {
		stripped := strings.ToLower(strings.TrimSuffix(words[0], "."))
		if canonical, ok := knownUnits[stripped]; ok {
			name := strings.TrimSpace(strings.Join(words[1:], " "))
			return name, amount, &canonical
		}
	}

	// No unit found — entire rest is the name.
	_ = lower
	return rest, amount, nil
}

// replaceUnicodeFractions replaces the first unicode fraction character in s with nothing
// and returns the replaced string and the fraction value.
func replaceUnicodeFractions(s string) (string, float64) {
	for _, r := range s {
		if val, ok := unicodeFractions[r]; ok {
			s = strings.Replace(s, string(r), "", 1)
			s = strings.TrimSpace(s)
			// Collapse double spaces left by removal.
			s = strings.Join(strings.Fields(s), " ")
			return s, val
		}
	}
	return s, 0
}

// parseFraction parses a fraction string like "1/2" or "3/4" into a float64.
func parseFraction(s string) float64 {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return 0
	}
	num, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	den, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil || den == 0 {
		return 0
	}
	return num / den
}
