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
  import { _ } from 'svelte-i18n';

  import type { TreeNode } from '$lib/stores/tree.svelte';

  interface Props {
    node: TreeNode;
    depth: number;
    isSelected: boolean;
    isDragging: boolean;
    isDragOver: boolean;
    dropPosition: 'before' | 'after' | 'into' | null;
    treeDragEnabled: boolean;
    noteExternalDragEnabled: boolean;
    showContextMenu: boolean;
    // Icon sizes
    folderIconSize: number;
    chevronIconSize: number;
    actionIconSize: number;
    // Event handlers
    onClick: (e?: MouseEvent) => void;
    onAuxClick?: (e: MouseEvent) => void;
    onExpandClick: (e: MouseEvent) => void;
    onContextMenu: (e: MouseEvent) => void;
    onKebabClick: (e: MouseEvent) => void;
    onRowKeydown: (e: KeyboardEvent) => void;
    // Drag handlers
    onNotePointerDown?: (e: PointerEvent) => void;
    onFolderDragStart?: (e: DragEvent) => void;
    onNoteDragStart?: (e: DragEvent) => void;
    onDragEnd?: () => void;
    onDragOver?: (e: DragEvent) => void;
    onDragLeave?: (e: DragEvent) => void;
    onDrop?: (e: DragEvent) => void;
  }

  const {
    node,
    depth,
    isSelected,
    isDragging,
    isDragOver,
    dropPosition,
    treeDragEnabled,
    noteExternalDragEnabled,
    showContextMenu,
    folderIconSize,
    chevronIconSize,
    actionIconSize,
    onClick,
    onAuxClick,
    onExpandClick,
    onContextMenu,
    onKebabClick,
    onRowKeydown,
    onNotePointerDown,
    onFolderDragStart,
    onNoteDragStart,
    onDragEnd,
    onDragOver,
    onDragLeave,
    onDrop,
  }: Props = $props();
</script>

<div
  class="tree-item"
  class:drop-before={dropPosition === 'before'}
  class:drop-after={dropPosition === 'after'}
  class:has-color={node.color}
  style="padding-left: {depth * 10}px"
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
      <button class="expand-button" data-no-drag onclick={onExpandClick} aria-label="Toggle folder">
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
        class:selected-row={isSelected}
        class:context-open={showContextMenu}
        oncontextmenu={onContextMenu}
        onkeydown={onRowKeydown}
      >
        <button
          draggable={treeDragEnabled}
          ondragstart={treeDragEnabled ? onFolderDragStart : undefined}
          ondragend={treeDragEnabled ? onDragEnd : undefined}
          ondragover={treeDragEnabled ? onDragOver : undefined}
          ondragleave={treeDragEnabled ? onDragLeave : undefined}
          ondrop={treeDragEnabled ? onDrop : undefined}
          class="tree-button"
          class:selected={isSelected}
          class:journal-parent={node.path === '/Journal'}
          class:drag-over={isDragOver}
          class:dragging={isDragging}
          onclick={onClick}
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
            onclick={onKebabClick}
            class="kebab-button"
            class:visible={isSelected || showContextMenu}
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
        class:selected-row={isSelected}
        class:context-open={showContextMenu}
        oncontextmenu={onContextMenu}
        onkeydown={onRowKeydown}
      >
        <button
          draggable={noteExternalDragEnabled}
          ondragstart={noteExternalDragEnabled ? onNoteDragStart : undefined}
          ondragend={noteExternalDragEnabled ? onDragEnd : undefined}
          ondragover={treeDragEnabled ? onDragOver : undefined}
          ondragleave={treeDragEnabled ? onDragLeave : undefined}
          ondrop={treeDragEnabled ? onDrop : undefined}
          onpointerdown={onNotePointerDown}
          class="tree-button note-button"
          class:selected={isSelected}
          class:journal-note={node.folderPath === '/Journal'}
          class:dragging={isDragging}
          onclick={(e) => onClick(e)}
          onauxclick={onAuxClick}
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
          onclick={onKebabClick}
          class="kebab-button"
          class:visible={isSelected || showContextMenu}
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
    width: 16px;
    height: 16px;
    padding: 0;
    background: none;
    border: none;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    color: color-mix(
      in oklch,
      var(--color-sidebar-foreground),
      var(--color-sidebar-background) 42%
    );
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
    width: 16px;
    flex-shrink: 0;
  }

  .tree-button {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 5px 10px;
    background: none;
    border: 1px solid transparent;
    cursor: pointer;
    font-size: var(--text-md);
    line-height: 1.2;
    color: var(--color-sidebar-foreground);
    border-radius: var(--radius-sm);
    max-width: 100%;
    text-align: left;
    transition:
      background var(--duration-fast) var(--ease-default),
      border-color var(--duration-fast) var(--ease-default),
      color var(--duration-fast) var(--ease-default);
    min-width: 0;
    min-height: 32px;
    -webkit-tap-highlight-color: transparent;
  }

  /* Only show hover on devices with a real pointer (mouse/trackpad) */
  @media (hover: hover) {
    .tree-button:hover {
      background: color-mix(in oklch, var(--color-sidebar-accent), white 8%);
    }
  }

  .tree-button.selected {
    background: color-mix(in oklch, var(--color-primary), transparent 84%);
    border-color: color-mix(in oklch, var(--color-primary), transparent 72%);
    color: color-mix(in oklch, var(--color-primary), white 10%);
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
    font-size: var(--text-2xs);
    color: color-mix(
      in oklch,
      var(--color-sidebar-foreground),
      var(--color-sidebar-background) 30%
    );
    background: color-mix(
      in oklch,
      var(--color-sidebar-accent),
      var(--color-sidebar-background) 20%
    );
    padding: 1px 5px;
    border-radius: var(--radius-xl);
    font-weight: 500;
    opacity: 0;
    transition: opacity var(--duration-fast) var(--ease-default);
  }

  .folder-row-container:focus-within .note-count {
    opacity: 1;
  }

  .folder-row-container.selected-row .note-count,
  .folder-row-container.context-open .note-count {
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
    margin-left: 1px;
  }

  @media (hover: hover) {
    .tree-button:hover .ai-badge {
      opacity: 1;
    }
  }

  .note-button {
    color: color-mix(
      in oklch,
      var(--color-sidebar-foreground),
      var(--color-sidebar-background) 35%
    );
  }

  .tree-button.journal-parent {
    color: color-mix(in oklch, var(--color-sidebar-foreground), var(--color-primary) 20%);
    font-weight: 500;
  }

  .note-button.journal-note {
    color: color-mix(
      in oklch,
      var(--color-sidebar-foreground),
      var(--color-sidebar-background) 42%
    );
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
    padding-right: 2px;
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
    padding-right: 2px;
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
    padding: 3px;
    background: none;
    border: none;
    cursor: pointer;
    color: color-mix(
      in oklch,
      var(--color-sidebar-foreground),
      var(--color-sidebar-background) 40%
    );
    border-radius: var(--radius-sm);
    flex-shrink: 0;
    margin-right: 1px;
    transition:
      opacity var(--duration-fast) var(--ease-default),
      background var(--duration-fast) var(--ease-default),
      color var(--duration-fast) var(--ease-default);
    -webkit-tap-highlight-color: transparent;
  }

  .kebab-button.visible {
    opacity: 1;
    pointer-events: auto;
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

  .folder-row-container.selected-row .kebab-button,
  .folder-row-container.context-open .kebab-button,
  .note-row-container.selected-row .kebab-button,
  .note-row-container.context-open .kebab-button {
    opacity: 1;
    pointer-events: auto;
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
      font-size: var(--text-lg);
      padding: 6px 10px;
      gap: 8px;
    }

    .expand-button {
      padding: 4px;
    }

    .note-count {
      font-size: var(--text-xs);
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
