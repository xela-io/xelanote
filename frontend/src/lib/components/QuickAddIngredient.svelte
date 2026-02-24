<script lang="ts">
  import { Circle, CircleCheck, Plus } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { RecipeIngredient } from '$lib/api';

  interface Props {
    ingredients: RecipeIngredient[];
    onadd: (ingredient: RecipeIngredient) => void;
  }

  const { ingredients, onadd }: Props = $props();

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

  const suggestedUnits = $derived(() => {
    const recent = ingredients
      .map((ing) => ing.unit?.trim())
      .filter((unit): unit is string => Boolean(unit))
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

  function addQuickIngredient() {
    if (!quickName.trim()) return;
    const amount = quickAmount.trim() ? Number.parseFloat(quickAmount) : null;
    onadd({
      name: quickName.trim(),
      amount: Number.isFinite(amount as number) ? amount : null,
      unit: quickUnit.trim() || null,
      group_name: null,
      display_order: ingredients.length,
      optional: quickOptional,
      scalable: true,
    });
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
</script>

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

<style>
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

  .quick-add-row :global(.lucide) {
    flex-shrink: 0;
  }

  .quick-add-row .quick-opt-chip {
    grid-area: opt;
  }

  .quick-add-row .quick-add-button {
    grid-area: add;
  }

  @media (min-width: 768px) {
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

  @media (max-width: 767px) {
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
  }
</style>
