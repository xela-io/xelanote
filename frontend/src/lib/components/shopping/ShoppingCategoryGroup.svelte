<script lang="ts">
  import { ChevronDown, ChevronRight, Plus } from 'lucide-svelte';
  import type { Snippet } from 'svelte';
  import { _ } from 'svelte-i18n';

  import type { ShoppingItem } from '$lib/api/types';
  import type { ParsedItem } from '$lib/utils/shopping-parser';
  import { parseShoppingInput } from '$lib/utils/shopping-parser';

  interface Props {
    category: string;
    items: ShoppingItem[];
    canEdit?: boolean;
    onadditem?: (items: ParsedItem[]) => void;
    children: Snippet;
  }

  const { category, items, canEdit = false, onadditem, children }: Props = $props();
  let collapsed = $state(false);
  let showInput = $state(false);
  let inputValue = $state('');

  function autofocus(node: HTMLElement) {
    node.focus();
  }

  function handleSubmit() {
    const parsed = parseShoppingInput(inputValue);
    if (parsed.length === 0) return;
    onadditem?.(parsed);
    inputValue = '';
    showInput = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
    if (e.key === 'Escape') {
      showInput = false;
      inputValue = '';
    }
  }
</script>

<div class="shopping-category-group">
  <button type="button" class="shopping-category-header" onclick={() => (collapsed = !collapsed)}>
    {#if collapsed}
      <ChevronRight size={16} />
    {:else}
      <ChevronDown size={16} />
    {/if}
    <span class="shopping-category-name">{category}</span>
    <span class="shopping-category-count">{items.length}</span>
  </button>

  {#if !collapsed}
    <div class="shopping-category-items">
      {@render children()}
      {#if canEdit && onadditem}
        {#if items.length === 0 || showInput}
          <div class="category-inline-input">
            <input
              type="text"
              bind:value={inputValue}
              onkeydown={handleKeydown}
              placeholder={$_('page.shopping.add_to_category')}
              class="category-inline-field"
              use:autofocus
            />
            <button
              type="button"
              class="category-inline-btn"
              onclick={handleSubmit}
              disabled={!inputValue.trim()}
            >
              <Plus size={14} />
            </button>
          </div>
        {:else}
          <button type="button" class="category-add-btn" onclick={() => (showInput = true)}>
            <Plus size={14} />
            <span>{$_('page.shopping.add_to_category')}</span>
          </button>
        {/if}
      {/if}
    </div>
  {/if}
</div>

<style>
  .shopping-category-group {
    border-bottom: 1px solid var(--color-surface-200);
  }

  .shopping-category-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.5rem 0.75rem;
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--color-text-muted);
    cursor: pointer;
    transition: background-color 0.1s;
  }

  .shopping-category-header:hover {
    background: var(--color-surface-100);
  }

  .shopping-category-name {
    flex: 1;
    text-align: left;
  }

  .shopping-category-count {
    font-size: var(--text-xs);
    font-weight: 400;
    color: var(--color-text-muted);
    opacity: 0.7;
  }

  .shopping-category-items {
    padding-left: 0.25rem;
  }

  .category-inline-input {
    display: flex;
    gap: 0.375rem;
    padding: 0.375rem 0.75rem 0.5rem;
  }

  .category-inline-field {
    flex: 1;
    padding: 0.25rem 0.5rem;
    border: 1px solid var(--color-surface-300);
    border-radius: var(--radius-sm);
    background: var(--color-surface-100);
    color: var(--color-text);
    font-size: var(--text-xs);
  }

  .category-inline-field:focus {
    outline: none;
    border-color: var(--color-primary);
  }

  .category-inline-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0.25rem;
    border-radius: var(--radius-sm);
    color: var(--color-primary);
    cursor: pointer;
  }

  .category-inline-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .category-add-btn {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    padding: 0.375rem 0.75rem;
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    opacity: 0.6;
    cursor: pointer;
    transition: opacity 0.1s;
  }

  .category-add-btn:hover {
    opacity: 1;
  }
</style>
