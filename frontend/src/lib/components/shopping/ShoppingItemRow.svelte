<script lang="ts">
  import { Pencil, Star, Trash2 } from 'lucide-svelte';

  import type { ShoppingItem } from '$lib/api/types';

  interface Props {
    item: ShoppingItem;
    readonly?: boolean;
    oncheck: (itemId: number, isChecked: boolean) => void;
    ondelete: (itemId: number) => void;
    onedit: (item: ShoppingItem) => void;
    onfavorite?: (item: ShoppingItem) => void;
  }

  const { item, readonly = false, oncheck, ondelete, onedit, onfavorite }: Props = $props();

  function formatQuantity(item: ShoppingItem): string {
    const parts: string[] = [];
    if (item.quantity != null) {
      // Format nicely: 1.0 → "1", 0.5 → "0.5", 2.5 → "2.5"
      const q = item.quantity % 1 === 0 ? item.quantity.toFixed(0) : item.quantity.toString();
      if (item.unit) {
        parts.push(`${q}${item.unit}`);
      } else {
        parts.push(`${q}x`);
      }
    }
    return parts.join(' ');
  }

  const quantityDisplay = $derived(formatQuantity(item));
</script>

<div class="shopping-item-row" class:checked={item.is_checked}>
  <label class="shopping-item-checkbox-label">
    <input
      type="checkbox"
      checked={item.is_checked}
      onchange={() => oncheck(item.id, !item.is_checked)}
      disabled={readonly}
      class="shopping-item-checkbox"
    />
  </label>

  <div class="shopping-item-content" class:line-through={item.is_checked}>
    {#if quantityDisplay}
      <span class="shopping-item-qty">{quantityDisplay}</span>
    {/if}
    <span class="shopping-item-name">{item.name}</span>
  </div>

  {#if !readonly && !item.is_checked}
    <div class="shopping-item-actions">
      {#if onfavorite}
        <button
          type="button"
          class="shopping-item-action-btn"
          onclick={() => onfavorite(item)}
          aria-label="Add to favorites"
        >
          <Star size={14} />
        </button>
      {/if}
      <button
        type="button"
        class="shopping-item-action-btn"
        onclick={() => onedit(item)}
        aria-label="Edit"
      >
        <Pencil size={14} />
      </button>
      <button
        type="button"
        class="shopping-item-action-btn text-red-500"
        onclick={() => ondelete(item.id)}
        aria-label="Delete"
      >
        <Trash2 size={14} />
      </button>
    </div>
  {/if}
</div>

<style>
  .shopping-item-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0.75rem;
    transition: background-color 0.1s;
  }

  .shopping-item-row:hover {
    background: var(--color-surface-200);
  }

  .shopping-item-row.checked {
    opacity: 0.6;
  }

  .shopping-item-checkbox-label {
    display: flex;
    align-items: center;
    cursor: pointer;
  }

  .shopping-item-checkbox {
    width: 1.125rem;
    height: 1.125rem;
    accent-color: var(--color-primary);
    cursor: pointer;
  }

  .shopping-item-content {
    flex: 1;
    display: flex;
    align-items: baseline;
    gap: 0.375rem;
    min-width: 0;
    font-size: var(--text-sm);
  }

  .shopping-item-content.line-through {
    text-decoration: line-through;
    color: var(--color-text-muted);
  }

  .shopping-item-qty {
    color: var(--color-text-muted);
    font-weight: 500;
    white-space: nowrap;
    font-size: var(--text-xs);
  }

  .shopping-item-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .shopping-item-actions {
    display: flex;
    gap: 0.25rem;
    opacity: 0;
    transition: opacity 0.15s;
  }

  .shopping-item-row:hover .shopping-item-actions {
    opacity: 1;
  }

  .shopping-item-action-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0.25rem;
    border-radius: var(--radius-sm);
    color: var(--color-text-muted);
    cursor: pointer;
    transition:
      color 0.1s,
      background-color 0.1s;
  }

  .shopping-item-action-btn:hover {
    background: var(--color-surface-300);
    color: var(--color-text);
  }
</style>
