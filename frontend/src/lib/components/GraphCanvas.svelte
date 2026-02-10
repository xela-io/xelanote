<script lang="ts">
  import type { ForceGraphGeneric,LinkObject, NodeObject } from 'force-graph';
  import { onMount } from 'svelte';

  import { goto } from '$app/navigation';
  import type { GraphEdge,GraphNode } from '$lib/api';

  type GraphNodeObj = NodeObject & GraphNode;
  type GraphLinkObj = LinkObject<GraphNodeObj> & { type: string };
  type ForceGraphInstance = ForceGraphGeneric<ForceGraphInstance, GraphNodeObj, GraphLinkObj>;

  const { nodes, edges } = $props<{ nodes: GraphNode[]; edges: GraphEdge[] }>();

  let containerRef = $state<HTMLDivElement | null>(null);
  let graphInstance = $state<ForceGraphInstance | null>(null);
  let selectedNode = $state<GraphNodeObj | null>(null);
  let tooltipPosition = $state<{ x: number; y: number }>({ x: 0, y: 0 });
  let loading = $state(true);

  let cleanup: (() => void) | null = null;

  onMount(() => {
    // Initialize graph asynchronously
    initGraph();

    // Cleanup on destroy
    return () => {
      if (cleanup) cleanup();
    };
  });

  async function initGraph() {
    // Lazy load force-graph
    const ForceGraphModule = await import('force-graph');
    // Type assertion needed - force-graph's default export is a factory function
    const ForceGraph = ForceGraphModule.default as unknown as () => (
      element: HTMLElement
    ) => ForceGraphInstance;
    loading = false;

    // Wait for next tick to ensure containerRef is bound
    await new Promise((resolve) => setTimeout(resolve, 0));

    if (!containerRef) {
      console.error('GraphCanvas: containerRef not available');
      return;
    }

    const width = containerRef.clientWidth || 1200;
    const height = containerRef.clientHeight || 600;

    // Initialize graph
    graphInstance = ForceGraph()(containerRef)
      .width(width)
      .height(height)
      .graphData({ nodes: [], links: [] })
      .nodeId('id')
      .nodeLabel('title')
      .nodeColor((node: GraphNodeObj) => (node.is_resolved ? '#3b82f6' : '#ef4444'))
      .nodeRelSize(6)
      .linkColor((link: GraphLinkObj) => (link.type === 'resolved' ? '#6366f1' : '#f87171'))
      .linkDirectionalArrowLength(3.5)
      .linkDirectionalArrowRelPos(1)
      .onNodeClick(handleNodeClick)
      .onBackgroundClick(() => {
        selectedNode = null;
      })
      .cooldownTicks(100)
      .onEngineStop(() => {
        if (graphInstance) {
          graphInstance.zoomToFit(400, 50);
        }
      });

    // Update with actual data if already available
    if (nodes && nodes.length > 0) {
      const graphData = {
        nodes: nodes,
        links: edges.map((e: GraphEdge) => ({
          source: e.source_id,
          target: e.target_id,
          type: e.type,
        })),
      };
      graphInstance.graphData(graphData);
    }

    // NOTE: Using force-graph defaults - works well out of the box
    // If custom forces needed later, install: npm install d3-force
    // Then import: import { forceManyBody, forceLink } from 'd3-force';
    // And configure: .d3Force('charge', forceManyBody().strength(-120))

    // Store cleanup function
    cleanup = () => {
      if (graphInstance) graphInstance._destructor();
    };
  }

  // Update graph when data changes
  $effect(() => {
    if (graphInstance && Array.isArray(nodes) && Array.isArray(edges)) {
      const graphData = {
        nodes: nodes,
        links: edges.map((e: GraphEdge) => ({
          source: e.source_id,
          target: e.target_id,
          type: e.type,
        })),
      };
      graphInstance.graphData(graphData);
    }
  });

  function handleNodeClick(node: GraphNodeObj, event: MouseEvent) {
    // Check if this is the already selected node
    if (selectedNode && selectedNode.id === node.id) {
      // Second tap - navigate to note if it's a resolved node
      if (node.is_resolved && !node.id.startsWith('unresolved:')) {
        goto(`/note/${node.id}`);
      }
      selectedNode = null;
    } else {
      // First tap - show tooltip
      selectedNode = node;
      // Position tooltip near the click
      tooltipPosition = {
        x: event.clientX,
        y: event.clientY,
      };
    }
  }
</script>

<div class="absolute inset-0">
  {#if loading}
    <div class="w-full h-full flex items-center justify-center text-muted-foreground">
      <div class="text-center">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-2"></div>
        <p>Loading graph...</p>
      </div>
    </div>
  {:else}
    <div
      bind:this={containerRef}
      class="w-full h-full bg-background"
      role="img"
      aria-label="Graph visualization"
    ></div>
  {/if}

  {#if selectedNode}
    <div
      class="absolute z-10 bg-popover text-popover-foreground border rounded-lg shadow-lg px-4 py-3 max-w-xs pointer-events-none"
      style="left: {tooltipPosition.x}px; top: {tooltipPosition.y}px; transform: translate(-50%, -100%) translateY(-8px);"
    >
      <div class="font-medium">{selectedNode.title}</div>
      <div class="text-xs text-muted-foreground mt-1">
        {selectedNode.is_resolved ? 'Tippe nochmal zum Öffnen' : 'Nicht aufgelöste Verknüpfung'}
      </div>
    </div>
  {/if}
</div>
