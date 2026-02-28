<script lang="ts">
  import { BookOpen, Loader2, Plus, ShoppingCart, Sparkles, Users } from 'lucide-svelte';
  import { SvelteMap, SvelteSet } from 'svelte/reactivity';
  import { _ } from 'svelte-i18n';

  import {
    getShoppingListShares,
    removeShoppingListShare,
    shareShoppingList,
    updateShoppingListShareRole,
  } from '$lib/api/shopping';
  import type { ShoppingItem } from '$lib/api/types';
  import type { ShoppingListShare } from '$lib/api/types';
  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import ShoppingCategoryGroup from '$lib/components/shopping/ShoppingCategoryGroup.svelte';
  import ShoppingCheckedSection from '$lib/components/shopping/ShoppingCheckedSection.svelte';
  import ShoppingImportRecipeDialog from '$lib/components/shopping/ShoppingImportRecipeDialog.svelte';
  import ShoppingItemRow from '$lib/components/shopping/ShoppingItemRow.svelte';
  import ShoppingListDialog from '$lib/components/shopping/ShoppingListDialog.svelte';
  import ShoppingListTabs from '$lib/components/shopping/ShoppingListTabs.svelte';
  import ShoppingQuickInput from '$lib/components/shopping/ShoppingQuickInput.svelte';
  import ShoppingShareDialog from '$lib/components/shopping/ShoppingShareDialog.svelte';
  import PageHeader from '$lib/components/ui/PageHeader.svelte';
  import * as features from '$lib/stores/features.svelte';
  import * as shopping from '$lib/stores/shopping.svelte';
  import * as toast from '$lib/stores/toast.svelte';

  // State
  let showCreateDialog = $state(false);
  let showImportDialog = $state(false);
  let showShareDialog = $state(false);
  let shares = $state<ShoppingListShare[]>([]);
  let _editingItem = $state<ShoppingItem | null>(null);
  const pendingCategories = new SvelteSet<string>();
  let showNewCategoryInput = $state(false);
  let newCategoryName = $state('');

  // Derived state
  const lists = $derived(shopping.getLists());
  const currentList = $derived(shopping.getCurrentList());
  const listsLoading = $derived(shopping.getListsLoading());
  const listLoading = $derived(shopping.getListLoading());
  const saving = $derived(shopping.getSaving());
  const sorting = $derived(shopping.getSorting());
  const isEnabled = $derived(
    features.getShoppingFeatureEnabled() && features.getShoppingFeatureLoaded()
  );
  const canEdit = $derived(currentList?.role === 'owner' || currentList?.role === 'editor');

  // Group items by category
  const uncheckedItems = $derived((currentList?.items ?? []).filter((i) => !i.is_checked));
  const checkedItems = $derived((currentList?.items ?? []).filter((i) => i.is_checked));

  // Group unchecked items by category (only items with explicit category)
  const categorizedItems = $derived(() => {
    const groups = new SvelteMap<string, ShoppingItem[]>();
    for (const item of uncheckedItems) {
      if (!item.category) continue;
      if (!groups.has(item.category)) groups.set(item.category, []);
      groups.get(item.category)!.push(item);
    }
    // Sort by category_order of first item in each group
    return [...groups.entries()].sort((a, b) => {
      const orderA = a[1][0]?.category_order ?? 99;
      const orderB = b[1][0]?.category_order ?? 99;
      return orderA - orderB;
    });
  });

  const uncategorizedItems = $derived(uncheckedItems.filter((i) => !i.category));
  const pendingCategoriesFiltered = $derived(
    [...pendingCategories].filter((c) => !uncheckedItems.some((i) => i.category === c))
  );

  const hasCategories = $derived(
    uncheckedItems.some((i) => !!i.category) || pendingCategories.size > 0
  );

  // Load lists on mount
  $effect(() => {
    if (isEnabled) {
      shopping.loadLists();
      shopping.loadFavorites();
    }
  });

  // Auto-select first list
  $effect(() => {
    if (lists.length > 0 && !currentList) {
      shopping.loadList(lists[0].id);
    }
  });

  // Clear pending categories when switching lists
  let lastListId: number | undefined;
  $effect(() => {
    const newId = currentList?.id;
    if (newId !== lastListId) {
      pendingCategories.clear();
      showNewCategoryInput = false;
      newCategoryName = '';
      lastListId = newId;
    }
  });

  async function handleCreateList(name: string, color: string | null) {
    try {
      const list = await shopping.addList(name, color);
      showCreateDialog = false;
      if (list) {
        shopping.loadList(list.id);
      }
    } catch {
      toast.error('Fehler beim Erstellen der Liste');
    }
  }

  async function handleAddItems(
    items: Array<{ name: string; quantity: number | null; unit: string | null }>
  ) {
    if (!currentList) return;
    try {
      const shoppingItems = items.map((i) => ({
        name: i.name,
        quantity: i.quantity,
        unit: i.unit,
      }));
      if (shoppingItems.length === 1) {
        await shopping.addItem(currentList.id, shoppingItems[0]);
      } else {
        await shopping.addItems(currentList.id, shoppingItems);
      }
    } catch {
      toast.error('Fehler beim Hinzufügen');
    }
  }

  async function handleCheck(itemId: number, isChecked: boolean) {
    if (!currentList) return;
    try {
      await shopping.setItemChecked(currentList.id, itemId, isChecked);
    } catch {
      // Error already handled in store
    }
  }

  async function handleDelete(itemId: number) {
    if (!currentList) return;
    try {
      await shopping.deleteItem(currentList.id, itemId);
    } catch {
      toast.error('Fehler beim Löschen');
    }
  }

  function handleEdit(item: ShoppingItem) {
    _editingItem = item;
  }

  async function handleFavorite(item: ShoppingItem) {
    try {
      await shopping.addFavorite(item.name, item.quantity, item.unit, item.category);
      toast.success(`"${item.name}" zu Favoriten hinzugefügt`);
    } catch {
      // Duplicate is ok
    }
  }

  function getCategoryOrder(category: string): number {
    const existing = uncheckedItems.filter((i) => i.category === category);
    if (existing.length > 0) {
      return existing[0].category_order ?? 15;
    }
    const orders = uncheckedItems.filter((i) => i.category).map((i) => i.category_order ?? 0);
    return Math.max(...orders, 14) + 1;
  }

  async function handleAddCategoryItem(
    category: string,
    items: Array<{ name: string; quantity: number | null; unit: string | null }>
  ) {
    if (!currentList) return;
    try {
      const order = getCategoryOrder(category);
      const shoppingItems = items.map((i) => ({
        name: i.name,
        quantity: i.quantity,
        unit: i.unit,
        category: category,
        category_order: order,
      }));
      if (shoppingItems.length === 1) {
        await shopping.addItem(currentList.id, shoppingItems[0]);
      } else {
        await shopping.addItems(currentList.id, shoppingItems);
      }
    } catch {
      toast.error('Fehler beim Hinzufügen');
    }
  }

  function handleCreateCategory() {
    const name = newCategoryName.trim();
    if (!name) return;
    const existingCategories = new Set(
      uncheckedItems.filter((i) => i.category).map((i) => i.category!)
    );
    if (existingCategories.has(name) || pendingCategories.has(name)) return;
    pendingCategories.add(name);
    showNewCategoryInput = false;
    newCategoryName = '';
  }

  function autofocus(node: HTMLElement) {
    node.focus();
  }

  function handleNewCategoryKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleCreateCategory();
    }
    if (e.key === 'Escape') {
      showNewCategoryInput = false;
      newCategoryName = '';
    }
  }

  async function handleClearChecked() {
    if (!currentList) return;
    try {
      await shopping.clearChecked(currentList.id);
    } catch {
      toast.error('Fehler beim Löschen');
    }
  }

  async function handleSort() {
    if (!currentList) return;
    if (uncheckedItems.some((i) => !!i.category)) {
      if (!confirm($_('page.shopping.sort_warning'))) return;
    }
    try {
      await shopping.sortByCategory(currentList.id);
      pendingCategories.clear();
      toast.success('Sortierung abgeschlossen');
    } catch {
      toast.error('KI-Sortierung fehlgeschlagen');
    }
  }

  async function handleImportRecipe(recipeNoteId: string) {
    if (!currentList) return;
    try {
      const items = await shopping.importRecipe(currentList.id, recipeNoteId);
      showImportDialog = false;
      toast.success(`${items.length} Zutaten importiert`);
    } catch {
      toast.error('Rezept-Import fehlgeschlagen');
    }
  }

  async function openShareDialog() {
    if (!currentList) return;
    try {
      shares = await getShoppingListShares(currentList.id);
      showShareDialog = true;
    } catch {
      toast.error('Fehler beim Laden der Freigaben');
    }
  }

  async function handleShare(userId: number, role: 'viewer' | 'editor') {
    if (!currentList) return;
    try {
      await shareShoppingList(currentList.id, userId, role);
      shares = await getShoppingListShares(currentList.id);
      toast.success('Liste geteilt');
    } catch {
      toast.error('Fehler beim Teilen');
    }
  }

  async function handleUpdateShareRole(userId: number, role: 'viewer' | 'editor') {
    if (!currentList) return;
    try {
      await updateShoppingListShareRole(currentList.id, userId, role);
      shares = await getShoppingListShares(currentList.id);
    } catch {
      toast.error('Fehler beim Aktualisieren');
    }
  }

  async function handleRemoveShare(userId: number) {
    if (!currentList) return;
    try {
      await removeShoppingListShare(currentList.id, userId);
      shares = shares.filter((s) => s.shared_with_user_id !== userId);
    } catch {
      toast.error('Fehler beim Entfernen');
    }
  }
</script>

<svelte:head>
  <title>{$_('page.shopping.title')} | xelanote</title>
</svelte:head>

<div class="shopping-page">
  <PageHeader title={$_('page.shopping.title')}>
    {#snippet leading()}
      <MobileSidebarInlineToggle />
      <ShoppingCart size={20} class="text-muted-foreground" />
    {/snippet}
    {#snippet actions()}
      {#if currentList && canEdit}
        <div class="flex items-center gap-1.5">
          <button
            type="button"
            class="action-btn"
            onclick={handleSort}
            disabled={sorting || uncheckedItems.length === 0}
            title={$_('page.shopping.sort_by_category')}
          >
            {#if sorting}
              <Loader2 size={16} class="animate-spin" />
            {:else}
              <Sparkles size={16} />
            {/if}
            <span class="hidden sm:inline">
              {sorting ? $_('page.shopping.sorting') : $_('page.shopping.sort_by_category')}
            </span>
          </button>
          <button
            type="button"
            class="action-btn"
            onclick={() => (showImportDialog = true)}
            title={$_('page.shopping.import_recipe')}
          >
            <BookOpen size={16} />
            <span class="hidden sm:inline">{$_('page.shopping.import_recipe')}</span>
          </button>
          {#if currentList.role === 'owner'}
            <button
              type="button"
              class="action-btn"
              onclick={openShareDialog}
              title={$_('page.shopping.share_list')}
            >
              <Users size={16} />
            </button>
          {/if}
        </div>
      {/if}
    {/snippet}
  </PageHeader>

  {#if !isEnabled}
    <div class="shopping-empty">
      <ShoppingCart size={48} class="text-muted-foreground opacity-30" />
      <p>{$_('page.shopping.title')}</p>
    </div>
  {:else if listsLoading}
    <div class="shopping-loading">
      <Loader2 size={24} class="animate-spin text-muted-foreground" />
    </div>
  {:else}
    <!-- List tabs -->
    <ShoppingListTabs
      {lists}
      activeListId={currentList?.id ?? null}
      onselect={(id) => shopping.loadList(id)}
      oncreate={() => (showCreateDialog = true)}
    />

    {#if currentList}
      <!-- Quick input -->
      {#if canEdit}
        <ShoppingQuickInput onsubmit={handleAddItems} disabled={saving} />
      {/if}

      <!-- Item list -->
      <div class="shopping-items">
        {#if listLoading}
          <div class="shopping-loading">
            <Loader2 size={20} class="animate-spin text-muted-foreground" />
          </div>
        {:else if uncheckedItems.length === 0 && checkedItems.length === 0 && pendingCategories.size === 0}
          <div class="shopping-empty-items">
            <p>{$_('page.shopping.no_items')}</p>
          </div>
        {:else}
          <!-- Unchecked items -->
          {#if hasCategories}
            {#each categorizedItems() as [category, catItems] (category)}
              <ShoppingCategoryGroup
                {category}
                items={catItems}
                {canEdit}
                onadditem={(parsed) => handleAddCategoryItem(category, parsed)}
              >
                {#each catItems as item (item.id)}
                  <ShoppingItemRow
                    {item}
                    readonly={!canEdit}
                    oncheck={handleCheck}
                    ondelete={handleDelete}
                    onedit={handleEdit}
                    onfavorite={handleFavorite}
                  />
                {/each}
              </ShoppingCategoryGroup>
            {/each}
            {#each pendingCategoriesFiltered as category (category)}
              <ShoppingCategoryGroup
                {category}
                items={[]}
                {canEdit}
                onadditem={(parsed) => handleAddCategoryItem(category, parsed)}
              ></ShoppingCategoryGroup>
            {/each}
            {#if uncategorizedItems.length > 0}
              <div class="uncategorized-header">
                <span>{$_('page.shopping.uncategorized')}</span>
              </div>
              {#each uncategorizedItems as item (item.id)}
                <ShoppingItemRow
                  {item}
                  readonly={!canEdit}
                  oncheck={handleCheck}
                  ondelete={handleDelete}
                  onedit={handleEdit}
                  onfavorite={handleFavorite}
                />
              {/each}
            {/if}
          {:else}
            {#each uncheckedItems as item (item.id)}
              <ShoppingItemRow
                {item}
                readonly={!canEdit}
                oncheck={handleCheck}
                ondelete={handleDelete}
                onedit={handleEdit}
                onfavorite={handleFavorite}
              />
            {/each}
          {/if}

          <!-- New Category -->
          {#if canEdit}
            {#if showNewCategoryInput}
              <div class="new-category-input">
                <input
                  type="text"
                  bind:value={newCategoryName}
                  onkeydown={handleNewCategoryKeydown}
                  placeholder={$_('page.shopping.category_name_placeholder')}
                  class="new-category-field"
                  use:autofocus
                />
              </div>
            {:else}
              <button
                type="button"
                class="new-category-btn"
                onclick={() => (showNewCategoryInput = true)}
              >
                <Plus size={14} />
                <span>{$_('page.shopping.new_category')}</span>
              </button>
            {/if}
          {/if}

          <!-- Checked items (collapsible) -->
          <ShoppingCheckedSection count={checkedItems.length} onclear={handleClearChecked}>
            {#each checkedItems as item (item.id)}
              <ShoppingItemRow
                {item}
                readonly={!canEdit}
                oncheck={handleCheck}
                ondelete={handleDelete}
                onedit={handleEdit}
              />
            {/each}
          </ShoppingCheckedSection>
        {/if}
      </div>
    {:else if lists.length === 0}
      <div class="shopping-empty">
        <ShoppingCart size={48} class="text-muted-foreground opacity-30" />
        <p>{$_('page.shopping.no_lists')}</p>
        <button type="button" class="btn-primary" onclick={() => (showCreateDialog = true)}>
          {$_('page.shopping.create_first')}
        </button>
      </div>
    {/if}
  {/if}
</div>

<!-- Dialogs -->
<ShoppingListDialog
  open={showCreateDialog}
  onsave={handleCreateList}
  onclose={() => (showCreateDialog = false)}
/>

<ShoppingImportRecipeDialog
  open={showImportDialog}
  onimport={handleImportRecipe}
  onclose={() => (showImportDialog = false)}
/>

<ShoppingShareDialog
  open={showShareDialog}
  {shares}
  onshare={handleShare}
  onupdaterole={handleUpdateShareRole}
  onremove={handleRemoveShare}
  onclose={() => (showShareDialog = false)}
/>

<style>
  .shopping-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
  }

  .shopping-items {
    flex: 1;
    overflow-y: auto;
  }

  .shopping-loading {
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 3rem;
  }

  .shopping-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    padding: 4rem 2rem;
    color: var(--color-text-muted);
    text-align: center;
  }

  .shopping-empty-items {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 3rem 2rem;
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }

  .action-btn {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    padding: 0.375rem 0.625rem;
    border-radius: var(--radius-md);
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    cursor: pointer;
    transition: all 0.15s;
  }

  .action-btn:hover {
    background: var(--color-surface-200);
    color: var(--color-text);
  }

  .action-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .uncategorized-header {
    display: flex;
    align-items: center;
    padding: 0.5rem 0.75rem;
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    opacity: 0.7;
  }

  .uncategorized-header::before,
  .uncategorized-header::after {
    content: '';
    flex: 1;
    border-top: 1px solid var(--color-surface-200);
  }

  .uncategorized-header::before {
    margin-right: 0.5rem;
  }

  .uncategorized-header::after {
    margin-left: 0.5rem;
  }

  .new-category-btn {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    width: 100%;
    padding: 0.5rem 0.75rem;
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    cursor: pointer;
    transition: all 0.1s;
    border-top: 1px solid var(--color-surface-200);
  }

  .new-category-btn:hover {
    background: var(--color-surface-100);
    color: var(--color-text);
  }

  .new-category-input {
    display: flex;
    padding: 0.5rem 0.75rem;
    border-top: 1px solid var(--color-surface-200);
  }

  .new-category-field {
    flex: 1;
    padding: 0.375rem 0.5rem;
    border: 1px solid var(--color-surface-300);
    border-radius: var(--radius-sm);
    background: var(--color-surface-100);
    color: var(--color-text);
    font-size: var(--text-sm);
  }

  .new-category-field:focus {
    outline: none;
    border-color: var(--color-primary);
  }

  .btn-primary {
    padding: 0.625rem 1.25rem;
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
    color: white;
    background: var(--color-primary);
    cursor: pointer;
  }

  .btn-primary:hover {
    opacity: 0.9;
  }
</style>
