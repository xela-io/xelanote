<script lang="ts">
  import {
    ChevronDown,
    ChevronRight,
    FileText,
    Folder,
    FolderOpen,
    LayoutDashboard,
    MoreVertical,
    Sparkles,
  } from 'lucide-svelte';
  import type { ComponentType } from 'svelte';
  import { createEventDispatcher } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import * as api from '$lib/api';
  import * as dialog from '$lib/stores/dialog.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as toast from '$lib/stores/toast.svelte';
  import * as trash from '$lib/stores/trash.svelte';
  import type { FolderTreeNode, NoteTreeNode, TreeNode } from '$lib/stores/tree.svelte';
  import * as tree from '$lib/stores/tree.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import { loadSvelteComponentFromModule } from '$lib/utils/lazy-component';

  import TreeContextMenu from './TreeContextMenu.svelte';
  import UnifiedTree from './UnifiedTree.svelte';
  const {
    node,
    depth = 0,
    isVirtualized = false,
  }: { node: TreeNode; depth?: number; isVirtualized?: boolean } = $props();

  // Event dispatcher for virtual tree scroll restoration
  const dispatch = createEventDispatcher();

  // Tree-internal drag & drop (reorder/move) is disabled in virtual mode and on mobile.
  const treeDragEnabled = $derived(!isVirtualized && !ui.getIsMobile());
  // External note drag (e.g. to Canvas) should still work on desktop in virtual mode.
  const noteExternalDragEnabled = $derived(!ui.getIsMobile());

  // Responsive icon sizes: 18px on mobile (< 640px), 16px on desktop
  const folderIconSize = $derived(ui.getIsMobile() ? 18 : 16);
  const chevronIconSize = $derived(ui.getIsMobile() ? 16 : 14);
  const actionIconSize = $derived(ui.getIsMobile() ? 16 : 14);

  // Drag & Drop state
  let isDragging = $state(false);
  let isDragOver = $state(false);
  let dropPosition = $state<'before' | 'after' | 'into' | null>(null);

  type DraggedTreeItem =
    | { type: 'folder'; id: number; path: string }
    | { type: 'note'; id: string; title: string; folder_path: string };

  function parseDraggedTreeItem(raw: string): DraggedTreeItem | null {
    let parsed: unknown;
    try {
      parsed = JSON.parse(raw);
    } catch {
      return null;
    }

    if (!parsed || typeof parsed !== 'object') return null;
    const data = parsed as {
      type?: unknown;
      id?: unknown;
      path?: unknown;
      title?: unknown;
      folder_path?: unknown;
    };

    if (data.type === 'folder') {
      if (typeof data.id !== 'number' || typeof data.path !== 'string') return null;
      return { type: 'folder', id: data.id, path: data.path };
    }
    if (data.type === 'note') {
      if (
        typeof data.id !== 'string' ||
        typeof data.title !== 'string' ||
        typeof data.folder_path !== 'string'
      ) {
        return null;
      }
      return { type: 'note', id: data.id, title: data.title, folder_path: data.folder_path };
    }
    return null;
  }

  // Rename dialog state
  let showRenameDialog = $state(false);
  let RenameFolderDialogComponent = $state<ComponentType | null>(null);

  // Delete dialog state
  let showDeleteDialog = $state(false);
  let DeleteFolderDialogComponent = $state<ComponentType | null>(null);

  // Color picker dialog state
  let showColorPicker = $state(false);
  let ColorPickerDialogComponent = $state<ComponentType | null>(null);

  // Share dialog state (unified for note + folder)
  let showShareDialog = $state(false);
  let ShareDialogComponent = $state<ComponentType | null>(null);

  // Rename note dialog state
  let showRenameNoteDialog = $state(false);
  let RenameNoteDialogComponent = $state<ComponentType | null>(null);

  // Derived selection state
  const isSelected = $derived.by(() => {
    if (node.type === 'folder') {
      return tree.getSelectedFolderPath() === node.path;
    } else {
      return tree.getSelectedNoteId() === node.id;
    }
  });

  // Click handler
  function handleClick() {
    if (node.type === 'folder') {
      tree.selectFolder(node.path);
    } else {
      tree.selectNote(node.id);
      // Navigate to note and close sidebar on mobile
      goto(`/note/${node.id}`);
      ui.closeSidebarOnMobile();
    }
  }

  // Expand/Collapse handler
  function handleExpandClick(e: MouseEvent) {
    e.stopPropagation();
    if (node.type === 'folder') {
      tree.toggleExpanded(node.path);
      // Dispatch expand event for virtual tree scroll restoration (Phase 4)
      dispatch('expand', { nodeId: node.id, nodePath: node.path });
    }
  }

  // Context menu state
  let showContextMenu = $state(false);
  let contextMenuPosition = $state({ x: 0, y: 0 });
  let contextMenuTrigger = $state<HTMLElement | null>(null);

  function handleContextMenu(e: MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    if (node.type === 'folder' && node.path === '/') return;
    contextMenuPosition = { x: e.clientX, y: e.clientY };
    contextMenuTrigger = e.currentTarget as HTMLElement;
    showContextMenu = true;
  }

  function handleKebabClick(e: MouseEvent) {
    e.stopPropagation();
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    contextMenuPosition = { x: rect.right, y: rect.bottom + 4 };
    contextMenuTrigger = e.currentTarget as HTMLElement;
    showContextMenu = true;
  }

  function closeContextMenu() {
    showContextMenu = false;
    contextMenuTrigger?.focus();
    contextMenuTrigger = null;
  }

  function handleRowKeydown(e: KeyboardEvent) {
    if (e.key === 'F10' && e.shiftKey) {
      e.preventDefault();
      if (node.type === 'folder' && node.path === '/') return;
      const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
      contextMenuPosition = { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
      contextMenuTrigger = e.currentTarget as HTMLElement;
      showContextMenu = true;
    }
  }

  // Rename handlers
  function handleRename() {
    if (node.type === 'folder' && node.path !== '/') {
      showRenameDialog = true;
    } else if (node.type === 'note') {
      showRenameNoteDialog = true;
    }
  }

  function closeRenameDialog() {
    showRenameDialog = false;
  }

  // Delete handlers
  async function handleDelete() {
    if (node.type === 'folder' && node.path !== '/') {
      showDeleteDialog = true;
    } else if (node.type === 'note') {
      const confirmed = await dialog.confirm({
        title: $_('dialog.confirm_title'),
        message: $_('dialog.delete_note_confirm'),
        confirmText: $_('common.delete'),
        cancelText: $_('dialog.cancel'),
        variant: 'danger',
      });
      if (!confirmed) return;
      try {
        await api.deleteNote(node.id);
        trash.incrementTrashCount();
        toast.success($_('component.editor.note_trashed'));
        notes.clearCurrentNote();
        await notes.loadNotes();
        await tree.loadTree();
        goto('/');
      } catch {
        toast.error($_('component.editor.error_delete'));
      }
    }
  }

  function closeDeleteDialog() {
    showDeleteDialog = false;
  }

  // Color picker handlers
  function handleColor() {
    showColorPicker = true;
  }

  function closeColorPicker() {
    showColorPicker = false;
  }

  async function handleColorSelect(color: string | null) {
    if (node.type === 'folder') {
      await tree.updateFolderColor(node.id, color);
    } else {
      await tree.updateNoteColor(node.id, color);
    }
  }

  // Share handler
  function handleShare() {
    if (node.type === 'folder' && node.path !== '/' && node.path !== '/Journal') {
      showShareDialog = true;
    } else if (node.type === 'note') {
      showShareDialog = true;
    }
  }

  function closeShareDialog() {
    showShareDialog = false;
  }

  function closeRenameNoteDialog() {
    showRenameNoteDialog = false;
  }

  // Drag Start Handlers
  function handleFolderDragStart(e: DragEvent) {
    if (node.type !== 'folder' || !e.dataTransfer) return;

    const dragData = {
      type: 'folder',
      id: node.id,
      path: node.path,
    };

    e.dataTransfer.setData('application/x-xelanote-item', JSON.stringify(dragData));
    e.dataTransfer.setData('text/plain', JSON.stringify(dragData));
    e.dataTransfer.effectAllowed = 'move';
    isDragging = true;
  }

  function handleNoteDragStart(e: DragEvent) {
    if (node.type !== 'note' || !e.dataTransfer) return;

    const dragData = {
      type: 'note',
      id: node.id,
      title: node.title,
      folder_path: node.folderPath,
    };

    e.dataTransfer.setData('application/x-xelanote-item', JSON.stringify(dragData));
    e.dataTransfer.setData('text/plain', JSON.stringify(dragData));
    e.dataTransfer.effectAllowed = 'move';
    isDragging = true;
  }

  function handleDragEnd() {
    isDragging = false;
  }

  // Drop Handlers - support reordering and moving into folders
  function handleDragOver(e: DragEvent) {
    if (!e.dataTransfer?.types.includes('application/x-xelanote-item')) return;

    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';

    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const mouseY = e.clientY - rect.top;
    const height = rect.height;

    // Determine drop position based on mouse position
    if (node.type === 'folder') {
      // For folders: top 25% = before, bottom 25% = after, middle 50% = into
      if (mouseY < height * 0.25) {
        dropPosition = 'before';
        isDragOver = false;
      } else if (mouseY > height * 0.75) {
        dropPosition = 'after';
        isDragOver = false;
      } else {
        dropPosition = 'into';
        isDragOver = true;
      }
    } else {
      // For notes: top 50% = before, bottom 50% = after
      if (mouseY < height * 0.5) {
        dropPosition = 'before';
      } else {
        dropPosition = 'after';
      }
      isDragOver = false;
    }
  }

  function handleDragLeave(e: DragEvent) {
    const relatedTarget = e.relatedTarget as Node | null;
    if (!e.currentTarget || !relatedTarget || !(e.currentTarget as Node).contains(relatedTarget)) {
      isDragOver = false;
      dropPosition = null;
    }
  }

  async function handleDrop(e: DragEvent) {
    e.preventDefault();
    const currentDropPosition = dropPosition;
    isDragOver = false;
    dropPosition = null;

    const data = e.dataTransfer?.getData('application/x-xelanote-item');
    if (!data) return;

    try {
      const dragData = parseDraggedTreeItem(data);
      if (!dragData) return;

      // Handle folder reordering (before/after) - only for sibling folders
      if (
        (currentDropPosition === 'before' || currentDropPosition === 'after') &&
        dragData.type === 'folder' &&
        node.type === 'folder'
      ) {
        const reordered = await handleFolderReorder(dragData.id, currentDropPosition);
        if (reordered) return;
        // Not siblings - fall through to move into folder
      }

      // Handle note reordering or cross-folder move (note dropped before/after another note)
      if (
        (currentDropPosition === 'before' || currentDropPosition === 'after') &&
        dragData.type === 'note' &&
        node.type === 'note'
      ) {
        if (dragData.folder_path === node.folderPath) {
          await handleNoteReorder(dragData.id, currentDropPosition);
          return;
        }
        // Cross-folder: move note to the target note's folder
        if (node.folderPath === '/Journal') {
          await dialog.alert({
            title: $_('common.error'),
            message: $_('component.tree.cannot_move_to_journal'),
            variant: 'warning',
          });
          return;
        }
        await notes.moveNote(dragData.id, node.folderPath);
        await tree.loadTree();
        return;
      }

      // Handle moving into folder (any drop position on a folder target)
      if (node.type === 'folder') {
        if (dragData.type === 'note') {
          if (dragData.folder_path === node.path) return;
          if (node.path === '/Journal') {
            await dialog.alert({
              title: $_('common.error'),
              message: $_('component.tree.cannot_move_to_journal'),
              variant: 'warning',
            });
            return;
          }
          await notes.moveNote(dragData.id, node.path);
          await tree.loadTree();
        } else if (dragData.type === 'folder') {
          if (dragData.path === node.path) return;
          if (node.path.startsWith(dragData.path + '/')) {
            await dialog.alert({
              title: $_('common.error'),
              message: $_('component.tree.cannot_move_into_self'),
              variant: 'warning',
            });
            return;
          }
          await tree.moveFolder(dragData.id, node.path);
        }
      }
    } catch (err) {
      console.error('Failed to move:', err);
      await dialog.alert({
        title: $_('common.error'),
        message: $_('component.tree.move_error'),
        variant: 'danger',
      });
    }
  }

  async function handleFolderReorder(
    draggedId: number,
    position: 'before' | 'after'
  ): Promise<boolean> {
    // Get parent context
    const treeData = tree.getTreeData();
    if (!treeData) return false;

    // Find all siblings at this level
    const parent = findParentOfNode(treeData, node);
    if (!parent || parent.type !== 'folder') return false;

    const siblings = parent.children.filter(
      (child: TreeNode) => child.type === 'folder'
    ) as FolderTreeNode[];
    const draggedIndex = siblings.findIndex((s) => s.id === draggedId);
    const targetIndex = siblings.findIndex((s) => s.id === (node as FolderTreeNode).id);

    // Not siblings - reorder not possible
    if (draggedIndex === -1 || targetIndex === -1) return false;

    // Calculate new order
    const newOrder = [...siblings];
    const [draggedItem] = newOrder.splice(draggedIndex, 1);

    const insertIndex = position === 'before' ? targetIndex : targetIndex + 1;
    newOrder.splice(draggedIndex < targetIndex ? insertIndex - 1 : insertIndex, 0, draggedItem);

    // Extract folder IDs in new order
    const folderIds = newOrder.map((s) => s.id).filter((id) => id !== 0);

    // Determine parent ID (parent is always FolderTreeNode here, id is number)
    const parentID: number | null = parent.id === 0 ? 1 : parent.id || null;

    // Call API
    await tree.reorderFolders(parentID, folderIds);
    return true;
  }

  async function handleNoteReorder(draggedId: string, position: 'before' | 'after') {
    const treeData = tree.getTreeData();
    if (!treeData || node.type !== 'note') return;

    const parent = findParentOfNode(treeData, node);
    if (!parent || parent.type !== 'folder') return;

    const siblings = parent.children.filter(
      (child: TreeNode) => child.type === 'note'
    ) as NoteTreeNode[];
    const draggedIndex = siblings.findIndex((s) => s.id === draggedId);
    const targetIndex = siblings.findIndex((s) => s.id === node.id);

    if (draggedIndex === -1 || targetIndex === -1) return;

    const newOrder = [...siblings];
    const [draggedItem] = newOrder.splice(draggedIndex, 1);

    const insertIndex = position === 'before' ? targetIndex : targetIndex + 1;
    newOrder.splice(draggedIndex < targetIndex ? insertIndex - 1 : insertIndex, 0, draggedItem);

    const noteIds = newOrder.map((s) => s.id);
    const folderPath = parent.path || '/';
    await tree.reorderNotes(folderPath, noteIds);
  }

  function findParentOfNode(root: FolderTreeNode, target: TreeNode): FolderTreeNode | null {
    for (const child of root.children) {
      if (child === target) return root;
      if (child.type === 'folder') {
        const found = findParentOfNode(child, target);
        if (found) return found;
      }
    }
    return null;
  }

  async function loadRenameFolderDialog() {
    if (RenameFolderDialogComponent) return;
    const module = await import('./RenameFolderDialog.svelte');
    RenameFolderDialogComponent = loadSvelteComponentFromModule(module, 'RenameFolderDialog');
  }

  async function loadDeleteFolderDialog() {
    if (DeleteFolderDialogComponent) return;
    const module = await import('./DeleteFolderDialog.svelte');
    DeleteFolderDialogComponent = loadSvelteComponentFromModule(module, 'DeleteFolderDialog');
  }

  async function loadColorPickerDialog() {
    if (ColorPickerDialogComponent) return;
    const module = await import('./ColorPickerDialog.svelte');
    ColorPickerDialogComponent = loadSvelteComponentFromModule(module, 'ColorPickerDialog');
  }

  async function loadShareDialog() {
    if (ShareDialogComponent) return;
    const module = await import('./ShareDialog.svelte');
    ShareDialogComponent = loadSvelteComponentFromModule(module, 'ShareDialog');
  }

  async function loadRenameNoteDialog() {
    if (RenameNoteDialogComponent) return;
    const module = await import('./RenameNoteDialog.svelte');
    RenameNoteDialogComponent = loadSvelteComponentFromModule(module, 'RenameNoteDialog');
  }

  $effect(() => {
    if (showRenameDialog) {
      loadRenameFolderDialog();
    }
  });

  $effect(() => {
    if (showDeleteDialog) {
      loadDeleteFolderDialog();
    }
  });

  $effect(() => {
    if (showColorPicker) {
      loadColorPickerDialog();
    }
  });

  $effect(() => {
    if (showShareDialog) {
      loadShareDialog();
    }
  });

  $effect(() => {
    if (showRenameNoteDialog) {
      loadRenameNoteDialog();
    }
  });
</script>

<div
  class="tree-item"
  class:drop-before={dropPosition === 'before'}
  class:drop-after={dropPosition === 'after'}
  class:has-color={node.color}
  style="padding-left: {depth * 12}px"
  data-drag-type={node.type}
  data-drag-id={String(node.id)}
  data-drag-path={node.type === 'folder' ? node.path : undefined}
  data-drag-folder-path={node.type === 'note' ? node.folderPath : undefined}
  data-drag-title={node.type === 'note' ? node.title : undefined}
>
  {#if node.color}
    <div class="color-bar" style="background-color: {node.color}"></div>
  {/if}
  <div class="tree-row">
    {#if node.type === 'folder' && node.children.length > 0}
      <button
        class="expand-button"
        data-no-drag
        onclick={handleExpandClick}
        aria-label="Toggle folder"
      >
        {#if node.isExpanded}
          <ChevronDown size={chevronIconSize} />
        {:else}
          <ChevronRight size={chevronIconSize} />
        {/if}
      </button>
    {:else}
      <span class="expand-spacer"></span>
    {/if}

    {#if node.type === 'folder'}
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        class="folder-row-container"
        oncontextmenu={handleContextMenu}
        onkeydown={handleRowKeydown}
      >
        <button
          draggable={treeDragEnabled}
          ondragstart={treeDragEnabled ? handleFolderDragStart : undefined}
          ondragend={treeDragEnabled ? handleDragEnd : undefined}
          ondragover={treeDragEnabled ? handleDragOver : undefined}
          ondragleave={treeDragEnabled ? handleDragLeave : undefined}
          ondrop={treeDragEnabled ? handleDrop : undefined}
          class="tree-button"
          class:selected={isSelected}
          class:drag-over={isDragOver}
          class:dragging={isDragging}
          onclick={handleClick}
        >
          {#if node.isExpanded && node.children.length > 0}
            <FolderOpen size={folderIconSize} />
          {:else}
            <Folder size={folderIconSize} />
          {/if}
          <span class="node-name">{node.name}</span>
          {#if node.noteCount > 0}
            <span class="note-count">{node.noteCount}</span>
          {/if}
        </button>
        {#if node.path !== '/'}
          <button
            data-no-drag
            onclick={handleKebabClick}
            class="kebab-button"
            aria-label={$_('component.tree.context_menu.more_options')}
            aria-haspopup="menu"
            aria-expanded={showContextMenu}
          >
            <MoreVertical size={actionIconSize} />
          </button>
        {/if}
      </div>
    {:else}
      <!-- Note node -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        class="note-row-container"
        oncontextmenu={handleContextMenu}
        onkeydown={handleRowKeydown}
      >
        <button
          draggable={noteExternalDragEnabled}
          ondragstart={noteExternalDragEnabled ? handleNoteDragStart : undefined}
          ondragend={noteExternalDragEnabled ? handleDragEnd : undefined}
          ondragover={treeDragEnabled ? handleDragOver : undefined}
          ondragleave={treeDragEnabled ? handleDragLeave : undefined}
          ondrop={treeDragEnabled ? handleDrop : undefined}
          class="tree-button note-button"
          class:selected={isSelected}
          class:dragging={isDragging}
          onclick={handleClick}
        >
          {#if node.noteType === 'canvas'}
            <LayoutDashboard size={folderIconSize} />
          {:else}
            <FileText size={folderIconSize} />
          {/if}
          <span class="node-name">{node.title}</span>
          {#if node.aiEnabled}
            <span class="ai-badge" title="KI aktiviert">
              <Sparkles size={12} />
            </span>
          {/if}
        </button>
        <button
          data-no-drag
          onclick={handleKebabClick}
          class="kebab-button"
          aria-label={$_('component.tree.context_menu.more_options')}
          aria-haspopup="menu"
          aria-expanded={showContextMenu}
        >
          <MoreVertical size={actionIconSize} />
        </button>
      </div>
    {/if}
  </div>
</div>

<!-- Recursive children rendering (only in non-virtual mode) -->
{#if !isVirtualized && node.type === 'folder' && node.isExpanded && node.children.length > 0}
  {#each node.children as child (child.type === 'folder' ? `f-${child.id}` : `n-${child.id}`)}
    <UnifiedTree node={child} depth={depth + 1} {isVirtualized} />
  {/each}
{/if}

<!-- Rename dialog -->
{#if node.type === 'folder' && showRenameDialog}
  {#if RenameFolderDialogComponent}
    <RenameFolderDialogComponent
      open={true}
      folderId={node.id}
      currentName={node.name}
      onClose={closeRenameDialog}
    />
  {/if}
{/if}

<!-- Delete dialog -->
{#if node.type === 'folder' && showDeleteDialog}
  {#if DeleteFolderDialogComponent}
    <DeleteFolderDialogComponent
      open={true}
      folderId={node.id}
      folderName={node.name}
      folderPath={node.path}
      noteCount={node.noteCount}
      onClose={closeDeleteDialog}
    />
  {/if}
{/if}

<!-- Color picker dialog -->
{#if showColorPicker}
  {#if ColorPickerDialogComponent}
    <ColorPickerDialogComponent
      currentColor={node.color}
      onClose={closeColorPicker}
      onSelect={handleColorSelect}
    />
  {/if}
{/if}

<!-- Share dialog (note or folder) -->
{#if showShareDialog && ShareDialogComponent}
  <ShareDialogComponent
    resourceType={node.type === 'folder' ? 'folder' : 'note'}
    resourceId={node.id}
    onClose={closeShareDialog}
  />
{/if}

<!-- Rename note dialog -->
{#if node.type === 'note' && showRenameNoteDialog}
  {#if RenameNoteDialogComponent}
    <RenameNoteDialogComponent
      open={true}
      noteId={node.id}
      currentTitle={node.title}
      onClose={closeRenameNoteDialog}
    />
  {/if}
{/if}

<!-- Context menu -->
{#if showContextMenu}
  <TreeContextMenu
    {node}
    position={contextMenuPosition}
    onClose={closeContextMenu}
    onRename={handleRename}
    onDelete={handleDelete}
    onColorPicker={handleColor}
    onShare={(node.type === 'folder' && node.path !== '/' && node.path !== '/Journal') ||
    node.type === 'note'
      ? handleShare
      : undefined}
  />
{/if}

<style>
  .tree-item {
    -webkit-user-select: none;
    user-select: none;
    position: relative;
  }

  .tree-item.has-color {
    padding-left: 8px;
  }

  .color-bar {
    position: absolute;
    left: 0;
    top: 2px;
    bottom: 2px;
    width: 3px;
    border-radius: var(--radius-xs);
  }

  .tree-item.drop-before::before,
  .tree-item.drop-after::before {
    content: '';
    position: absolute;
    left: 0;
    right: 0;
    height: 2px;
    background: var(--color-primary);
    z-index: 10;
  }

  .tree-item.drop-before::before {
    top: -1px;
  }

  .tree-item.drop-after::before {
    bottom: -1px;
  }

  .tree-row {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .expand-button {
    padding: 2px;
    background: none;
    border: none;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--color-muted-foreground);
    border-radius: var(--radius-sm);
    flex-shrink: 0;
    -webkit-tap-highlight-color: transparent;
  }

  @media (hover: hover) {
    .expand-button:hover {
      background: var(--color-sidebar-accent);
      color: var(--color-sidebar-foreground);
    }
  }

  .expand-spacer {
    width: 18px;
    flex-shrink: 0;
  }

  .tree-button {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px;
    background: none;
    border: none;
    cursor: pointer;
    font-size: 13px;
    color: var(--color-sidebar-foreground);
    border-radius: var(--radius-sm);
    max-width: 100%;
    text-align: left;
    transition: background var(--duration-fast) var(--ease-default);
    min-width: 0;
    -webkit-tap-highlight-color: transparent;
  }

  /* Only show hover on devices with a real pointer (mouse/trackpad) */
  @media (hover: hover) {
    .tree-button:hover {
      background: var(--color-sidebar-accent);
    }
  }

  .tree-button.selected {
    background: var(--color-sidebar-accent);
    color: var(--color-primary);
  }

  .tree-button.drag-over {
    background: color-mix(in oklch, var(--color-primary), transparent 90%);
    border: 2px dashed var(--color-primary);
  }

  .tree-button.dragging {
    opacity: 0.5;
  }

  .node-name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .note-count {
    font-size: 11px;
    color: var(--color-muted-foreground);
    background: var(--color-sidebar-accent);
    padding: 2px 6px;
    border-radius: var(--radius-xl);
    font-weight: 500;
    opacity: 0;
    transition: opacity var(--duration-fast) var(--ease-default);
  }

  .folder-row-container:focus-within .note-count {
    opacity: 1;
  }

  @media (hover: hover) {
    .folder-row-container:hover .note-count {
      opacity: 1;
    }
  }

  .ai-badge {
    flex-shrink: 0;
    color: var(--color-primary);
    opacity: 0.7;
    display: flex;
    align-items: center;
  }

  @media (hover: hover) {
    .tree-button:hover .ai-badge {
      opacity: 1;
    }
  }

  .note-button {
    color: var(--color-muted-foreground);
  }

  @media (hover: hover) {
    .note-button:hover {
      color: var(--color-sidebar-foreground);
    }
  }

  /* Mobile: allow wrapped note titles instead of single-line truncation. */
  @media (max-width: 639px) {
    .note-button .node-name {
      white-space: normal;
      text-overflow: clip;
      overflow-wrap: anywhere;
      word-break: break-word;
    }
  }

  .folder-row-container {
    display: flex;
    align-items: center;
    gap: 4px;
    flex: 1;
    min-width: 0;
  }

  .folder-row-container .tree-button {
    flex: 1;
    min-width: 0;
    overflow: hidden;
  }

  .note-row-container {
    display: flex;
    align-items: center;
    gap: 4px;
    flex: 1;
    min-width: 0;
  }

  .note-row-container .tree-button {
    flex: 1;
    min-width: 0;
    overflow: hidden;
  }

  .kebab-button {
    opacity: 0;
    pointer-events: none;
    display: flex;
    align-items: center;
    padding: 2px;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--color-muted-foreground);
    border-radius: var(--radius-sm);
    flex-shrink: 0;
    transition: opacity var(--duration-fast) var(--ease-default);
    -webkit-tap-highlight-color: transparent;
  }

  .folder-row-container:focus-within .kebab-button,
  .note-row-container:focus-within .kebab-button {
    opacity: 1;
    pointer-events: auto;
  }

  @media (hover: hover) {
    .folder-row-container:hover .kebab-button,
    .note-row-container:hover .kebab-button {
      opacity: 1;
      pointer-events: auto;
    }

    .kebab-button:hover {
      background: var(--color-sidebar-accent);
      color: var(--color-sidebar-foreground);
    }
  }

  .kebab-button:focus-visible {
    outline: 2px solid var(--color-ring);
    outline-offset: 2px;
  }

  /* Focus indicators */
  .tree-button:focus-visible,
  .expand-button:focus-visible {
    outline: 2px solid var(--color-ring);
    outline-offset: 2px;
  }

  /* Touch targets */
  @media (pointer: coarse) {
    .tree-button {
      padding: 10px 8px;
      min-height: 44px;
    }
    .expand-button {
      min-width: 44px;
      min-height: 44px;
    }
    .kebab-button {
      min-width: 36px;
      min-height: 36px;
      padding: 6px;
    }
  }

  /* Touch-optimized sizes */
  @media (pointer: coarse) {
    .tree-button {
      font-size: 15px;
      padding: 6px 10px;
      gap: 8px;
    }

    .expand-button {
      padding: 4px;
    }

    .note-count {
      font-size: 12px;
      padding: 3px 8px;
    }

    .tree-row {
      gap: 6px;
    }

    .folder-row-container {
      gap: 6px;
    }

    /* Always show kebab on touch devices for context menu access */
    .kebab-button {
      opacity: 1;
      pointer-events: auto;
    }
  }

  /* Touch drag & drop visual indicators (classes applied by touchdrag action) */
  :global(.tree-item.touch-drop-before)::after {
    content: '';
    position: absolute;
    left: 0;
    right: 0;
    top: -1px;
    height: 2px;
    background: var(--color-primary);
    z-index: 10;
    pointer-events: none;
  }

  :global(.tree-item.touch-drop-after)::after {
    content: '';
    position: absolute;
    left: 0;
    right: 0;
    bottom: -1px;
    height: 2px;
    background: var(--color-primary);
    z-index: 10;
    pointer-events: none;
  }

  :global(.tree-item.touch-drop-into) .tree-button {
    border: 2px dashed var(--color-primary);
    background: color-mix(in oklch, var(--color-primary), transparent 90%);
  }

  :global(.tree-item.touch-dragging-source) {
    opacity: 0.4;
  }
</style>
