package htmlutil

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// jsonldScriptRe matches <script type="application/ld+json">…</script> blocks.
var jsonldScriptRe = regexp.MustCompile(`(?is)<script[^>]*type\s*=\s*["']application/ld\+json["'][^>]*>(.*?)</script>`)

// SchemaRecipeData holds the relevant fields extracted from a schema.org Recipe JSON-LD block.
type SchemaRecipeData struct {
	Title           string
	Servings        int
	PrepTimeMinutes *int
	CookTimeMinutes *int
	Ingredients     []string // Raw strings from recipeIngredient
	Instructions    string   // Markdown
}

// ExtractJSONLDBlocks returns all JSON-LD script block contents from raw HTML.
func ExtractJSONLDBlocks(rawHTML string) []string {
	matches := jsonldScriptRe.FindAllStringSubmatch(rawHTML, -1)
	blocks := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 {
			content := strings.TrimSpace(m[1])
			if content != "" {
				blocks = append(blocks, content)
			}
		}
	}
	return blocks
}

// ExtractRecipeJSONLD extracts a schema.org Recipe from JSON-LD blocks in raw HTML.
// Returns nil, false if no valid Recipe block is found.
func ExtractRecipeJSONLD(rawHTML string) (*SchemaRecipeData, bool) {
	blocks := ExtractJSONLDBlocks(rawHTML)
	for _, block := range blocks {
		raw := findRecipeInJSONLD([]byte(block))
		if raw == nil {
			continue
		}
		recipe := parseRecipeObject(*raw)
		if recipe != nil {
			return recipe, true
		}
	}
	return nil, false
}

// findRecipeInJSONLD locates a Recipe object in a parsed JSON-LD blob.
// Handles top-level objects, arrays, and @graph wrappers.
func findRecipeInJSONLD(data []byte) *json.RawMessage {
	// Try as a single object first.
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err == nil {
		if isRecipeType(obj) {
			raw := json.RawMessage(data)
			return &raw
		}
		// Check @graph array.
		if graphRaw, ok := obj["@graph"]; ok {
			if found := findRecipeInArray(graphRaw); found != nil {
				return found
			}
		}
		return nil
	}

	// Try as a top-level array.
	return findRecipeInArray(data)
}

// findRecipeInArray searches an array of JSON-LD objects for one with @type "Recipe".
func findRecipeInArray(data []byte) *json.RawMessage {
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil
	}
	for _, item := range arr {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(item, &obj); err != nil {
			continue
		}
		if isRecipeType(obj) {
			raw := json.RawMessage(item)
			return &raw
		}
	}
	return nil
}

// isRecipeType checks whether a JSON-LD object has @type "Recipe".
func isRecipeType(obj map[string]json.RawMessage) bool {
	typeRaw, ok := obj["@type"]
	if !ok {
		return false
	}

	// @type can be a string or an array of strings.
	var typeStr string
	if err := json.Unmarshal(typeRaw, &typeStr); err == nil {
		return typeStr == "Recipe"
	}
	var typeArr []string
	if err := json.Unmarshal(typeRaw, &typeArr); err == nil {
		for _, t := range typeArr {
			if t == "Recipe" {
				return true
			}
		}
	}
	return false
}

// parseRecipeObject converts a raw JSON-LD Recipe object into SchemaRecipeData.
// Returns nil if the data is too incomplete (missing title, ingredients, or instructions).
func parseRecipeObject(data json.RawMessage) *SchemaRecipeData {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	title, _ := raw["name"].(string)
	title = strings.TrimSpace(title)
	if title == "" {
		return nil
	}

	ingredients := parseStringArray(raw["recipeIngredient"])
	if len(ingredients) == 0 {
		return nil
	}

	instructions := parseInstructions(raw["recipeInstructions"])
	if instructions == "" {
		return nil
	}

	recipe := &SchemaRecipeData{
		Title:        title,
		Servings:     parseServings(raw["recipeYield"]),
		Ingredients:  ingredients,
		Instructions: instructions,
	}

	if pt, ok := raw["prepTime"].(string); ok {
		recipe.PrepTimeMinutes = parseISO8601Duration(pt)
	}
	if ct, ok := raw["cookTime"].(string); ok {
		recipe.CookTimeMinutes = parseISO8601Duration(ct)
	}

	return recipe
}

// parseStringArray extracts a []string from a JSON value that may be a string or array.
func parseStringArray(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					result = append(result, s)
				}
			}
		}
		return result
	case string:
		s := strings.TrimSpace(val)
		if s == "" {
			return nil
		}
		return []string{s}
	}
	return nil
}

// parseServings extracts a numeric servings value from the polymorphic recipeYield field.
func parseServings(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case string:
		return extractFirstNumber(val)
	case []interface{}:
		// Take the first element that yields a number.
		for _, item := range val {
			switch inner := item.(type) {
			case float64:
				return int(inner)
			case string:
				if n := extractFirstNumber(inner); n > 0 {
					return n
				}
			}
		}
	}
	return 0
}

// extractFirstNumber extracts the first integer from a string like "4 Portionen" or "4-6".
var firstNumberRe = regexp.MustCompile(`\d+`)

func extractFirstNumber(s string) int {
	m := firstNumberRe.FindString(s)
	if m == "" {
		return 0
	}
	n, _ := strconv.Atoi(m)
	return n
}

// parseInstructions converts recipeInstructions into a Markdown string.
// Handles: plain string, array of strings, array of HowToStep, array of HowToSection.
func parseInstructions(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return strings.TrimSpace(StripHTML(val))
	case []interface{}:
		return parseInstructionArray(val)
	}
	return ""
}

func parseInstructionArray(arr []interface{}) string {
	var parts []string
	stepNum := 1

	for _, item := range arr {
		switch val := item.(type) {
		case string:
			s := strings.TrimSpace(StripHTML(val))
			if s != "" {
				parts = append(parts, strconv.Itoa(stepNum)+". "+s)
				stepNum++
			}
		case map[string]interface{}:
			typ, _ := val["@type"].(string)
			switch typ {
			case "HowToSection":
				name, _ := val["name"].(string)
				if name != "" {
					parts = append(parts, "\n## "+strings.TrimSpace(name)+"\n")
					stepNum = 1
				}
				if items, ok := val["itemListElement"].([]interface{}); ok {
					sectionSteps := parseInstructionArray(items)
					if sectionSteps != "" {
						parts = append(parts, sectionSteps)
					}
				}
			case "HowToStep":
				text, _ := val["text"].(string)
				text = strings.TrimSpace(StripHTML(text))
				if text != "" {
					parts = append(parts, strconv.Itoa(stepNum)+". "+text)
					stepNum++
				}
			default:
				// Try "text" field for generic objects.
				if text, ok := val["text"].(string); ok {
					text = strings.TrimSpace(StripHTML(text))
					if text != "" {
						parts = append(parts, strconv.Itoa(stepNum)+". "+text)
						stepNum++
					}
				}
			}
		}
	}

	return strings.Join(parts, "\n")
}

// iso8601DurationRe matches ISO 8601 duration strings like PT1H30M, PT45M, PT2H.
var iso8601DurationRe = regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)

// parseISO8601Duration parses an ISO 8601 duration string into total minutes.
// Returns nil if the string is not a valid duration or results in 0 minutes.
func parseISO8601Duration(s string) *int {
	s = strings.TrimSpace(strings.ToUpper(s))
	m := iso8601DurationRe.FindStringSubmatch(s)
	if m == nil {
		return nil
	}

	total := 0
	if m[1] != "" {
		h, _ := strconv.Atoi(m[1])
		total += h * 60
	}
	if m[2] != "" {
		min, _ := strconv.Atoi(m[2])
		total += min
	}
	if m[3] != "" {
		sec, _ := strconv.Atoi(m[3])
		if sec >= 30 {
			total++
		}
	}

	if total == 0 {
		return nil
	}
	return &total
}
