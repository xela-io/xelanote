<script lang="ts">
  import FilterChip from './FilterChip.svelte';
  import * as searchStore from '$lib/stores/search.svelte';
  import type { DateFilter } from '$lib/stores/search.svelte';
  import { X } from 'lucide-svelte';

  const folders = $derived(searchStore.getFolders());
  const tags = $derived(searchStore.getTags());
  const createdDate = $derived(searchStore.getCreatedDate());
  const updatedDate = $derived(searchStore.getUpdatedDate());

  function formatDateFilter(dateFilter: DateFilter): string {
    if (!dateFilter) return '';

    if (dateFilter.type === 'relative') {
      switch (dateFilter.preset) {
        case 'today':
          return 'Today';
        case 'yesterday':
          return 'Yesterday';
        case 'last7days':
          return 'Last 7 days';
        case 'last30days':
          return 'Last 30 days';
        case 'thisWeek':
          return 'This week';
        case 'thisMonth':
          return 'This month';
        default:
          return '';
      }
    } else if (dateFilter.type === 'absolute') {
      const parts = [];
      if (dateFilter.after) {
        const date = new Date(dateFilter.after);
        parts.push(`After ${date.toLocaleDateString()}`);
      }
      if (dateFilter.before) {
        const date = new Date(dateFilter.before);
        parts.push(`Before ${date.toLocaleDateString()}`);
      }
      return parts.join(' - ');
    }

    return '';
  }
</script>

{#if searchStore.hasActiveFilters()}
  <div
    class="flex items-center gap-2 flex-wrap px-4 py-2 bg-gray-50 dark:bg-gray-900/50 border-t border-gray-200 dark:border-gray-700"
  >
    <span class="text-xs text-gray-500 dark:text-gray-400 font-medium">Filters:</span>

    {#each folders as folder (folder)}
      <FilterChip
        type="folder"
        label={folder}
        onRemove={() => searchStore.removeFolderFilter(folder)}
      />
    {/each}

    {#each tags as tag (tag)}
      <FilterChip type="tag" label={tag} onRemove={() => searchStore.removeTagFilter(tag)} />
    {/each}

    {#if createdDate}
      <FilterChip
        type="date"
        label="Created: {formatDateFilter(createdDate)}"
        onRemove={() => searchStore.setCreatedDateFilter(undefined)}
      />
    {/if}

    {#if updatedDate}
      <FilterChip
        type="date"
        label="Updated: {formatDateFilter(updatedDate)}"
        onRemove={() => searchStore.setUpdatedDateFilter(undefined)}
      />
    {/if}

    <button
      type="button"
      onclick={() => searchStore.clearAllFilters()}
      class="ml-2 inline-flex items-center gap-1 px-2 py-1 text-xs text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-800 rounded transition-colors"
    >
      <X size={12} />
      <span>Clear all</span>
    </button>
  </div>
{/if}
