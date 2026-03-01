package htmlutil

import (
	"testing"
)

func TestExtractJSONLDBlocks_Single(t *testing.T) {
	html := `<html><head>
		<script type="application/ld+json">{"@type":"Recipe","name":"Test"}</script>
	</head></html>`
	blocks := ExtractJSONLDBlocks(html)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
}

func TestExtractJSONLDBlocks_Multiple(t *testing.T) {
	html := `<html>
		<script type="application/ld+json">{"@type":"WebSite"}</script>
		<script type="application/ld+json">{"@type":"Recipe","name":"Cake"}</script>
	</html>`
	blocks := ExtractJSONLDBlocks(html)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
}

func TestExtractJSONLDBlocks_None(t *testing.T) {
	html := `<html><body>No JSON-LD here</body></html>`
	blocks := ExtractJSONLDBlocks(html)
	if len(blocks) != 0 {
		t.Fatalf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestExtractRecipeJSONLD_Standard(t *testing.T) {
	html := `<html><head><script type="application/ld+json">{
		"@context": "https://schema.org",
		"@type": "Recipe",
		"name": "Chocolate Cake",
		"recipeYield": "8 servings",
		"prepTime": "PT20M",
		"cookTime": "PT1H30M",
		"recipeIngredient": ["200g flour", "100g sugar", "3 eggs"],
		"recipeInstructions": "Mix all ingredients. Bake at 350°F for 1 hour."
	}</script></head></html>`

	recipe, ok := ExtractRecipeJSONLD(html)
	if !ok || recipe == nil {
		t.Fatal("expected recipe to be found")
	}
	if recipe.Title != "Chocolate Cake" {
		t.Fatalf("expected 'Chocolate Cake', got %q", recipe.Title)
	}
	if recipe.Servings != 8 {
		t.Fatalf("expected 8 servings, got %d", recipe.Servings)
	}
	if recipe.PrepTimeMinutes == nil || *recipe.PrepTimeMinutes != 20 {
		t.Fatalf("expected 20 min prep time, got %v", recipe.PrepTimeMinutes)
	}
	if recipe.CookTimeMinutes == nil || *recipe.CookTimeMinutes != 90 {
		t.Fatalf("expected 90 min cook time, got %v", recipe.CookTimeMinutes)
	}
	if len(recipe.Ingredients) != 3 {
		t.Fatalf("expected 3 ingredients, got %d", len(recipe.Ingredients))
	}
	if recipe.Instructions == "" {
		t.Fatal("expected non-empty instructions")
	}
}

func TestExtractRecipeJSONLD_HowToStep(t *testing.T) {
	html := `<html><head><script type="application/ld+json">{
		"@type": "Recipe",
		"name": "Pasta",
		"recipeIngredient": ["500g pasta", "salt"],
		"recipeInstructions": [
			{"@type": "HowToStep", "text": "Boil water."},
			{"@type": "HowToStep", "text": "Add pasta."},
			{"@type": "HowToStep", "text": "Cook for 10 minutes."}
		]
	}</script></head></html>`

	recipe, ok := ExtractRecipeJSONLD(html)
	if !ok || recipe == nil {
		t.Fatal("expected recipe")
	}
	if recipe.Instructions != "1. Boil water.\n2. Add pasta.\n3. Cook for 10 minutes." {
		t.Fatalf("unexpected instructions: %q", recipe.Instructions)
	}
}

func TestExtractRecipeJSONLD_HowToSection(t *testing.T) {
	html := `<html><head><script type="application/ld+json">{
		"@type": "Recipe",
		"name": "Layered Cake",
		"recipeIngredient": ["flour", "sugar"],
		"recipeInstructions": [
			{
				"@type": "HowToSection",
				"name": "Dough",
				"itemListElement": [
					{"@type": "HowToStep", "text": "Mix flour and sugar."},
					{"@type": "HowToStep", "text": "Knead the dough."}
				]
			},
			{
				"@type": "HowToSection",
				"name": "Frosting",
				"itemListElement": [
					{"@type": "HowToStep", "text": "Whip cream."}
				]
			}
		]
	}</script></head></html>`

	recipe, ok := ExtractRecipeJSONLD(html)
	if !ok {
		t.Fatal("expected recipe")
	}
	if recipe.Title != "Layered Cake" {
		t.Fatalf("unexpected title: %q", recipe.Title)
	}
	// Should contain section headers and numbered steps.
	if recipe.Instructions == "" {
		t.Fatal("expected non-empty instructions")
	}
	// Check that section names appear.
	if !containsSubstring(recipe.Instructions, "Dough") {
		t.Fatalf("expected section 'Dough' in instructions: %q", recipe.Instructions)
	}
	if !containsSubstring(recipe.Instructions, "Frosting") {
		t.Fatalf("expected section 'Frosting' in instructions: %q", recipe.Instructions)
	}
}

func TestExtractRecipeJSONLD_GraphWrapper(t *testing.T) {
	html := `<html><head><script type="application/ld+json">{
		"@context": "https://schema.org",
		"@graph": [
			{"@type": "WebSite", "name": "My Blog"},
			{"@type": "Recipe", "name": "Soup", "recipeIngredient": ["water", "salt"], "recipeInstructions": "Boil water. Add salt."},
			{"@type": "Person", "name": "Chef"}
		]
	}</script></head></html>`

	recipe, ok := ExtractRecipeJSONLD(html)
	if !ok || recipe == nil {
		t.Fatal("expected recipe from @graph")
	}
	if recipe.Title != "Soup" {
		t.Fatalf("expected 'Soup', got %q", recipe.Title)
	}
}

func TestExtractRecipeJSONLD_TopLevelArray(t *testing.T) {
	html := `<html><head><script type="application/ld+json">[
		{"@type": "WebSite", "name": "My Blog"},
		{"@type": "Recipe", "name": "Salad", "recipeIngredient": ["lettuce"], "recipeInstructions": "Wash lettuce."}
	]</script></head></html>`

	recipe, ok := ExtractRecipeJSONLD(html)
	if !ok || recipe == nil {
		t.Fatal("expected recipe from top-level array")
	}
	if recipe.Title != "Salad" {
		t.Fatalf("expected 'Salad', got %q", recipe.Title)
	}
}

func TestExtractRecipeJSONLD_MissingTitle(t *testing.T) {
	html := `<html><head><script type="application/ld+json">{
		"@type": "Recipe",
		"recipeIngredient": ["flour"],
		"recipeInstructions": "Mix."
	}</script></head></html>`

	_, ok := ExtractRecipeJSONLD(html)
	if ok {
		t.Fatal("expected no recipe when title is missing")
	}
}

func TestExtractRecipeJSONLD_MissingIngredients(t *testing.T) {
	html := `<html><head><script type="application/ld+json">{
		"@type": "Recipe",
		"name": "Empty Recipe",
		"recipeInstructions": "Do nothing."
	}</script></head></html>`

	_, ok := ExtractRecipeJSONLD(html)
	if ok {
		t.Fatal("expected no recipe when ingredients are missing")
	}
}

func TestExtractRecipeJSONLD_MissingInstructions(t *testing.T) {
	html := `<html><head><script type="application/ld+json">{
		"@type": "Recipe",
		"name": "No Instructions",
		"recipeIngredient": ["flour"]
	}</script></head></html>`

	_, ok := ExtractRecipeJSONLD(html)
	if ok {
		t.Fatal("expected no recipe when instructions are missing")
	}
}

func TestExtractRecipeJSONLD_InvalidJSON(t *testing.T) {
	html := `<html><head><script type="application/ld+json">{not valid json}</script></head></html>`
	_, ok := ExtractRecipeJSONLD(html)
	if ok {
		t.Fatal("expected no recipe for invalid JSON")
	}
}

func TestExtractRecipeJSONLD_HTMLInInstructions(t *testing.T) {
	html := `<html><head><script type="application/ld+json">{
		"@type": "Recipe",
		"name": "HTML Recipe",
		"recipeIngredient": ["flour"],
		"recipeInstructions": "<p>Mix <b>flour</b> with water.</p><p>Bake for 30 min.</p>"
	}</script></head></html>`

	recipe, ok := ExtractRecipeJSONLD(html)
	if !ok || recipe == nil {
		t.Fatal("expected recipe")
	}
	// HTML should be stripped.
	if containsSubstring(recipe.Instructions, "<p>") || containsSubstring(recipe.Instructions, "<b>") {
		t.Fatalf("expected HTML to be stripped from instructions: %q", recipe.Instructions)
	}
}

func TestExtractRecipeJSONLD_TypeAsArray(t *testing.T) {
	html := `<html><head><script type="application/ld+json">{
		"@type": ["Recipe"],
		"name": "Array Type Recipe",
		"recipeIngredient": ["flour"],
		"recipeInstructions": "Mix."
	}</script></head></html>`

	recipe, ok := ExtractRecipeJSONLD(html)
	if !ok || recipe == nil {
		t.Fatal("expected recipe with @type as array")
	}
	if recipe.Title != "Array Type Recipe" {
		t.Fatalf("expected 'Array Type Recipe', got %q", recipe.Title)
	}
}

func TestParseISO8601Duration(t *testing.T) {
	tests := []struct {
		input string
		want  *int
	}{
		{"PT1H30M", intPtr(90)},
		{"PT45M", intPtr(45)},
		{"PT2H", intPtr(120)},
		{"PT0H15M", intPtr(15)},
		{"PT1H0M", intPtr(60)},
		{"pt30m", intPtr(30)},
		{"PT1H30M45S", intPtr(91)}, // 45s rounds up
		{"PT1H30M10S", intPtr(90)}, // 10s doesn't round up
		{"", nil},
		{"invalid", nil},
		{"P1D", nil},
		{"PT0M", nil},
	}

	for _, tc := range tests {
		got := parseISO8601Duration(tc.input)
		if tc.want == nil && got != nil {
			t.Errorf("parseISO8601Duration(%q) = %d, want nil", tc.input, *got)
		} else if tc.want != nil && got == nil {
			t.Errorf("parseISO8601Duration(%q) = nil, want %d", tc.input, *tc.want)
		} else if tc.want != nil && got != nil && *got != *tc.want {
			t.Errorf("parseISO8601Duration(%q) = %d, want %d", tc.input, *got, *tc.want)
		}
	}
}

func TestParseServings(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  int
	}{
		{"nil", nil, 0},
		{"number", float64(4), 4},
		{"string digits", "4", 4},
		{"string with text", "4 Portionen", 4},
		{"string range", "4-6", 4},
		{"array with string", []interface{}{"8 servings"}, 8},
		{"array with number", []interface{}{float64(6)}, 6},
		{"empty string", "", 0},
		{"no number string", "many", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseServings(tc.input)
			if got != tc.want {
				t.Errorf("parseServings(%v) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseInstructions_StringArray(t *testing.T) {
	input := []interface{}{
		"Preheat oven.",
		"Mix ingredients.",
		"Bake.",
	}
	got := parseInstructions(input)
	want := "1. Preheat oven.\n2. Mix ingredients.\n3. Bake."
	if got != want {
		t.Errorf("parseInstructions(string array) = %q, want %q", got, want)
	}
}

func TestExtractRecipeJSONLD_ServingsNumeric(t *testing.T) {
	html := `<html><head><script type="application/ld+json">{
		"@type": "Recipe",
		"name": "Test",
		"recipeYield": 4,
		"recipeIngredient": ["flour"],
		"recipeInstructions": "Mix."
	}</script></head></html>`

	recipe, ok := ExtractRecipeJSONLD(html)
	if !ok {
		t.Fatal("expected recipe")
	}
	if recipe.Servings != 4 {
		t.Fatalf("expected 4 servings, got %d", recipe.Servings)
	}
}

func TestExtractRecipeJSONLD_NonRecipeJSONLD(t *testing.T) {
	html := `<html><head><script type="application/ld+json">{
		"@type": "WebSite",
		"name": "My Blog"
	}</script></head></html>`

	_, ok := ExtractRecipeJSONLD(html)
	if ok {
		t.Fatal("expected no recipe for WebSite type")
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && contains(s, sub))
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func intPtr(v int) *int {
	return &v
}
