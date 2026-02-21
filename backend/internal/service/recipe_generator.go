package service

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/llm"
)

// fahrenheitPattern matches temperature values in Fahrenheit within recipe text.
// Matches patterns like: 350°F, 350 °F, 350 degrees F, 350 degrees Fahrenheit, 350 Grad Fahrenheit.
var fahrenheitPattern = regexp.MustCompile(`(\d+)\s*(?:°\s*|degrees?\s+|Grad\s+)[Ff](?:ahrenheit)?(?:\b|(?:[^a-zA-Z]))`)

// SaveGeneratedRecipeRequest is the input for saving a generated recipe.
type SaveGeneratedRecipeRequest struct {
	Title           string                `json:"title"`
	Instructions    string                `json:"instructions"`
	Servings        int                   `json:"servings"`
	PrepTimeMinutes *int                  `json:"prep_time_minutes"`
	CookTimeMinutes *int                  `json:"cook_time_minutes"`
	Difficulty      *string               `json:"difficulty"`
	SourceURL       *string               `json:"source_url,omitempty"`
	Ingredients     []GeneratedIngredient `json:"ingredients"`
	FolderPath      string                `json:"folder_path"`
}

// SaveGeneratedRecipe saves a generated recipe as a new note with metadata and ingredients.
func (s *RecipeSuggestionService) SaveGeneratedRecipe(
	userID int, req SaveGeneratedRecipeRequest,
) (*db.Note, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	if strings.TrimSpace(req.Instructions) == "" {
		return nil, fmt.Errorf("instructions are required")
	}
	if req.Servings < 1 {
		req.Servings = 4
	}
	if req.Servings > 999 {
		req.Servings = 999
	}
	if req.FolderPath == "" {
		req.FolderPath = "/Rezepte"
	}

	// Validate difficulty
	if req.Difficulty != nil {
		d := *req.Difficulty
		if d != "easy" && d != "medium" && d != "hard" {
			req.Difficulty = nil
		}
	}

	metadata := db.RecipeMetadata{
		Servings:        req.Servings,
		PrepTimeMinutes: req.PrepTimeMinutes,
		CookTimeMinutes: req.CookTimeMinutes,
		Difficulty:      req.Difficulty,
		SourceURL:       req.SourceURL,
	}

	ingredients := make([]db.RecipeIngredient, 0, len(req.Ingredients))
	for _, ing := range req.Ingredients {
		name := strings.TrimSpace(ing.Name)
		if name == "" {
			continue
		}
		ingredients = append(ingredients, db.RecipeIngredient{
			Name:      name,
			Amount:    ing.Amount,
			Unit:      ing.Unit,
			GroupName: ing.GroupName,
			Optional:  ing.Optional,
			Scalable:  ing.Scalable,
		})
	}

	note, err := s.db.CreateRecipeNoteWithIngredients(
		userID, req.Title, req.Instructions, req.FolderPath,
		metadata, ingredients,
	)
	if err != nil {
		return nil, err
	}

	s.invalidateRecipeSummaryCache(userID)
	return note, nil
}

// validateGeneratedRecipe corrects/clamps values from LLM output.
func validateGeneratedRecipe(r *GeneratedRecipe) {
	if r.Servings < 1 {
		r.Servings = 4
	}
	if r.Servings > 999 {
		r.Servings = 999
	}
	r.Title = strings.TrimSpace(r.Title)
	r.Instructions = strings.TrimSpace(r.Instructions)
	if r.Difficulty != nil {
		d := *r.Difficulty
		if d != "easy" && d != "medium" && d != "hard" {
			r.Difficulty = nil
		}
	}
	if r.SourceURL != nil {
		trimmed := strings.TrimSpace(*r.SourceURL)
		if trimmed == "" {
			r.SourceURL = nil
		} else {
			if len(trimmed) > 2048 {
				trimmed = trimmed[:2048]
			}
			r.SourceURL = &trimmed
		}
	}
	for i := range r.Ingredients {
		r.Ingredients[i].Name = strings.TrimSpace(r.Ingredients[i].Name)
		if len(r.Ingredients[i].Name) > 200 {
			r.Ingredients[i].Name = r.Ingredients[i].Name[:200]
		}
		if r.Ingredients[i].Unit != nil {
			trimmed := strings.TrimSpace(*r.Ingredients[i].Unit)
			if len(trimmed) > 50 {
				trimmed = trimmed[:50]
			}
			r.Ingredients[i].Unit = &trimmed
		}
		if r.Ingredients[i].GroupName != nil {
			trimmed := strings.TrimSpace(*r.Ingredients[i].GroupName)
			if trimmed == "" {
				r.Ingredients[i].GroupName = nil
			} else {
				if len(trimmed) > 100 {
					trimmed = trimmed[:100]
				}
				r.Ingredients[i].GroupName = &trimmed
			}
		}
	}
}

func parseExtractedRecipe(rawResponse string) (*GeneratedRecipe, error) {
	cleaned := llm.CleanMarkdownCodeBlock(rawResponse)

	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(cleaned), &errResp); err == nil && errResp.Error == "no_recipe_found" {
		return nil, ErrNoRecipeFound
	}

	var recipe GeneratedRecipe
	if err := json.Unmarshal([]byte(cleaned), &recipe); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}
	validateGeneratedRecipe(&recipe)
	if recipe.Title == "" || recipe.Instructions == "" || len(recipe.Ingredients) == 0 {
		return nil, ErrNoRecipeFound
	}
	return &recipe, nil
}

func convertRecipeToMetricUnits(recipe *GeneratedRecipe) {
	for i := range recipe.Ingredients {
		convertIngredientToMetricUnits(&recipe.Ingredients[i])
	}
}

func convertIngredientToMetricUnits(ingredient *GeneratedIngredient) {
	if ingredient.Unit == nil {
		return
	}

	normalizedUnit := normalizeUnit(*ingredient.Unit)
	if normalizedUnit == "" {
		return
	}

	amount, convertedUnit, ok := convertToMetricAmount(ingredient.Amount, normalizedUnit)
	if !ok {
		return
	}

	ingredient.Amount = amount
	ingredient.Unit = &convertedUnit
}

func normalizeUnit(unit string) string {
	u := strings.ToLower(strings.TrimSpace(unit))
	if u == "" {
		return ""
	}

	replacer := strings.NewReplacer(".", "", "-", " ")
	u = replacer.Replace(u)
	u = strings.Join(strings.Fields(u), " ")

	aliases := map[string]string{
		"c":            "cup",
		"cup":          "cup",
		"cups":         "cup",
		"tbsp":         "tbsp",
		"tbsps":        "tbsp",
		"tablespoon":   "tbsp",
		"tablespoons":  "tbsp",
		"tsp":          "tsp",
		"tsps":         "tsp",
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
		"pt":           "pint",
		"pint":         "pint",
		"pints":        "pint",
		"qt":           "quart",
		"quart":        "quart",
		"quarts":       "quart",
		"gal":          "gallon",
		"gallon":       "gallon",
		"gallons":      "gallon",
	}

	if canonical, ok := aliases[u]; ok {
		return canonical
	}
	return ""
}

func convertToMetricAmount(amount *float64, normalizedUnit string) (*float64, string, bool) {
	if amount == nil {
		return nil, "", false
	}

	converted := *amount
	unit := ""

	switch normalizedUnit {
	case "cup":
		converted = converted * 236.588
		unit = "ml"
	case "tbsp":
		converted = converted * 14.7868
		unit = "ml"
	case "tsp":
		converted = converted * 4.92892
		unit = "ml"
	case "fl oz":
		converted = converted * 29.5735
		unit = "ml"
	case "pint":
		converted = converted * 473.176
		unit = "ml"
	case "quart":
		converted = converted * 946.353
		unit = "ml"
	case "gallon":
		converted = converted * 3785.41
		unit = "ml"
	case "oz":
		converted = converted * 28.3495
		unit = "g"
	case "lb":
		converted = converted * 453.592
		unit = "g"
	default:
		return nil, "", false
	}

	if unit == "ml" && converted >= 1000 {
		converted = converted / 1000
		unit = "l"
	} else if unit == "g" && converted >= 1000 {
		converted = converted / 1000
		unit = "kg"
	}

	converted = math.Round(converted*100) / 100
	return &converted, unit, true
}

// convertRecipeTemperatures converts Fahrenheit temperatures to Celsius in the recipe instructions.
func convertRecipeTemperatures(recipe *GeneratedRecipe) {
	recipe.Instructions = convertFahrenheitToCelsius(recipe.Instructions)
}

// convertFahrenheitToCelsius replaces Fahrenheit temperatures in text with Celsius equivalents.
// Values are rounded to the nearest 5°C, which is conventional for cooking temperatures.
func convertFahrenheitToCelsius(text string) string {
	return fahrenheitPattern.ReplaceAllStringFunc(text, func(match string) string {
		sub := fahrenheitPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		f, err := strconv.Atoi(sub[1])
		if err != nil {
			return match
		}
		c := fahrenheitToCelsius(f)
		// Preserve any trailing character that was part of the match but not the temperature.
		suffix := ""
		trimmed := strings.TrimRight(match, " ")
		if len(trimmed) > 0 {
			last := trimmed[len(trimmed)-1]
			if last != 'F' && last != 'f' && last != 't' { // not ending on F/f/fahrenheit
				suffix = string(last)
			}
		}
		return fmt.Sprintf("%d°C%s", c, suffix)
	})
}

// fahrenheitToCelsius converts a Fahrenheit value to Celsius, rounded to the nearest 5.
func fahrenheitToCelsius(f int) int {
	c := float64(f-32) * 5.0 / 9.0
	return int(math.Round(c/5.0)) * 5
}
