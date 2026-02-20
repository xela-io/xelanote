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
  import { ExternalLink, FileText, Group as GroupIcon, Type } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import type { Note } from '$lib/api/types';
  import { uploadImage } from '$lib/api/uploads';
  import { DeleteCommand } from '$lib/commands/DeleteCommand';
  import {
    CANVAS_LEGACY_NOTE_MIME,
    CANVAS_TREE_ITEM_MIME,
    parseDroppedSidebarNote,
  } from '$lib/components/canvas/canvas-note-drop';
  import { TOOL_DRAG_MIME, type ToolbarAction } from '$lib/components/canvas/canvas-toolbar-tools';
  import CanvasContextMenu from '$lib/components/canvas/CanvasContextMenu.svelte';
  import CanvasEditorToolbar from '$lib/components/canvas/CanvasEditorToolbar.svelte';
  import CanvasFileNode from '$lib/components/canvas/CanvasFileNode.svelte';
  import CanvasGroupNode from '$lib/components/canvas/CanvasGroupNode.svelte';
  import CanvasLinkNode from '$lib/components/canvas/CanvasLinkNode.svelte';
  import CanvasMoreMenu from '$lib/components/canvas/CanvasMoreMenu.svelte';
  import CanvasNotePicker from '$lib/components/canvas/CanvasNotePicker.svelte';
  import CanvasTextNode from '$lib/components/canvas/CanvasTextNode.svelte';
  import CanvasToolbar from '$lib/components/canvas/CanvasToolbar.svelte';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import {
    type DialogLoaderState,
    loadMoveToFolderDialog,
    loadShareDialog,
    loadVersionHistoryDialog,
    maybeLoadDialog,
  } from '$lib/editor/dialog-loaders';
  import { handleTitleInput as handleTitleInputAction } from '$lib/editor/editor-actions';
  import {
    handleDeleteNote,
    handleWikilinkClick as handleWikilinkAction,
  } from '$lib/editor/note-actions';
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
  import * as folders from '$lib/stores/folders.svelte';
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
    const noteError = notes.getError();
    const currentNote = notes.getCurrentNote();
    if (currentNote?.id === noteId) return;
    if (noteError === 'NOT_FOUND') return;
    notes.loadNote(noteId);
  });

  // Flow state
  let flowNodes = $state<FlowNode[]>([]);
  let flowEdges = $state<FlowEdge[]>([]);
  let saveStatus = $state<'saved' | 'saving' | 'unsaved'>('saved');
  let saveTimeout: ReturnType<typeof setTimeout> | null = null;
  let lastSavedContent = '';
  let copiedNodes: FlowNode[] = [];
  let copiedEdges: FlowEdge[] = [];
  let pasteCount = 0;

  // Viewport state for coordinate conversion (bound to SvelteFlow)
  let viewport = $state<{ x: number; y: number; zoom: number } | undefined>(undefined);
  let containerEl: HTMLDivElement | undefined = $state();
  let draggingOver = $state(false);
  let toolDragPreview = $state<{ action: ToolbarAction; x: number; y: number } | null>(null);

  // Allowed image MIME types (must match backend allowedTypes)
  const allowedImageTypes = ['image/png', 'image/jpeg', 'image/gif', 'image/webp'];

  // Context menu state
  let contextMenu = $state<{ x: number; y: number; nodeId: string } | null>(null);

  // Note picker state
  let notePickerOpen = $state(false);
  let pendingFileNodePosition = $state<{ x: number; y: number } | null>(null);
  let linkDialogOpen = $state(false);
  let pendingLinkNodePosition = $state<{ x: number; y: number } | null>(null);
  let linkUrl = $state('');
  let linkUrlError = $state<string | null>(null);

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

  // Mirror real note save state from notes store for toolbar save button/status.
  $effect(() => {
    const isDirty = notes.getIsDirty();
    const autoSaveStatus = notes.getAutoSaveStatus();
    if (!isDirty) {
      saveStatus = 'saved';
      return;
    }
    if (autoSaveStatus === 'saving' || autoSaveStatus === 'pending') {
      saveStatus = 'saving';
      return;
    }
    saveStatus = 'unsaved';
  });

  // Debounced save
  function scheduleSave() {
    if (saveTimeout) clearTimeout(saveTimeout);
    saveTimeout = setTimeout(() => {
      saveCanvasSnapshot();
      notes.scheduleAutoSave();
    }, 3000);
  }

  function saveCanvasSnapshot() {
    try {
      const data = flowToCanvas(flowNodes, flowEdges);
      const json = serializeCanvas(data);
      lastSavedContent = json;
      notes.updateCurrentNoteContent(json);
      return true;
    } catch (err) {
      console.error('Failed to save canvas:', err);
      return false;
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
  function handleToolbarAction(action: ToolbarAction, flowPosition?: { x: number; y: number }) {
    const position = flowPosition ?? { x: Math.random() * 400 + 100, y: Math.random() * 300 + 100 };

    switch (action) {
      case 'add-text': {
        const node = createTextNode(position.x, position.y);
        const { nodes } = canvasToFlow({ nodes: [node], edges: [] });
        flowNodes = [...flowNodes, ...nodes];
        scheduleSave();
        break;
      }
      case 'add-file': {
        pendingFileNodePosition = { x: position.x, y: position.y };
        notePickerOpen = true;
        break;
      }
      case 'add-link': {
        pendingLinkNodePosition = { x: position.x, y: position.y };
        linkUrl = '';
        linkUrlError = null;
        linkDialogOpen = true;
        break;
      }
      case 'add-group': {
        const node = createGroupNode(position.x, position.y);
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
    duplicateNodes([contextMenu.nodeId]);
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

  function handleRenameGroup() {
    if (!contextMenu) return;
    const nodeId = contextMenu.nodeId;
    const node = flowNodes.find((n) => n.id === nodeId);
    if (!node || node.type !== 'canvas-group') return;
    const currentLabel = (node.data.label as string) || 'Group';
    const newLabel = prompt('Group name:', currentLabel);
    if (newLabel !== null && newLabel.trim() && newLabel.trim() !== currentLabel) {
      flowNodes = flowNodes.map((n) => {
        if (n.id === nodeId) {
          return { ...n, data: { ...n.data, label: newLabel.trim() } };
        }
        return n;
      });
      scheduleSave();
    }
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

  function flowToContainerPosition(flowX: number, flowY: number): { x: number; y: number } {
    const vp = viewport ?? { x: 0, y: 0, zoom: 1 };
    return {
      x: flowX * vp.zoom + vp.x,
      y: flowY * vp.zoom + vp.y,
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
    const types = e.dataTransfer?.types;
    if (!types) return;
    const isFileDrag = types.includes('Files');
    const isToolDrag = types.includes(TOOL_DRAG_MIME);
    const isSidebarNoteDrag =
      types.includes(CANVAS_TREE_ITEM_MIME) || types.includes(CANVAS_LEGACY_NOTE_MIME);
    if (!isFileDrag && !isToolDrag && !isSidebarNoteDrag) return;
    e.preventDefault();
    // Sidebar drags use effectAllowed='move' – dropEffect must match or the
    // browser will suppress the drop event entirely (per HTML5 DnD spec).
    e.dataTransfer.dropEffect = isSidebarNoteDrag ? 'move' : 'copy';
    draggingOver = true;
    if (isToolDrag) {
      const action = parseDroppedToolAction(e);
      if (action) {
        const pos = clientToFlowPosition(e.clientX, e.clientY);
        toolDragPreview = { action, x: pos.x, y: pos.y };
      }
      return;
    }
    toolDragPreview = null;
  }

  function parseDroppedToolAction(e: DragEvent): ToolbarAction | null {
    const action = e.dataTransfer?.getData(TOOL_DRAG_MIME);
    if (
      action === 'add-text' ||
      action === 'add-file' ||
      action === 'add-link' ||
      action === 'add-group'
    ) {
      return action;
    }
    return null;
  }

  function getToolLabel(action: ToolbarAction): string {
    switch (action) {
      case 'add-text':
        return 'Text';
      case 'add-file':
        return 'Note';
      case 'add-link':
        return 'Link';
      case 'add-group':
        return 'Group';
    }
  }

  function getToolSize(action: ToolbarAction): { width: number; height: number } {
    if (action === 'add-group') return { width: 500, height: 400 };
    return { width: 300, height: 200 };
  }

  function getDropOriginForTool(
    _action: ToolbarAction,
    cursorFlowPos: { x: number; y: number }
  ): { x: number; y: number } {
    return {
      x: cursorFlowPos.x,
      y: cursorFlowPos.y,
    };
  }

  function handleDrop(e: DragEvent) {
    draggingOver = false;
    const droppedTool = parseDroppedToolAction(e);
    if (droppedTool) {
      e.preventDefault();
      const cursorPos = clientToFlowPosition(e.clientX, e.clientY);
      const originPos = getDropOriginForTool(droppedTool, cursorPos);
      handleToolbarAction(droppedTool, originPos);
      toolDragPreview = null;
      return;
    }
    const droppedNote = e.dataTransfer ? parseDroppedSidebarNote(e.dataTransfer) : null;
    if (droppedNote) {
      e.preventDefault();
      e.stopPropagation();
      const pos = clientToFlowPosition(e.clientX, e.clientY);
      const node = createFileNode(pos.x, pos.y, droppedNote.title, droppedNote.id);
      const { nodes } = canvasToFlow({ nodes: [node], edges: [] });
      flowNodes = [...flowNodes, ...nodes];
      scheduleSave();
      toolDragPreview = null;
      return;
    }
    if (!e.dataTransfer?.types.includes('Files')) return;
    e.preventDefault();
    const files = Array.from(e.dataTransfer.files);
    handleImageFiles(files, e.clientX, e.clientY);
    toolDragPreview = null;
  }

  function closeLinkDialog() {
    linkDialogOpen = false;
    linkUrl = '';
    linkUrlError = null;
    pendingLinkNodePosition = null;
  }

  function normalizeLinkUrl(raw: string): string {
    const trimmed = raw.trim();
    if (!trimmed) return '';
    if (/^[a-zA-Z][a-zA-Z\d+\-.]*:/.test(trimmed)) return trimmed;
    return `https://${trimmed}`;
  }

  function submitLinkDialog() {
    const normalized = normalizeLinkUrl(linkUrl);
    if (!normalized) {
      linkUrlError = $_('component.canvas.link_dialog.error_empty');
      return;
    }
    try {
      const parsed = new URL(normalized);
      if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
        linkUrlError = $_('component.canvas.link_dialog.error_invalid');
        return;
      }
      const pos = pendingLinkNodePosition ?? { x: 200, y: 200 };
      const node = createLinkNode(pos.x, pos.y, normalized);
      const { nodes } = canvasToFlow({ nodes: [node], edges: [] });
      flowNodes = [...flowNodes, ...nodes];
      scheduleSave();
      closeLinkDialog();
    } catch {
      linkUrlError = $_('component.canvas.link_dialog.error_invalid');
    }
  }

  function handleDragLeave(e: DragEvent) {
    // Only reset when leaving the container (not entering a child)
    if (containerEl && !containerEl.contains(e.relatedTarget as Node)) {
      draggingOver = false;
      toolDragPreview = null;
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

  function duplicateNodes(nodeIds: string[]) {
    const idSet = new Set(nodeIds);
    const originals = flowNodes.filter((n) => idSet.has(n.id));
    if (originals.length === 0) return;

    const duplicates = originals.map((original, index) => {
      const offset = 40 * (index + 1);
      const newNode: FlowNode = {
        ...original,
        id: `node-${crypto.randomUUID().slice(0, 8)}`,
        position: { x: original.position.x + offset, y: original.position.y + offset },
        selected: false,
        data: { ...original.data },
      };
      return newNode;
    });

    flowNodes = [...flowNodes, ...duplicates];
    scheduleSave();
  }

  function copySelectionToClipboard(): boolean {
    const selectedNodes = flowNodes.filter((n) => n.selected);
    if (selectedNodes.length === 0) return false;

    const selectedNodeIds = new Set(selectedNodes.map((n) => n.id));
    const connectedEdges = flowEdges.filter(
      (e) => selectedNodeIds.has(e.source) && selectedNodeIds.has(e.target)
    );

    copiedNodes = selectedNodes.map((node) => ({
      ...node,
      selected: false,
      data: { ...node.data },
      position: { ...node.position },
    }));
    copiedEdges = connectedEdges.map((edge) => ({
      ...edge,
      selected: false,
      data: edge.data ? { ...edge.data } : edge.data,
    }));
    pasteCount = 0;
    return true;
  }

  function pasteCopiedSelection() {
    if (copiedNodes.length === 0) return;

    pasteCount += 1;
    const offset = 40 * pasteCount;
    const idMap: Record<string, string> = {};

    const newNodes: FlowNode[] = copiedNodes.map((node) => {
      const newId = `node-${crypto.randomUUID().slice(0, 8)}`;
      idMap[node.id] = newId;
      return {
        ...node,
        id: newId,
        selected: true,
        position: {
          x: node.position.x + offset,
          y: node.position.y + offset,
        },
        data: { ...node.data },
      };
    });

    // Keep group-child relations when both were copied; otherwise detach from old parent.
    const remappedNodes = newNodes.map((node) => {
      if (!node.parentId) return node;
      const newParentId = idMap[node.parentId];
      return {
        ...node,
        parentId: newParentId,
      };
    });

    const newEdges: FlowEdge[] = copiedEdges
      .map((edge) => {
        const source = idMap[edge.source];
        const target = idMap[edge.target];
        if (!source || !target) return null;
        return {
          ...edge,
          id: generateEdgeId(),
          source,
          target,
          selected: true,
          data: edge.data
            ? {
                ...edge.data,
                fromNode: source,
                toNode: target,
              }
            : edge.data,
        } as FlowEdge;
      })
      .filter((edge): edge is FlowEdge => edge !== null);

    flowNodes = flowNodes
      .map((node): FlowNode => ({ ...node, selected: false }))
      .concat(remappedNodes);
    flowEdges = flowEdges.map((edge): FlowEdge => ({ ...edge, selected: false })).concat(newEdges);
    scheduleSave();
  }

  // Handle keyboard shortcuts
  function handleKeydown(e: KeyboardEvent) {
    if (isEditingText(e)) return;

    const key = e.key.toLowerCase();
    const hasModifier = e.ctrlKey || e.metaKey || e.altKey;

    if (e.key === 'Escape') {
      contextMenu = null;
      showMoreMenu = false;
      if (notePickerOpen) {
        notePickerOpen = false;
        pendingFileNodePosition = null;
      }
      if (linkDialogOpen) {
        closeLinkDialog();
      }
      return;
    }

    if (notePickerOpen || linkDialogOpen || showMoreMenu || contextMenu) return;

    if ((e.ctrlKey || e.metaKey) && key === 'c') {
      if (copySelectionToClipboard()) {
        e.preventDefault();
      }
      return;
    }

    if ((e.ctrlKey || e.metaKey) && key === 'v') {
      if (copiedNodes.length > 0) {
        e.preventDefault();
        pasteCopiedSelection();
      }
      return;
    }

    if (e.key === 'Delete' || e.key === 'Backspace') {
      const selectedNodes = flowNodes.filter((n) => n.selected);
      const selectedEdges = flowEdges.filter((ed) => ed.selected);
      if (selectedNodes.length > 0 || selectedEdges.length > 0) {
        e.preventDefault();
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

    if ((e.ctrlKey || e.metaKey) && key === 'd') {
      const selectedIds = flowNodes.filter((n) => n.selected).map((n) => n.id);
      if (selectedIds.length > 0) {
        e.preventDefault();
        duplicateNodes(selectedIds);
      }
      return;
    }

    if (hasModifier || e.repeat) return;

    switch (key) {
      case 't':
        e.preventDefault();
        handleToolbarAction('add-text');
        return;
      case 'n':
        e.preventDefault();
        handleToolbarAction('add-file');
        return;
      case 'l':
        e.preventDefault();
        handleToolbarAction('add-link');
        return;
      case 'g':
        e.preventDefault();
        handleToolbarAction('add-group');
        return;
    }
  }

  function isEditingText(e: KeyboardEvent): boolean {
    const target = e.target as HTMLElement;
    return (
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.isContentEditable ||
      !!target.closest('.cm-editor')
    );
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
    if (saveTimeout) {
      clearTimeout(saveTimeout);
      saveTimeout = null;
    }
    if (!saveCanvasSnapshot()) return;
    void notes.saveNote();
  }

  // Listen for custom events bubbling up from CanvasTextNode CodeMirror instances
  $effect(() => {
    const el = containerEl;
    if (!el) return;
    const onTextChange = () => scheduleSave();
    const onGroupLabelChange = () => scheduleSave();
    const onSave = () => handleManualSave();
    const onWikilinkClick = (e: Event) => {
      const detail = (e as CustomEvent<{ title: string }>).detail;
      if (detail?.title) handleCanvasWikilinkClick(detail.title);
    };
    el.addEventListener('canvastextchange', onTextChange);
    el.addEventListener('canvasgrouplabelchange', onGroupLabelChange);
    el.addEventListener('canvassave', onSave);
    el.addEventListener('wikilinkclick', onWikilinkClick);
    return () => {
      el.removeEventListener('canvastextchange', onTextChange);
      el.removeEventListener('canvasgrouplabelchange', onGroupLabelChange);
      el.removeEventListener('canvassave', onSave);
      el.removeEventListener('wikilinkclick', onWikilinkClick);
    };
  });

  function handleCanvasWikilinkClick(title: string) {
    handleWikilinkAction(title, {
      goto,
      confirm: dialog.confirm,
      getCurrentNote: () => notes.getCurrentNote(),
      getAllNotes: () => notes.getNotes(),
      createNote: notes.createNote,
      loadFolders: folders.loadFolders,
      strings: {
        confirmTitle: $_('dialog.confirm_title'),
        cancelText: $_('dialog.cancel'),
        createMissingMessage: $_('dialog.create_missing_note'),
        createMissingConfirmText: $_('common.confirm'),
      },
    });
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

<svelte:window
  onkeydown={handleKeydown}
  onpaste={handlePaste}
  ondragend={() => {
    draggingOver = false;
    toolDragPreview = null;
  }}
/>

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
    ondragovercapture={handleDragOver}
    ondropcapture={handleDrop}
    ondragleavecapture={handleDragLeave}
    role="application"
  >
    <SvelteFlow
      bind:nodes={flowNodes}
      bind:edges={flowEdges}
      {nodeTypes}
      fitView
      snapGrid={[20, 20]}
      onlyRenderVisibleElements
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

    {#if toolDragPreview}
      {@const previewSize = getToolSize(toolDragPreview.action)}
      {@const previewOrigin = getDropOriginForTool(toolDragPreview.action, {
        x: toolDragPreview.x,
        y: toolDragPreview.y,
      })}
      {@const previewPos = flowToContainerPosition(previewOrigin.x, previewOrigin.y)}
      <div
        class={`tool-drag-preview tool-drag-preview--${toolDragPreview.action}`}
        style={`left:${previewPos.x}px;top:${previewPos.y}px;width:${previewSize.width}px;height:${previewSize.height}px;`}
        aria-hidden="true"
      >
        {#if toolDragPreview.action === 'add-text'}
          <div class="tool-drag-preview-header">
            <Type size={14} />
            <span>{getToolLabel(toolDragPreview.action)}</span>
          </div>
          <div class="tool-drag-preview-text-lines">
            <span></span>
            <span></span>
            <span></span>
          </div>
        {:else if toolDragPreview.action === 'add-file'}
          <div class="tool-drag-preview-header">
            <FileText size={14} />
            <span>{getToolLabel(toolDragPreview.action)}</span>
          </div>
          <div class="tool-drag-preview-text-lines">
            <span></span>
            <span></span>
          </div>
        {:else if toolDragPreview.action === 'add-link'}
          <div class="tool-drag-preview-header">
            <ExternalLink size={14} />
            <span>{getToolLabel(toolDragPreview.action)}</span>
          </div>
          <div class="tool-drag-preview-link-url">example.com/path</div>
          <div class="tool-drag-preview-text-lines">
            <span></span>
          </div>
        {:else}
          <div class="tool-drag-preview-group-label">
            <GroupIcon size={14} />
            <span>{getToolLabel(toolDragPreview.action)}</span>
          </div>
        {/if}
      </div>
    {/if}
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
      onRename={flowNodes.find((n) => n.id === contextMenu?.nodeId)?.type === 'canvas-group'
        ? handleRenameGroup
        : undefined}
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

<!-- Add Link Dialog -->
<BaseDialog
  open={linkDialogOpen}
  title={$_('component.canvas.link_dialog.title')}
  onClose={closeLinkDialog}
  size="sm"
>
  {#snippet content()}
    <div class="space-y-3">
      <label for="canvas-link-input" class="text-sm font-medium text-foreground">
        {$_('component.canvas.link_dialog.url_label')}
      </label>
      <input
        id="canvas-link-input"
        type="text"
        bind:value={linkUrl}
        placeholder={$_('component.canvas.link_dialog.placeholder')}
        class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
        onkeydown={(e) => {
          if (e.key === 'Enter') {
            e.preventDefault();
            submitLinkDialog();
          }
        }}
      />
      {#if linkUrlError}
        <p class="text-sm text-red-600">{linkUrlError}</p>
      {/if}
    </div>
  {/snippet}
  {#snippet footer()}
    <button
      type="button"
      onclick={closeLinkDialog}
      class="px-4 py-2 text-sm hover:bg-accent rounded-md"
    >
      {$_('common.cancel')}
    </button>
    <button
      type="button"
      onclick={submitLinkDialog}
      class="px-4 py-2 text-sm bg-primary text-primary-foreground hover:bg-primary/90 rounded-md"
    >
      {$_('component.canvas.link_dialog.add_button')}
    </button>
  {/snippet}
</BaseDialog>

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

  .tool-drag-preview {
    position: absolute;
    z-index: 20;
    border-radius: 0.6rem;
    border: 1px solid var(--color-border);
    background: color-mix(in oklch, var(--color-card) 90%, transparent);
    box-shadow:
      0 0 0 1px color-mix(in oklch, var(--color-ring) 30%, transparent),
      0 14px 32px color-mix(in oklch, var(--color-foreground) 16%, transparent);
    pointer-events: none;
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    overflow: hidden;
  }

  .tool-drag-preview-header {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--color-foreground);
  }

  .tool-drag-preview-text-lines {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .tool-drag-preview-text-lines span {
    display: block;
    height: 8px;
    border-radius: 999px;
    background: color-mix(in oklch, var(--color-muted-foreground) 26%, transparent);
  }

  .tool-drag-preview-text-lines span:nth-child(1) {
    width: 88%;
  }

  .tool-drag-preview-text-lines span:nth-child(2) {
    width: 70%;
  }

  .tool-drag-preview-text-lines span:nth-child(3) {
    width: 55%;
  }

  .tool-drag-preview-link-url {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--color-muted-foreground);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tool-drag-preview-group-label {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: color-mix(in oklch, var(--color-foreground) 76%, var(--color-primary));
  }

  .tool-drag-preview--add-group {
    border-style: dashed;
    border-width: 1.5px;
    background: color-mix(in oklch, var(--color-primary) 10%, transparent);
  }

  .tool-drag-preview--add-text,
  .tool-drag-preview--add-file,
  .tool-drag-preview--add-link {
    border-left-width: 3px;
    border-left-color: var(--color-primary);
  }

  /* Svelte Flow theme overrides */
  .canvas-flow-container :global(.svelte-flow) {
    --xy-background-color: var(--color-background);
    --xy-node-border-radius: 0.5rem;
    --xy-edge-stroke: var(--color-muted-foreground);
    --xy-edge-stroke-width: 3px;
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
