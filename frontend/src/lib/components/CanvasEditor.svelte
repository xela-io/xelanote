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
  import { ArrowLeft, Download } from 'lucide-svelte';

  import { goto } from '$app/navigation';
  import type { Note } from '$lib/api/types';
  import CanvasContextMenu from '$lib/components/canvas/CanvasContextMenu.svelte';
  import CanvasFileNode from '$lib/components/canvas/CanvasFileNode.svelte';
  import CanvasGroupNode from '$lib/components/canvas/CanvasGroupNode.svelte';
  import CanvasLinkNode from '$lib/components/canvas/CanvasLinkNode.svelte';
  import CanvasNotePicker from '$lib/components/canvas/CanvasNotePicker.svelte';
  import CanvasTextNode from '$lib/components/canvas/CanvasTextNode.svelte';
  import CanvasToolbar from '$lib/components/canvas/CanvasToolbar.svelte';
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
  import * as notes from '$lib/stores/notes.svelte';

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
  const zoom = $state(1);

  // Context menu state
  let contextMenu = $state<{ x: number; y: number; nodeId: string } | null>(null);

  // Note picker state
  let notePickerOpen = $state(false);
  let pendingFileNodePosition = $state<{ x: number; y: number } | null>(null);

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

  // Toolbar actions
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
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="canvas-editor">
  <!-- Header -->
  <div class="canvas-header">
    <button class="canvas-header-btn" onclick={() => goto('/')} title="Back">
      <ArrowLeft size={16} />
    </button>
    <h1 class="canvas-title">{noteTitle}</h1>
    <div class="canvas-header-right">
      <span
        class="canvas-save-status"
        class:saving={saveStatus === 'saving'}
        class:unsaved={saveStatus === 'unsaved'}
      >
        {saveStatus === 'saving' ? 'Saving...' : saveStatus === 'unsaved' ? 'Unsaved' : ''}
      </span>
      <span class="canvas-zoom">{Math.round(zoom * 100)}%</span>
      <button class="canvas-header-btn" onclick={handleExport} title="Export .canvas">
        <Download size={16} />
      </button>
    </div>
  </div>

  <!-- Canvas -->
  <div class="canvas-flow-container">
    <SvelteFlow
      bind:nodes={flowNodes}
      bind:edges={flowEdges}
      {nodeTypes}
      fitView
      snapGrid={[20, 20]}
      onnodedragstop={handleNodeDragStop}
      onconnect={handleConnect}
      onnodecontextmenu={handleNodeContextMenu}
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

<style>
  .canvas-editor {
    display: flex;
    flex-direction: column;
    height: 100vh;
    height: 100dvh;
    background: var(--color-background);
  }

  .canvas-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 16px;
    border-bottom: 1px solid var(--color-border);
    background: var(--color-background);
    height: 48px;
    gap: 12px;
    flex-shrink: 0;
  }

  .canvas-header-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: none;
    background: transparent;
    border-radius: 6px;
    color: var(--color-foreground);
    cursor: pointer;
  }

  .canvas-header-btn:hover {
    background: var(--color-accent);
  }

  .canvas-title {
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--color-foreground);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    min-width: 0;
  }

  .canvas-header-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .canvas-save-status {
    font-size: 0.75rem;
    color: var(--color-muted-foreground);
    transition: opacity 200ms ease;
  }

  .canvas-save-status.saving {
    color: oklch(62% 0.12 230);
  }

  .canvas-save-status.unsaved {
    color: var(--color-warning);
  }

  .canvas-zoom {
    font-size: 0.75rem;
    color: var(--color-muted-foreground);
  }

  .canvas-flow-container {
    flex: 1;
    min-height: 0;
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
