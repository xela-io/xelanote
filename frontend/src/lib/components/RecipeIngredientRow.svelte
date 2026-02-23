<script lang="ts">
  import { Circle, CircleCheck, GripVertical, Trash2 } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { RecipeIngredient } from '$lib/api';

  interface Props {
    ingredient: RecipeIngredient;
    readonly?: boolean;
    scaled?: { display_amount: string } | null;
    unitDatalistId?: string;
    unitSuggestions?: string[];
    onupdate: (ingredient: RecipeIngredient) => void;
    onremove: () => void;
  }

  const {
    ingredient,
    readonly = false,
    scaled = null,
    unitDatalistId,
    unitSuggestions = [],
    onupdate,
    onremove,
  }: Props = $props();

  let rowUnitInput = $state<HTMLInputElement | null>(null);
  let rowNameInput = $state<HTMLInputElement | null>(null);
  let unitMenuOpen = $state(false);
  let unitMenuIndex = $state(0);
  let unitBlurTimeout: ReturnType<typeof setTimeout> | null = null;

  const filteredUnitSuggestions = $derived(() => {
    const query = (ingredient.unit ?? '').trim().toLowerCase();
    if (!query) return unitSuggestions.slice(0, 8);
    return unitSuggestions.filter((u) => u.toLowerCase().includes(query)).slice(0, 8);
  });

  const rowUnitListboxId = $derived(
    `row-unit-listbox-${ingredient.display_order}-${(ingredient.name || 'ing')
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .slice(0, 16)}`
  );

  function handleChange(field: string, value: unknown) {
    onupdate({ ...ingredient, [field]: value, scalable: true });
  }

  function handleAmountChange(e: Event) {
    const val = (e.target as HTMLInputElement).value;
    handleChange('amount', val ? parseFloat(val) : null);
  }

  function handleUnitInput(e: Event) {
    handleChange('unit', (e.target as HTMLInputElement).value || null);
    unitMenuIndex = 0;
    unitMenuOpen = filteredUnitSuggestions().length > 0;
  }

  function openUnitMenu() {
    if (unitBlurTimeout) {
      clearTimeout(unitBlurTimeout);
      unitBlurTimeout = null;
    }
    if (filteredUnitSuggestions().length > 0) {
      unitMenuOpen = true;
    }
  }

  function closeUnitMenuDelayed() {
    if (unitBlurTimeout) clearTimeout(unitBlurTimeout);
    unitBlurTimeout = setTimeout(() => {
      unitMenuOpen = false;
      unitBlurTimeout = null;
    }, 120);
  }

  function selectUnitSuggestion(unit: string, focusName = true) {
    handleChange('unit', unit);
    unitMenuOpen = false;
    unitMenuIndex = 0;
    if (focusName) {
      queueMicrotask(() => rowNameInput?.focus());
    }
  }

  function handleUnitKeydown(e: KeyboardEvent) {
    const options = filteredUnitSuggestions();
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (!unitMenuOpen && options.length > 0) {
        unitMenuOpen = true;
        unitMenuIndex = 0;
        return;
      }
      if (options.length > 0) unitMenuIndex = (unitMenuIndex + 1) % options.length;
      return;
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (!unitMenuOpen && options.length > 0) {
        unitMenuOpen = true;
        unitMenuIndex = Math.max(options.length - 1, 0);
        return;
      }
      if (options.length > 0) unitMenuIndex = (unitMenuIndex - 1 + options.length) % options.length;
      return;
    }
    if (e.key === 'Enter') {
      if (unitMenuOpen && options.length > 0) {
        e.preventDefault();
        const selected = options[unitMenuIndex] ?? options[0];
        if (selected) selectUnitSuggestion(selected);
        return;
      }
      e.preventDefault();
      rowNameInput?.focus();
      return;
    }
    if (e.key === 'Escape') {
      unitMenuOpen = false;
    }
  }
</script>

<div
  class="ui-list-item ingredient-row group"
  class:optional={ingredient.optional}
  class:readonly-row={readonly}
>
  {#if !readonly}
    <div
      class="grip ingredient-cell-grip cursor-grab text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100"
    >
      <GripVertical size={14} />
    </div>
  {/if}

  <!-- Amount -->
  <div class="ingredient-cell-amount">
    {#if readonly}
      <span class="text-sm font-medium">
        {scaled?.display_amount ?? ingredient.amount ?? ''}
      </span>
    {:else}
      <input
        type="number"
        value={ingredient.amount ?? ''}
        oninput={handleAmountChange}
        step="0.01"
        min="0"
        placeholder="–"
        class="ui-input ingredient-input"
      />
    {/if}
  </div>

  <!-- Unit -->
  <div class="ingredient-cell-unit">
    {#if readonly}
      <span class="text-sm text-muted-foreground">{ingredient.unit ?? ''}</span>
    {:else}
      <div class="row-unit-combobox">
        <input
          bind:this={rowUnitInput}
          type="text"
          value={ingredient.unit ?? ''}
          oninput={handleUnitInput}
          onfocus={openUnitMenu}
          onblur={closeUnitMenuDelayed}
          onkeydown={handleUnitKeydown}
          placeholder={$_('page.recipes.unit')}
          list={unitDatalistId}
          class="ui-input ingredient-input"
          aria-expanded={unitMenuOpen}
          aria-autocomplete="list"
          aria-haspopup="listbox"
          aria-controls={rowUnitListboxId}
        />
        {#if unitMenuOpen && filteredUnitSuggestions().length > 0}
          <div id={rowUnitListboxId} class="ui-panel row-unit-menu" role="listbox">
            {#each filteredUnitSuggestions() as unit, idx (unit)}
              <button
                type="button"
                class="ui-button ui-button-ghost row-unit-option"
                class:active={idx === unitMenuIndex}
                role="option"
                aria-selected={idx === unitMenuIndex}
                onmousedown={(e) => {
                  e.preventDefault();
                  selectUnitSuggestion(unit);
                }}
              >
                {unit}
              </button>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>

  <!-- Name -->
  <div class="ingredient-cell-name min-w-0">
    {#if readonly}
      <span class="text-sm" class:italic={ingredient.optional}>
        {ingredient.name}
        {#if ingredient.optional}
          <span class="text-xs text-muted-foreground">({$_('page.recipes.optional')})</span>
        {/if}
      </span>
    {:else}
      <input
        bind:this={rowNameInput}
        type="text"
        value={ingredient.name}
        oninput={(e) => handleChange('name', (e.target as HTMLInputElement).value)}
        placeholder={$_('page.recipes.ingredient_name')}
        class="ui-input ingredient-input"
      />
    {/if}
  </div>

  {#if !readonly}
    <div class="ingredient-cell-options">
      <!-- Optional Toggle -->
      <label
        class="ui-toggle-chip option-chip"
        class:is-active={ingredient.optional}
        title={$_('page.recipes.optional')}
      >
        <input
          type="checkbox"
          checked={ingredient.optional}
          onchange={(e) => handleChange('optional', (e.target as HTMLInputElement).checked)}
          class="w-3 h-3"
        />
        {#if ingredient.optional}
          <CircleCheck size={13} />
        {:else}
          <Circle size={13} />
        {/if}
        <span>opt</span>
      </label>

      <!-- Remove -->
      <button
        onclick={onremove}
        class="ui-icon-button remove-button shrink-0 p-1.5 opacity-0 transition-opacity group-hover:opacity-100"
        title={$_('page.recipes.remove_ingredient')}
      >
        <Trash2 size={14} />
      </button>
    </div>
  {/if}
</div>

<style>
  .ingredient-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 5.25rem;
    grid-template-areas:
      'name amount'
      'unit options';
    align-items: center;
    gap: 0.5rem;
    padding: 0.52rem 0.55rem;
    border-radius: 0.85rem;
    transition:
      border-color 120ms ease,
      background-color 120ms ease,
      box-shadow 120ms ease,
      transform 120ms ease;
    box-shadow:
      inset 0 1px 0 hsl(var(--border) / 0.14),
      0 1px 1px hsl(var(--background) / 0.18);
  }

  .ingredient-row:hover {
    background: linear-gradient(
      180deg,
      hsl(var(--background) / 0.96) 0%,
      hsl(var(--background) / 0.86) 100%
    );
    box-shadow:
      inset 0 1px 0 hsl(var(--border) / 0.18),
      0 2px 6px hsl(var(--background) / 0.22);
  }

  .ingredient-row.optional {
    opacity: 0.7;
  }

  .ingredient-cell-grip {
    display: none;
    grid-area: grip;
    align-items: center;
    justify-content: center;
  }

  .ingredient-cell-amount {
    grid-area: amount;
    min-width: 0;
  }

  .ingredient-cell-unit {
    grid-area: unit;
    min-width: 0;
    position: relative;
  }

  .ingredient-cell-name {
    grid-area: name;
  }

  .ingredient-cell-options {
    grid-area: options;
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 0.35rem;
    min-width: 0;
  }

  .ingredient-input {
    width: 100%;
    padding: 0.5rem 0.55rem;
    font-size: 0.9rem;
    line-height: 1.2;
  }

  .row-unit-combobox {
    position: relative;
  }

  .row-unit-menu {
    position: absolute;
    top: calc(100% + 0.2rem);
    left: 0;
    right: 0;
    z-index: 15;
    display: grid;
    gap: 0.15rem;
    border-radius: 0.65rem;
    padding: 0.2rem;
    box-shadow: 0 8px 24px hsl(var(--background) / 0.3);
  }

  .row-unit-option {
    width: 100%;
    border-radius: 0.45rem;
    text-align: left;
    padding: 0.3rem 0.4rem;
    font-size: 0.82rem;
    justify-content: flex-start;
  }

  .row-unit-option:hover,
  .row-unit-option.active {
    background: hsl(var(--accent) / 0.6);
  }

  .option-chip {
    flex-shrink: 0;
    font-size: 0.76rem;
  }

  .remove-button:hover {
    border-color: hsl(var(--destructive) / 0.35);
  }

  @media (min-width: 768px) {
    .ingredient-row {
      grid-template-columns: 1.15rem 4.1rem 4.6rem minmax(0, 1fr) 3.6rem;
      grid-template-areas: 'grip amount unit name options';
      gap: 0.5rem;
      padding: 0.6rem;
    }

    .ingredient-cell-grip {
      display: flex;
    }

    .ingredient-cell-options {
      justify-content: flex-start;
    }

    .ingredient-row.readonly-row {
      grid-template-columns: 4.1rem 4.6rem minmax(0, 1fr);
      grid-template-areas: 'amount unit name';
    }

    .option-chip {
      justify-content: center;
      min-width: 0;
      width: auto;
      padding-inline: 0.4rem;
    }
  }

  @media (hover: none) {
    .remove-button {
      opacity: 1;
    }
  }

  @media (max-width: 767px) {
    .ingredient-row {
      grid-template-columns: minmax(0, 1fr) auto;
      grid-template-areas:
        'name amount'
        'unit options';
      row-gap: 0.45rem;
      column-gap: 0.45rem;
      padding: 0.62rem;
      border-radius: 0.9rem;
    }

    .ingredient-cell-options {
      justify-content: flex-end;
      width: 100%;
      gap: 0.3rem;
    }

    .remove-button {
      opacity: 1;
      padding: 0.4rem;
    }

    .option-chip {
      min-width: 2rem;
      justify-content: center;
      padding: 0.38rem 0.5rem;
      gap: 0.2rem;
    }

    .option-chip span {
      display: none;
    }
  }
</style>
