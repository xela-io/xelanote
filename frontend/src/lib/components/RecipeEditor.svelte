<script lang="ts">
  import { Loader2, Lock, MoreVertical, Plus, Sparkles, X } from 'lucide-svelte';
  import type { ComponentType } from 'svelte';
  import { onMount, untrack } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { bottomsheet } from '$lib/actions/bottomsheet';
  import type { RecipeCollection, RecipeIngredient } from '$lib/api';
  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as recipes from '$lib/stores/recipes.svelte';
  import { loadSvelteComponentFromModule } from '$lib/utils/lazy-component';
  import { formatRelativeTime } from '$lib/utils/time';

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
  let EditorComponent = $state<ComponentType | null>(null);
  let editorLoading = $state(true);

  // Collection dialogs
  let showCollectionDialog = $state(false);
  let editingCollection = $state<RecipeCollection | null>(null);
  let showAddToCollectionDialog = $state(false);

  // Suggestion dialog
  let showSuggestionDialog = $state(false);
  let showMobileActionsSheet = $state(false);

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
  const isReadonly = $derived(
    currentRecipe?.note?.share_role === 'viewer' || currentNote?.share_role === 'viewer'
  );

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

  // Load recipe data reactively when noteId changes (not in onMount,
  // because the component can stay mounted when noteId changes via routing).
  // Cleanup clears stale recipe state on noteId change or unmount.
  $effect(() => {
    const id = noteId;
    recipes.loadRecipeDetail(id);
    recipes.loadCollections();

    return () => {
      recipes.clearCurrentRecipe();
    };
  });

  // Load markdown editor component (only once)
  onMount(async () => {
    try {
      const module = await import('$lib/components/Editor.svelte');
      EditorComponent = loadSvelteComponentFromModule(module, 'RecipeEditorInstructions');
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
    const valid = localIngredients
      .filter((ing) => ing.name.trim())
      .map((ing) => ({ ...ing, scalable: true }));
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

  function closeMobileActionsSheet() {
    showMobileActionsSheet = false;
  }

  function handleMobileActionsKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      closeMobileActionsSheet();
    }
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
    <div
      class="ui-page-header flex items-center gap-2 px-2 py-2 ui-mobile-topbar sm:px-4 sm:py-3 shrink-0"
    >
      <div class="ui-mobile-topbar-leading">
        <MobileSidebarInlineToggle />
      </div>
      <div class="ui-mobile-topbar-scroll">
        <div class="ui-mobile-topbar-nowrap">
          <div class="recipe-tab-slider" role="tablist">
            <div
              class="recipe-tab-slider-indicator"
              style="transform: translateX({activeTab === 'ingredients'
                ? 0
                : activeTab === 'instructions'
                  ? 100
                  : 200}%)"
            ></div>
            <button
              onclick={() => switchTab('ingredients')}
              class="recipe-tab-slider-tab"
              class:active={activeTab === 'ingredients'}
              role="tab"
              aria-selected={activeTab === 'ingredients'}
            >
              {$_('page.recipes.tab_ingredients')}
            </button>
            <button
              onclick={() => switchTab('instructions')}
              class="recipe-tab-slider-tab"
              class:active={activeTab === 'instructions'}
              role="tab"
              aria-selected={activeTab === 'instructions'}
            >
              {$_('page.recipes.tab_instructions')}
            </button>
            <button
              onclick={() => switchTab('preview')}
              class="recipe-tab-slider-tab"
              class:active={activeTab === 'preview'}
              role="tab"
              aria-selected={activeTab === 'preview'}
            >
              {$_('page.recipes.tab_preview')}
            </button>
          </div>
        </div>
      </div>

      <div class="hidden sm:block flex-1"></div>

      <!-- Find similar button -->
      {#if !isEncrypted && !isReadonly}
        <button
          onclick={() => (showSuggestionDialog = true)}
          class="ui-button ui-button-secondary px-2 py-1 text-xs ui-mobile-hide-overflow-actions"
          title={$_('page.recipes.suggestions.find_similar')}
        >
          <Sparkles size={14} />
          <span class="hidden sm:inline">{$_('page.recipes.suggestions.find_similar')}</span>
        </button>
      {/if}

      <!-- Last updated + Saving indicator -->
      <div class="flex items-center gap-2 ui-mobile-hide-overflow-actions">
        {#if saving}
          <div class="flex items-center gap-1 text-xs text-muted-foreground">
            <Loader2 size={12} class="animate-spin" />
            {$_('page.recipes.saving')}
          </div>
        {:else if ingredientsDirty}
          <span class="text-xs text-amber-500">{$_('page.recipes.unsaved')}</span>
        {:else if currentRecipe?.note?.updated_at}
          <span
            class="text-xs text-muted-foreground hidden sm:inline"
            title={new Date(currentRecipe.note.updated_at).toLocaleString()}
          >
            {$_('component.editor.last_updated', {
              values: { date: formatRelativeTime(currentRecipe.note.updated_at, $_) },
            })}
          </span>
        {/if}
      </div>

      <div class="ui-mobile-topbar-actions sm:hidden">
        <button
          type="button"
          onclick={() => (showMobileActionsSheet = true)}
          class="ui-mobile-topbar-icon ui-mobile-topbar-icon--ghost"
          aria-label={$_('nav.more')}
          title={$_('nav.more')}
          aria-haspopup="menu"
          aria-expanded={showMobileActionsSheet}
        >
          <MoreVertical size={20} />
        </button>
      </div>
    </div>

    <!-- Tab Content -->
    <div class="flex-1 overflow-y-auto">
      {#if activeTab === 'ingredients'}
        <div class="max-w-4xl p-4 pb-28 sm:p-5 sm:pb-5">
          <div class="ui-panel ui-panel-mobile-flat recipe-pane-shell space-y-4 sm:space-y-5">
            <div class="recipe-top-grid">
              <!-- Metadata -->
              <section class="ui-panel-soft recipe-section-card">
                <div class="ui-kicker recipe-section-header">
                  {$_('page.recipes.tab_ingredients')}
                </div>
                <RecipeMetadataForm
                  metadata={currentRecipe.metadata}
                  readonly={isReadonly}
                  onupdate={handleMetadataUpdate}
                />
              </section>

              <section class="ui-panel-soft recipe-section-card recipe-image-card space-y-4">
                <!-- Images -->
                <div>
                  <div class="ui-kicker recipe-section-header">{$_('page.recipes.images')}</div>
                  <RecipeImageGallery
                    images={currentRecipe.images ?? []}
                    {noteId}
                    readonly={isReadonly}
                  />
                </div>

                <!-- Scale Control -->
                <div class="recipe-subsection">
                  <div class="ui-kicker recipe-section-header recipe-section-header-plain">
                    {$_('page.recipes.servings')}
                  </div>
                  <RecipeScaleControl
                    servings={targetServings}
                    baseServings={currentRecipe.metadata?.servings ?? 4}
                    onchange={handleServingsChange}
                    disabled={isReadonly}
                  />
                </div>
              </section>
            </div>

            <!-- Ingredients -->
            <section class="ui-panel-soft recipe-section-card recipe-collections-card">
              <div class="ui-section-head mb-3">
                <h3 class="text-sm font-semibold tracking-tight">
                  {$_('page.recipes.ingredients')}
                </h3>
                <span class="text-xs text-muted-foreground">
                  {localIngredients.length}
                </span>
              </div>
              <RecipeIngredientEditor
                ingredients={localIngredients}
                scaledIngredients={targetServings !== (currentRecipe.metadata?.servings ?? 4)
                  ? scaledIngredients
                  : undefined}
                readonly={isReadonly}
                onupdate={handleIngredientsChange}
              />
            </section>

            <!-- Collections -->
            <section class="ui-panel-soft recipe-section-card">
              <div class="ui-section-head mb-3">
                <h3 class="text-sm font-semibold tracking-tight">
                  {$_('page.recipes.collections')}
                </h3>
                <button
                  onclick={() => (showAddToCollectionDialog = true)}
                  class="ui-button ui-button-secondary px-2 py-1 text-xs"
                >
                  <Plus size={12} />
                  {$_('page.recipes.add_to_collection')}
                </button>
              </div>
              {#if currentRecipe.collections.length > 0}
                <div class="flex flex-wrap gap-1.5">
                  {#each currentRecipe.collections as coll (coll.id)}
                    <span
                      class="inline-flex items-center gap-1 rounded-full border border-border/70 bg-accent/50 px-2.5 py-1 text-xs"
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
            </section>
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
        <div class="p-4 sm:p-5">
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

  {#if showMobileActionsSheet}
    <div
      class="fixed inset-0 z-40 bg-black/50 sm:hidden"
      onclick={closeMobileActionsSheet}
      onkeydown={handleMobileActionsKeydown}
      tabindex="-1"
      role="presentation"
    ></div>

    <div
      class="fixed z-50 bottom-0 left-0 right-0 bg-background rounded-t-2xl animate-bottom-sheet p-4 sm:hidden"
      role="menu"
      aria-label={$_('nav.more_options')}
      tabindex="-1"
      onkeydown={handleMobileActionsKeydown}
      use:bottomsheet={{ onClose: closeMobileActionsSheet }}
    >
      <div class="w-12 h-1 bg-muted rounded-full mx-auto mb-4"></div>

      <div class="space-y-2">
        {#if !isEncrypted && !isReadonly}
          <button
            type="button"
            onclick={() => {
              showSuggestionDialog = true;
              closeMobileActionsSheet();
            }}
            class="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-accent rounded-md transition-colors"
            role="menuitem"
          >
            <Sparkles size={18} />
            {$_('page.recipes.suggestions.find_similar')}
          </button>
        {/if}

        <div class="rounded-lg border border-border/60 px-3 py-2.5 text-sm text-muted-foreground">
          {#if saving}
            <div class="flex items-center gap-2">
              <Loader2 size={14} class="animate-spin" />
              <span>{$_('page.recipes.saving')}</span>
            </div>
          {:else if ingredientsDirty}
            <span>{$_('page.recipes.unsaved')}</span>
          {:else if currentRecipe?.note?.updated_at}
            <span>
              {$_('component.editor.last_updated', {
                values: { date: formatRelativeTime(currentRecipe.note.updated_at, $_) },
              })}
            </span>
          {:else}
            <span>{$_('common.no_data')}</span>
          {/if}
        </div>
      </div>
    </div>
  {/if}
{/if}

<style>
  .recipe-section-card {
    padding: 0.9rem;
  }

  .recipe-top-grid {
    display: grid;
    gap: 1rem;
  }

  .recipe-pane-shell {
    padding: 0.95rem;
  }

  .recipe-image-card {
    align-self: start;
  }

  .recipe-section-header {
    margin-bottom: 0.7rem;
    border-bottom: 1px dashed color-mix(in oklch, var(--color-border), transparent 26%);
    padding-bottom: 0.45rem;
  }

  .recipe-subsection {
    border-top: 1px solid color-mix(in oklch, var(--color-border), transparent 26%);
    padding-top: 0.75rem;
  }

  .recipe-section-header-plain {
    border-bottom: 0;
    padding-bottom: 0;
    margin-bottom: 0.6rem;
  }

  .recipe-collections-card {
    padding-top: 0.8rem;
    padding-bottom: 0.8rem;
  }

  @media (max-width: 639px) {
    :global(.recipe-editor .ui-input),
    :global(.recipe-editor .ui-select),
    :global(.recipe-editor .ui-textarea) {
      border-color: color-mix(in oklch, var(--color-border), transparent 44%);
      background: color-mix(in oklch, var(--color-background), transparent 14%);
      box-shadow: none;
    }

    :global(.recipe-editor .ui-input:hover),
    :global(.recipe-editor .ui-select:hover),
    :global(.recipe-editor .ui-textarea:hover) {
      border-color: color-mix(in oklch, var(--color-border), transparent 34%);
    }

    :global(.recipe-editor .ui-form-row) {
      gap: 0.3rem;
    }

    :global(.recipe-editor .ui-label) {
      margin-bottom: 0.25rem;
    }

    .recipe-section-header {
      margin-bottom: 0.55rem;
      padding-bottom: 0.35rem;
      border-bottom-color: color-mix(in oklch, var(--color-border), transparent 42%);
    }

    .recipe-subsection {
      border-top-color: color-mix(in oklch, var(--color-border), transparent 42%);
      padding-top: 0.65rem;
    }
  }

  @media (min-width: 640px) {
    .recipe-section-card {
      padding: 1rem;
    }

    .recipe-collections-card {
      padding-top: 0.85rem;
      padding-bottom: 0.85rem;
    }

    .recipe-pane-shell {
      padding: 1.1rem;
    }
  }

  @media (min-width: 1024px) {
    .recipe-top-grid {
      grid-template-columns: minmax(0, 1.3fr) minmax(16rem, 0.9fr);
      align-items: start;
    }

    .recipe-image-card {
      gap: 0.75rem;
    }

    .recipe-subsection {
      padding-top: 0.65rem;
    }
  }
</style>
