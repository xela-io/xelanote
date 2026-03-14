<script lang="ts">
  import type { ComponentType } from 'svelte';
  import { createEventDispatcher } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import * as api from '$lib/api';
  import * as dialog from '$lib/stores/dialog.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as tabs from '$lib/stores/tabs.svelte';
  import * as toast from '$lib/stores/toast.svelte';
  import * as trash from '$lib/stores/trash.svelte';
  import type { FolderTreeNode, NoteTreeNode, TreeNode } from '$lib/stores/tree.svelte';
  import * as tree from '$lib/stores/tree.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import { loadSvelteComponentFromModule } from '$lib/utils/lazy-component';

  import TreeNodeDialogs from './tree/TreeNodeDialogs.svelte';
  import TreeNodeRow from './tree/TreeNodeRow.svelte';
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
  // Reordering should only be active in manual sort mode.
  const manualSortMode = $derived(tree.isManualSortMode());
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
  function handleClick(e?: MouseEvent) {
    if (node.type === 'folder') {
      tree.selectFolder(node.path);
    } else {
      tree.selectNote(node.id);
      if (e && (e.ctrlKey || e.metaKey)) tabs.requestNewTab();
      goto(`/note/${node.id}`);
      ui.closeSidebarOnMobile();
    }
  }

  // Middle-click handler: open in new tab
  function handleAuxClick(e: MouseEvent) {
    if (e.button !== 1 || node.type !== 'note') return;
    e.preventDefault();
    tabs.requestNewTab();
    goto(`/note/${node.id}`);
  }

  // Prefetch note data on pointerdown for faster perceived navigation
  function handleNotePointerDown() {
    if (node.type === 'note') {
      notes.prefetchNote(node.id);
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

  // Open in new tab handler (context menu)
  function handleOpenInNewTab() {
    if (node.type !== 'note') return;
    tabs.requestNewTab();
    goto(`/note/${node.id}`);
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
      if (!manualSortMode) {
        // In auto sort modes, allow moving into folders but not before/after reordering.
        dropPosition = 'into';
        isDragOver = true;
        return;
      }

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
      if (!manualSortMode) {
        // In auto sort modes, dropping on notes is not used for ordering.
        dropPosition = null;
        isDragOver = false;
        return;
      }

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
        manualSortMode &&
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
        manualSortMode &&
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

<!-- Tree node row rendering -->
<TreeNodeRow
  {node}
  {depth}
  {isSelected}
  {isDragging}
  {isDragOver}
  {dropPosition}
  {treeDragEnabled}
  {noteExternalDragEnabled}
  {showContextMenu}
  {folderIconSize}
  {chevronIconSize}
  {actionIconSize}
  onClick={handleClick}
  onAuxClick={handleAuxClick}
  onExpandClick={handleExpandClick}
  onContextMenu={handleContextMenu}
  onKebabClick={handleKebabClick}
  onRowKeydown={handleRowKeydown}
  onNotePointerDown={handleNotePointerDown}
  onFolderDragStart={handleFolderDragStart}
  onNoteDragStart={handleNoteDragStart}
  onDragEnd={handleDragEnd}
  onDragOver={handleDragOver}
  onDragLeave={handleDragLeave}
  onDrop={handleDrop}
/>

<!-- Recursive children rendering (only in non-virtual mode) -->
{#if !isVirtualized && node.type === 'folder' && node.isExpanded && node.children.length > 0}
  {#each node.children as child (child.type === 'folder' ? `f-${child.id}` : `n-${child.id}`)}
    <UnifiedTree node={child} depth={depth + 1} {isVirtualized} />
  {/each}
{/if}

<!-- All dialogs (rename, delete, color, share) -->
<TreeNodeDialogs
  {node}
  {showRenameDialog}
  {RenameFolderDialogComponent}
  onCloseRenameDialog={closeRenameDialog}
  {showDeleteDialog}
  {DeleteFolderDialogComponent}
  onCloseDeleteDialog={closeDeleteDialog}
  {showColorPicker}
  {ColorPickerDialogComponent}
  onCloseColorPicker={closeColorPicker}
  onColorSelect={handleColorSelect}
  {showShareDialog}
  {ShareDialogComponent}
  onCloseShareDialog={closeShareDialog}
  {showRenameNoteDialog}
  {RenameNoteDialogComponent}
  onCloseRenameNoteDialog={closeRenameNoteDialog}
/>

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
    onOpenInNewTab={node.type === 'note' ? handleOpenInNewTab : undefined}
  />
{/if}

<!-- Styles moved to TreeNodeRow.svelte -->
