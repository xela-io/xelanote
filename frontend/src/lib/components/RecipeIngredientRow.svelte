<script lang="ts">
  import { GripVertical, Trash2 } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { RecipeIngredient } from '$lib/api';

  interface Props {
    ingredient: RecipeIngredient;
    readonly?: boolean;
    scaled?: { display_amount: string } | null;
    onupdate: (ingredient: RecipeIngredient) => void;
    onremove: () => void;
  }

  const { ingredient, readonly = false, scaled = null, onupdate, onremove }: Props = $props();

  function handleChange(field: string, value: unknown) {
    onupdate({ ...ingredient, [field]: value });
  }

  function handleAmountChange(e: Event) {
    const val = (e.target as HTMLInputElement).value;
    handleChange('amount', val ? parseFloat(val) : null);
  }
</script>

<div class="ingredient-row group" class:optional={ingredient.optional}>
  {#if !readonly}
    <div class="grip cursor-grab text-muted-foreground opacity-0 group-hover:opacity-100">
      <GripVertical size={14} />
    </div>
  {/if}

  <div class="flex-1 flex items-center gap-2">
    <!-- Amount -->
    <div class="w-16 shrink-0">
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
          class="w-full px-1.5 py-1 text-sm bg-background border border-border rounded focus:outline-none focus:ring-1 focus:ring-ring"
        />
      {/if}
    </div>

    <!-- Unit -->
    <div class="w-16 shrink-0">
      {#if readonly}
        <span class="text-sm text-muted-foreground">{ingredient.unit ?? ''}</span>
      {:else}
        <input
          type="text"
          value={ingredient.unit ?? ''}
          oninput={(e) => handleChange('unit', (e.target as HTMLInputElement).value || null)}
          placeholder={$_('page.recipes.unit')}
          class="w-full px-1.5 py-1 text-sm bg-background border border-border rounded focus:outline-none focus:ring-1 focus:ring-ring"
        />
      {/if}
    </div>

    <!-- Name -->
    <div class="flex-1">
      {#if readonly}
        <span class="text-sm" class:italic={ingredient.optional}>
          {ingredient.name}
          {#if ingredient.optional}
            <span class="text-xs text-muted-foreground">({$_('page.recipes.optional')})</span>
          {/if}
        </span>
      {:else}
        <input
          type="text"
          value={ingredient.name}
          oninput={(e) => handleChange('name', (e.target as HTMLInputElement).value)}
          placeholder={$_('page.recipes.ingredient_name')}
          class="w-full px-1.5 py-1 text-sm bg-background border border-border rounded focus:outline-none focus:ring-1 focus:ring-ring"
        />
      {/if}
    </div>

    {#if !readonly}
      <!-- Optional Toggle -->
      <label
        class="flex items-center gap-1 text-xs text-muted-foreground shrink-0"
        title={$_('page.recipes.optional')}
      >
        <input
          type="checkbox"
          checked={ingredient.optional}
          onchange={(e) => handleChange('optional', (e.target as HTMLInputElement).checked)}
          class="w-3 h-3"
        />
        opt
      </label>

      <!-- Scalable Toggle -->
      <label
        class="flex items-center gap-1 text-xs text-muted-foreground shrink-0"
        title={$_('page.recipes.scalable')}
      >
        <input
          type="checkbox"
          checked={ingredient.scalable}
          onchange={(e) => handleChange('scalable', (e.target as HTMLInputElement).checked)}
          class="w-3 h-3"
        />
        scl
      </label>

      <!-- Remove -->
      <button
        onclick={onremove}
        class="remove-button p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 shrink-0"
        title={$_('page.recipes.remove_ingredient')}
      >
        <Trash2 size={14} />
      </button>
    {/if}
  </div>
</div>

<style>
  .ingredient-row {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.25rem 0;
    border-bottom: 1px solid var(--color-border, hsl(var(--border)));
  }

  .ingredient-row:last-child {
    border-bottom: none;
  }

  .ingredient-row.optional {
    opacity: 0.7;
  }

  @media (hover: none) {
    .remove-button {
      opacity: 1;
    }
  }
</style>
