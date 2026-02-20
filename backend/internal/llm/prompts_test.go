package llm

import (
	"strings"
	"testing"
)

func TestBuildSimilarRecipePrompt_WithDietaryPreference(t *testing.T) {
	current := RecipeContext{
		NoteID:          "note-1",
		Title:           "Pasta Bolognese",
		IngredientNames: []string{"pasta", "tomato", "beef"},
	}
	candidates := []RecipeContext{
		{NoteID: "note-2", Title: "Lasagna", IngredientNames: []string{"pasta", "cheese"}},
	}

	prompt := BuildSimilarRecipePrompt(current, candidates, "en", "vegan")

	if !strings.Contains(prompt, "vegan") {
		t.Error("prompt should contain dietary preference 'vegan'")
	}
	if !strings.Contains(prompt, "Prioritize recipes that are compatible") {
		t.Error("prompt should contain prioritization instruction")
	}
}

func TestBuildSimilarRecipePrompt_NoDietaryBlock_WhenNone(t *testing.T) {
	current := RecipeContext{
		NoteID:          "note-1",
		Title:           "Pasta Bolognese",
		IngredientNames: []string{"pasta", "tomato", "beef"},
	}
	candidates := []RecipeContext{
		{NoteID: "note-2", Title: "Lasagna", IngredientNames: []string{"pasta", "cheese"}},
	}

	prompt := BuildSimilarRecipePrompt(current, candidates, "en", "none")

	if strings.Contains(prompt, "dietary preference") {
		t.Error("prompt should NOT contain dietary preference block when set to 'none'")
	}
}

func TestBuildSimilarRecipePrompt_NoDietaryBlock_WhenEmpty(t *testing.T) {
	current := RecipeContext{
		NoteID:          "note-1",
		Title:           "Pasta",
		IngredientNames: []string{"pasta"},
	}

	prompt := BuildSimilarRecipePrompt(current, nil, "en", "")

	if strings.Contains(prompt, "dietary preference") {
		t.Error("prompt should NOT contain dietary preference block when empty")
	}
}

func TestBuildIngredientMatchPrompt_WithDietaryPreference(t *testing.T) {
	ingredients := []string{"tofu", "rice", "soy sauce"}
	recipes := []RecipeContext{
		{NoteID: "note-1", Title: "Fried Rice", IngredientNames: []string{"rice", "soy sauce"}},
	}

	prompt := BuildIngredientMatchPrompt(ingredients, recipes, "en", "vegetarian")

	if !strings.Contains(prompt, "vegetarian") {
		t.Error("prompt should contain dietary preference 'vegetarian'")
	}
	if !strings.Contains(prompt, "Rank compatible recipes higher") {
		t.Error("prompt should contain ranking instruction")
	}
}

func TestBuildRecipeGenerationPrompt_WithDietaryPreference(t *testing.T) {
	ingredients := []string{"lentils", "rice", "spinach"}

	prompt := BuildRecipeGenerationPrompt(ingredients, nil, "en", "vegetarian")

	if !strings.Contains(prompt, "MUST be compatible") {
		t.Error("prompt should contain strict compatibility requirement for generated recipes")
	}
	if !strings.Contains(prompt, "vegetarian") {
		t.Error("prompt should contain 'vegetarian'")
	}
}

func TestBuildRecipeGenerationPrompt_NoDietaryBlock_WhenNone(t *testing.T) {
	ingredients := []string{"chicken", "rice"}

	prompt := BuildRecipeGenerationPrompt(ingredients, nil, "en", "none")

	if strings.Contains(prompt, "MUST be compatible") {
		t.Error("prompt should NOT contain dietary restriction when set to 'none'")
	}
}
