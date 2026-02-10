<script lang="ts">
  import { Loader2, Lock, Menu, Plus, Sparkles,X } from 'lucide-svelte';
  import type { Component } from 'svelte';
  import { onMount, untrack } from 'svelte';
  import { _ } from 'svelte-i18n';

  import type { RecipeCollection,RecipeIngredient } from '$lib/api';
  import * as notes from '$lib/stores/notes.svelte';
  import * as recipes from '$lib/stores/recipes.svelte';
  import * as ui from '$lib/stores/ui.svelte';

  import AddToCollectionDialog from './AddToCollectionDialog.svelte';
  import RecipeCollectionDialog from './RecipeCollectionDialog.svelte';
  import RecipeImageGallery from './RecipeImageGallery.svelte';
  import RecipeIngredientEditor from './RecipeIngredientEditor.svelte';
  import RecipeMetadataForm from './RecipeMetadataForm.svelte';
  import RecipePreview from './RecipePreview.svelte';
  import RecipeScaleControl from './RecipeScaleControl.svelte';
  import RecipeSuggestionDialog from './RecipeSuggestionDialog.svelte';

  interface Props {
    noteId: string;
  }

  const { noteId }: Props = $props();

  let activeTab = $state<'ingredients' | 'instructions' | 'preview'>('ingredients');
  let EditorComponent = $state<Component<Record<string, unknown>> | null>(null);
  let editorLoading = $state(true);

  // Collection dialogs
  let showCollectionDialog = $state(false);
  let editingCollection = $state<RecipeCollection | null>(null);
  let showAddToCollectionDialog = $state(false);

  // Suggestion dialog
  let showSuggestionDialog = $state(false);

  // Local ingredients state for batched updates
  let localIngredients = $state<RecipeIngredient[]>([]);
  let ingredientsDirty = $state(false);
  let ingredientSaveTimeout: ReturnType<typeof setTimeout> | null = null;

  const currentRecipe = $derived(recipes.getCurrentRecipe());
  const targetServings = $derived(recipes.getTargetServings());
  const scaledIngredients = $derived(recipes.getScaledIngredients());
  const loading = $derived(recipes.getRecipeDetailLoading());
  const saving = $derived(recipes.getSaving());
  const lastError = $derived(recipes.getLastError());
  const collections = $derived(recipes.getCollections());
  const currentNote = $derived(notes.getCurrentNote());

  // Determine access level
  const isEncrypted = $derived(currentRecipe?.encrypted ?? false);
  const isReadonly = $derived(false); // TODO: Check share role for viewer

  // Sync local ingredients from server state (initial load + remote updates).
  // Uses untrack on ingredientsDirty so it doesn't become a dependency —
  // this prevents the effect from re-running when ingredientsDirty changes,
  // which would overwrite in-progress edits after auto-save completes.
  $effect(() => {
    if (currentRecipe?.ingredients) {
      const dirty = untrack(() => ingredientsDirty);
      if (!dirty) {
        localIngredients = [...currentRecipe.ingredients];
        ingredientsDirty = false;
      }
    }
  });

  // Current collection IDs for the recipe
  const currentCollectionIds = $derived(currentRecipe?.collections?.map((c) => c.id) ?? []);

  onMount(async () => {
    // Load recipe detail
    await recipes.loadRecipeDetail(noteId);
    await recipes.loadCollections();

    // Load markdown editor for instructions tab
    try {
      const module = await import('$lib/components/Editor.svelte');
      EditorComponent = module.default;
    } catch (err) {
      console.error('Failed to load editor:', err);
    }
    editorLoading = false;
  });

  function handleServingsChange(servings: number) {
    recipes.setTargetServings(servings);
  }

  async function handleMetadataUpdate(data: {
    servings: number;
    prep_time_minutes?: number | null;
    cook_time_minutes?: number | null;
    source_url?: string | null;
    difficulty?: string | null;
  }) {
    if (!noteId || isReadonly) return;
    await recipes.updateMetadata(noteId, data);
  }

  function handleIngredientsChange(updated: RecipeIngredient[]) {
    localIngredients = updated;
    ingredientsDirty = true;
    // Debounce save
    if (ingredientSaveTimeout) clearTimeout(ingredientSaveTimeout);
    ingredientSaveTimeout = setTimeout(saveIngredients, 2000);
  }

  async function saveIngredients() {
    if (!ingredientsDirty || isReadonly || !noteId) return;
    // Filter out empty-name ingredients
    const valid = localIngredients.filter((ing) => ing.name.trim());
    const success = await recipes.updateIngredients(noteId, valid);
    if (success) {
      ingredientsDirty = false;
    }
  }

  // Save ingredients before tab switch
  async function switchTab(tab: 'ingredients' | 'instructions' | 'preview') {
    if (ingredientsDirty && activeTab === 'ingredients') {
      if (ingredientSaveTimeout) clearTimeout(ingredientSaveTimeout);
      await saveIngredients();
    }
    activeTab = tab;
  }

  // Collection handlers
  function handleCreateCollection() {
    editingCollection = null;
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

  async function handleAddToCollection(collectionId: number) {
    await recipes.addToCollection(collectionId, noteId);
  }

  async function handleRemoveFromCollection(collectionId: number) {
    await recipes.removeFromCollection(collectionId, noteId);
  }
</script>

{#if loading}
  <div class="flex items-center justify-center h-screen-safe">
    <div class="text-center">
      <Loader2 class="w-8 h-8 animate-spin mx-auto mb-2" />
      <p class="text-sm text-muted-foreground">{$_('page.recipes.loading')}</p>
    </div>
  </div>
{:else if isEncrypted}
  <div class="flex items-center justify-center h-screen-safe">
    <div class="text-center space-y-3">
      <Lock class="w-12 h-12 mx-auto text-muted-foreground" />
      <p class="text-muted-foreground">{$_('page.recipes.encrypted_message')}</p>
    </div>
  </div>
{:else if currentRecipe}
  <div class="recipe-editor h-full flex flex-col">
    <!-- Error Banner -->
    {#if lastError}
      <div class="px-4 py-2 bg-destructive/10 text-destructive text-sm flex items-center gap-2">
        <span class="flex-1">{lastError}</span>
        <button onclick={() => recipes.loadRecipeDetail(noteId)} class="underline text-xs">
          {$_('common.retry')}
        </button>
      </div>
    {/if}

    <!-- Tabs -->
    <div class="flex items-center border-b border-border px-4 shrink-0">
      <!-- Sidebar toggle on mobile (MobileHeader is hidden on /note/ routes) -->
      {#if ui.getIsMobile()}
        <button
          type="button"
          onclick={() => ui.setSidebarOpen(true)}
          class="p-2 -ml-2 mr-1 rounded-md hover:bg-accent toolbar-btn"
          aria-label="Menü öffnen"
        >
          <Menu size={16} />
        </button>
      {/if}
      <div class="flex gap-0 -mb-px">
        <button
          onclick={() => switchTab('ingredients')}
          class="tab-button"
          class:active={activeTab === 'ingredients'}
        >
          {$_('page.recipes.tab_ingredients')}
        </button>
        <button
          onclick={() => switchTab('instructions')}
          class="tab-button"
          class:active={activeTab === 'instructions'}
        >
          {$_('page.recipes.tab_instructions')}
        </button>
        <button
          onclick={() => switchTab('preview')}
          class="tab-button"
          class:active={activeTab === 'preview'}
        >
          {$_('page.recipes.tab_preview')}
        </button>
      </div>

      <div class="flex-1"></div>

      <!-- Find similar button -->
      {#if !isEncrypted && !isReadonly}
        <button
          onclick={() => (showSuggestionDialog = true)}
          class="flex items-center gap-1 px-2 py-1 text-xs rounded-md text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
          title={$_('page.recipes.suggestions.find_similar')}
        >
          <Sparkles size={14} />
          <span class="hidden sm:inline">{$_('page.recipes.suggestions.find_similar')}</span>
        </button>
      {/if}

      <!-- Saving indicator -->
      {#if saving}
        <div class="flex items-center gap-1 text-xs text-muted-foreground">
          <Loader2 size={12} class="animate-spin" />
          {$_('page.recipes.saving')}
        </div>
      {/if}
      {#if ingredientsDirty}
        <span class="text-xs text-amber-500 ml-2">{$_('page.recipes.unsaved')}</span>
      {/if}
    </div>

    <!-- Tab Content -->
    <div class="flex-1 overflow-y-auto">
      {#if activeTab === 'ingredients'}
        <div class="p-4 space-y-4">
          <!-- Metadata -->
          <RecipeMetadataForm
            metadata={currentRecipe.metadata}
            readonly={isReadonly}
            onupdate={handleMetadataUpdate}
          />

          <!-- Images -->
          <RecipeImageGallery images={currentRecipe.images ?? []} {noteId} readonly={isReadonly} />

          <!-- Scale Control -->
          <RecipeScaleControl
            servings={targetServings}
            baseServings={currentRecipe.metadata?.servings ?? 4}
            onchange={handleServingsChange}
            disabled={isReadonly}
          />

          <!-- Ingredients -->
          <div>
            <h3 class="text-sm font-semibold mb-2">{$_('page.recipes.ingredients')}</h3>
            <RecipeIngredientEditor
              ingredients={localIngredients}
              scaledIngredients={targetServings !== (currentRecipe.metadata?.servings ?? 4)
                ? scaledIngredients
                : undefined}
              readonly={isReadonly}
              onupdate={handleIngredientsChange}
            />
          </div>

          <!-- Collections -->
          <div>
            <div class="flex items-center justify-between mb-2">
              <h3 class="text-sm font-semibold">{$_('page.recipes.collections')}</h3>
              <button
                onclick={() => (showAddToCollectionDialog = true)}
                class="text-xs text-muted-foreground hover:text-foreground flex items-center gap-1"
              >
                <Plus size={12} />
                {$_('page.recipes.add_to_collection')}
              </button>
            </div>
            {#if currentRecipe.collections.length > 0}
              <div class="flex flex-wrap gap-1">
                {#each currentRecipe.collections as coll (coll.id)}
                  <span
                    class="inline-flex items-center gap-1 px-2 py-0.5 text-xs rounded-full bg-accent"
                  >
                    {#if coll.color}
                      <span class="w-2 h-2 rounded-full" style="background-color: {coll.color}"
                      ></span>
                    {/if}
                    {coll.name}
                    {#if !isReadonly}
                      <button
                        onclick={() => handleRemoveFromCollection(coll.id)}
                        class="hover:text-destructive"
                      >
                        <X size={10} />
                      </button>
                    {/if}
                  </span>
                {/each}
              </div>
            {:else}
              <p class="text-xs text-muted-foreground italic">
                {$_('page.recipes.not_in_collection')}
              </p>
            {/if}
          </div>
        </div>
      {:else if activeTab === 'instructions'}
        {#if editorLoading}
          <div class="flex items-center justify-center py-12">
            <Loader2 class="w-6 h-6 animate-spin" />
          </div>
        {:else if EditorComponent}
          <EditorComponent {noteId} />
        {/if}
      {:else if activeTab === 'preview'}
        <div class="p-4">
          <RecipePreview
            title={currentNote?.title ?? ''}
            metadata={currentRecipe.metadata}
            images={currentRecipe.images ?? []}
            {scaledIngredients}
            content={currentNote?.content ?? ''}
            {targetServings}
            baseServings={currentRecipe.metadata?.servings ?? 4}
            onServingsChange={(s) => recipes.setTargetServings(s)}
          />
        </div>
      {/if}
    </div>
  </div>

  <!-- Dialogs -->
  <RecipeCollectionDialog
    open={showCollectionDialog}
    collection={editingCollection}
    onClose={() => (showCollectionDialog = false)}
    onSave={handleSaveCollection}
  />

  <AddToCollectionDialog
    open={showAddToCollectionDialog}
    {collections}
    {currentCollectionIds}
    onClose={() => (showAddToCollectionDialog = false)}
    onAdd={handleAddToCollection}
    onCreateNew={handleCreateCollection}
  />

  <RecipeSuggestionDialog
    open={showSuggestionDialog}
    {noteId}
    mode="similar"
    onClose={() => (showSuggestionDialog = false)}
  />
{/if}

<style>
  .tab-button {
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    color: var(--color-text-muted, hsl(var(--muted-foreground)));
    border-bottom: 2px solid transparent;
    transition: all 0.15s ease;
  }

  .tab-button:hover {
    color: var(--color-text, hsl(var(--foreground)));
  }

  .tab-button.active {
    color: var(--color-text, hsl(var(--foreground)));
    border-bottom-color: hsl(var(--primary));
  }
</style>
