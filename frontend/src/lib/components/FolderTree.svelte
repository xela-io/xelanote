<script lang="ts">
  import { ChevronDown, ChevronRight, Folder, FolderOpen } from 'lucide-svelte';

  import type { FolderNode } from '$lib/stores/folders.svelte';
  import * as folders from '$lib/stores/folders.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as toast from '$lib/stores/toast.svelte';

  import FolderTree from './FolderTree.svelte';

  const { node, depth = 0 }: { node: FolderNode; depth?: number } = $props();

  // Use $derived.by for proper dependency tracking
  const isExpanded = $derived.by(() => folders.isExpanded(node.path));
  const isSelected = $derived.by(() => folders.getSelectedFolder() === node.path);
  const hasChildren = $derived.by(() => node.children.length > 0);

  // Drag and drop state
  let isDragOver = $state(false);

  type DraggedNoteData = { id: string; title: string; folder_path: string };

  function parseDraggedNoteData(raw: string): DraggedNoteData | null {
    let parsed: unknown;
    try {
      parsed = JSON.parse(raw);
    } catch {
      return null;
    }

    if (!parsed || typeof parsed !== 'object') return null;
    const data = parsed as { id?: unknown; title?: unknown; folder_path?: unknown };
    if (
      typeof data.id !== 'string' ||
      typeof data.title !== 'string' ||
      typeof data.folder_path !== 'string'
    ) {
      return null;
    }
    return { id: data.id, title: data.title, folder_path: data.folder_path };
  }

  function handleExpandClick(e: MouseEvent) {
    e.stopPropagation();
    folders.toggleExpanded(node.path);
  }

  function handleFolderClick() {
    // Toggle: if already selected and it's root, deselect (null). Otherwise select this folder.
    if (node.path === '/' && isSelected) {
      folders.selectFolder(null);
    } else {
      folders.selectFolder(node.path);
    }
  }

  function handleDragOver(e: DragEvent) {
    // Only accept our custom note type
    if (e.dataTransfer?.types.includes('application/x-xelanote-note')) {
      e.preventDefault();
      e.dataTransfer.dropEffect = 'move';
      isDragOver = true;
    }
  }

  function handleDragLeave(e: DragEvent) {
    // Only reset if leaving this element (not entering a child)
    const relatedTarget = e.relatedTarget as Node | null;
    if (!e.currentTarget || !relatedTarget || !(e.currentTarget as Node).contains(relatedTarget)) {
      isDragOver = false;
    }
  }

  async function handleDrop(e: DragEvent) {
    e.preventDefault();
    isDragOver = false;

    const data = e.dataTransfer?.getData('application/x-xelanote-note');
    if (!data) return;

    try {
      const dragData = parseDraggedNoteData(data);
      if (!dragData) return;

      // Don't move if already in this folder
      if (dragData.folder_path === node.path) return;

      await notes.moveNote(dragData.id, node.path);
      await folders.loadFolders();
    } catch (err) {
      console.error('Failed to move note:', err);
      toast.error('Failed to move note');
    }
  }
</script>

<div class="folder-item" style="padding-left: {depth * 12}px">
  <div class="folder-row">
    {#if hasChildren}
      <button class="expand-button" onclick={handleExpandClick}>
        {#if isExpanded}
          <ChevronDown size={14} />
        {:else}
          <ChevronRight size={14} />
        {/if}
      </button>
    {:else}
      <span class="expand-spacer"></span>
    {/if}

    <button
      class="folder-button"
      class:selected={isSelected}
      class:drag-over={isDragOver}
      onclick={handleFolderClick}
      ondragover={handleDragOver}
      ondragleave={handleDragLeave}
      ondrop={handleDrop}
    >
      {#if isExpanded && hasChildren}
        <FolderOpen size={16} />
      {:else}
        <Folder size={16} />
      {/if}

      <span class="folder-name">{node.name}</span>

      {#if node.noteCount > 0}
        <span class="note-count">{node.noteCount}</span>
      {/if}
    </button>
  </div>
</div>

{#if isExpanded && hasChildren}
  {#each node.children as child (child.path)}
    <FolderTree node={child} depth={depth + 1} />
  {/each}
{/if}

<style>
  .folder-item {
    width: 100%;
  }

  .folder-row {
    display: flex;
    align-items: center;
    gap: 2px;
  }

  .folder-button {
    display: flex;
    align-items: center;
    gap: 4px;
    flex: 1;
    padding: 4px 8px;
    border: none;
    background: transparent;
    color: var(--color-sidebar-foreground);
    font-size: 13px;
    text-align: left;
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: background-color var(--duration-fast) var(--ease-default);
  }

  .folder-button:hover {
    background-color: var(--color-sidebar-accent);
  }

  .folder-button.selected {
    background-color: var(--color-sidebar-accent);
    font-weight: 500;
  }

  .folder-button.drag-over {
    background-color: var(--color-primary);
    color: var(--color-primary-foreground);
    outline: 2px solid var(--color-primary);
    outline-offset: -2px;
  }

  .expand-button {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    padding: 0;
    border: none;
    background: transparent;
    color: var(--color-muted-foreground);
    cursor: pointer;
    border-radius: var(--radius-xs);
    flex-shrink: 0;
  }

  .expand-button:hover {
    background-color: var(--color-sidebar-accent);
    color: var(--color-sidebar-foreground);
  }

  .expand-spacer {
    width: 18px;
    flex-shrink: 0;
  }

  .folder-name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .note-count {
    font-size: 11px;
    color: var(--color-muted-foreground);
    background-color: var(--color-sidebar-accent);
    padding: 1px 6px;
    border-radius: var(--radius-xl);
    flex-shrink: 0;
  }
</style>
