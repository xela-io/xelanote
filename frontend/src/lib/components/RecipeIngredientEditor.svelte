<script lang="ts">
  import { Check, Circle, CircleCheck, GripVertical, Pencil, Plus, Trash2, X } from 'lucide-svelte';
  import { SvelteMap } from 'svelte/reactivity';
  import { _ } from 'svelte-i18n';

  import type { RecipeIngredient, ScaledIngredient } from '$lib/api';

  import RecipeIngredientRow from './RecipeIngredientRow.svelte';

  interface Props {
    ingredients: RecipeIngredient[];
    scaledIngredients?: ScaledIngredient[];
    readonly?: boolean;
    onupdate: (ingredients: RecipeIngredient[]) => void;
  }

  const { ingredients, scaledIngredients, readonly = false, onupdate }: Props = $props();
  const unitDatalistId = 'recipe-ingredient-unit-options';
  const quickUnitListboxId = 'recipe-quick-unit-listbox';
  const commonUnits = ['g', 'kg', 'mg', 'ml', 'l', 'TL', 'EL', 'Stk', 'Prise', 'Bund', 'Dose'];
  let quickAmount = $state('');
  let quickUnit = $state('');
  let quickName = $state('');
  let quickOptional = $state(false);
  let quickAmountInput = $state<HTMLInputElement | null>(null);
  let quickUnitInput = $state<HTMLInputElement | null>(null);
  let quickNameInput = $state<HTMLInputElement | null>(null);
  let quickUnitOpen = $state(false);
  let quickUnitHighlightIndex = $state(0);
  let quickUnitBlurTimeout: ReturnType<typeof setTimeout> | null = null;
  let groupComposerOpen = $state(false);
  let groupNameDraft = $state('');
  let groupNameInput = $state<HTMLInputElement | null>(null);
  let editingGroupName = $state<string | null>(null);
  let editingGroupDraft = $state('');
  let draggedIngredientIndex = $state<number | null>(null);
  let draggedGroupName = $state<string | null>(null);
  let dragHoverRowIndex = $state<number | null>(null);
  let dragHoverGroupName = $state<string | null>(null);

  // Group ingredients by group_name
  const groups = $derived(() => {
    const map = new SvelteMap<string, RecipeIngredient[]>();
    for (const ing of ingredients) {
      const key = ing.group_name ?? '';
      if (!map.has(key)) map.set(key, []);
      map.get(key)!.push(ing);
    }
    return map;
  });

  const suggestedUnits = $derived(() => {
    const recent = ingredients
      .map((ing) => ing.unit?.trim())
      .filter((unit): unit is string => Boolean(unit))
      // Latest entries first, dedupe preserving order
      .reverse()
      .filter((unit, idx, arr) => arr.indexOf(unit) === idx);

    const merged = [...recent, ...commonUnits];
    return merged.filter((unit, idx) => merged.indexOf(unit) === idx);
  });

  const filteredQuickUnits = $derived(() => {
    const query = quickUnit.trim().toLowerCase();
    const units = suggestedUnits();
    if (!query) return units.slice(0, 8);
    return units.filter((unit) => unit.toLowerCase().includes(query)).slice(0, 8);
  });

  function addIngredient(groupName?: string) {
    const newIng: RecipeIngredient = {
      name: '',
      amount: null,
      unit: null,
      group_name: groupName || null,
      display_order: ingredients.length,
      optional: false,
      scalable: true,
    };
    onupdate([...ingredients, newIng]);
  }

  function addQuickIngredient() {
    if (!quickName.trim()) return;
    const amount = quickAmount.trim() ? Number.parseFloat(quickAmount) : null;
    const newIng: RecipeIngredient = {
      name: quickName.trim(),
      amount: Number.isFinite(amount as number) ? amount : null,
      unit: quickUnit.trim() || null,
      group_name: null,
      display_order: ingredients.length,
      optional: quickOptional,
      scalable: true,
    };
    onupdate([...ingredients, newIng]);
    quickAmount = '';
    quickUnit = '';
    quickName = '';
    quickOptional = false;
    quickUnitOpen = false;
    quickUnitHighlightIndex = 0;
    queueMicrotask(() => {
      quickNameInput?.focus();
    });
  }

  function handleQuickKeydown(e: KeyboardEvent, field: 'amount' | 'unit' | 'name') {
    if (e.key !== 'Enter') return;
    e.preventDefault();
    if (field === 'amount') {
      quickUnitInput?.focus();
      return;
    }
    if (field === 'unit') {
      if (quickUnitOpen && filteredQuickUnits().length > 0) {
        const selected = filteredQuickUnits()[quickUnitHighlightIndex] ?? filteredQuickUnits()[0];
        if (selected) quickUnit = selected;
        quickUnitOpen = false;
      }
      quickNameInput?.focus();
      return;
    }
    addQuickIngredient();
  }

  function handleQuickUnitKeydown(e: KeyboardEvent) {
    const options = filteredQuickUnits();
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (!quickUnitOpen && options.length > 0) {
        quickUnitOpen = true;
        quickUnitHighlightIndex = 0;
        return;
      }
      if (options.length > 0) {
        quickUnitHighlightIndex = (quickUnitHighlightIndex + 1) % options.length;
      }
      return;
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (!quickUnitOpen && options.length > 0) {
        quickUnitOpen = true;
        quickUnitHighlightIndex = Math.max(options.length - 1, 0);
        return;
      }
      if (options.length > 0) {
        quickUnitHighlightIndex = (quickUnitHighlightIndex - 1 + options.length) % options.length;
      }
      return;
    }
    if (e.key === 'Escape') {
      quickUnitOpen = false;
    }
  }

  function handleQuickUnitCombinedKeydown(e: KeyboardEvent) {
    handleQuickUnitKeydown(e);
    if (e.defaultPrevented) return;
    handleQuickKeydown(e, 'unit');
  }

  function handleQuickUnitInput() {
    quickUnitHighlightIndex = 0;
    quickUnitOpen = filteredQuickUnits().length > 0;
  }

  function openQuickUnitMenu() {
    if (quickUnitBlurTimeout) {
      clearTimeout(quickUnitBlurTimeout);
      quickUnitBlurTimeout = null;
    }
    if (filteredQuickUnits().length > 0) {
      quickUnitOpen = true;
    }
  }

  function closeQuickUnitMenuDelayed() {
    if (quickUnitBlurTimeout) clearTimeout(quickUnitBlurTimeout);
    quickUnitBlurTimeout = setTimeout(() => {
      quickUnitOpen = false;
      quickUnitBlurTimeout = null;
    }, 120);
  }

  function selectQuickUnit(unit: string) {
    quickUnit = unit;
    quickUnitOpen = false;
    quickUnitHighlightIndex = 0;
    queueMicrotask(() => quickNameInput?.focus());
  }

  function updateIngredient(index: number, updated: RecipeIngredient) {
    const newList = [...ingredients];
    newList[index] = updated;
    onupdate(newList);
  }

  function removeIngredient(index: number) {
    const newList = ingredients.filter((_, i) => i !== index);
    emitNormalized(newList);
  }

  function emitNormalized(next: RecipeIngredient[]) {
    const normalized = next.map((ing, i) => ({ ...ing, display_order: i, scalable: true }));
    onupdate(normalized);
  }

  function openGroupComposer() {
    groupComposerOpen = true;
    queueMicrotask(() => groupNameInput?.focus());
  }

  function closeGroupComposer() {
    groupComposerOpen = false;
    groupNameDraft = '';
  }

  function addGroupFromDraft() {
    const name = groupNameDraft.trim();
    if (!name) return;
    addIngredient(name);
    closeGroupComposer();
  }

  function startEditGroup(name: string) {
    editingGroupName = name;
    editingGroupDraft = name;
  }

  function cancelEditGroup() {
    editingGroupName = null;
    editingGroupDraft = '';
  }

  function saveEditGroup(oldName: string) {
    const nextName = editingGroupDraft.trim();
    if (!nextName) return;
    emitNormalized(
      ingredients.map((ing) =>
        ing.group_name === oldName ? { ...ing, group_name: nextName } : ing
      )
    );
    cancelEditGroup();
  }

  function handleEditGroupKeydown(e: KeyboardEvent, oldName: string) {
    if (e.key === 'Enter') {
      e.preventDefault();
      saveEditGroup(oldName);
      return;
    }
    if (e.key === 'Escape') {
      e.preventDefault();
      cancelEditGroup();
    }
  }

  function deleteGroup(name: string) {
    emitNormalized(
      ingredients.map((ing) => (ing.group_name === name ? { ...ing, group_name: null } : ing))
    );
    if (editingGroupName === name) cancelEditGroup();
  }

  function clearDragState() {
    draggedIngredientIndex = null;
    draggedGroupName = null;
    dragHoverRowIndex = null;
    dragHoverGroupName = null;
  }

  function handleIngredientDragStart(e: DragEvent, index: number) {
    const target = e.target as HTMLElement | null;
    if (!target?.closest('.ingredient-cell-grip')) {
      e.preventDefault();
      return;
    }
    draggedIngredientIndex = index;
    draggedGroupName = null;
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/plain', `ingredient:${index}`);
    }
  }

  function handleGroupDragStart(e: DragEvent, groupName: string) {
    draggedGroupName = groupName;
    draggedIngredientIndex = null;
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/plain', `group:${groupName}`);
    }
  }

  function moveIngredientBeforeTarget(
    fromIndex: number,
    targetIndex: number,
    targetGroupName: string | null
  ) {
    if (fromIndex === targetIndex) return;
    const list = [...ingredients];
    const [item] = list.splice(fromIndex, 1);
    if (!item) return;
    const adjustedTarget = fromIndex < targetIndex ? targetIndex - 1 : targetIndex;
    item.group_name = targetGroupName;
    list.splice(Math.max(0, adjustedTarget), 0, item);
    emitNormalized(list);
  }

  function moveIngredientToGroupStart(fromIndex: number, groupName: string) {
    const firstIndex = ingredients.findIndex((ing) => (ing.group_name ?? '') === groupName);
    if (firstIndex < 0) return;
    moveIngredientBeforeTarget(fromIndex, firstIndex, groupName || null);
  }

  function moveGroupBeforeGroup(sourceGroup: string, targetGroup: string) {
    if (!sourceGroup || !targetGroup || sourceGroup === targetGroup) return;
    const sourceItems = ingredients.filter((ing) => (ing.group_name ?? '') === sourceGroup);
    if (sourceItems.length === 0) return;
    const remaining = ingredients.filter((ing) => (ing.group_name ?? '') !== sourceGroup);
    const targetIdx = remaining.findIndex((ing) => (ing.group_name ?? '') === targetGroup);
    if (targetIdx < 0) return;
    remaining.splice(targetIdx, 0, ...sourceItems);
    emitNormalized(remaining);
  }

  function handleRowDragOver(e: DragEvent, index: number) {
    if (draggedIngredientIndex === null) return;
    e.preventDefault();
    dragHoverRowIndex = index;
    dragHoverGroupName = null;
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
  }

  function handleRowDrop(e: DragEvent, index: number, groupName: string | null) {
    if (draggedIngredientIndex === null) return;
    e.preventDefault();
    moveIngredientBeforeTarget(draggedIngredientIndex, index, groupName);
    clearDragState();
  }

  function handleGroupHeaderDragOver(e: DragEvent, groupName: string) {
    if (draggedIngredientIndex === null && draggedGroupName === null) return;
    e.preventDefault();
    dragHoverGroupName = groupName;
    dragHoverRowIndex = null;
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
  }

  function handleGroupHeaderDrop(e: DragEvent, groupName: string) {
    e.preventDefault();
    if (draggedIngredientIndex !== null) {
      moveIngredientToGroupStart(draggedIngredientIndex, groupName);
    } else if (draggedGroupName) {
      moveGroupBeforeGroup(draggedGroupName, groupName);
    }
    clearDragState();
  }

  function handleGroupComposerKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      addGroupFromDraft();
      return;
    }
    if (e.key === 'Escape') {
      e.preventDefault();
      closeGroupComposer();
    }
  }

  function getScaledForIndex(index: number): ScaledIngredient | null {
    if (!scaledIngredients || index >= scaledIngredients.length) return null;
    return scaledIngredients[index];
  }
</script>

<div class="ingredient-editor space-y-3">
  {#if ingredients.length === 0}
    <p
      class="ui-panel-soft rounded-lg border-dashed px-3 py-4 text-sm text-muted-foreground italic"
    >
      {$_('page.recipes.no_ingredients')}
    </p>
  {:else}
    {#if !readonly}
      <div class="ingredient-grid-header" aria-hidden="true">
        <span></span>
        <span>Menge</span>
        <span>{$_('page.recipes.unit')}</span>
        <span>{$_('page.recipes.ingredient_name')}</span>
        <span>{$_('page.recipes.optional')}</span>
      </div>
    {/if}

    {#if !readonly}
      <div class="ui-panel-soft quick-add-composer">
        <div class="ui-kicker quick-add-label">Neue Zutat</div>
        <div class="quick-add-row">
          <input
            bind:this={quickAmountInput}
            type="number"
            step="0.01"
            min="0"
            bind:value={quickAmount}
            onkeydown={(e) => handleQuickKeydown(e, 'amount')}
            placeholder="–"
            class="ui-input quick-input quick-amount"
          />
          <div class="quick-unit-combobox quick-unit">
            <input
              bind:this={quickUnitInput}
              type="text"
              bind:value={quickUnit}
              oninput={handleQuickUnitInput}
              onfocus={openQuickUnitMenu}
              onblur={closeQuickUnitMenuDelayed}
              onkeydown={handleQuickUnitCombinedKeydown}
              placeholder={$_('page.recipes.unit')}
              class="ui-input quick-input"
              role="combobox"
              aria-expanded={quickUnitOpen}
              aria-autocomplete="list"
              aria-haspopup="listbox"
              aria-controls={quickUnitListboxId}
            />
            {#if quickUnitOpen && filteredQuickUnits().length > 0}
              <div id={quickUnitListboxId} class="ui-panel quick-unit-menu" role="listbox">
                {#each filteredQuickUnits() as unit, idx (unit)}
                  <button
                    type="button"
                    class="ui-button ui-button-ghost quick-unit-option"
                    class:active={idx === quickUnitHighlightIndex}
                    role="option"
                    aria-selected={idx === quickUnitHighlightIndex}
                    onmousedown={(e) => {
                      e.preventDefault();
                      selectQuickUnit(unit);
                    }}
                  >
                    {unit}
                  </button>
                {/each}
              </div>
            {/if}
          </div>
          <input
            bind:this={quickNameInput}
            type="text"
            bind:value={quickName}
            onkeydown={(e) => handleQuickKeydown(e, 'name')}
            placeholder={$_('page.recipes.add_ingredient')}
            class="ui-input quick-input quick-name"
          />
          <label
            class="ui-toggle-chip quick-opt-chip"
            class:is-active={quickOptional}
            title={$_('page.recipes.optional')}
          >
            <input type="checkbox" bind:checked={quickOptional} />
            {#if quickOptional}
              <CircleCheck size={13} />
            {:else}
              <Circle size={13} />
            {/if}
            <span>opt</span>
          </label>
          <button
            type="button"
            onclick={addQuickIngredient}
            disabled={!quickName.trim()}
            class="ui-icon-button quick-add-button"
          >
            <Plus size={14} />
          </button>
        </div>
      </div>
    {/if}
    {@const grouped = groups()}
    <div class="ui-panel-soft ingredient-table-shell">
      {#each [...grouped.entries()] as [groupName, groupIngredients] (groupName)}
        {#if groupName}
          <div
            class="group-header mt-4 first:mt-0"
            class:drag-hover={dragHoverGroupName === groupName}
            role="group"
            draggable={!readonly}
            ondragstart={(e) => handleGroupDragStart(e, groupName)}
            ondragend={clearDragState}
            ondragover={(e) => handleGroupHeaderDragOver(e, groupName)}
            ondrop={(e) => handleGroupHeaderDrop(e, groupName)}
          >
            <div class="group-header-main">
              {#if !readonly}
                <span class="group-header-grip text-muted-foreground" title="Gruppe verschieben">
                  <GripVertical size={14} />
                </span>
              {/if}
              {#if editingGroupName === groupName}
                <input
                  type="text"
                  bind:value={editingGroupDraft}
                  onkeydown={(e) => handleEditGroupKeydown(e, groupName)}
                  class="ui-input group-header-input"
                />
              {:else}
                <h4 class="group-header-title">{groupName}</h4>
              {/if}
            </div>
            {#if !readonly}
              <div class="group-header-actions">
                {#if editingGroupName === groupName}
                  <button
                    type="button"
                    class="ui-icon-button p-1.5"
                    onclick={() => saveEditGroup(groupName)}
                    disabled={!editingGroupDraft.trim()}
                    title={$_('common.save')}
                  >
                    <Check size={13} />
                  </button>
                  <button
                    type="button"
                    class="ui-icon-button p-1.5"
                    onclick={cancelEditGroup}
                    title={$_('dialog.cancel')}
                  >
                    <X size={13} />
                  </button>
                {:else}
                  <button
                    type="button"
                    class="ui-icon-button p-1.5"
                    onclick={() => startEditGroup(groupName)}
                    title={$_('common.edit')}
                  >
                    <Pencil size={13} />
                  </button>
                  <button
                    type="button"
                    class="ui-icon-button p-1.5 text-destructive hover:text-destructive hover:bg-destructive/10"
                    onclick={() => deleteGroup(groupName)}
                    title={$_('common.delete')}
                  >
                    <Trash2 size={13} />
                  </button>
                {/if}
              </div>
            {/if}
          </div>
        {/if}

        {#each groupIngredients as ingredient, gi (gi)}
          {@const globalIndex = ingredients.indexOf(ingredient)}
          <div
            class="ingredient-dnd-wrap"
            class:drag-hover={dragHoverRowIndex === globalIndex}
            role="listitem"
            draggable={!readonly}
            ondragstart={(e) => handleIngredientDragStart(e, globalIndex)}
            ondragend={clearDragState}
            ondragover={(e) => handleRowDragOver(e, globalIndex)}
            ondrop={(e) => handleRowDrop(e, globalIndex, ingredient.group_name ?? null)}
          >
            <RecipeIngredientRow
              {ingredient}
              {readonly}
              scaled={getScaledForIndex(globalIndex)}
              {unitDatalistId}
              unitSuggestions={suggestedUnits()}
              onupdate={(updated) => updateIngredient(globalIndex, updated)}
              onremove={() => removeIngredient(globalIndex)}
            />
          </div>
        {/each}
      {/each}
    </div>
  {/if}

  {#if !readonly}
    <div class="ingredient-footer-actions flex flex-wrap gap-2 pt-1">
      <button
        onclick={() => addIngredient()}
        class="ui-button ui-button-secondary text-sm px-2.5 py-1.5"
      >
        <Plus size={14} />
        {$_('page.recipes.add_ingredient')}
      </button>
      <button
        onclick={openGroupComposer}
        class="ui-button ui-button-secondary text-sm px-2.5 py-1.5"
      >
        <Plus size={14} />
        {$_('page.recipes.add_group')}
      </button>
    </div>

    {#if groupComposerOpen}
      <div class="ui-panel-soft group-composer">
        <div class="ui-kicker group-composer-label">{$_('page.recipes.add_group')}</div>
        <div class="group-composer-row">
          <input
            bind:this={groupNameInput}
            type="text"
            bind:value={groupNameDraft}
            onkeydown={handleGroupComposerKeydown}
            placeholder={$_('page.recipes.group_name_prompt')}
            class="ui-input"
          />
          <button
            type="button"
            onclick={closeGroupComposer}
            class="ui-button ui-button-ghost px-2.5 py-1.5 text-sm"
          >
            {$_('dialog.cancel')}
          </button>
          <button
            type="button"
            onclick={addGroupFromDraft}
            disabled={!groupNameDraft.trim()}
            class="ui-button ui-button-primary px-2.5 py-1.5 text-sm"
          >
            {$_('common.create')}
          </button>
        </div>
      </div>
    {/if}
  {/if}

  <datalist id={unitDatalistId}>
    {#each suggestedUnits() as unit (unit)}
      <option value={unit}></option>
    {/each}
  </datalist>
</div>

<style>
  .ingredient-grid-header {
    display: none;
  }

  .ingredient-table-shell {
    border: 1px solid hsl(var(--border) / 0.75);
    border-radius: 0.9rem;
    background: hsl(var(--background) / 0.22);
    padding: 0.45rem;
    display: grid;
    gap: 0.42rem;
  }

  .ingredient-dnd-wrap.drag-hover {
    border-radius: 0.9rem;
    box-shadow: 0 0 0 2px hsl(var(--ring) / 0.22);
  }

  .group-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    margin-bottom: 0.35rem;
    padding: 0.15rem 0.2rem;
    border-radius: 0.7rem;
  }

  .group-header.drag-hover {
    background: hsl(var(--accent) / 0.35);
    box-shadow: 0 0 0 1px hsl(var(--border) / 0.8) inset;
  }

  .group-header-main {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    min-width: 0;
    flex: 1;
  }

  .group-header-grip {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    cursor: grab;
    opacity: 0.75;
    flex-shrink: 0;
  }

  .group-header-title {
    margin: 0;
    font-size: 0.74rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: hsl(var(--muted-foreground));
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .group-header-input {
    min-width: 0;
    max-width: 18rem;
    padding: 0.35rem 0.55rem;
    font-size: 0.82rem;
  }

  .group-header-actions {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    flex-shrink: 0;
  }

  .quick-add-composer {
    border: 1px solid hsl(var(--border) / 0.9);
    border-radius: 0.95rem;
    background: linear-gradient(
      180deg,
      hsl(var(--accent) / 0.16) 0%,
      hsl(var(--background) / 0.22) 100%
    );
    padding: 0.5rem;
    margin-bottom: 0.25rem;
    box-shadow:
      inset 0 1px 0 hsl(var(--border) / 0.12),
      0 1px 6px hsl(var(--background) / 0.14);
  }

  .quick-add-label {
    margin: 0 0 0.4rem 0.15rem;
  }

  .quick-add-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto auto;
    grid-template-areas:
      'name amount amount'
      'unit opt add';
    gap: 0.5rem;
    padding: 0.5rem;
    border: 1px dashed hsl(var(--border) / 0.85);
    border-radius: 0.9rem;
    background: hsl(var(--background) / 0.62);
  }

  .quick-amount {
    grid-area: amount;
  }

  .quick-unit {
    grid-area: unit;
  }

  .quick-name {
    grid-area: name;
  }

  .quick-input {
    width: 100%;
    padding: 0.5rem 0.55rem;
    font-size: 0.9rem;
    line-height: 1.2;
  }

  .quick-unit-combobox {
    position: relative;
  }

  .quick-unit-menu {
    position: absolute;
    top: calc(100% + 0.25rem);
    left: 0;
    right: 0;
    z-index: 20;
    display: grid;
    gap: 0.2rem;
    border-radius: 0.7rem;
    padding: 0.25rem;
    box-shadow: 0 8px 24px hsl(var(--background) / 0.35);
  }

  .quick-unit-option {
    width: 100%;
    border-radius: 0.5rem;
    text-align: left;
    padding: 0.35rem 0.45rem;
    font-size: 0.85rem;
    justify-content: flex-start;
  }

  .quick-unit-option:hover,
  .quick-unit-option.active {
    background: hsl(var(--accent) / 0.6);
  }

  .quick-add-button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0.45rem;
  }

  .quick-add-button:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .group-composer {
    padding: 0.5rem;
  }

  .group-composer-label {
    margin: 0 0 0.4rem 0.15rem;
  }

  .group-composer-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto auto;
    gap: 0.5rem;
    align-items: center;
  }

  @media (min-width: 768px) {
    .ingredient-grid-header {
      display: grid;
      grid-template-columns: 1.15rem 4.1rem 4.6rem minmax(0, 1fr) 3.6rem;
      align-items: center;
      gap: 0.5rem;
      padding: 0 0.5rem;
      font-size: 0.76rem;
      text-transform: uppercase;
      letter-spacing: 0.04em;
      color: hsl(var(--muted-foreground));
    }

    .ingredient-editor {
      max-width: 49rem;
    }

    .ingredient-table-shell {
      padding: 0.5rem;
      gap: 0.4rem;
      border-radius: 1rem;
      margin-top: 0.25rem;
    }

    .quick-add-composer {
      padding: 0.5rem;
      border-radius: 1rem;
    }

    .quick-add-row {
      grid-template-columns: 1.15rem 4.1rem 4.6rem minmax(0, 1fr) 3.6rem 2.45rem;
      grid-template-areas: '. amount unit name opt add';
      align-items: center;
      padding: 0.6rem;
      gap: 0.5rem;
    }
  }

  .quick-add-row :global(.lucide) {
    flex-shrink: 0;
  }

  .quick-add-row .quick-opt-chip {
    grid-area: opt;
  }

  .quick-add-row .quick-add-button {
    grid-area: add;
  }

  @media (max-width: 767px) {
    .ingredient-table-shell {
      border: none;
      background: transparent;
      padding: 0;
      gap: 0.5rem;
    }

    .quick-add-composer {
      padding: 0.4rem;
      border-radius: 0.9rem;
      margin-bottom: 0.15rem;
    }

    .quick-add-row {
      align-items: center;
      gap: 0.4rem;
      padding: 0.45rem;
    }

    .quick-add-row .quick-opt-chip {
      grid-area: opt;
      justify-self: end;
    }

    .quick-add-row .quick-add-button {
      grid-area: add;
      min-width: 2.25rem;
    }

    .ingredient-footer-actions > button:first-child {
      display: none;
    }

    .group-composer-row {
      grid-template-columns: minmax(0, 1fr);
      gap: 0.4rem;
    }

    .group-composer-row > :global(button) {
      width: 100%;
    }

    .group-header {
      padding-inline: 0.1rem;
    }

    .group-header-title {
      font-size: 0.78rem;
    }
  }
</style>
