<script lang="ts">
  import { ChefHat, Clock, Loader2, Lock, Plus, Sparkles } from 'lucide-svelte';
  import { ArrowLeft, Edit, Eye, Pencil, Trash2, Users, Users as UsersIcon } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import type { RecipeCollection } from '$lib/api';
  import RecipeCollectionDialog from '$lib/components/RecipeCollectionDialog.svelte';
  import RecipeCollectionList from '$lib/components/RecipeCollectionList.svelte';
  import RecipeSuggestionDialog from '$lib/components/RecipeSuggestionDialog.svelte';
  import ShareDialog from '$lib/components/ShareDialog.svelte';
  import * as features from '$lib/stores/features.svelte';
  import * as recipes from '$lib/stores/recipes.svelte';

  let showCreateDialog = $state(false);
  let newTitle = $state('');
  let creating = $state(false);
  let createError = $state<string | null>(null);
  let showCollectionDialog = $state(false);
  let editingCollection = $state<RecipeCollection | null>(null);
  let sharingCollectionId = $state<number | null>(null);
  let showIngredientSuggestions = $state(false);

  const recipeList = $derived(recipes.getRecipes());
  const loading = $derived(recipes.getRecipesLoading());
  const collections = $derived(recipes.getCollections());
  const featureEnabled = $derived(features.getRecipeFeatureEnabled());
  const sharedRecipesList = $derived(recipes.getSharedRecipes());
  const selectedCollectionId = $derived(recipes.getSelectedCollectionId());
  const collectionItems = $derived(recipes.getCollectionItems());
  const collectionItemsLoading = $derived(recipes.getCollectionItemsLoading());
  const selectedCollection = $derived(collections.find((c) => c.id === selectedCollectionId));
  const displayedRecipes = $derived(selectedCollectionId ? collectionItems : recipeList);

  onMount(async () => {
    if (!featureEnabled) {
      goto('/');
      return;
    }
    await Promise.all([
      recipes.loadRecipes(),
      recipes.loadCollections(),
      recipes.loadSharedRecipes(),
    ]);
  });

  async function handleCreate() {
    if (!newTitle.trim() || creating) return;
    creating = true;
    createError = null;
    try {
      const id = await recipes.createRecipe(newTitle.trim());
      if (id) {
        showCreateDialog = false;
        newTitle = '';
      } else {
        createError = recipes.getLastError() || $_('page.recipes.create_failed');
      }
    } catch (err) {
      createError = err instanceof Error ? err.message : $_('page.recipes.create_failed');
    } finally {
      creating = false;
    }
  }

  function handleCreateKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') handleCreate();
    if (e.key === 'Escape') showCreateDialog = false;
  }

  function difficultyLabel(d: string | null | undefined): string {
    if (!d) return '';
    switch (d) {
      case 'easy':
        return $_('page.recipes.difficulty_easy');
      case 'medium':
        return $_('page.recipes.difficulty_medium');
      case 'hard':
        return $_('page.recipes.difficulty_hard');
      default:
        return d;
    }
  }

  function handleEditCollection(coll: RecipeCollection) {
    editingCollection = coll;
    showCollectionDialog = true;
  }

  async function handleSaveCollection(
    name: string,
    description: string | null,
    color: string | null
  ) {
    if (editingCollection) {
      await recipes.updateCollection(editingCollection.id, name, description, color);
    } else {
      await recipes.createCollection(name, description, color);
    }
  }

  async function handleDeleteCollection(id: number) {
    if (confirm($_('page.recipes.confirm_delete_collection'))) {
      await recipes.deleteCollection(id);
    }
  }

  async function handleDeleteCollectionFromDetail(id: number) {
    if (confirm($_('page.recipes.confirm_delete_collection'))) {
      recipes.clearCollectionFilter();
      await recipes.deleteCollection(id);
    }
  }

  function handleSelectCollection(id: number) {
    recipes.selectCollection(id);
  }

  function handleShareCollection(collectionId: number) {
    sharingCollectionId = collectionId;
  }
</script>

<svelte:head>
  <title>{$_('page.recipes.title')} - xelanote</title>
</svelte:head>

<div class="h-full flex flex-col">
  <!-- Header -->
  <div class="flex items-center justify-between px-6 py-4 border-b border-border shrink-0">
    <h1 class="text-xl font-bold">{$_('page.recipes.title')}</h1>
    <div class="flex items-center gap-2">
      <button
        onclick={() => (showIngredientSuggestions = true)}
        class="flex items-center gap-2 px-3 py-1.5 text-sm rounded-md border border-border hover:bg-accent transition-colors"
      >
        <Sparkles size={16} />
        {$_('page.recipes.suggestions.what_can_i_cook')}
      </button>
      <button
        onclick={() => (showCreateDialog = true)}
        class="flex items-center gap-2 px-3 py-1.5 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
      >
        <Plus size={16} />
        {$_('page.recipes.create')}
      </button>
    </div>
  </div>

  {#if loading}
    <div class="flex items-center justify-center flex-1">
      <Loader2 class="w-8 h-8 animate-spin" />
    </div>
  {:else}
    <div class="flex-1 overflow-y-auto p-6">
      {#if selectedCollectionId && selectedCollection}
        <!-- Collection Detail View -->
        <div class="max-w-5xl">
          <button
            onclick={() => recipes.clearCollectionFilter()}
            class="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground mb-4"
          >
            <ArrowLeft size={14} />
            {$_('page.recipes.all_recipes')}
          </button>
          <div class="flex items-center justify-between mb-3">
            <h2 class="text-lg font-semibold flex items-center gap-2">
              {#if selectedCollection.color}
                <span
                  class="w-3 h-3 rounded-full shrink-0"
                  style="background-color: {selectedCollection.color}"
                ></span>
              {/if}
              {selectedCollection.name}
            </h2>
            <div class="flex items-center gap-1">
              <button
                onclick={() => handleShareCollection(selectedCollection.id)}
                class="p-2 rounded hover:bg-accent text-muted-foreground hover:text-foreground"
                title={$_('sharing.collection_title')}
              >
                <Users size={16} />
              </button>
              <button
                onclick={() => handleEditCollection(selectedCollection)}
                class="p-2 rounded hover:bg-accent text-muted-foreground hover:text-foreground"
                title={$_('common.edit')}
              >
                <Pencil size={16} />
              </button>
              <button
                onclick={() => handleDeleteCollectionFromDetail(selectedCollection.id)}
                class="p-2 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive"
                title={$_('common.delete')}
              >
                <Trash2 size={16} />
              </button>
            </div>
          </div>
          {#if selectedCollection.description}
            <p class="text-sm text-muted-foreground mb-4">{selectedCollection.description}</p>
          {/if}

          {#if collectionItemsLoading}
            <div class="flex items-center justify-center py-12">
              <Loader2 class="w-6 h-6 animate-spin" />
            </div>
          {:else if displayedRecipes.length === 0}
            <div class="text-center py-12 text-muted-foreground">
              <ChefHat class="w-12 h-12 mx-auto mb-3 opacity-50" />
              <p>{$_('page.recipes.collection_empty')}</p>
            </div>
          {:else}
            <div class="space-y-2">
              {#each displayedRecipes as recipe (recipe.id)}
                <button
                  onclick={() => goto(`/note/${recipe.id}`)}
                  class="w-full text-left p-3 rounded-lg border border-border hover:bg-accent transition-colors"
                >
                  <div class="flex items-center gap-2">
                    <span class="font-medium text-sm flex-1 truncate">{recipe.title}</span>
                    {#if recipe.content_encrypted}
                      <Lock size={12} class="text-muted-foreground shrink-0" />
                    {/if}
                  </div>
                  <div class="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                    {#if recipe.servings}
                      <span>{recipe.servings} {$_('page.recipes.servings')}</span>
                    {/if}
                    {#if recipe.prep_time_minutes}
                      <span class="flex items-center gap-0.5">
                        <Clock size={10} />
                        {recipe.prep_time_minutes} min
                      </span>
                    {/if}
                    {#if recipe.difficulty}
                      <span>{difficultyLabel(recipe.difficulty)}</span>
                    {/if}
                  </div>
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6 max-w-5xl">
          <!-- Recipe List -->
          <div>
            <h2 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide mb-3">
              {$_('page.recipes.all_recipes')} ({recipeList.length})
            </h2>

            {#if recipeList.length === 0}
              <div class="text-center py-12 text-muted-foreground">
                <ChefHat class="w-12 h-12 mx-auto mb-3 opacity-50" />
                <p>{$_('page.recipes.no_recipes')}</p>
                <button
                  onclick={() => (showCreateDialog = true)}
                  class="mt-3 text-sm text-primary hover:underline"
                >
                  {$_('page.recipes.create_first')}
                </button>
              </div>
            {:else}
              <div class="space-y-2">
                {#each recipeList as recipe (recipe.id)}
                  <button
                    onclick={() => goto(`/note/${recipe.id}`)}
                    class="w-full text-left p-3 rounded-lg border border-border hover:bg-accent transition-colors"
                  >
                    <div class="flex items-center gap-2">
                      <span class="font-medium text-sm flex-1 truncate">{recipe.title}</span>
                      {#if recipe.content_encrypted}
                        <Lock size={12} class="text-muted-foreground shrink-0" />
                      {/if}
                    </div>
                    <div class="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                      {#if recipe.servings}
                        <span>{recipe.servings} {$_('page.recipes.servings')}</span>
                      {/if}
                      {#if recipe.prep_time_minutes}
                        <span class="flex items-center gap-0.5">
                          <Clock size={10} />
                          {recipe.prep_time_minutes} min
                        </span>
                      {/if}
                      {#if recipe.difficulty}
                        <span>{difficultyLabel(recipe.difficulty)}</span>
                      {/if}
                    </div>
                  </button>
                {/each}
              </div>
            {/if}
          </div>

          <!-- Collections -->
          <div>
            <h2 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide mb-3">
              {$_('page.recipes.collections')}
            </h2>
            <RecipeCollectionList
              {collections}
              onEdit={handleEditCollection}
              onDelete={handleDeleteCollection}
              onCreate={() => {
                editingCollection = null;
                showCollectionDialog = true;
              }}
              onSelect={handleSelectCollection}
              onShare={handleShareCollection}
            />
          </div>
        </div>
      {/if}

      <!-- Shared Recipes -->
      {#if sharedRecipesList.length > 0}
        <div class="max-w-5xl mt-8">
          <h2
            class="text-sm font-semibold text-muted-foreground uppercase tracking-wide mb-3 flex items-center gap-2"
          >
            <UsersIcon size={14} />
            {$_('page.recipes.shared_recipes')} ({sharedRecipesList.length})
          </h2>
          <div class="space-y-2">
            {#each sharedRecipesList as recipe (recipe.id)}
              <button
                onclick={() => goto(`/note/${recipe.id}`)}
                class="w-full text-left p-3 rounded-lg border border-border hover:bg-accent transition-colors"
              >
                <div class="flex items-center gap-2">
                  <span class="font-medium text-sm flex-1 truncate">{recipe.title}</span>
                  <span
                    class="flex items-center gap-1 text-xs px-2 py-0.5 rounded-full {recipe.share_role ===
                    'editor'
                      ? 'bg-primary/10 text-primary'
                      : 'bg-muted text-muted-foreground'}"
                  >
                    {#if recipe.share_role === 'editor'}
                      <Edit size={10} />
                      {$_('sharing.editable')}
                    {:else}
                      <Eye size={10} />
                      {$_('sharing.read_only')}
                    {/if}
                  </span>
                </div>
                <div class="text-xs text-muted-foreground mt-1">
                  {$_('sharing.shared_by', { values: { name: recipe.shared_by } })}
                </div>
              </button>
            {/each}
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>

<!-- Create Recipe Dialog -->
{#if showCreateDialog}
  <div
    class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
    role="button"
    tabindex="0"
    onclick={() => (showCreateDialog = false)}
    onkeydown={(e) => {
      if (e.key === 'Escape' || e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        showCreateDialog = false;
      }
    }}
  >
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div
      class="bg-background border border-border rounded-lg p-6 w-full max-w-md shadow-xl"
      onclick={(e) => e.stopPropagation()}
      role="dialog"
      tabindex="-1"
    >
      <h2 class="text-lg font-semibold mb-4">{$_('page.recipes.create')}</h2>
      {#if createError}
        <div class="text-sm text-destructive mb-3 p-2 bg-destructive/10 rounded">{createError}</div>
      {/if}
      <input
        type="text"
        bind:value={newTitle}
        onkeydown={handleCreateKeydown}
        placeholder={$_('page.recipes.title_placeholder')}
        class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-ring mb-4"
      />
      <div class="flex justify-end gap-2">
        <button
          onclick={() => (showCreateDialog = false)}
          class="px-4 py-2 text-sm hover:bg-accent rounded-md"
        >
          {$_('dialog.cancel')}
        </button>
        <button
          onclick={handleCreate}
          disabled={!newTitle.trim() || creating}
          class="px-4 py-2 text-sm bg-primary text-primary-foreground hover:bg-primary/90 rounded-md disabled:opacity-50"
        >
          {#if creating}
            <Loader2 size={14} class="animate-spin" />
          {:else}
            {$_('common.create')}
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<RecipeCollectionDialog
  open={showCollectionDialog}
  collection={editingCollection}
  onClose={() => (showCollectionDialog = false)}
  onSave={handleSaveCollection}
/>

{#if sharingCollectionId !== null}
  <ShareDialog
    resourceType="collection"
    resourceId={sharingCollectionId}
    onClose={() => (sharingCollectionId = null)}
  />
{/if}

<RecipeSuggestionDialog
  open={showIngredientSuggestions}
  mode="ingredients"
  collectionId={selectedCollectionId}
  onClose={() => (showIngredientSuggestions = false)}
/>
