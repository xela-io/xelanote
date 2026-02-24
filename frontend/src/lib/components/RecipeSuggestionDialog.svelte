<script lang="ts">
  import { Camera, ChefHat, Loader2, Search, Sparkles } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';
  import { locale } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import type { GeneratedRecipe, IngredientMatchResult, SimilarRecipeResult } from '$lib/api';
  import { extractIngredientsFromPhoto, findSimilarRecipes, suggestByIngredients } from '$lib/api';

  import RecipePreviewSaveDialog from './RecipePreviewSaveDialog.svelte';
  import BaseDialog from './ui/BaseDialog.svelte';

  interface Props {
    open: boolean;
    noteId?: string;
    mode: 'similar' | 'ingredients';
    collectionId?: number | null;
    onClose: () => void;
  }

  const { open, noteId, mode, collectionId, onClose }: Props = $props();

  // State
  let loading = $state(false);
  let error = $state<string | null>(null);
  let scopeAll = $state(true);

  // Similar results
  let similarResults = $state<SimilarRecipeResult[]>([]);

  // Ingredient mode
  let ingredientText = $state('');
  let matchResults = $state<IngredientMatchResult[]>([]);
  let generatedRecipes = $state<GeneratedRecipe[]>([]);
  let hasSearched = $state(false);

  // Photo upload
  let _photoFile = $state<File | null>(null);
  let photoPreview = $state<string | null>(null);
  let extracting = $state(false);

  // Preview dialog for generated recipes
  let previewRecipe = $state<GeneratedRecipe | null>(null);

  const currentLocale = $derived($locale?.substring(0, 2) ?? 'en');
  const effectiveCollectionId = $derived(scopeAll ? undefined : collectionId);

  function reset() {
    loading = false;
    error = null;
    similarResults = [];
    ingredientText = '';
    matchResults = [];
    generatedRecipes = [];
    hasSearched = false;
    _photoFile = null;
    photoPreview = null;
    extracting = false;
    previewRecipe = null;
  }

  // Auto-search when opening in similar mode
  $effect(() => {
    if (open && mode === 'similar' && noteId) {
      searchSimilar();
    }
    if (!open) {
      reset();
    }
  });

  async function searchSimilar() {
    if (!noteId) return;
    loading = true;
    error = null;
    try {
      similarResults = await findSimilarRecipes(noteId, currentLocale, effectiveCollectionId);
    } catch (err: unknown) {
      error = mapError(err);
    } finally {
      loading = false;
    }
  }

  async function searchByIngredients() {
    const ingredients = ingredientText
      .split('\n')
      .map((l) => l.trim())
      .filter(Boolean);
    if (ingredients.length === 0) return;

    loading = true;
    error = null;
    hasSearched = true;
    try {
      const result = await suggestByIngredients(ingredients, currentLocale, effectiveCollectionId);
      matchResults = result.matches;
      generatedRecipes = result.generated;
    } catch (err: unknown) {
      error = mapError(err);
    } finally {
      loading = false;
    }
  }

  async function handlePhotoUpload(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    _photoFile = file;
    photoPreview = URL.createObjectURL(file);

    extracting = true;
    try {
      const ingredients = await extractIngredientsFromPhoto(file, currentLocale);
      if (ingredients.length > 0) {
        ingredientText = ingredients.join('\n');
      }
    } catch (err: unknown) {
      error = mapError(err);
    } finally {
      extracting = false;
    }
  }

  function handleNavigate(noteId: string) {
    onClose();
    goto(`/note/${noteId}`);
  }

  function mapError(err: unknown): string {
    const error = err as { status?: number; message?: string };
    const status = error?.status;
    if (status === 424) {
      const msg = error?.message || '';
      if (msg.includes('vision')) {
        return $_('page.recipes.suggestions.no_vision_provider');
      }
      return $_('page.recipes.suggestions.no_provider');
    }
    if (status === 403) return $_('page.recipes.suggestions.encrypted_not_available');
    if (status === 502 || status === 504) return $_('page.recipes.suggestions.ai_unavailable');
    return $_('page.recipes.suggestions.ai_error');
  }

  function scoreColor(score: number): string {
    if (score >= 0.7) return 'text-success';
    if (score >= 0.4) return 'text-warning';
    return 'text-muted-foreground';
  }

  function scorePercent(score: number): string {
    return Math.round(score * 100) + '%';
  }

  const dialogTitle = $derived(
    mode === 'similar'
      ? $_('page.recipes.suggestions.similar_recipes')
      : $_('page.recipes.suggestions.ingredient_suggestions')
  );
</script>

<BaseDialog {open} title={dialogTitle} {onClose} size="lg" scrollable>
  {#snippet content()}
    <!-- Scope toggle (only if collection context exists) -->
    {#if collectionId && mode === 'similar'}
      <div class="flex gap-2 mb-4">
        <button
          onclick={() => {
            scopeAll = true;
            if (mode === 'similar') searchSimilar();
          }}
          class="ui-button text-sm"
          class:ui-button-primary={scopeAll}
          class:ui-button-secondary={!scopeAll}
        >
          {$_('page.recipes.suggestions.scope_all')}
        </button>
        <button
          onclick={() => {
            scopeAll = false;
            if (mode === 'similar') searchSimilar();
          }}
          class="ui-button text-sm"
          class:ui-button-primary={!scopeAll}
          class:ui-button-secondary={scopeAll}
        >
          {$_('page.recipes.suggestions.scope_collection')}
        </button>
      </div>
    {/if}

    <!-- Error -->
    {#if error}
      <div class="ui-alert ui-alert-danger mb-4 text-sm">
        <p>{error}</p>
        <button
          onclick={() => {
            error = null;
            if (mode === 'similar') searchSimilar();
          }}
          class="mt-1 underline text-xs"
        >
          {$_('page.recipes.suggestions.retry')}
        </button>
      </div>
    {/if}

    <!-- Loading -->
    {#if loading}
      <div class="flex flex-col items-center justify-center py-12 gap-3">
        <Loader2 class="w-8 h-8 animate-spin text-primary" />
        <p class="text-sm text-muted-foreground">
          {$_('page.recipes.suggestions.loading')}
        </p>
      </div>

      <!-- SIMILAR MODE -->
    {:else if mode === 'similar'}
      {#if similarResults.length === 0 && !error}
        <div class="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <ChefHat class="w-10 h-10 mb-2 opacity-50" />
          <p class="text-sm">{$_('page.recipes.suggestions.no_similar')}</p>
        </div>
      {:else}
        <div class="space-y-2">
          {#each similarResults as result (result.note_id)}
            <button
              onclick={() => handleNavigate(result.note_id)}
              class="ui-list-item w-full text-left p-3"
            >
              <div class="flex items-center justify-between">
                <span class="font-medium">{result.title}</span>
                <span class="text-xs {scoreColor(result.similarity_score)}">
                  {scorePercent(result.similarity_score)}
                  {$_('page.recipes.suggestions.similarity')}
                </span>
              </div>
              {#if result.reason}
                <p class="text-xs text-muted-foreground mt-1">{result.reason}</p>
              {/if}
            </button>
          {/each}
        </div>
      {/if}

      <!-- INGREDIENT MODE -->
    {:else}
      <!-- Input Phase -->
      {#if !hasSearched || (!loading && matchResults.length === 0 && generatedRecipes.length === 0)}
        <div class="space-y-3">
          <div class="relative">
            <textarea
              bind:value={ingredientText}
              placeholder={$_('page.recipes.suggestions.enter_ingredients')}
              class="ui-textarea w-full min-h-[120px] p-3 text-sm resize-y"
              onkeydown={(e) => {
                if (e.key === 'Enter' && e.ctrlKey) searchByIngredients();
              }}
            ></textarea>
          </div>

          <!-- Photo upload -->
          <div class="flex items-center gap-2">
            <label class="ui-button ui-button-secondary text-sm cursor-pointer">
              <Camera size={14} />
              {$_('page.recipes.suggestions.upload_photo')}
              <input
                type="file"
                accept="image/jpeg,image/png,image/webp"
                onchange={handlePhotoUpload}
                class="hidden"
              />
            </label>
            <span class="text-xs text-muted-foreground">
              {$_('page.recipes.suggestions.photo_formats')}
            </span>
          </div>

          {#if extracting}
            <div class="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 size={14} class="animate-spin" />
              {$_('page.recipes.suggestions.extracting')}
            </div>
          {/if}

          {#if photoPreview}
            <img src={photoPreview} alt="Uploaded" class="w-32 h-32 object-cover rounded-lg" />
          {/if}

          <button
            onclick={searchByIngredients}
            disabled={!ingredientText.trim() || loading}
            class="ui-button ui-button-primary text-sm"
          >
            <Search size={14} />
            {$_('page.recipes.suggestions.search')}
          </button>
        </div>
      {/if}

      <!-- Results Phase -->
      {#if hasSearched && !loading}
        <!-- Matching existing recipes -->
        {#if matchResults.length > 0}
          <div class="mt-4">
            <h3 class="text-sm font-semibold mb-2">
              {$_('page.recipes.suggestions.matching_recipes')}
            </h3>
            <div class="space-y-2">
              {#each matchResults as match (match.note_id)}
                <button
                  onclick={() => handleNavigate(match.note_id)}
                  class="ui-list-item w-full text-left p-3"
                >
                  <div class="flex items-center justify-between">
                    <span class="font-medium">{match.title}</span>
                    <span class="text-xs {scoreColor(match.match_score)}">
                      {scorePercent(match.match_score)}
                      {$_('page.recipes.suggestions.match')}
                    </span>
                  </div>
                  {#if match.matched_ingredients.length > 0}
                    <p class="text-xs text-success mt-1">
                      {$_('page.recipes.suggestions.matched_ingredients')}: {match.matched_ingredients.join(
                        ', '
                      )}
                    </p>
                  {/if}
                  {#if match.missing_ingredients.length > 0}
                    <p class="text-xs text-warning mt-1">
                      {$_('page.recipes.suggestions.missing_ingredients')}: {match.missing_ingredients.join(
                        ', '
                      )}
                    </p>
                  {/if}
                </button>
              {/each}
            </div>
          </div>
        {/if}

        <!-- Generated recipe ideas -->
        {#if generatedRecipes.length > 0}
          <div class="mt-4">
            <h3 class="text-sm font-semibold mb-2">
              <Sparkles class="inline w-4 h-4 mr-1" />
              {$_('page.recipes.suggestions.new_ideas')}
            </h3>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {#each generatedRecipes as recipe (recipe.title)}
                <div class="ui-panel-soft p-3">
                  <h4 class="font-medium text-sm">{recipe.title}</h4>
                  <p class="text-xs text-muted-foreground mt-1">
                    {recipe.servings}
                    {$_('page.recipes.servings')}
                    {#if recipe.difficulty}
                      &middot; {recipe.difficulty}
                    {/if}
                  </p>
                  <p class="text-xs text-muted-foreground mt-1 line-clamp-2">
                    {recipe.ingredients.map((i) => i.name).join(', ')}
                  </p>
                  <button
                    onclick={() => (previewRecipe = recipe)}
                    class="ui-button ui-button-ghost mt-2 px-0 py-0 text-xs text-primary"
                  >
                    {$_('page.recipes.suggestions.preview_save')}
                  </button>
                </div>
              {/each}
            </div>
          </div>
        {/if}

        <!-- No results -->
        {#if matchResults.length === 0 && generatedRecipes.length === 0 && !error}
          <div class="ui-empty-state ui-empty-state-compact py-8">
            <ChefHat class="w-10 h-10 mb-2 opacity-50" />
            <p class="text-sm">{$_('page.recipes.suggestions.no_matches')}</p>
          </div>
        {/if}

        <!-- Back to input -->
        <button
          onclick={() => {
            hasSearched = false;
            matchResults = [];
            generatedRecipes = [];
          }}
          class="ui-button ui-button-ghost mt-3 px-0 py-0 text-xs"
        >
          {$_('common.back')}
        </button>
      {/if}
    {/if}
  {/snippet}
</BaseDialog>

<!-- Preview + Save dialog for generated recipes -->
{#if previewRecipe}
  <RecipePreviewSaveDialog
    open={!!previewRecipe}
    recipe={previewRecipe}
    onClose={() => (previewRecipe = null)}
    onSaved={(noteId) => {
      previewRecipe = null;
      onClose();
      goto(`/note/${noteId}`);
    }}
  />
{/if}
