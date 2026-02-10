<script lang="ts">
  import { _ } from 'svelte-i18n';
  import { goto } from '$app/navigation';
  import { getJournalFeatureEnabled, getJournalFeatureLoaded } from '$lib/stores/features.svelte';
  import { closeSidebarOnMobile } from '$lib/stores/ui.svelte';

  const isEnabled = $derived(getJournalFeatureEnabled() && getJournalFeatureLoaded());

  function handleClick() {
    goto('/journal');
    closeSidebarOnMobile();
  }
</script>

{#if isEnabled}
  <button class="journal-button" onclick={handleClick} title={$_('page.journal.title')}>
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
    >
      <rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect>
      <line x1="16" y1="2" x2="16" y2="6"></line>
      <line x1="8" y1="2" x2="8" y2="6"></line>
      <line x1="3" y1="10" x2="21" y2="10"></line>
    </svg>
    <span class="journal-label">{$_('page.journal.title')}</span>
  </button>
{/if}

<style>
  .journal-button {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.5rem 0.75rem;
    background: var(--color-surface-200);
    border: 1px solid var(--color-surface-300);
    border-radius: 0.5rem;
    color: var(--color-text);
    cursor: pointer;
    transition: all 0.15s ease;
    font-size: 0.875rem;
  }

  .journal-button:hover {
    background: var(--color-surface-300);
  }

  .journal-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
