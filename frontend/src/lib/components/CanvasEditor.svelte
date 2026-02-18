<script lang="ts">
  import '@xyflow/svelte/dist/style.css';

  import type { Edge as FlowEdge, Node as FlowNode, NodeTypes } from '@xyflow/svelte';
  import {
    Background,
    BackgroundVariant,
    Controls,
    MiniMap,
    Panel,
    SvelteFlow,
  } from '@xyflow/svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import type { Note } from '$lib/api/types';
  import { uploadImage } from '$lib/api/uploads';
  import { DeleteCommand } from '$lib/commands/DeleteCommand';
  import CanvasContextMenu from '$lib/components/canvas/CanvasContextMenu.svelte';
  import CanvasEditorToolbar from '$lib/components/canvas/CanvasEditorToolbar.svelte';
  import CanvasFileNode from '$lib/components/canvas/CanvasFileNode.svelte';
  import CanvasGroupNode from '$lib/components/canvas/CanvasGroupNode.svelte';
  import CanvasLinkNode from '$lib/components/canvas/CanvasLinkNode.svelte';
  import CanvasMoreMenu from '$lib/components/canvas/CanvasMoreMenu.svelte';
  import CanvasNotePicker from '$lib/components/canvas/CanvasNotePicker.svelte';
  import CanvasTextNode from '$lib/components/canvas/CanvasTextNode.svelte';
  import CanvasToolbar from '$lib/components/canvas/CanvasToolbar.svelte';
  import {
    type DialogLoaderState,
    loadMoveToFolderDialog,
    loadShareDialog,
    loadVersionHistoryDialog,
    maybeLoadDialog,
  } from '$lib/editor/dialog-loaders';
  import { handleTitleInput as handleTitleInputAction } from '$lib/editor/editor-actions';
  import { handleDeleteNote } from '$lib/editor/note-actions';
  import { getIsSyncing, getPendingCount, getSyncProgress } from '$lib/offline/sync-manager.svelte';
  import * as auth from '$lib/stores/auth.svelte';
  import {
    canvasToFlow,
    createFileNode,
    createGroupNode,
    createLinkNode,
    createTextNode,
    flowToCanvas,
    generateEdgeId,
    parseCanvas,
    serializeCanvas,
  } from '$lib/stores/canvas.svelte';
  import * as dialog from '$lib/stores/dialog.svelte';
  import * as encryption from '$lib/stores/encryption.svelte';
  import * as focusMode from '$lib/stores/focus-mode.svelte';
  import * as history from '$lib/stores/history.svelte';
  import * as network from '$lib/stores/network.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as toast from '$lib/stores/toast.svelte';
  import * as trash from '$lib/stores/trash.svelte';
  import * as tree from '$lib/stores/tree.svelte';
  import * as ui from '$lib/stores/ui.svelte';

  const { noteId }: { noteId: string } = $props();

  // Load note when ID changes (mirrors Editor.svelte logic).
  // Without this, navigating between canvas notes or arriving via direct URL
  // would leave currentNote stale because CanvasEditor never triggered loadNote.
  $effect(() => {
    if (!noteId || !auth.isAuthenticated()) return;
    const currentNote = notes.getCurrentNote();
    if (currentNote?.id === noteId) return;
    notes.loadNote(noteId);
  });

  // Flow state
  let flowNodes = $state<FlowNode[]>([]);
  let flowEdges = $state<FlowEdge[]>([]);
  let saveStatus = $state<'saved' | 'saving' | 'unsaved'>('saved');
  let saveTimeout: ReturnType<typeof setTimeout> | null = null;
  let lastSavedContent = '';

  // Viewport state for coordinate conversion (bound to SvelteFlow)
  let viewport = $state<{ x: number; y: number; zoom: number } | undefined>(undefined);
  let containerEl: HTMLDivElement | undefined = $state();
  let draggingOver = $state(false);

  // Allowed image MIME types (must match backend allowedTypes)
  const allowedImageTypes = ['image/png', 'image/jpeg', 'image/gif', 'image/webp'];

  // Context menu state
  let contextMenu = $state<{ x: number; y: number; nodeId: string } | null>(null);

  // Note picker state
  let notePickerOpen = $state(false);
  let pendingFileNodePosition = $state<{ x: number; y: number } | null>(null);

  // More menu & dialog state
  let showMoreMenu = $state(false);
  let moreMenuTriggerRect = $state<DOMRect | null>(null);
  let showMoveDialog = $state(false);
  let showVersionHistory = $state(false);
  let showShareDialog = $state(false);
  let uploading = $state(false);
  let fileInput: HTMLInputElement;

  // Lazy-loaded dialog state
  let dialogLoaders = $state<DialogLoaderState>({});
  const setDialogLoaders = (s: DialogLoaderState) => {
    dialogLoaders = s;
  };

  // Trigger lazy loading when dialogs are requested
  $effect(() => {
    maybeLoadDialog(showMoveDialog, dialogLoaders, loadMoveToFolderDialog, setDialogLoaders);
  });
  $effect(() => {
    maybeLoadDialog(showVersionHistory, dialogLoaders, loadVersionHistoryDialog, setDialogLoaders);
  });
  $effect(() => {
    maybeLoadDialog(showShareDialog, dialogLoaders, loadShareDialog, setDialogLoaders);
  });

  // Custom node types
  const nodeTypes: NodeTypes = {
    'canvas-text': CanvasTextNode as unknown as NodeTypes[string],
    'canvas-file': CanvasFileNode as unknown as NodeTypes[string],
    'canvas-link': CanvasLinkNode as unknown as NodeTypes[string],
    'canvas-group': CanvasGroupNode as unknown as NodeTypes[string],
  };

  // Load canvas data from note content
  $effect(() => {
    const note = notes.getCurrentNote();
    if (note && note.id === noteId) {
      const content = note.content || '';
      // Only reload if content changed externally (not from our own save)
      if (content !== lastSavedContent) {
        const data = parseCanvas(content);
        if (data) {
          const { nodes, edges } = canvasToFlow(data);
          flowNodes = nodes;
          flowEdges = edges;
          lastSavedContent = content;
        }
      }
    }
  });

  // Get the note title
  const noteTitle = $derived(notes.getCurrentNote()?.title ?? 'Canvas');

  // Debounced save
  function scheduleSave() {
    if (saveTimeout) clearTimeout(saveTimeout);
    saveStatus = 'unsaved';
    saveTimeout = setTimeout(() => {
      saveCanvas();
    }, 3000);
  }

  async function saveCanvas() {
    saveStatus = 'saving';
    try {
      const data = flowToCanvas(flowNodes, flowEdges);
      const json = serializeCanvas(data);
      lastSavedContent = json;
      notes.updateCurrentNoteContent(json);
      saveStatus = 'saved';
    } catch (err) {
      console.error('Failed to save canvas:', err);
      saveStatus = 'unsaved';
    }
  }

  // Handle node drag stop (save after drag completes)
  function handleNodeDragStop() {
    scheduleSave();
  }

  // Handle new connections
  function handleConnect(params: {
    source: string;
    target: string;
    sourceHandle?: string | null;
    targetHandle?: string | null;
    edgeType?: string;
  }) {
    const newEdge: FlowEdge = {
      id: generateEdgeId(),
      source: params.source,
      target: params.target,
      type: 'default',
      markerEnd: { type: 'arrowclosed' as const },
      data: {
        fromNode: params.source,
        toNode: params.target,
        fromSide: params.sourceHandle || '',
        toSide: params.targetHandle || '',
        toEnd: 'arrow',
      },
    };
    flowEdges = [...flowEdges, newEdge];
    scheduleSave();
  }

  // Handle context menu on nodes
  function handleNodeContextMenu({ event, node }: { event: MouseEvent; node: FlowNode }) {
    event.preventDefault();
    contextMenu = {
      x: event.clientX,
      y: event.clientY,
      nodeId: node.id,
    };
  }

  // Toolbar actions (floating bottom toolbar)
  function handleToolbarAction(action: string) {
    const centerX = Math.random() * 400 + 100;
    const centerY = Math.random() * 300 + 100;

    switch (action) {
      case 'add-text': {
        const node = createTextNode(centerX, centerY);
        const { nodes } = canvasToFlow({ nodes: [node], edges: [] });
        flowNodes = [...flowNodes, ...nodes];
        scheduleSave();
        break;
      }
      case 'add-file': {
        pendingFileNodePosition = { x: centerX, y: centerY };
        notePickerOpen = true;
        break;
      }
      case 'add-link': {
        const url = prompt('Enter URL:');
        if (url) {
          const node = createLinkNode(centerX, centerY, url);
          const { nodes } = canvasToFlow({ nodes: [node], edges: [] });
          flowNodes = [...flowNodes, ...nodes];
          scheduleSave();
        }
        break;
      }
      case 'add-group': {
        const node = createGroupNode(centerX, centerY);
        const { nodes } = canvasToFlow({ nodes: [node], edges: [] });
        flowNodes = [...flowNodes, ...nodes];
        scheduleSave();
        break;
      }
    }
  }

  // Note picker selection
  function handleNoteSelected(note: Note) {
    const pos = pendingFileNodePosition ?? { x: 200, y: 200 };
    const node = createFileNode(pos.x, pos.y, note.title, note.id);
    const { nodes } = canvasToFlow({ nodes: [node], edges: [] });
    flowNodes = [...flowNodes, ...nodes];
    pendingFileNodePosition = null;
    scheduleSave();
  }

  // Context menu actions
  function handleDeleteNode() {
    if (!contextMenu) return;
    flowNodes = flowNodes.filter((n) => n.id !== contextMenu!.nodeId);
    flowEdges = flowEdges.filter(
      (e) => e.source !== contextMenu!.nodeId && e.target !== contextMenu!.nodeId
    );
    scheduleSave();
  }

  function handleDuplicateNode() {
    if (!contextMenu) return;
    const original = flowNodes.find((n) => n.id === contextMenu!.nodeId);
    if (!original) return;

    const newNode: FlowNode = {
      ...original,
      id: `node-${crypto.randomUUID().slice(0, 8)}`,
      position: { x: original.position.x + 40, y: original.position.y + 40 },
      selected: false,
      data: { ...original.data },
    };
    flowNodes = [...flowNodes, newNode];
    scheduleSave();
  }

  function handleColorChange(color: string) {
    if (!contextMenu) return;
    flowNodes = flowNodes.map((n) => {
      if (n.id === contextMenu!.nodeId) {
        return { ...n, data: { ...n.data, color: color || undefined } };
      }
      return n;
    });
    scheduleSave();
  }

  // Convert client coordinates to flow coordinates using bound viewport
  function clientToFlowPosition(clientX: number, clientY: number): { x: number; y: number } {
    const rect = containerEl?.getBoundingClientRect();
    if (!rect) return { x: clientX, y: clientY };
    const vp = viewport ?? { x: 0, y: 0, zoom: 1 };
    return {
      x: (clientX - rect.left - vp.x) / vp.zoom,
      y: (clientY - rect.top - vp.y) / vp.zoom,
    };
  }

  // Upload image files and create file nodes at given position
  async function handleImageFiles(files: File[], clientX: number, clientY: number) {
    const imageFiles = files.filter((f) => allowedImageTypes.includes(f.type));
    if (imageFiles.length === 0) return;

    const basePos = clientToFlowPosition(clientX, clientY);

    for (let i = 0; i < imageFiles.length; i++) {
      try {
        const result = await uploadImage(imageFiles[i]);
        const pos = { x: basePos.x + i * 40, y: basePos.y + i * 40 };
        const node = createFileNode(pos.x, pos.y, result.url);
        const { nodes } = canvasToFlow({ nodes: [node], edges: [] });
        flowNodes = [...flowNodes, ...nodes];
      } catch (err) {
        console.error('Failed to upload image:', err);
      }
    }
    scheduleSave();
  }

  // Drag-and-drop handlers for external files
  function handleDragOver(e: DragEvent) {
    if (!e.dataTransfer?.types.includes('Files')) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = 'copy';
    draggingOver = true;
  }

  function handleDrop(e: DragEvent) {
    draggingOver = false;
    if (!e.dataTransfer?.types.includes('Files')) return;
    e.preventDefault();
    const files = Array.from(e.dataTransfer.files);
    handleImageFiles(files, e.clientX, e.clientY);
  }

  function handleDragLeave(e: DragEvent) {
    // Only reset when leaving the container (not entering a child)
    if (containerEl && !containerEl.contains(e.relatedTarget as Node)) {
      draggingOver = false;
    }
  }

  // Clipboard paste for images (only when not editing text)
  function handlePaste(e: ClipboardEvent) {
    const target = e.target as HTMLElement;
    if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
      return;
    }
    if (!e.clipboardData?.files.length) return;
    const files = Array.from(e.clipboardData.files);
    if (!files.some((f) => allowedImageTypes.includes(f.type))) return;
    e.preventDefault();
    // Place pasted images at center of the visible canvas area
    const rect = containerEl?.getBoundingClientRect();
    const centerX = rect ? rect.left + rect.width / 2 : window.innerWidth / 2;
    const centerY = rect ? rect.top + rect.height / 2 : window.innerHeight / 2;
    handleImageFiles(files, centerX, centerY);
  }

  // Handle keyboard shortcuts
  function handleKeydown(e: KeyboardEvent) {
    if ((e.key === 'Delete' || e.key === 'Backspace') && !isEditingText(e)) {
      const selectedNodes = flowNodes.filter((n) => n.selected);
      const selectedEdges = flowEdges.filter((ed) => ed.selected);
      if (selectedNodes.length > 0 || selectedEdges.length > 0) {
        const selectedNodeIds = new Set(selectedNodes.map((n) => n.id));
        const selectedEdgeIds = new Set(selectedEdges.map((ed) => ed.id));
        flowNodes = flowNodes.filter((n) => !selectedNodeIds.has(n.id));
        flowEdges = flowEdges.filter(
          (e) =>
            !selectedEdgeIds.has(e.id) &&
            !selectedNodeIds.has(e.source) &&
            !selectedNodeIds.has(e.target)
        );
        scheduleSave();
      }
    }
  }

  function isEditingText(e: KeyboardEvent): boolean {
    const target = e.target as HTMLElement;
    return target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable;
  }

  // Export as .canvas file
  function handleExport() {
    const data = flowToCanvas(flowNodes, flowEdges);
    const json = serializeCanvas(data);
    const blob = new Blob([json], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${noteTitle}.canvas`;
    a.click();
    URL.revokeObjectURL(url);
  }

  // --- Top toolbar handlers ---

  function handleTitleInput(e: Event) {
    handleTitleInputAction(e, {
      updateTitle: notes.updateCurrentNoteTitle,
      scheduleAutoSave: notes.scheduleAutoSave,
    });
  }

  function handleManualSave() {
    if (saveTimeout) clearTimeout(saveTimeout);
    saveCanvas();
  }

  function handleUploadButtonClick() {
    fileInput.click();
  }

  function handleFileInputChange(e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    const files = input.files ? Array.from(input.files) : [];
    if (files.length === 0) return;

    uploading = true;
    // Place uploaded images at center of the visible canvas area
    const rect = containerEl?.getBoundingClientRect();
    const centerX = rect ? rect.left + rect.width / 2 : window.innerWidth / 2;
    const centerY = rect ? rect.top + rect.height / 2 : window.innerHeight / 2;
    handleImageFiles(files, centerX, centerY).finally(() => {
      uploading = false;
    });
    input.value = '';
  }

  function openMoreMenu(rect: DOMRect) {
    moreMenuTriggerRect = rect;
    showMoreMenu = true;
  }

  async function handleDelete() {
    await handleDeleteNote({
      goto,
      confirm: dialog.confirm,
      createDeleteCommand: (snapshot) => new DeleteCommand(snapshot),
      executeCommand: history.executeCommand,
      undoCommand: history.undo,
      getCurrentNote: () => notes.getCurrentNote(),
      loadNotes: notes.loadNotes,
      loadTree: tree.loadTree,
      clearCurrentNote: notes.clearCurrentNote,
      incrementTrash: trash.incrementTrashCount,
      decrementTrash: trash.decrementTrashCount,
      toast,
      strings: {
        confirmTitle: $_('dialog.confirm_title'),
        deleteConfirmMessage: $_('dialog.delete_note_confirm'),
        deleteConfirmText: $_('common.delete'),
        cancelText: $_('dialog.cancel'),
        deleteError: $_('component.editor.error_delete'),
        noteTrashed: $_('component.editor.note_trashed'),
        noteRestored: $_('component.editor.note_restored'),
        restoreError: $_('component.editor.error_restore'),
      },
    });
  }
</script>

<svelte:window onkeydown={handleKeydown} onpaste={handlePaste} />

<div class="canvas-editor">
  <!-- Toolbar -->
  {#if notes.getCurrentNote()}
    <CanvasEditorToolbar
      note={notes.getCurrentNote()}
      isMobile={ui.getIsMobile()}
      {saveStatus}
      {uploading}
      {showMoreMenu}
      syncing={getIsSyncing()}
      syncProgress={getSyncProgress()}
      pendingCount={getPendingCount()}
      isOnline={network.getIsOnline()}
      isEncryptionUnlocked={encryption.isEncryptionUnlocked()}
      focusModeActive={focusMode.isActive()}
      onTitleInput={handleTitleInput}
      onOpenSidebar={() => ui.setSidebarOpen(true)}
      onSave={handleManualSave}
      onUpload={handleUploadButtonClick}
      onShowHistory={() => (showVersionHistory = true)}
      onToggleFocus={focusMode.toggle}
      onOpenMoreMenu={openMoreMenu}
    />
  {/if}

  <!-- Canvas -->
  <div
    class="canvas-flow-container"
    class:drag-over={draggingOver}
    bind:this={containerEl}
    ondragover={handleDragOver}
    ondrop={handleDrop}
    ondragleave={handleDragLeave}
    role="application"
  >
    <SvelteFlow
      bind:nodes={flowNodes}
      bind:edges={flowEdges}
      {nodeTypes}
      fitView
      snapGrid={[20, 20]}
      onnodedragstop={handleNodeDragStop}
      onconnect={handleConnect}
      onnodecontextmenu={handleNodeContextMenu}
      bind:viewport
    >
      <Background variant={BackgroundVariant.Dots} gap={20} size={1} />
      <Controls />
      <MiniMap
        style="width: 160px; height: 120px; background: color-mix(in oklch, var(--color-card) 92%, transparent); border: 1px solid var(--color-border); border-radius: 8px;"
      />
      <Panel position="bottom-center">
        <CanvasToolbar onAction={handleToolbarAction} />
      </Panel>
    </SvelteFlow>
  </div>

  <!-- Context Menu -->
  {#if contextMenu}
    <CanvasContextMenu
      x={contextMenu.x}
      y={contextMenu.y}
      currentColor={flowNodes.find((n) => n.id === contextMenu?.nodeId)?.data?.color as
        | string
        | undefined}
      onClose={() => (contextMenu = null)}
      onDelete={handleDeleteNode}
      onDuplicate={handleDuplicateNode}
      onColorChange={handleColorChange}
    />
  {/if}

  <!-- Note Picker -->
  <CanvasNotePicker
    open={notePickerOpen}
    onSelect={handleNoteSelected}
    onClose={() => {
      notePickerOpen = false;
      pendingFileNodePosition = null;
    }}
  />
</div>

<!-- More Menu (rendered outside .canvas-editor for proper z-index) -->
{#if showMoreMenu}
  <CanvasMoreMenu
    onExport={handleExport}
    onShare={() => (showShareDialog = true)}
    onMove={() => (showMoveDialog = true)}
    onDelete={handleDelete}
    onClose={() => (showMoreMenu = false)}
    isEncrypted={notes.getCurrentNote()?.content_encrypted ?? false}
    triggerRect={moreMenuTriggerRect}
  />
{/if}

<!-- Move to folder dialog -->
{#if showMoveDialog && notes.getCurrentNote()}
  {#if dialogLoaders.moveToFolderDialog}
    {@const MoveToFolderDialog = dialogLoaders.moveToFolderDialog}
    <MoveToFolderDialog
      noteId={notes.getCurrentNote()!.id}
      currentFolder={notes.getCurrentNote()!.folder_path}
      onClose={() => (showMoveDialog = false)}
    />
  {/if}
{/if}

<!-- Version History Dialog -->
{#if showVersionHistory && notes.getCurrentNote() && dialogLoaders.versionHistoryDialog}
  {@const VersionHistoryDialog = dialogLoaders.versionHistoryDialog}
  <VersionHistoryDialog
    noteId={notes.getCurrentNote()!.id}
    noteTitle={notes.getCurrentNote()!.title}
    currentVersion={notes.getCurrentNote()!.version}
    currentContent={notes.getCurrentNote()!.content}
    onClose={() => (showVersionHistory = false)}
    onRestored={async () => {
      await notes.loadNote(noteId);
      toast.success($_('component.editor.version_restored'));
    }}
  />
{/if}

<!-- Share Dialog (lazy-loaded) -->
{#if showShareDialog && notes.getCurrentNote() && dialogLoaders.shareDialog}
  <dialogLoaders.shareDialog
    resourceType="note"
    resourceId={notes.getCurrentNote()!.id}
    isEncrypted={notes.getCurrentNote()!.content_encrypted ?? false}
    onClose={() => (showShareDialog = false)}
  />
{/if}

<!-- Hidden file input for image upload -->
<input
  type="file"
  accept="image/*"
  multiple
  bind:this={fileInput}
  onchange={handleFileInputChange}
  style="display:none"
/>

<style>
  .canvas-editor {
    display: flex;
    flex-direction: column;
    height: 100vh;
    height: 100dvh;
    background: var(--color-background);
  }

  .canvas-flow-container {
    flex: 1;
    min-height: 0;
    position: relative;
  }

  .canvas-flow-container.drag-over {
    outline: 2px dashed var(--color-ring);
    outline-offset: -2px;
  }

  /* Svelte Flow theme overrides */
  .canvas-flow-container :global(.svelte-flow) {
    --xy-background-color: var(--color-background);
    --xy-node-border-radius: 0.5rem;
    --xy-edge-stroke: var(--color-muted-foreground);
    --xy-edge-stroke-width: 1.5px;
    --xy-edge-stroke-selected: var(--color-ring);
    --xy-handle-background-color: var(--color-ring);
    --xy-handle-border-color: var(--color-background);
    --xy-minimap-background-color: color-mix(in oklch, var(--color-card) 92%, transparent);
    --xy-controls-button-background-color: var(--color-card);
    --xy-controls-button-border-color: var(--color-border);
    --xy-controls-button-color: var(--color-foreground);
  }

  .canvas-flow-container :global(.svelte-flow__controls) {
    border-radius: 8px;
    border: 1px solid var(--color-border);
    box-shadow: 0 4px 12px color-mix(in oklch, var(--color-foreground) 8%, transparent);
  }
</style>
