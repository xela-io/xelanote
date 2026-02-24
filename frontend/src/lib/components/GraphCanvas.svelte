<script lang="ts">
  import type { ForceGraphGeneric, LinkObject, NodeObject } from 'force-graph';
  import { Crosshair, Focus, ScanSearch, Target } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { SvelteSet } from 'svelte/reactivity';

  import { goto } from '$app/navigation';
  import type { GraphEdge, GraphNode } from '$lib/api';

  type GraphNodeObj = NodeObject & GraphNode;
  type GraphLinkObj = LinkObject<GraphNodeObj> & {
    type: string;
    sourceId?: string;
    targetId?: string;
  };
  type ForceGraphInstance = ForceGraphGeneric<ForceGraphInstance, GraphNodeObj, GraphLinkObj>;
  type ForceGraphFactory = () => (element: HTMLElement) => ForceGraphInstance;
  type GraphThemeColors = {
    canvas: string;
    nodeResolved: string;
    nodeUnresolved: string;
    nodeMuted: string;
    nodeHighlight: string;
    nodeBorder: string;
    nodeText: string;
    linkResolved: string;
    linkUnresolved: string;
    linkMuted: string;
    tooltipBg: string;
    tooltipBorder: string;
  };

  const { nodes, edges } = $props<{ nodes: GraphNode[]; edges: GraphEdge[] }>();

  let containerRef = $state<HTMLDivElement | null>(null);
  let graphInstance = $state<ForceGraphInstance | null>(null);
  let selectedNode = $state<GraphNodeObj | null>(null);
  let connectedNodeIds = $state<Set<string>>(new Set());
  let tooltipPosition = $state<{ x: number; y: number }>({ x: 0, y: 0 });
  let tooltipHidden = $state(false);
  let focusNeighborsOnly = $state(false);
  let zoomLevel = $state(1);
  let loading = $state(true);
  let initError = $state<string | null>(null);
  let themeColors = $state<GraphThemeColors>({
    canvas: 'rgb(245, 245, 245)',
    nodeResolved: 'rgb(59, 130, 246)',
    nodeUnresolved: 'rgb(239, 68, 68)',
    nodeMuted: 'rgb(156, 163, 175)',
    nodeHighlight: 'rgb(16, 185, 129)',
    nodeBorder: 'rgb(75, 85, 99)',
    nodeText: 'rgb(17, 24, 39)',
    linkResolved: 'rgba(59, 130, 246, 0.45)',
    linkUnresolved: 'rgba(239, 68, 68, 0.45)',
    linkMuted: 'rgba(107, 114, 128, 0.2)',
    tooltipBg: 'rgb(255, 255, 255)',
    tooltipBorder: 'rgb(209, 213, 219)',
  });

  let cleanup: (() => void) | null = null;

  function parseForceGraphFactory(module: unknown): ForceGraphFactory {
    if (!module || typeof module !== 'object') {
      throw new Error('Invalid force-graph module');
    }
    const maybe = module as { default?: unknown };
    if (typeof maybe.default !== 'function') {
      throw new Error('force-graph default export is not a function');
    }
    return maybe.default as ForceGraphFactory;
  }

  onMount(() => {
    // Initialize graph asynchronously
    initGraph().catch((err) => {
      console.error('[GraphCanvas]', err);
      initError = err instanceof Error ? err.message : String(err);
      loading = false;
    });

    // Cleanup on destroy
    return () => {
      if (cleanup) cleanup();
    };
  });

  async function initGraph() {
    // Lazy load force-graph
    const ForceGraphModule = await import('force-graph');
    const ForceGraph = parseForceGraphFactory(ForceGraphModule);
    loading = false;

    // Wait for next tick to ensure containerRef is bound
    await new Promise((resolve) => setTimeout(resolve, 0));

    if (!containerRef) {
      console.error('GraphCanvas: containerRef not available');
      return;
    }

    themeColors = readThemeColors();

    const width = containerRef.clientWidth || 1200;
    const height = containerRef.clientHeight || 600;

    // Initialize graph
    graphInstance = ForceGraph()(containerRef)
      .width(width)
      .height(height)
      .graphData({ nodes: [], links: [] })
      .nodeId('id')
      .nodeLabel('title')
      .backgroundColor(themeColors.canvas)
      .nodeColor((node: GraphNodeObj) => getNodeColor(node))
      .nodeRelSize(6)
      .nodeCanvasObject(nodeCanvasObject)
      .linkColor((link: GraphLinkObj) => getLinkColor(link))
      .linkWidth((link: GraphLinkObj) => getLinkWidth(link))
      .linkCurvature(0.18)
      .linkDirectionalArrowColor((link: GraphLinkObj) => getLinkColor(link))
      .linkDirectionalArrowLength(3.5)
      .linkDirectionalArrowRelPos(1)
      .onNodeClick(handleNodeClick)
      .onBackgroundClick(() => {
        selectedNode = null;
        tooltipHidden = true;
        updateConnectedNodeIds(null);
        updateGraphData();
        applyVisualConfig();
        refreshGraph();
      })
      .onZoom((transform: { k?: number }) => {
        zoomLevel = typeof transform?.k === 'number' ? transform.k : 1;
      })
      .cooldownTicks(100)
      .onEngineStop(() => {
        if (graphInstance) {
          graphInstance.zoomToFit(400, 50);
        }
      });

    // Update with actual data if already available
    if (nodes && nodes.length > 0) {
      updateGraphData();
    }

    // NOTE: Using force-graph defaults - works well out of the box
    // If custom forces needed later, install: npm install d3-force
    // Then import: import { forceManyBody, forceLink } from 'd3-force';
    // And configure: .d3Force('charge', forceManyBody().strength(-120))

    // Store cleanup function
    cleanup = () => {
      if (graphInstance) graphInstance._destructor();
    };

    const resizeObserver = new ResizeObserver(() => {
      if (!graphInstance || !containerRef) return;
      const nextWidth = containerRef.clientWidth || 1200;
      const nextHeight = containerRef.clientHeight || 600;
      graphInstance.width(nextWidth).height(nextHeight);
      refreshGraph();
    });
    resizeObserver.observe(containerRef);

    const themeObserver = new MutationObserver(() => {
      themeColors = readThemeColors();
      applyVisualConfig();
    });
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class', 'style', 'data-theme'],
    });
    if (document.body) {
      themeObserver.observe(document.body, {
        attributes: true,
        attributeFilter: ['class', 'style', 'data-theme'],
      });
    }

    const oldCleanup = cleanup;
    cleanup = () => {
      resizeObserver.disconnect();
      themeObserver.disconnect();
      oldCleanup?.();
    };
  }

  // Update graph when data changes
  $effect(() => {
    if (graphInstance && Array.isArray(nodes) && Array.isArray(edges)) {
      const selectedId = selectedNode ? String(selectedNode.id) : null;
      const focusOnly = focusNeighborsOnly;
      void selectedId;
      void focusOnly;
      updateConnectedNodeIds(selectedNode ? String(selectedNode.id) : null);
      updateGraphData();
      applyVisualConfig();
      refreshGraph();
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
      tooltipHidden = true;
      updateConnectedNodeIds(null);
      applyVisualConfig();
      refreshGraph();
    } else {
      // First tap - show tooltip
      selectedNode = node;
      tooltipHidden = false;
      updateConnectedNodeIds(String(node.id));
      applyVisualConfig();
      refreshGraph();
      // Position tooltip near the click
      const rect = containerRef?.getBoundingClientRect();
      tooltipPosition = {
        x: rect ? event.clientX - rect.left : event.clientX,
        y: rect ? event.clientY - rect.top : event.clientY,
      };
    }
  }

  function updateConnectedNodeIds(selectedId: string | null) {
    if (!selectedId) {
      connectedNodeIds = new Set();
      return;
    }
    const next = new SvelteSet<string>([selectedId]);
    for (const edge of edges) {
      const sourceId = String(edge.source_id);
      const targetId = String(edge.target_id);
      if (sourceId === selectedId) next.add(targetId);
      if (targetId === selectedId) next.add(sourceId);
    }
    connectedNodeIds = next;
  }

  function getNodeColor(node: GraphNodeObj): string {
    if (!selectedNode) {
      return node.is_resolved ? themeColors.nodeResolved : themeColors.nodeUnresolved;
    }

    const nodeId = String(node.id);
    const selectedId = String(selectedNode.id);
    if (nodeId === selectedId) return themeColors.nodeHighlight;
    if (connectedNodeIds.has(nodeId)) {
      return node.is_resolved ? themeColors.nodeResolved : themeColors.nodeUnresolved;
    }
    return themeColors.nodeMuted;
  }

  function getLinkColor(link: GraphLinkObj): string {
    if (!selectedNode) {
      return link.type === 'resolved' ? themeColors.linkResolved : themeColors.linkUnresolved;
    }
    return isLinkConnectedToSelected(link)
      ? link.type === 'resolved'
        ? themeColors.linkResolved
        : themeColors.linkUnresolved
      : themeColors.linkMuted;
  }

  function getLinkWidth(link: GraphLinkObj): number {
    const base = zoomLevel < 0.75 ? 0.7 : zoomLevel < 1.2 ? 1.1 : 1.6;
    if (!selectedNode) return base;
    if (isLinkConnectedToSelected(link)) return base + 1.1;
    return focusNeighborsOnly ? 0.35 : Math.max(0.55, base - 0.4);
  }

  function isLinkConnectedToSelected(link: GraphLinkObj): boolean {
    if (!selectedNode) return false;
    const selectedId = String(selectedNode.id);
    const source = getLinkEndpointId(link.source, link.sourceId);
    const target = getLinkEndpointId(link.target, link.targetId);
    return source === selectedId || target === selectedId;
  }

  function getLinkEndpointId(
    endpoint: GraphLinkObj['source'] | GraphLinkObj['target'],
    fallback?: string
  ): string {
    if (typeof endpoint === 'string' || typeof endpoint === 'number') return String(endpoint);
    if (endpoint && typeof endpoint === 'object' && 'id' in endpoint) {
      return String((endpoint as { id: string | number }).id);
    }
    return fallback || '';
  }

  function nodeCanvasObject(
    node: GraphNodeObj,
    ctx: CanvasRenderingContext2D,
    globalScale: number
  ): void {
    const nodeColor = getNodeColor(node);
    const label = node.title || '';
    const fontSize = Math.max((zoomLevel < 0.8 ? 9 : 10) / globalScale, 3.3);
    const baseRadius = zoomLevel < 0.75 ? 4.2 : 5;
    const radius =
      selectedNode && String(node.id) === String(selectedNode.id) ? baseRadius + 1.7 : baseRadius;
    const x = Number(node.x || 0);
    const y = Number(node.y || 0);

    ctx.beginPath();
    ctx.arc(x, y, radius, 0, 2 * Math.PI, false);
    ctx.fillStyle = nodeColor;
    ctx.fill();
    ctx.strokeStyle = themeColors.nodeBorder;
    ctx.lineWidth = selectedNode && String(node.id) === String(selectedNode.id) ? 1.8 : 1;
    ctx.stroke();

    if (!shouldRenderNodeLabel(node, globalScale) || !label) return;

    ctx.font = `500 ${fontSize}px ui-sans-serif, system-ui, sans-serif`;
    const textWidth = ctx.measureText(label).width;
    const backgroundPadding = 4 / globalScale;
    const textY = y - 10;

    ctx.fillStyle = toRgba(themeColors.tooltipBg, 0.82);
    drawRoundedRect(
      ctx,
      x - textWidth / 2 - backgroundPadding,
      textY - fontSize + 1,
      textWidth + backgroundPadding * 2,
      fontSize + 4 / globalScale,
      4 / globalScale
    );
    ctx.fill();

    ctx.fillStyle = themeColors.nodeText;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'alphabetic';
    ctx.fillText(label, x, textY);
  }

  function drawRoundedRect(
    ctx: CanvasRenderingContext2D,
    x: number,
    y: number,
    width: number,
    height: number,
    radius: number
  ): void {
    const r = Math.min(radius, width / 2, height / 2);
    ctx.beginPath();
    ctx.moveTo(x + r, y);
    ctx.arcTo(x + width, y, x + width, y + height, r);
    ctx.arcTo(x + width, y + height, x, y + height, r);
    ctx.arcTo(x, y + height, x, y, r);
    ctx.arcTo(x, y, x + width, y, r);
    ctx.closePath();
  }

  function applyVisualConfig() {
    if (!graphInstance) return;
    graphInstance
      .backgroundColor(themeColors.canvas)
      .nodeColor((node: GraphNodeObj) => getNodeColor(node))
      .linkColor((link: GraphLinkObj) => getLinkColor(link))
      .linkWidth((link: GraphLinkObj) => getLinkWidth(link))
      .linkDirectionalArrowColor((link: GraphLinkObj) => getLinkColor(link));
  }

  function refreshGraph() {
    const instance = graphInstance as unknown as { refresh?: () => void };
    instance.refresh?.();
  }

  function updateGraphData() {
    if (!graphInstance || !Array.isArray(nodes) || !Array.isArray(edges)) return;

    const links = edges.map((e: GraphEdge) => ({
      source: e.source_id,
      target: e.target_id,
      type: e.type,
      sourceId: String(e.source_id),
      targetId: String(e.target_id),
    }));

    if (!focusNeighborsOnly || !selectedNode) {
      graphInstance.graphData({ nodes, links });
      return;
    }

    const selectedId = String(selectedNode.id);
    const visibleIds = new SvelteSet<string>([selectedId]);
    for (const edge of edges) {
      const sourceId = String(edge.source_id);
      const targetId = String(edge.target_id);
      if (sourceId === selectedId) visibleIds.add(targetId);
      if (targetId === selectedId) visibleIds.add(sourceId);
    }

    graphInstance.graphData({
      nodes: nodes.filter((node) => visibleIds.has(String(node.id))),
      links: links.filter((link) => link.sourceId === selectedId || link.targetId === selectedId),
    });
  }

  function shouldRenderNodeLabel(node: GraphNodeObj, globalScale: number): boolean {
    if (selectedNode) {
      const nodeId = String(node.id);
      if (nodeId === String(selectedNode.id)) return globalScale >= 0.8;
      if (connectedNodeIds.has(nodeId)) return globalScale >= 1;
    }
    return globalScale >= 1.35;
  }

  function fitGraph() {
    if (!graphInstance) return;
    graphInstance.zoomToFit(500, 70);
  }

  function resetGraphView() {
    if (!graphInstance) return;
    selectedNode = null;
    tooltipHidden = true;
    updateConnectedNodeIds(null);
    graphInstance.centerAt(0, 0, 350);
    graphInstance.zoom(1, 350);
    updateGraphData();
    applyVisualConfig();
    refreshGraph();
  }

  function centerSelectedNode() {
    if (!graphInstance || !selectedNode) return;
    graphInstance.centerAt(Number(selectedNode.x || 0), Number(selectedNode.y || 0), 350);
    graphInstance.zoom(Math.max(1.8, zoomLevel), 350);
    refreshGraph();
  }

  function toggleFocusNeighborsOnly() {
    focusNeighborsOnly = !focusNeighborsOnly;
    updateGraphData();
    applyVisualConfig();
    refreshGraph();
  }

  function readThemeColors(): GraphThemeColors {
    const resolved = (token: string, fallback: string) => {
      const cssValue = getComputedStyle(document.documentElement).getPropertyValue(token).trim();
      return normalizeColor(cssValue || fallback, fallback);
    };
    const primary = resolved('--color-primary', 'rgb(59, 130, 246)');
    const destructive = resolved('--color-destructive', 'rgb(239, 68, 68)');
    const mutedForeground = resolved('--color-muted-foreground', 'rgb(107, 114, 128)');
    const border = resolved('--color-border', 'rgb(209, 213, 219)');
    const background = resolved('--color-background', 'rgb(245, 245, 245)');
    const card = resolved('--color-card', 'rgb(255, 255, 255)');
    const foreground = resolved('--color-foreground', 'rgb(17, 24, 39)');
    const ring = resolved('--color-ring', 'rgb(34, 197, 94)');

    return {
      canvas: toRgba(background, 0.98),
      nodeResolved: toRgba(primary, 0.92),
      nodeUnresolved: toRgba(destructive, 0.88),
      nodeMuted: toRgba(mutedForeground, 0.38),
      nodeHighlight: toRgba(ring, 0.95),
      nodeBorder: toRgba(border, 0.75),
      nodeText: foreground,
      linkResolved: toRgba(primary, 0.42),
      linkUnresolved: toRgba(destructive, 0.42),
      linkMuted: toRgba(mutedForeground, 0.2),
      tooltipBg: toRgba(card, 0.96),
      tooltipBorder: toRgba(border, 0.9),
    };
  }

  function normalizeColor(color: string, fallback: string): string {
    const probe = document.createElement('span');
    probe.style.color = fallback;
    probe.style.color = color;
    probe.style.position = 'fixed';
    probe.style.opacity = '0';
    probe.style.pointerEvents = 'none';
    document.body.appendChild(probe);
    const resolved = getComputedStyle(probe).color || fallback;
    probe.remove();
    return resolved;
  }

  function toRgba(color: string, alpha: number): string {
    const match = color.match(/rgba?\(([^)]+)\)/i);
    if (!match) return color;
    const channels = match[1]
      .split(',')
      .map((part) => Number.parseFloat(part.trim()))
      .filter((num) => Number.isFinite(num));
    if (channels.length < 3) return color;
    return `rgba(${Math.round(channels[0])}, ${Math.round(channels[1])}, ${Math.round(channels[2])}, ${alpha})`;
  }
</script>

<div class="absolute inset-0 graph-canvas-wrapper">
  {#if initError}
    <div class="w-full h-full flex items-center justify-center text-muted-foreground">
      <div class="text-center">
        <p class="text-destructive mb-2">Failed to load graph</p>
        <p class="text-sm">{initError}</p>
      </div>
    </div>
  {:else if loading}
    <div class="w-full h-full flex items-center justify-center text-muted-foreground">
      <div class="text-center">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-2"></div>
        <p>Loading graph...</p>
      </div>
    </div>
  {:else}
    <div
      bind:this={containerRef}
      class="w-full h-full"
      role="img"
      aria-label="Graph visualization"
    ></div>
  {/if}

  {#if selectedNode && !tooltipHidden}
    <div
      class="absolute z-10 text-popover-foreground rounded-xl shadow-lg px-4 py-3 max-w-xs pointer-events-none graph-tooltip"
      class:tooltip-resolved={selectedNode.is_resolved}
      class:tooltip-unresolved={!selectedNode.is_resolved}
      style="left: {tooltipPosition.x}px; top: {tooltipPosition.y}px; transform: translate(-50%, -100%) translateY(-8px);"
    >
      <div class="font-medium">{selectedNode.title}</div>
      <div class="text-xs text-muted-foreground mt-1">
        {selectedNode.is_resolved ? 'Tippe nochmal zum Öffnen' : 'Nicht aufgelöste Verknüpfung'}
      </div>
    </div>
  {/if}

  <div class="floating-toolbar">
    <button
      class="tool-btn"
      type="button"
      onclick={fitGraph}
      title="Fit graph"
      aria-label="Fit graph"
    >
      <ScanSearch size={15} />
    </button>
    <button
      class="tool-btn"
      type="button"
      onclick={resetGraphView}
      title="Reset view"
      aria-label="Reset view"
    >
      <Target size={15} />
    </button>
    <button
      class="tool-btn"
      class:tool-btn-disabled={!selectedNode}
      type="button"
      onclick={centerSelectedNode}
      title="Center selected"
      aria-label="Center selected"
      disabled={!selectedNode}
    >
      <Crosshair size={15} />
    </button>
    <button
      class="tool-btn"
      class:tool-btn-active={focusNeighborsOnly}
      type="button"
      onclick={toggleFocusNeighborsOnly}
      title="Nur Nachbarn anzeigen"
      aria-label="Nur Nachbarn anzeigen"
    >
      <Focus size={15} />
    </button>
    <div class="zoom-pill">{Math.round(zoomLevel * 100)}%</div>
  </div>
</div>

<style>
  .graph-canvas-wrapper {
    background:
      radial-gradient(
        circle at 15% 12%,
        color-mix(in oklch, var(--color-primary), transparent 92%),
        transparent 55%
      ),
      radial-gradient(
        circle at 85% 84%,
        color-mix(in oklch, var(--color-accent), transparent 90%),
        transparent 50%
      ),
      var(--color-background);
  }

  .graph-tooltip {
    border: 1px solid color-mix(in oklch, var(--color-border), transparent 10%);
    background: color-mix(in oklch, var(--color-card), transparent 8%);
    backdrop-filter: blur(8px);
  }

  .graph-tooltip.tooltip-resolved {
    box-shadow:
      0 10px 30px color-mix(in oklch, var(--color-primary), transparent 84%),
      0 4px 12px color-mix(in oklch, var(--color-foreground), transparent 90%);
  }

  .graph-tooltip.tooltip-unresolved {
    box-shadow:
      0 10px 30px color-mix(in oklch, var(--color-destructive), transparent 84%),
      0 4px 12px color-mix(in oklch, var(--color-foreground), transparent 90%);
  }

  .floating-toolbar {
    position: absolute;
    right: max(12px, calc(env(safe-area-inset-right, 0px) + 8px));
    top: max(12px, calc(env(safe-area-inset-top, 0px) + 8px));
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px;
    border-radius: 999px;
    border: 1px solid color-mix(in oklch, var(--color-border), transparent 12%);
    background: color-mix(in oklch, var(--color-card), transparent 12%);
    backdrop-filter: blur(10px);
    box-shadow: 0 8px 28px color-mix(in oklch, var(--color-foreground), transparent 90%);
    z-index: 40;
    pointer-events: auto;
  }

  .tool-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border-radius: 999px;
    color: var(--color-foreground);
    transition:
      background-color 0.14s ease,
      color 0.14s ease,
      transform 0.14s ease;
  }

  .tool-btn:hover:not(.tool-btn-disabled) {
    background: color-mix(in oklch, var(--color-accent), transparent 70%);
    transform: translateY(-1px);
  }

  .tool-btn-active {
    color: var(--color-primary);
    background: color-mix(in oklch, var(--color-primary), transparent 84%);
  }

  .tool-btn-disabled {
    opacity: 0.45;
    cursor: default;
  }

  .zoom-pill {
    padding: 0 8px;
    height: 24px;
    border-radius: 999px;
    border: 1px solid color-mix(in oklch, var(--color-border), transparent 16%);
    color: var(--color-muted-foreground);
    font-size: 11px;
    font-weight: 600;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 50px;
  }

  @media (pointer: coarse) {
    .floating-toolbar {
      gap: 8px;
      padding: 8px;
    }

    .tool-btn {
      width: 40px;
      height: 40px;
    }

    .zoom-pill {
      height: 28px;
      min-width: 58px;
      font-size: 12px;
    }
  }
</style>
