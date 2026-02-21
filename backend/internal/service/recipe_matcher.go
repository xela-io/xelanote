package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/llm"
)

// FindSimilarRecipes finds recipes similar to the given recipe using LLM.
func (s *RecipeSuggestionService) FindSimilarRecipes(
	ctx context.Context, userID int, noteID string,
	collectionID *int, locale string,
) ([]SimilarRecipeResult, error) {
	locale = normalizeLocale(locale)

	// Get provider
	provider, err := s.router.GetAnyProvider(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Load current recipe detail
	detail, err := s.recipe.GetRecipeDetail(userID, noteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe detail: %w", err)
	}
	if detail.Encrypted {
		return nil, ErrRecipeEncrypted
	}

	// Build current recipe context
	currentIngNames := make([]string, 0, len(detail.Ingredients))
	for _, ing := range detail.Ingredients {
		currentIngNames = append(currentIngNames, ing.Name)
	}
	var difficulty *string
	if detail.Metadata != nil {
		difficulty = detail.Metadata.Difficulty
	}
	current := llm.RecipeContext{
		NoteID:          noteID,
		Title:           detail.Note.Title,
		IngredientNames: currentIngNames,
		ContentSnippet:  truncate(detail.Note.Content, MaxSnippetLength),
		Difficulty:      difficulty,
	}

	// Load candidate summaries (cached)
	snippetLen := computeSnippetLength(provider.Name())
	var summaries []db.RecipeSummary
	if collectionID != nil {
		summaries, err = s.getRecipeSummariesInCollection(userID, *collectionID, snippetLen)
	} else {
		summaries, err = s.getRecipeSummaries(userID, snippetLen)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe summaries: %w", err)
	}

	// Remove current recipe from candidates (service-level self-match exclusion)
	candidates := make([]db.RecipeSummary, 0, len(summaries))
	for _, s := range summaries {
		if s.NoteID != noteID {
			candidates = append(candidates, s)
		}
	}

	if len(candidates) < 2 {
		return []SimilarRecipeResult{}, nil
	}

	// Pre-filter if too many candidates
	if len(candidates) > MaxRecipesForPrompt {
		candidates = preFilterByJaccard(candidates, currentIngNames, MaxRecipesForPrompt)
	}

	// Build prompt contexts
	promptCandidates := make([]llm.RecipeContext, len(candidates))
	for i, c := range candidates {
		promptCandidates[i] = llm.RecipeContext{
			NoteID:          c.NoteID,
			Title:           c.Title,
			IngredientNames: c.IngredientNames,
			ContentSnippet:  c.ContentSnippet,
			Difficulty:      c.Difficulty,
			Servings:        c.Servings,
		}
	}

	// Load dietary preference (graceful fallback to "none")
	dietaryPref, err := s.db.GetDietaryPreference(userID)
	if err != nil {
		s.logger.Warn("failed to load dietary preference, using default", slog.String("error", err.Error()))
		dietaryPref = "none"
	}

	prompt := llm.BuildSimilarRecipePrompt(current, promptCandidates, locale, dietaryPref)

	response, err := provider.Generate(ctx, prompt, 2000)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse JSON response
	var results []SimilarRecipeResult
	if err := json.Unmarshal([]byte(llm.CleanMarkdownCodeBlock(response)), &results); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Validate note_ids exist in our candidate set
	candidateIDs := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		candidateIDs[c.NoteID] = true
	}
	validated := make([]SimilarRecipeResult, 0, len(results))
	for _, r := range results {
		if candidateIDs[r.NoteID] && r.NoteID != noteID {
			validated = append(validated, r)
		}
	}

	return validated, nil
}

// preFilterByJaccard selects the top-N candidates by Jaccard similarity on ingredient names.
func preFilterByJaccard(candidates []db.RecipeSummary, targetIngredients []string, limit int) []db.RecipeSummary {
	targetSet := make(map[string]bool, len(targetIngredients))
	for _, ing := range targetIngredients {
		targetSet[strings.ToLower(strings.TrimSpace(ing))] = true
	}

	type scored struct {
		index int
		score float64
	}

	scores := make([]scored, len(candidates))
	for i, c := range candidates {
		candSet := make(map[string]bool, len(c.IngredientNames))
		for _, ing := range c.IngredientNames {
			candSet[strings.ToLower(strings.TrimSpace(ing))] = true
		}

		intersection := 0
		for k := range targetSet {
			if candSet[k] {
				intersection++
			}
		}

		union := len(targetSet) + len(candSet) - intersection
		score := 0.0
		if union > 0 {
			score = float64(intersection) / float64(union)
		}
		scores[i] = scored{index: i, score: score}
	}

	sort.Slice(scores, func(a, b int) bool {
		return scores[a].score > scores[b].score
	})

	result := make([]db.RecipeSummary, 0, limit)
	for i := 0; i < limit && i < len(scores); i++ {
		result = append(result, candidates[scores[i].index])
	}
	return result
}
