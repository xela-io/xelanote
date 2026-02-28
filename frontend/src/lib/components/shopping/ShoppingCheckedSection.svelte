<script lang="ts">
  import { ChevronDown, ChevronRight, Trash2 } from 'lucide-svelte';
  import type { Snippet } from 'svelte';
  import { _ } from 'svelte-i18n';

  interface Props {
    count: number;
    onclear: () => void;
    children: Snippet;
  }

  const { count, onclear, children }: Props = $props();
  let expanded = $state(false);
</script>

{#if count > 0}
  <div class="shopping-checked-section">
    <div class="shopping-checked-header">
      <button type="button" class="shopping-checked-toggle" onclick={() => (expanded = !expanded)}>
        {#if expanded}
          <ChevronDown size={16} />
        {:else}
          <ChevronRight size={16} />
        {/if}
        <span>{$_('page.shopping.checked_items', { values: { count } })}</span>
      </button>

      <button
        type="button"
        class="shopping-clear-btn"
        onclick={onclear}
        title={$_('page.shopping.clear_checked')}
      >
        <Trash2 size={14} />
        <span class="hidden sm:inline">{$_('page.shopping.clear_checked')}</span>
      </button>
    </div>

    {#if expanded}
      <div class="shopping-checked-items">
        {@render children()}
      </div>
    {/if}
  </div>
{/if}

<style>
  .shopping-checked-section {
    border-top: 1px solid var(--color-surface-300);
  }

  .shopping-checked-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.5rem 0.75rem;
  }

  .shopping-checked-toggle {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    cursor: pointer;
  }

  .shopping-checked-toggle:hover {
    color: var(--color-text);
  }

  .shopping-clear-btn {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    padding: 0.25rem 0.5rem;
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all 0.15s;
  }

  .shopping-clear-btn:hover {
    background: var(--color-surface-200);
    color: var(--color-error, #ef4444);
  }

  .shopping-checked-items {
    opacity: 0.6;
  }
</style>
