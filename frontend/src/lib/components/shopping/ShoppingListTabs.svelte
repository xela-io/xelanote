<script lang="ts">
  import { Plus } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { ShoppingListSummary } from '$lib/api/types';

  interface Props {
    lists: ShoppingListSummary[];
    activeListId: number | null;
    onselect: (listId: number) => void;
    oncreate: () => void;
  }

  const { lists, activeListId, onselect, oncreate }: Props = $props();
</script>

<div class="shopping-tabs">
  <div class="shopping-tabs-scroll">
    {#each lists as list (list.id)}
      <button
        type="button"
        class="shopping-tab"
        class:active={list.id === activeListId}
        onclick={() => onselect(list.id)}
        style={list.color ? `--tab-color: ${list.color}` : ''}
      >
        {#if list.color}
          <span class="tab-color-dot" style="background-color: {list.color}"></span>
        {/if}
        <span class="tab-name">{list.name}</span>
        {#if list.item_count > 0}
          <span class="tab-count">
            {list.item_count - list.checked_count}/{list.item_count}
          </span>
        {/if}
        {#if list.shared_by}
          <span
            class="tab-shared"
            title={$_('page.shopping.shared_by', { values: { name: list.shared_by } })}
          >
            &middot;
          </span>
        {/if}
      </button>
    {/each}
  </div>

  <button
    type="button"
    class="shopping-tab-add"
    onclick={oncreate}
    title={$_('page.shopping.new_list')}
  >
    <Plus size={16} />
  </button>
</div>

<style>
  .shopping-tabs {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.5rem 0.75rem;
    border-bottom: 1px solid var(--color-surface-300);
    overflow: hidden;
  }

  .shopping-tabs-scroll {
    display: flex;
    gap: 0.25rem;
    overflow-x: auto;
    flex: 1;
    scrollbar-width: none;
  }

  .shopping-tabs-scroll::-webkit-scrollbar {
    display: none;
  }

  .shopping-tab {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    padding: 0.375rem 0.75rem;
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
    white-space: nowrap;
    color: var(--color-text-muted);
    background: transparent;
    cursor: pointer;
    transition: all 0.15s;
    border: 1px solid transparent;
  }

  .shopping-tab:hover {
    background: var(--color-surface-200);
  }

  .shopping-tab.active {
    background: var(--color-surface-200);
    color: var(--color-text);
    border-color: var(--color-surface-400, var(--color-surface-300));
    font-weight: 500;
  }

  .tab-color-dot {
    width: 0.5rem;
    height: 0.5rem;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .tab-name {
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 8rem;
  }

  .tab-count {
    font-size: var(--text-xs);
    opacity: 0.7;
  }

  .tab-shared {
    font-size: var(--text-xs);
    opacity: 0.5;
  }

  .shopping-tab-add {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0.375rem;
    border-radius: var(--radius-md);
    color: var(--color-text-muted);
    cursor: pointer;
    transition: all 0.15s;
    flex-shrink: 0;
  }

  .shopping-tab-add:hover {
    background: var(--color-surface-200);
    color: var(--color-text);
  }
</style>
