<script lang="ts">
  import { ChefHat, Clock, Loader2, Lock, Plus, Sparkles, Upload } from 'lucide-svelte';
  import { ArrowLeft, Edit, Eye, Pencil, Trash2, Users, Users as UsersIcon } from 'lucide-svelte';
  import type { ComponentType } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import type { RecipeCollection } from '$lib/api';
  import { deleteNote } from '$lib/api/notes';
  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import RecipeCollectionDialog from '$lib/components/RecipeCollectionDialog.svelte';
  import RecipeCollectionList from '$lib/components/RecipeCollectionList.svelte';
  import PageHeader from '$lib/components/ui/PageHeader.svelte';
  import * as dialog from '$lib/stores/dialog.svelte';
  import * as features from '$lib/stores/features.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as recipes from '$lib/stores/recipes.svelte';
  import * as toast from '$lib/stores/toast.svelte';
  import * as trash from '$lib/stores/trash.svelte';
  import * as tree from '$lib/stores/tree.svelte';
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
  const featureLoaded = $derived(features.getRecipeFeatureLoaded());
  const sharedRecipesList = $derived(recipes.getSharedRecipes());
  const selectedCollectionId = $derived(recipes.getSelectedCollectionId());
  const collectionItems = $derived(recipes.getCollectionItems());
  const collectionItemsLoading = $derived(recipes.getCollectionItemsLoading());
  const selectedCollection = $derived(collections.find((c) => c.id === selectedCollectionId));
  const displayedRecipes = $derived(selectedCollectionId ? collectionItems : recipeList);

  let dataLoaded = false;

  // Redirect when feature is confirmed disabled (wait for load to complete)
  $effect(() => {
    if (featureLoaded && !featureEnabled) {
      goto('/');
    }
  });

  // Load data reactively once feature is confirmed enabled
  $effect(() => {
    if (featureLoaded && featureEnabled && !dataLoaded) {
      dataLoaded = true;
      Promise.all([recipes.loadRecipes(), recipes.loadCollections(), recipes.loadSharedRecipes()]);
    }
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
    const confirmed = await dialog.confirm({
      title: $_('dialog.confirm_title'),
      message: $_('page.recipes.confirm_delete_collection'),
      confirmText: $_('common.delete'),
      cancelText: $_('dialog.cancel'),
      variant: 'danger',
    });
    if (confirmed) {
      await recipes.deleteCollection(id);
    }
  }

  async function handleDeleteCollectionFromDetail(id: number) {
    const confirmed = await dialog.confirm({
      title: $_('dialog.confirm_title'),
      message: $_('page.recipes.confirm_delete_collection'),
      confirmText: $_('common.delete'),
      cancelText: $_('dialog.cancel'),
      variant: 'danger',
    });
    if (confirmed) {
      recipes.clearCollectionFilter();
      await recipes.deleteCollection(id);
    }
  }

  function handleSelectCollection(id: number) {
    recipes.selectCollection(id);
  }

  async function handleDeleteRecipe(id: string, event: MouseEvent) {
    event.stopPropagation();
    const confirmed = await dialog.confirm({
      title: $_('dialog.confirm_title'),
      message: $_('dialog.delete_note_confirm'),
      confirmText: $_('common.delete'),
      cancelText: $_('dialog.cancel'),
      variant: 'danger',
    });
    if (!confirmed) return;
    try {
      await deleteNote(id);
      trash.incrementTrashCount();
      await Promise.all([recipes.loadRecipes(), notes.loadNotes(), tree.loadTree()]);
      if (selectedCollectionId) {
        await recipes.selectCollection(selectedCollectionId);
      }
      toast.success($_('component.editor.note_trashed'));
    } catch {
      toast.error($_('page.recipes.delete_failed'));
    }
  }

  async function handleShareCollection(collectionId: number) {
    await ensureShareDialog();
    sharingCollectionId = collectionId;
  }
</script>

<svelte:head>
  <title>{$_('page.recipes.title')} - xelanote</title>
</svelte:head>

<div data-body-scroll class="h-full flex flex-col">
  <!-- Header -->
  <PageHeader
    title={$_('page.recipes.title')}
    class="sticky top-0 z-10 shrink-0 px-4 py-3 sm:px-6 sm:py-4"
    titleClass="min-w-0 truncate text-xl font-bold"
  >
    {#snippet leading()}
      <div
        class="grid grid-cols-[2.5rem_minmax(0,1fr)] items-center gap-2 sm:flex sm:items-center sm:gap-3"
      >
        <MobileSidebarInlineToggle />
      </div>
    {/snippet}
    {#snippet actions()}
      <div class="grid grid-cols-2 gap-2 min-w-0 sm:flex sm:items-center sm:min-w-auto">
        <div
          class="col-span-2 flex min-w-0 gap-2 overflow-x-auto pb-1 -mb-1 sm:col-span-1 sm:mb-0 sm:pb-0 sm:overflow-visible sm:flex-none scrollbar-none"
        >
          <button
            onclick={async () => {
              await ensureSuggestionDialog();
              showIngredientSuggestions = true;
            }}
            class="ui-button ui-button-secondary recipe-header-secondary-action flex flex-1 items-center justify-center text-sm sm:flex-none sm:justify-start"
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
            class="ui-button ui-button-secondary recipe-header-secondary-action flex flex-1 items-center justify-center text-sm sm:flex-none sm:justify-start"
          >
            <Upload size={16} />
            <span class="leading-tight whitespace-nowrap">{$_('page.recipes.import.button')}</span>
          </button>
        </div>
        <button
          onclick={() => (showCreateDialog = true)}
          class="ui-button ui-button-primary recipe-header-primary-action col-span-2 flex w-full items-center justify-center text-sm sm:col-span-1 sm:w-auto sm:justify-start"
        >
          <Plus size={16} />
          <span class="leading-tight whitespace-nowrap">{$_('page.recipes.create')}</span>
        </button>
      </div>
    {/snippet}
  </PageHeader>

  {#if !featureLoaded || loading}
    <div class="flex items-center justify-center flex-1">
      <Loader2 class="w-8 h-8 animate-spin" />
    </div>
  {:else}
    <div class="flex-1 overflow-y-auto overscroll-contain p-4 pb-24 sm:p-6 sm:pb-6">
      {#if selectedCollectionId && selectedCollection}
        <!-- Collection Detail View -->
        <div class="max-w-5xl">
          <button
            onclick={() => recipes.clearCollectionFilter()}
            class="ui-button ui-button-ghost mb-4 gap-1 px-2 py-1 text-sm"
          >
            <ArrowLeft size={14} />
            {$_('page.recipes.all_recipes')}
          </button>
          <div class="ui-panel p-4 sm:p-5">
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
              <div class="ui-panel-soft flex items-center gap-1 self-start p-1">
                <button
                  onclick={() => handleShareCollection(selectedCollection.id)}
                  class="ui-icon-button ui-icon-button-sm"
                  title={$_('sharing.collection_title')}
                >
                  <Users size={16} />
                </button>
                <button
                  onclick={() => handleEditCollection(selectedCollection)}
                  class="ui-icon-button ui-icon-button-sm"
                  title={$_('common.edit')}
                >
                  <Pencil size={16} />
                </button>
                <button
                  onclick={() => handleDeleteCollectionFromDetail(selectedCollection.id)}
                  class="ui-icon-button ui-icon-button-sm hover:bg-destructive/10 hover:text-destructive"
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
              <div class="space-y-2.5">
                {#each displayedRecipes as recipe (recipe.id)}
                  <div
                    class="ui-list-item group flex w-full items-center p-3 text-left hover:-translate-y-px"
                  >
                    <button
                      onclick={() => goto(`/note/${recipe.id}`)}
                      class="flex-1 min-w-0 text-left"
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
                    <button
                      onclick={(e) => handleDeleteRecipe(recipe.id, e)}
                      class="shrink-0 p-1.5 rounded opacity-100 sm:opacity-0 sm:group-hover:opacity-100 focus:opacity-100 hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-opacity"
                      aria-label={$_('common.delete')}
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        </div>
      {:else}
        <div class="grid grid-cols-1 gap-4 md:gap-6 md:grid-cols-2 max-w-5xl">
          <!-- Recipe List -->
          <section class="ui-panel p-4 sm:p-5">
            <h2 class="ui-kicker mb-3">
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
              <div class="space-y-2.5">
                {#each recipeList as recipe, i (recipe.id)}
                  <div
                    class="ui-list-item group flex w-full items-center p-3 text-left hover:-translate-y-px animate-stagger-item"
                    style="animation-delay: {i * 40}ms"
                  >
                    <button
                      onclick={() => goto(`/note/${recipe.id}`)}
                      class="flex-1 min-w-0 text-left"
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
                    <button
                      onclick={(e) => handleDeleteRecipe(recipe.id, e)}
                      class="shrink-0 p-1.5 rounded opacity-100 sm:opacity-0 sm:group-hover:opacity-100 focus:opacity-100 hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-opacity"
                      aria-label={$_('common.delete')}
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                {/each}
              </div>
            {/if}
          </section>

          <!-- Collections -->
          <section class="ui-panel p-4 sm:p-5">
            <h2 class="ui-kicker mb-3">
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
        <div class="ui-panel max-w-5xl mt-6 sm:mt-8 p-4 sm:p-5">
          <h2 class="ui-kicker mb-3 flex items-center gap-2">
            <UsersIcon size={14} />
            {$_('page.recipes.shared_recipes')} ({sharedRecipesList.length})
          </h2>
          <div class="space-y-2.5">
            {#each sharedRecipesList as recipe (recipe.id)}
              <button
                onclick={() => goto(`/note/${recipe.id}`)}
                class="ui-list-item w-full p-3 text-left hover:-translate-y-px"
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
      class="ui-panel w-full max-w-md p-6"
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
        class="ui-input mb-4"
      />
      <div class="flex justify-end gap-2">
        <button
          onclick={() => (showCreateDialog = false)}
          class="ui-button ui-button-secondary text-sm"
        >
          {$_('dialog.cancel')}
        </button>
        <button
          onclick={handleCreate}
          disabled={!newTitle.trim() || creating}
          class="ui-button ui-button-primary text-sm"
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

<style>
  @media (max-width: 639px) {
    .recipe-header-secondary-action {
      border-color: color-mix(in oklch, var(--color-border), transparent 60%);
      background: color-mix(in oklch, var(--color-background), transparent 78%);
      box-shadow: none;
    }

    .recipe-header-secondary-action:hover {
      background: color-mix(in oklch, var(--color-accent), transparent 82%);
    }

    .recipe-header-primary-action {
      box-shadow: inset 0 1px 0 color-mix(in oklch, white, transparent 88%);
    }
  }
</style>
