<script lang="ts">
  import { ChevronRight, Home } from 'lucide-svelte';
  import * as tree from '$lib/stores/tree.svelte';

  interface Props {
    folderPath: string;
    noteTitle?: string;
  }

  const { folderPath, noteTitle }: Props = $props();

  interface BreadcrumbItem {
    label: string;
    path: string;
    isFolder: boolean;
  }

  const items = $derived.by(() => {
    const result: BreadcrumbItem[] = [{ label: 'Home', path: '/', isFolder: true }];

    if (folderPath && folderPath !== '/') {
      const parts = folderPath.split('/').filter(Boolean);
      let currentPath = '';
      for (const part of parts) {
        currentPath += '/' + part;
        result.push({ label: part, path: currentPath, isFolder: true });
      }
    }

    if (noteTitle) {
      result.push({ label: noteTitle, path: '', isFolder: false });
    }
    return result;
  });

  function handleFolderClick(path: string) {
    tree.setSelectedFolder(path);
  }
</script>

<nav class="flex items-center gap-1 text-sm text-muted-foreground overflow-x-auto py-1 px-1">
  {#each items as item, i (item.path)}
    {#if i > 0}
      <ChevronRight size={14} class="opacity-50 flex-shrink-0" />
    {/if}
    {#if item.isFolder}
      <button
        onclick={() => handleFolderClick(item.path)}
        class="breadcrumb-btn hover:text-foreground truncate max-w-[150px] flex items-center gap-1 py-1 px-1"
        title={item.path}
      >
        {#if i === 0}
          <Home size={14} />
        {:else}
          {item.label}
        {/if}
      </button>
    {:else}
      <span class="text-foreground font-medium truncate max-w-[200px]" title={item.label}>
        {item.label}
      </span>
    {/if}
  {/each}
</nav>

<style>
  @media (pointer: coarse) {
    .breadcrumb-btn {
      min-height: 36px;
    }
  }
</style>
