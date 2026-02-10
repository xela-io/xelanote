<script lang="ts">
  import { _ } from 'svelte-i18n';
  import { goto } from '$app/navigation';
  import { getRecipeFeatureEnabled, getRecipeFeatureLoaded } from '$lib/stores/features.svelte';

  const isEnabled = $derived(getRecipeFeatureEnabled() && getRecipeFeatureLoaded());

  function handleClick() {
    goto('/recipes');
  }
</script>

{#if isEnabled}
  <button class="recipe-button" onclick={handleClick} title={$_('page.recipes.title')}>
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
      <path
        d="M6 13.87A4 4 0 0 1 7.41 6a5.11 5.11 0 0 1 1.05-1.54 5 5 0 0 1 7.08 0A5.11 5.11 0 0 1 16.59 6 4 4 0 0 1 18 13.87V21H6Z"
      ></path>
      <line x1="6" y1="17" x2="18" y2="17"></line>
    </svg>
    <span class="recipe-label">{$_('page.recipes.title')}</span>
  </button>
{/if}

<style>
  .recipe-button {
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

  .recipe-button:hover {
    background: var(--color-surface-300);
  }

  .recipe-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
