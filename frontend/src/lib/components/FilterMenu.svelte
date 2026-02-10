<script lang="ts">
  import { Calendar, Folder, Tag } from 'lucide-svelte';
  import { onMount } from 'svelte';

  import * as api from '$lib/api';
  import * as foldersStore from '$lib/stores/folders.svelte';
  import type { DateFilter } from '$lib/stores/search.svelte';
  import * as searchStore from '$lib/stores/search.svelte';

  interface Props {
    isOpen: boolean;
    onClose: () => void;
  }

  const { isOpen, onClose }: Props = $props();

  // State
  let tags = $state<api.Tag[]>([]);
  let loadingTags = $state(false);
  let selectedSection = $state<'folders' | 'tags' | 'date'>('folders');

  const folders = $derived(foldersStore.getFolders());
  const currentFilters = $derived(searchStore.getFilters());

  onMount(async () => {
    await loadTags();
  });

  async function loadTags() {
    loadingTags = true;
    try {
      tags = await api.getTags();
    } catch (err) {
      console.error('Failed to load tags:', err);
    } finally {
      loadingTags = false;
    }
  }

  function toggleFolder(folderPath: string) {
    if (currentFilters.folders.includes(folderPath)) {
      searchStore.removeFolderFilter(folderPath);
    } else {
      searchStore.addFolderFilter(folderPath);
    }
  }

  function toggleTag(tagName: string) {
    if (currentFilters.tags.includes(tagName)) {
      searchStore.removeTagFilter(tagName);
    } else {
      searchStore.addTagFilter(tagName);
    }
  }

  function setDatePreset(type: 'created' | 'updated', preset: DateFilter['preset']) {
    const dateFilter = { type: 'relative' as const, preset };
    if (type === 'created') {
      searchStore.setCreatedDateFilter(dateFilter);
    } else {
      searchStore.setUpdatedDateFilter(dateFilter);
    }
  }
</script>

{#if isOpen}
  <!-- Backdrop -->
  <button type="button" class="fixed inset-0 z-40" onclick={onClose} aria-label="Close filter menu"
  ></button>

  <!-- Menu -->
  <div
    class="absolute top-full left-0 right-0 mt-2 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-xl z-50 max-h-96 overflow-hidden flex flex-col"
  >
    <!-- Tabs -->
    <div class="flex border-b border-border">
      <button
        type="button"
        class="flex-1 flex items-center justify-center gap-2 px-4 py-3 text-base md:text-sm font-medium transition-colors {selectedSection ===
        'folders'
          ? 'text-primary border-b-2 border-primary'
          : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => (selectedSection = 'folders')}
      >
        <Folder size={16} />
        Folders
      </button>
      <button
        type="button"
        class="flex-1 flex items-center justify-center gap-2 px-4 py-3 text-base md:text-sm font-medium transition-colors {selectedSection ===
        'tags'
          ? 'text-success border-b-2 border-success'
          : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => (selectedSection = 'tags')}
      >
        <Tag size={16} />
        Tags
      </button>
      <button
        type="button"
        class="flex-1 flex items-center justify-center gap-2 px-4 py-3 text-base md:text-sm font-medium transition-colors {selectedSection ===
        'date'
          ? 'text-purple-600 dark:text-purple-400 border-b-2 border-purple-600 dark:border-purple-400'
          : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => (selectedSection = 'date')}
      >
        <Calendar size={16} />
        Date
      </button>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-y-auto p-4">
      {#if selectedSection === 'folders'}
        <div class="space-y-1">
          {#if folders.length === 0}
            <p class="text-base md:text-sm text-gray-500 dark:text-gray-400">
              No folders available
            </p>
          {:else}
            {#each folders as folder (folder.path)}
              <label
                class="flex items-center gap-2 px-2 py-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded cursor-pointer"
              >
                <input
                  type="checkbox"
                  checked={currentFilters.folders.includes(folder.path)}
                  onchange={() => toggleFolder(folder.path)}
                  class="rounded"
                />
                <Folder size={14} class="text-primary" />
                <span class="text-base md:text-sm text-gray-700 dark:text-gray-300"
                  >{folder.path}</span
                >
                <span class="ml-auto text-xs text-gray-500 dark:text-gray-400"
                  >({folder.note_count})</span
                >
              </label>
            {/each}
          {/if}
        </div>
      {:else if selectedSection === 'tags'}
        <div class="space-y-1">
          {#if loadingTags}
            <p class="text-base md:text-sm text-gray-500 dark:text-gray-400">Loading tags...</p>
          {:else if tags.length === 0}
            <p class="text-base md:text-sm text-gray-500 dark:text-gray-400">
              No tags yet. Add tags to your notes to see them here.
            </p>
          {:else}
            {#each tags as tag (tag.name)}
              <label
                class="flex items-center gap-2 px-2 py-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded cursor-pointer"
              >
                <input
                  type="checkbox"
                  checked={currentFilters.tags.includes(tag.name)}
                  onchange={() => toggleTag(tag.name)}
                  class="rounded"
                />
                <Tag size={14} class="text-success" />
                <span class="text-base md:text-sm text-gray-700 dark:text-gray-300">{tag.name}</span
                >
              </label>
            {/each}
          {/if}
        </div>
      {:else if selectedSection === 'date'}
        <div class="space-y-4">
          <!-- Created Date Presets -->
          <div>
            <h3 class="text-xs font-medium text-gray-500 dark:text-gray-400 mb-2">Created Date</h3>
            <div class="grid grid-cols-2 gap-2">
              <button
                type="button"
                class="px-3 py-2 text-base md:text-sm text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
                onclick={() => setDatePreset('created', 'today')}
              >
                Today
              </button>
              <button
                type="button"
                class="px-3 py-2 text-base md:text-sm text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
                onclick={() => setDatePreset('created', 'yesterday')}
              >
                Yesterday
              </button>
              <button
                type="button"
                class="px-3 py-2 text-base md:text-sm text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
                onclick={() => setDatePreset('created', 'last7days')}
              >
                Last 7 days
              </button>
              <button
                type="button"
                class="px-3 py-2 text-base md:text-sm text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
                onclick={() => setDatePreset('created', 'last30days')}
              >
                Last 30 days
              </button>
              <button
                type="button"
                class="px-3 py-2 text-base md:text-sm text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
                onclick={() => setDatePreset('created', 'thisWeek')}
              >
                This week
              </button>
              <button
                type="button"
                class="px-3 py-2 text-base md:text-sm text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
                onclick={() => setDatePreset('created', 'thisMonth')}
              >
                This month
              </button>
            </div>
          </div>

          <!-- Updated Date Presets -->
          <div>
            <h3 class="text-xs font-medium text-gray-500 dark:text-gray-400 mb-2">Updated Date</h3>
            <div class="grid grid-cols-2 gap-2">
              <button
                type="button"
                class="px-3 py-2 text-base md:text-sm text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
                onclick={() => setDatePreset('updated', 'today')}
              >
                Today
              </button>
              <button
                type="button"
                class="px-3 py-2 text-base md:text-sm text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
                onclick={() => setDatePreset('updated', 'yesterday')}
              >
                Yesterday
              </button>
              <button
                type="button"
                class="px-3 py-2 text-base md:text-sm text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
                onclick={() => setDatePreset('updated', 'last7days')}
              >
                Last 7 days
              </button>
              <button
                type="button"
                class="px-3 py-2 text-base md:text-sm text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
                onclick={() => setDatePreset('updated', 'last30days')}
              >
                Last 30 days
              </button>
              <button
                type="button"
                class="px-3 py-2 text-base md:text-sm text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
                onclick={() => setDatePreset('updated', 'thisWeek')}
              >
                This week
              </button>
              <button
                type="button"
                class="px-3 py-2 text-base md:text-sm text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
                onclick={() => setDatePreset('updated', 'thisMonth')}
              >
                This month
              </button>
            </div>
          </div>
        </div>
      {/if}
    </div>
  </div>
{/if}
