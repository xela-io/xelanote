package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/llm"
)

func TestNormalizeLocale(t *testing.T) {
	if got := normalizeLocale("de"); got != "de" {
		t.Fatalf("expected de, got %s", got)
	}
	if got := normalizeLocale("en"); got != "en" {
		t.Fatalf("expected en, got %s", got)
	}
	if got := normalizeLocale("fr"); got != "en" {
		t.Fatalf("expected fallback en, got %s", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Fatalf("unexpected truncate: %s", got)
	}
	if got := truncate("longer-string", 6); got != "longer" {
		t.Fatalf("unexpected truncate: %s", got)
	}
}

func TestComputeSnippetLengthBounds(t *testing.T) {
	for _, name := range []string{"claude", "gemini", "unknown"} {
		length := computeSnippetLength(name)
		if length < 50 || length > MaxSnippetLength {
			t.Fatalf("snippet length out of bounds for %s: %d", name, length)
		}
	}
}

func TestPreFilterByJaccard(t *testing.T) {
	candidates := []db.RecipeSummary{
		{NoteID: "a", IngredientNames: []string{"egg", "flour"}},
		{NoteID: "b", IngredientNames: []string{"egg"}},
		{NoteID: "c", IngredientNames: []string{"tomato"}},
	}
	target := []string{"egg", "flour"}

	filtered := preFilterByJaccard(candidates, target, 2)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 results, got %d", len(filtered))
	}
	if filtered[0].NoteID != "a" || filtered[1].NoteID != "b" {
		t.Fatalf("unexpected order: %+v", filtered)
	}
}

func TestValidateGeneratedRecipe_ClampsAndTrims(t *testing.T) {
	longName := strings.Repeat("a", 250)
	longUnit := strings.Repeat("b", 80)
	longGroup := strings.Repeat("g", 130)
	badDifficulty := "impossible"
	source := "  https://example.com/recipe  "
	r := &GeneratedRecipe{
		Servings:   0,
		Difficulty: &badDifficulty,
		SourceURL:  &source,
		Ingredients: []GeneratedIngredient{
			{Name: "  name  ", Unit: &longUnit, GroupName: &longGroup},
			{Name: longName},
		},
	}

	validateGeneratedRecipe(r)
	if r.Servings != 4 {
		t.Fatalf("expected default servings 4, got %d", r.Servings)
	}
	if r.Difficulty != nil {
		t.Fatalf("expected difficulty reset")
	}
	if r.Ingredients[0].Name != "name" {
		t.Fatalf("expected trimmed name")
	}
	if r.Ingredients[0].Unit == nil || len(*r.Ingredients[0].Unit) != 50 {
		t.Fatalf("expected trimmed unit length 50")
	}
	if r.Ingredients[0].GroupName == nil || len(*r.Ingredients[0].GroupName) != 100 {
		t.Fatalf("expected trimmed group_name length 100")
	}
	if len(r.Ingredients[1].Name) != 200 {
		t.Fatalf("expected name trimmed to 200")
	}
	if r.SourceURL == nil || *r.SourceURL != "https://example.com/recipe" {
		t.Fatalf("expected trimmed source_url, got %+v", r.SourceURL)
	}
}

func TestSaveGeneratedRecipe_ValidationsAndDefaults(t *testing.T) {
	database := setupTestDB(t)
	user := createTestUser(t, database, "user1")

	service := NewRecipeSuggestionService(database, &llm.ProviderRouter{}, NewRecipeService(database, NewNoteService(database)))

	_, err := service.SaveGeneratedRecipe(user.ID, SaveGeneratedRecipeRequest{})
	if err == nil {
		t.Fatalf("expected error for missing fields")
	}

	req := SaveGeneratedRecipeRequest{
		Title:        "My Recipe",
		Instructions: "Mix it",
		Servings:     0,
		Difficulty:   func() *string { v := "invalid"; return &v }(),
		Ingredients: []GeneratedIngredient{
			{Name: "  sugar ", Scalable: true, GroupName: func() *string { v := "Teig"; return &v }()},
			{Name: "   "},
		},
		SourceURL: func() *string { v := "https://example.com/cake"; return &v }(),
	}
	note, err := service.SaveGeneratedRecipe(user.ID, req)
	if err != nil {
		t.Fatalf("save generated: %v", err)
	}
	if note == nil || note.NoteType != "recipe" {
		t.Fatalf("expected recipe note")
	}

	meta, err := database.GetRecipeMetadata(note.ID)
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	if meta.Servings != 4 {
		t.Fatalf("expected default servings 4, got %d", meta.Servings)
	}
	if meta.Difficulty != nil {
		t.Fatalf("expected difficulty nil")
	}
	if meta.SourceURL == nil || *meta.SourceURL != "https://example.com/cake" {
		t.Fatalf("expected source_url to be set")
	}

	ingredients, err := database.GetRecipeIngredients(note.ID)
	if err != nil {
		t.Fatalf("get ingredients: %v", err)
	}
	if len(ingredients) != 1 || ingredients[0].Name != "sugar" {
		t.Fatalf("unexpected ingredients: %+v", ingredients)
	}
	if ingredients[0].GroupName == nil || *ingredients[0].GroupName != "Teig" {
		t.Fatalf("expected ingredient group_name to be set")
	}
}

func TestParseExtractedRecipe_NoRecipeFoundSentinel(t *testing.T) {
	_, err := parseExtractedRecipe(`{"error":"no_recipe_found"}`)
	if err == nil || !errors.Is(err, ErrNoRecipeFound) {
		t.Fatalf("expected ErrNoRecipeFound, got: %v", err)
	}
}

func TestNormalizeUnit(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: " Cups ", want: "cup"},
		{in: "tbsp", want: "tbsp"},
		{in: "fluid ounces", want: "fl oz"},
		{in: "lbs", want: "lb"},
		{in: "unknown", want: ""},
	}

	for _, tc := range cases {
		if got := normalizeUnit(tc.in); got != tc.want {
			t.Fatalf("normalizeUnit(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestConvertRecipeToMetricUnits(t *testing.T) {
	cupAmount := 1.0
	poundAmount := 2.0
	unknownAmount := 3.0

	recipe := &GeneratedRecipe{
		Ingredients: []GeneratedIngredient{
			{Name: "Milk", Amount: &cupAmount, Unit: ptrString("cups")},
			{Name: "Flour", Amount: &poundAmount, Unit: ptrString("lb")},
			{Name: "Pepper", Amount: &unknownAmount, Unit: ptrString("pinch")},
			{Name: "Salt", Amount: nil, Unit: ptrString("tbsp")},
		},
	}

	convertRecipeToMetricUnits(recipe)

	if recipe.Ingredients[0].Amount == nil || *recipe.Ingredients[0].Amount != 236.59 {
		t.Fatalf("expected cups -> 236.59 ml, got %+v", recipe.Ingredients[0].Amount)
	}
	if recipe.Ingredients[0].Unit == nil || *recipe.Ingredients[0].Unit != "ml" {
		t.Fatalf("expected cups unit to become ml, got %+v", recipe.Ingredients[0].Unit)
	}

	if recipe.Ingredients[1].Amount == nil || *recipe.Ingredients[1].Amount != 907.18 {
		t.Fatalf("expected lb -> 907.18 g, got %+v", recipe.Ingredients[1].Amount)
	}
	if recipe.Ingredients[1].Unit == nil || *recipe.Ingredients[1].Unit != "g" {
		t.Fatalf("expected lb unit to become g, got %+v", recipe.Ingredients[1].Unit)
	}

	if recipe.Ingredients[2].Amount == nil || *recipe.Ingredients[2].Amount != 3.0 {
		t.Fatalf("expected unknown unit amount unchanged, got %+v", recipe.Ingredients[2].Amount)
	}
	if recipe.Ingredients[2].Unit == nil || *recipe.Ingredients[2].Unit != "pinch" {
		t.Fatalf("expected unknown unit unchanged, got %+v", recipe.Ingredients[2].Unit)
	}

	if recipe.Ingredients[3].Unit == nil || *recipe.Ingredients[3].Unit != "tbsp" {
		t.Fatalf("expected nil amount to keep original unit, got %+v", recipe.Ingredients[3].Unit)
	}
}

func ptrString(v string) *string {
	return &v
}

func TestParseSelectedImageCandidates(t *testing.T) {
	candidates := []string{
		"https://example.com/a.jpg",
		"https://example.com/b.jpg",
		"https://example.com/c.jpg",
	}

	got := parseSelectedImageCandidates(`{"selected_indexes":[2,2,4,1]}`, candidates, 3)
	if len(got) != 2 {
		t.Fatalf("expected 2 selected images, got %d: %#v", len(got), got)
	}
	if got[0] != candidates[1] || got[1] != candidates[0] {
		t.Fatalf("unexpected selection order: %#v", got)
	}
}
