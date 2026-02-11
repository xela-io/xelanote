// Graph visualization store using Svelte 5 runes
import * as api from '$lib/api';

// State
let nodes = $state<api.GraphNode[]>([]);
let edges = $state<api.GraphEdge[]>([]);
let metadata = $state<api.GraphMetadata | null>(null);
let loading = $state(false);
let error = $state<string | null>(null);

// Filters
let folderFilter = $state<string | null>(null);
let searchFilter = $state<string | null>(null);

// Derived/computed state for filtered nodes
const filteredNodes = $derived(
  searchFilter
    ? nodes.filter((n) => n.title.toLowerCase().includes(searchFilter!.toLowerCase()))
    : nodes
);

// Derived/computed state for filtered edges (only edges where both nodes exist)
// Optimized: O(n+e) instead of O(n*e) by using Set for O(1) lookups
const filteredEdges = $derived.by(() => {
  const nodeIds = new Set(filteredNodes.map((n) => n.id));
  return edges.filter((e) => nodeIds.has(e.source_id) && nodeIds.has(e.target_id));
});

// Export getters
export function getNodes() {
  return filteredNodes;
}

export function getEdges() {
  return filteredEdges;
}

export function getLoading() {
  return loading;
}

export function getError() {
  return error;
}

export function getMetadata() {
  return metadata;
}

export function getFolderFilter() {
  return folderFilter;
}

export function getSearchFilter() {
  return searchFilter;
}

// Load global graph
export async function loadGlobalGraph(options?: { folder?: string }) {
  loading = true;
  error = null;
  try {
    const data = await api.getGlobalGraph(options);
    // Defensive null handling (similar to tree.svelte.ts fix)
    nodes = data.nodes ?? [];
    edges = data.edges ?? [];
    metadata = data.metadata ?? null;
  } catch (e) {
    error = e instanceof Error ? e.message : 'Failed to load graph';
    console.error('Failed to load graph:', e);
  } finally {
    loading = false;
  }
}

// Set folder filter and reload
export function setFolderFilter(folder: string | null) {
  folderFilter = folder;
  loadGlobalGraph(folder ? { folder } : undefined);
}

// Set search filter (client-side filtering via $derived)
export function setSearchFilter(query: string | null) {
  searchFilter = query;
}

// LocalStorage persistence for graph layout
export function saveLayout(layout: Record<string, { x: number; y: number }>) {
  if (typeof localStorage !== 'undefined') {
    try {
      localStorage.setItem('xelanote_graph_layout', JSON.stringify(layout));
    } catch (err) {
      console.error('Failed to save graph layout:', err);
    }
  }
}

export function loadLayout(): Record<string, { x: number; y: number }> | null {
  if (typeof localStorage !== 'undefined') {
    try {
      const saved = localStorage.getItem('xelanote_graph_layout');
      return saved ? parseGraphLayout(saved) : null;
    } catch (err) {
      console.error('Failed to load graph layout:', err);
    }
  }
  return null;
}

function parseGraphLayout(raw: string): Record<string, { x: number; y: number }> | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }

  if (!parsed || typeof parsed !== 'object') return null;
  const entries = Object.entries(parsed as Record<string, unknown>);
  const layout: Record<string, { x: number; y: number }> = {};

  for (const [id, value] of entries) {
    if (!value || typeof value !== 'object') return null;
    const point = value as { x?: unknown; y?: unknown };
    if (typeof point.x !== 'number' || typeof point.y !== 'number') return null;
    layout[id] = { x: point.x, y: point.y };
  }

  return layout;
}

// Clear all state (useful for logout)
export function clearGraphState() {
  nodes = [];
  edges = [];
  metadata = null;
  loading = false;
  error = null;
  folderFilter = null;
  searchFilter = null;
}
