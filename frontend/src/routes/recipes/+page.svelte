<script lang="ts">
  import { ChefHat, Clock, Loader2, Lock, Plus, Sparkles, Upload } from 'lucide-svelte';
  import { ArrowLeft, Edit, Eye, Pencil, Trash2, Users, Users as UsersIcon } from 'lucide-svelte';
  import type { ComponentType } from 'svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import type { RecipeCollection } from '$lib/api';
  import RecipeCollectionDialog from '$lib/components/RecipeCollectionDialog.svelte';
  import RecipeCollectionList from '$lib/components/RecipeCollectionList.svelte';
  import * as features from '$lib/stores/features.svelte';
  import * as recipes from '$lib/stores/recipes.svelte';
  import { loadSvelteComponentFromModule } from '$lib/utils/lazy-component';

  let showCreateDialog = $state(false);
  let newTitle = $state('');
  let creating = $state(false);
  let createError = $state<string | null>(null);
  let showCollectionDialog = $state(false);
  let editingCollection = $state<RecipeCollection | null>(null);
  let sharingCollectionId = $state<number | null>(null);
  let showIngredientSuggestions = $state(false);
  let showImportDialog = $state(false);

  // Lazy-loaded dialog components (only imported when user triggers them)
  let SuggestionDialog = $state<ComponentType | null>(null);
  let ImportDialog = $state<ComponentType | null>(null);
  let ShareDialogComponent = $state<ComponentType | null>(null);

  async function ensureSuggestionDialog() {
    if (!SuggestionDialog) {
      const mod = await import('$lib/components/RecipeSuggestionDialog.svelte');
      SuggestionDialog = loadSvelteComponentFromModule(mod, 'RecipeSuggestionDialog');
    }
  }

  async function ensureImportDialog() {
    if (!ImportDialog) {
      const mod = await import('$lib/components/RecipeImportDialog.svelte');
      ImportDialog = loadSvelteComponentFromModule(mod, 'RecipeImportDialog');
    }
  }

  async function ensureShareDialog() {
    if (!ShareDialogComponent) {
      const mod = await import('$lib/components/ShareDialog.svelte');
      ShareDialogComponent = loadSvelteComponentFromModule(mod, 'ShareDialog');
    }
  }

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

  async function handleShareCollection(collectionId: number) {
    await ensureShareDialog();
    sharingCollectionId = collectionId;
  }
</script>

<svelte:head>
  <title>{$_('page.recipes.title')} - xelanote</title>
</svelte:head>

<div class="h-full flex flex-col">
  <!-- Header -->
  <div class="border-b border-border shrink-0 px-4 py-3 sm:px-6 sm:py-4">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <h1
        class="text-xl font-bold pl-[calc(var(--safe-area-inset-left)+2.75rem)] mt-2 sm:mt-0 sm:pl-0"
      >
        {$_('page.recipes.title')}
      </h1>
      <div class="grid grid-cols-2 gap-2 min-w-0 sm:flex sm:items-center sm:min-w-auto">
        <div
          class="col-span-2 flex min-w-0 gap-2 overflow-x-auto pb-1 -mb-1 sm:col-span-1 sm:mb-0 sm:pb-0 sm:overflow-visible sm:flex-none scrollbar-none"
        >
          <button
            onclick={async () => {
              await ensureSuggestionDialog();
              showIngredientSuggestions = true;
            }}
            class="flex flex-1 items-center justify-center gap-2 rounded-md border border-border px-3 py-2 text-sm transition-colors hover:bg-accent sm:flex-none sm:justify-start sm:py-1.5"
          >
            <Sparkles size={16} />
            <span class="leading-tight whitespace-nowrap">
              {$_('page.recipes.suggestions.what_can_i_cook')}
            </span>
          </button>
          <button
            onclick={async () => {
              await ensureImportDialog();
              showImportDialog = true;
            }}
            class="flex flex-1 items-center justify-center gap-2 rounded-md border border-border px-3 py-2 text-sm transition-colors hover:bg-accent sm:flex-none sm:justify-start sm:py-1.5"
          >
            <Upload size={16} />
            <span class="leading-tight whitespace-nowrap">{$_('page.recipes.import.button')}</span>
          </button>
        </div>
        <button
          onclick={() => (showCreateDialog = true)}
          class="col-span-2 flex w-full items-center justify-center gap-2 rounded-md bg-primary px-3 py-2 text-sm text-primary-foreground hover:bg-primary/90 sm:col-span-1 sm:w-auto sm:justify-start sm:py-1.5"
        >
          <Plus size={16} />
          <span class="leading-tight whitespace-nowrap">{$_('page.recipes.create')}</span>
        </button>
      </div>
    </div>
  </div>

  {#if loading}
    <div class="flex items-center justify-center flex-1">
      <Loader2 class="w-8 h-8 animate-spin" />
    </div>
  {:else}
    <div class="flex-1 overflow-y-auto p-4 pb-24 sm:p-6 sm:pb-6">
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
          <div class="rounded-xl border border-border bg-background/50 p-4 sm:p-5">
            <div class="mb-3 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div class="min-w-0">
                <h2 class="text-lg font-semibold flex items-center gap-2">
                  {#if selectedCollection.color}
                    <span
                      class="w-3 h-3 rounded-full shrink-0"
                      style="background-color: {selectedCollection.color}"
                    ></span>
                  {/if}
                  <span class="truncate">{selectedCollection.name}</span>
                </h2>
                {#if selectedCollection.description}
                  <p class="text-sm text-muted-foreground mt-2">{selectedCollection.description}</p>
                {/if}
              </div>
              <div
                class="flex items-center gap-1 self-start rounded-lg border border-border bg-background/40 p-1"
              >
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
                    class="w-full text-left p-3 rounded-lg border border-border bg-background/40 hover:bg-accent transition-colors"
                  >
                    <div class="flex items-center gap-2">
                      <span class="font-medium text-sm flex-1 truncate">{recipe.title}</span>
                      {#if recipe.content_encrypted}
                        <Lock size={12} class="text-muted-foreground shrink-0" />
                      {/if}
                    </div>
                    <div
                      class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground"
                    >
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
        </div>
      {:else}
        <div class="grid grid-cols-1 gap-4 md:gap-6 md:grid-cols-2 max-w-5xl">
          <!-- Recipe List -->
          <section class="rounded-xl border border-border bg-background/50 p-4 sm:p-5">
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
                    class="w-full text-left p-3 rounded-lg border border-border bg-background/40 hover:bg-accent transition-colors"
                  >
                    <div class="flex items-center gap-2">
                      <span class="font-medium text-sm flex-1 truncate">{recipe.title}</span>
                      {#if recipe.content_encrypted}
                        <Lock size={12} class="text-muted-foreground shrink-0" />
                      {/if}
                    </div>
                    <div
                      class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground"
                    >
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
          </section>

          <!-- Collections -->
          <section class="rounded-xl border border-border bg-background/50 p-4 sm:p-5">
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
          </section>
        </div>
      {/if}

      <!-- Shared Recipes -->
      {#if sharedRecipesList.length > 0}
        <div
          class="max-w-5xl mt-6 sm:mt-8 rounded-xl border border-border bg-background/50 p-4 sm:p-5"
        >
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
                class="w-full text-left p-3 rounded-lg border border-border bg-background/40 hover:bg-accent transition-colors"
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

{#if sharingCollectionId !== null && ShareDialogComponent}
  <ShareDialogComponent
    resourceType="collection"
    resourceId={sharingCollectionId}
    onClose={() => (sharingCollectionId = null)}
  />
{/if}

{#if SuggestionDialog}
  <SuggestionDialog
    open={showIngredientSuggestions}
    mode="ingredients"
    collectionId={selectedCollectionId}
    onClose={() => (showIngredientSuggestions = false)}
  />
{/if}

{#if ImportDialog}
  <ImportDialog open={showImportDialog} onClose={() => (showImportDialog = false)} />
{/if}
