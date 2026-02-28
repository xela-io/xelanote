<script lang="ts">
  import { ChevronDown, ChevronRight } from 'lucide-svelte';
  import type { Snippet } from 'svelte';

  import type { ShoppingItem } from '$lib/api/types';

  interface Props {
    category: string;
    items: ShoppingItem[];
    children: Snippet;
  }

  const { category, items, children }: Props = $props();
  let collapsed = $state(false);
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
</style>
