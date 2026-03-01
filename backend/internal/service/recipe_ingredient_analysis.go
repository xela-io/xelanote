package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/xela-io/xelanote/internal/htmlutil"
	"github.com/xela-io/xelanote/internal/llm"
)

// ExtractIngredientsFromPhoto uses a vision model to identify ingredients in a photo.
func (s *RecipeSuggestionService) ExtractIngredientsFromPhoto(
	ctx context.Context, userID int,
	imageData []byte, mimeType string, locale string,
) ([]string, error) {
	locale = normalizeLocale(locale)

	visionProvider, err := s.router.GetVisionProvider(ctx, userID)
	if err != nil {
		if errors.Is(err, llm.ErrVisionNotAvailable) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get vision provider: %w", err)
	}

	prompt := llm.BuildFridgePhotoPrompt(locale)
	response, err := visionProvider.GenerateWithImage(ctx, prompt, imageData, mimeType, 1000)
	if err != nil {
		return nil, fmt.Errorf("vision API call failed: %w", err)
	}

	var ingredients []string
	if err := json.Unmarshal([]byte(llm.CleanMarkdownCodeBlock(response)), &ingredients); err != nil {
		return nil, fmt.Errorf("failed to parse vision response: %w", err)
	}

	return ingredients, nil
}

// ExtractRecipeFromImage uses a vision model to extract a full recipe from an image.
func (s *RecipeSuggestionService) ExtractRecipeFromImage(
	ctx context.Context, userID int,
	imageData []byte, mimeType string, locale string,
) (*GeneratedRecipe, error) {
	locale = normalizeLocale(locale)

	visionProvider, err := s.router.GetVisionProvider(ctx, userID)
	if err != nil {
		if errors.Is(err, llm.ErrVisionNotAvailable) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get vision provider: %w", err)
	}

	prompt := llm.BuildRecipeExtractionFromImagePrompt(locale)
	response, err := visionProvider.GenerateWithImage(ctx, prompt, imageData, mimeType, 4000)
	if err != nil {
		return nil, fmt.Errorf("vision API call failed: %w", err)
	}

	recipe, err := parseExtractedRecipe(response)
	if err != nil {
		return nil, err
	}
	convertRecipeToMetricUnits(recipe)
	convertRecipeTemperatures(recipe)
	return recipe, nil
}

// ExtractRecipeFromURL fetches a webpage and extracts a recipe.
// It first tries to extract structured JSON-LD data (no LLM needed).
// If no JSON-LD Recipe is found, it falls back to LLM-based extraction.
func (s *RecipeSuggestionService) ExtractRecipeFromURL(
	ctx context.Context, userID int, rawURL string, locale string,
) (*GeneratedRecipe, error) {
	locale = normalizeLocale(locale)

	rawHTML, _, err := htmlutil.FetchHTML(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	// Try JSON-LD extraction first (fast, no LLM needed).
	if schemaRecipe, ok := htmlutil.ExtractRecipeJSONLD(rawHTML); ok {
		s.logger.Info("recipe extracted from JSON-LD, skipping LLM",
			slog.String("url", rawURL), slog.String("title", schemaRecipe.Title))

		recipe := mapSchemaRecipeToGenerated(schemaRecipe)
		convertRecipeToMetricUnits(recipe)
		convertRecipeTemperatures(recipe)

		normalizedURL := strings.TrimSpace(rawURL)
		if normalizedURL != "" {
			recipe.SourceURL = &normalizedURL
		}
		validateGeneratedRecipe(recipe)
		return recipe, nil
	}

	// Fallback: strip HTML and use LLM.
	pageText := htmlutil.StripHTML(rawHTML)
	if len(pageText) > htmlutil.MaxTextChars {
		pageText = pageText[:htmlutil.MaxTextChars]
	}

	provider, err := s.router.GetAnyProvider(ctx, userID)
	if err != nil {
		return nil, err
	}

	prompt := llm.BuildRecipeExtractionFromTextPrompt(pageText, locale)
	response, err := provider.Generate(ctx, prompt, 4000)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	recipe, err := parseExtractedRecipe(response)
	if err != nil {
		return nil, err
	}
	convertRecipeToMetricUnits(recipe)
	convertRecipeTemperatures(recipe)

	normalizedURL := strings.TrimSpace(rawURL)
	if normalizedURL != "" {
		recipe.SourceURL = &normalizedURL
	}
	validateGeneratedRecipe(recipe)

	return recipe, nil
}

// mapSchemaRecipeToGenerated converts structured JSON-LD recipe data into a GeneratedRecipe.
func mapSchemaRecipeToGenerated(schema *htmlutil.SchemaRecipeData) *GeneratedRecipe {
	ingredients := make([]GeneratedIngredient, 0, len(schema.Ingredients))
	for _, raw := range schema.Ingredients {
		name, amount, unit := htmlutil.ParseIngredientString(raw)
		if strings.TrimSpace(name) == "" && amount == nil {
			continue
		}
		ing := GeneratedIngredient{
			Name:     name,
			Amount:   amount,
			Unit:     unit,
			Scalable: amount != nil,
		}
		ingredients = append(ingredients, ing)
	}

	return &GeneratedRecipe{
		Title:           schema.Title,
		Servings:        schema.Servings,
		PrepTimeMinutes: schema.PrepTimeMinutes,
		CookTimeMinutes: schema.CookTimeMinutes,
		Ingredients:     ingredients,
		Instructions:    schema.Instructions,
	}
}

// SelectMainRecipeImages uses an LLM to choose the best recipe images from URL candidates.
// Falls back to the first items if no provider is available or parsing fails.
func (s *RecipeSuggestionService) SelectMainRecipeImages(
	ctx context.Context, userID int, pageURL string, candidates []string, limit int,
) []string {
	if limit < 1 {
		return []string{}
	}
	if len(candidates) <= limit {
		return candidates
	}

	provider, err := s.router.GetAnyProvider(ctx, userID)
	if err != nil {
		return candidates[:limit]
	}

	prompt := llm.BuildRecipeMainImageSelectionPrompt(pageURL, candidates)
	response, err := provider.Generate(ctx, prompt, 400)
	if err != nil {
		s.logger.Warn("recipe image selection LLM call failed", slog.String("error", err.Error()))
		return candidates[:limit]
	}

	selected := parseSelectedImageCandidates(response, candidates, limit)
	if len(selected) == 0 {
		return candidates[:limit]
	}
	return selected
}

func parseSelectedImageCandidates(rawResponse string, candidates []string, limit int) []string {
	var parsed struct {
		SelectedIndexes []int `json:"selected_indexes"`
	}
	if err := json.Unmarshal([]byte(llm.CleanMarkdownCodeBlock(rawResponse)), &parsed); err != nil {
		return []string{}
	}

	selected := make([]string, 0, limit)
	seen := make(map[int]bool)
	for _, idx := range parsed.SelectedIndexes {
		if idx < 1 || idx > len(candidates) {
			continue
		}
		if seen[idx] {
			continue
		}
		seen[idx] = true
		selected = append(selected, candidates[idx-1])
		if len(selected) >= limit {
			break
		}
	}
	return selected
}
