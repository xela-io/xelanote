<script lang="ts">
  import { Check, GripVertical, Pencil, Plus, Trash2, X } from 'lucide-svelte';
  import { SvelteMap } from 'svelte/reactivity';
  import { _ } from 'svelte-i18n';

  import type { RecipeIngredient, ScaledIngredient } from '$lib/api';

  import QuickAddIngredient from './QuickAddIngredient.svelte';
  import RecipeIngredientRow from './RecipeIngredientRow.svelte';

  interface Props {
    ingredients: RecipeIngredient[];
    scaledIngredients?: ScaledIngredient[];
    readonly?: boolean;
    onupdate: (ingredients: RecipeIngredient[]) => void;
  }

  const { ingredients, scaledIngredients, readonly = false, onupdate }: Props = $props();
  const unitDatalistId = 'recipe-ingredient-unit-options';
  const commonUnits = ['g', 'kg', 'mg', 'ml', 'l', 'TL', 'EL', 'Stk', 'Prise', 'Bund', 'Dose'];
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
      .reverse()
      .filter((unit, idx, arr) => arr.indexOf(unit) === idx);
    const merged = [...recent, ...commonUnits];
    return merged.filter((unit, idx) => merged.indexOf(unit) === idx);
  });

  function handleQuickAdd(ingredient: RecipeIngredient) {
    onupdate([...ingredients, ingredient]);
  }

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
      <QuickAddIngredient {ingredients} onadd={handleQuickAdd} />
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
  }

  @media (max-width: 767px) {
    .ingredient-table-shell {
      border: none;
      background: transparent;
      padding: 0;
      gap: 0.5rem;
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
