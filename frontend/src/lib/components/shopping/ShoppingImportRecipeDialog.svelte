<script lang="ts">
  import { Loader2, X } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { getRecipes } from '$lib/api/recipes';
  import type { RecipeListItem } from '$lib/api/types';

  interface Props {
    open: boolean;
    onimport: (recipeNoteId: string) => void;
    onclose: () => void;
  }

  const { open, onimport, onclose }: Props = $props();

  let recipes = $state<RecipeListItem[]>([]);
  let loading = $state(false);
  let search = $state('');

  const filtered = $derived(
    search ? recipes.filter((r) => r.title.toLowerCase().includes(search.toLowerCase())) : recipes
  );

  $effect(() => {
    if (open && recipes.length === 0) {
      loadRecipes();
    }
  });

  async function loadRecipes() {
    loading = true;
    try {
      recipes = await getRecipes();
    } catch (error) {
      console.error('Failed to load recipes:', error);
    } finally {
      loading = false;
    }
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="dialog-backdrop" onmousedown={onclose}>
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="dialog-content" onmousedown={(e) => e.stopPropagation()}>
      <div class="dialog-header">
        <h2 class="dialog-title">{$_('page.shopping.import_recipe')}</h2>
        <button type="button" class="dialog-close" onclick={onclose}>
          <X size={18} />
        </button>
      </div>

      <div class="dialog-body">
        <input
          type="text"
          bind:value={search}
          placeholder={$_('page.shopping.select_recipe')}
          class="recipe-search"
        />

        <div class="recipe-list">
          {#if loading}
            <div class="recipe-loading">
              <Loader2 size={20} class="animate-spin" />
            </div>
          {:else if filtered.length === 0}
            <p class="recipe-empty">Keine Rezepte gefunden</p>
          {:else}
            {#each filtered as recipe (recipe.id)}
              <button type="button" class="recipe-item" onclick={() => onimport(recipe.id)}>
                <span class="recipe-title">{recipe.title}</span>
              </button>
            {/each}
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  .dialog-backdrop {
    position: fixed;
    inset: 0;
    z-index: 50;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.5);
  }

  .dialog-content {
    background: var(--color-surface-100);
    border: 1px solid var(--color-surface-300);
    border-radius: var(--radius-lg);
    width: 100%;
    max-width: 28rem;
    max-height: 80vh;
    margin: 1rem;
    display: flex;
    flex-direction: column;
  }

  .dialog-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1rem 1.25rem;
    border-bottom: 1px solid var(--color-surface-300);
  }

  .dialog-title {
    font-size: var(--text-base);
    font-weight: 600;
  }

  .dialog-close {
    color: var(--color-text-muted);
    cursor: pointer;
  }

  .dialog-body {
    padding: 1rem 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    overflow: hidden;
  }

  .recipe-search {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--color-surface-300);
    border-radius: var(--radius-md);
    background: var(--color-surface-100);
    color: var(--color-text);
    font-size: var(--text-sm);
  }

  .recipe-list {
    overflow-y: auto;
    max-height: 20rem;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .recipe-loading {
    display: flex;
    justify-content: center;
    padding: 2rem;
    color: var(--color-text-muted);
  }

  .recipe-empty {
    text-align: center;
    padding: 2rem;
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }

  .recipe-item {
    display: flex;
    align-items: center;
    padding: 0.5rem 0.75rem;
    border-radius: var(--radius-md);
    cursor: pointer;
    transition: background-color 0.1s;
    text-align: left;
  }

  .recipe-item:hover {
    background: var(--color-surface-200);
  }

  .recipe-title {
    font-size: var(--text-sm);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
