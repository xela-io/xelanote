<script lang="ts">
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import { getShoppingFeatureEnabled, getShoppingFeatureLoaded } from '$lib/stores/features.svelte';
  import { closeSidebarOnMobile } from '$lib/stores/ui.svelte';

  interface Props {
    iconOnly?: boolean;
  }

  const { iconOnly = false }: Props = $props();

  const isEnabled = $derived(getShoppingFeatureEnabled() && getShoppingFeatureLoaded());

  function handleClick() {
    goto('/shopping');
    closeSidebarOnMobile();
  }
</script>

{#if isEnabled}
  {#if iconOnly}
    <button
      class="p-2 rounded hover:bg-sidebar-accent text-sidebar-foreground"
      onclick={handleClick}
      title={$_('page.shopping.title')}
      aria-label={$_('page.shopping.title')}
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        ><circle cx="8" cy="21" r="1"></circle><circle cx="19" cy="21" r="1"></circle><path
          d="M2.05 2.05h2l2.66 12.42a2 2 0 0 0 2 1.58h9.78a2 2 0 0 0 1.95-1.57l1.65-7.43H5.12"
        ></path></svg
      >
    </button>
  {:else}
    <button class="shopping-button" onclick={handleClick} title={$_('page.shopping.title')}>
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="18"
        height="18"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        ><circle cx="8" cy="21" r="1"></circle><circle cx="19" cy="21" r="1"></circle><path
          d="M2.05 2.05h2l2.66 12.42a2 2 0 0 0 2 1.58h9.78a2 2 0 0 0 1.95-1.57l1.65-7.43H5.12"
        ></path></svg
      >
      <span class="shopping-label">{$_('page.shopping.title')}</span>
    </button>
  {/if}
{/if}

<style>
  .shopping-button {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.5rem 0.75rem;
    background: var(--color-surface-200);
    border: 1px solid var(--color-surface-300);
    border-radius: var(--radius-lg);
    color: var(--color-text);
    cursor: pointer;
    transition: all var(--duration-fast) var(--ease-default);
    font-size: var(--text-sm);
  }

  .shopping-button:hover {
    background: var(--color-surface-300);
  }

  .shopping-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
