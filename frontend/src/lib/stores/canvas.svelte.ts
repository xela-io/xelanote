// Canvas store for JSON Canvas <-> Svelte Flow conversion
// Using Svelte 5 runes

import type { Edge as FlowEdge, Node as FlowNode } from '@xyflow/svelte';

import type {
  CanvasData,
  CanvasEdge,
  CanvasFileNode,
  CanvasGroupNode,
  CanvasLinkNode,
  CanvasNode,
  CanvasTextNode,
} from '$lib/api/types';

// Default empty canvas
export const EMPTY_CANVAS: CanvasData = { nodes: [], edges: [] };

/**
 * Parse canvas JSON content, returning CanvasData or null on failure.
 */
export function parseCanvas(content: string): CanvasData | null {
  if (!content || content.trim() === '' || content.trim() === '{}') {
    return { ...EMPTY_CANVAS };
  }
  try {
    const data = JSON.parse(content);
    if (!data || typeof data !== 'object') return null;
    return {
      nodes: Array.isArray(data.nodes) ? data.nodes : [],
      edges: Array.isArray(data.edges) ? data.edges : [],
    };
  } catch {
    return null;
  }
}

/**
 * Serialize CanvasData to JSON string.
 */
export function serializeCanvas(data: CanvasData): string {
  return JSON.stringify(data, null, 2);
}

/**
 * Convert JSON Canvas data to Svelte Flow nodes and edges.
 * Computes parentId from spatial containment (group bounding box contains node).
 */
export function canvasToFlow(data: CanvasData): { nodes: FlowNode[]; edges: FlowEdge[] } {
  // First pass: identify groups for containment
  const groups = data.nodes.filter((n) => n.type === 'group');

  const flowNodes: FlowNode[] = data.nodes.map((node) => {
    const base: FlowNode = {
      id: node.id,
      type: `canvas-${node.type}`,
      position: { x: node.x, y: node.y },
      measured: { width: node.width, height: node.height },
      data: { ...node },
      style: `width: ${node.width}px; height: ${node.height}px;`,
    };

    if (node.type === 'group') {
      base.type = 'canvas-group';
      // Groups have a lower z-index
      base.zIndex = -1;
    } else {
      // Check if node is contained within a group (spatial containment)
      const parent = groups.find(
        (g) =>
          g.id !== node.id &&
          node.x >= g.x &&
          node.y >= g.y &&
          node.x + node.width <= g.x + g.width &&
          node.y + node.height <= g.y + g.height
      );
      if (parent) {
        base.parentId = parent.id;
        // Position relative to parent
        base.position = { x: node.x - parent.x, y: node.y - parent.y };
        base.extent = 'parent';
      }
    }

    return base;
  });

  // Sort: groups first (so they render behind children)
  flowNodes.sort((a, b) => {
    const aIsGroup = a.type === 'canvas-group' ? 0 : 1;
    const bIsGroup = b.type === 'canvas-group' ? 0 : 1;
    return aIsGroup - bIsGroup;
  });

  const flowEdges: FlowEdge[] = data.edges.map((edge) => ({
    id: edge.id,
    source: edge.fromNode,
    target: edge.toNode,
    sourceHandle: edge.fromSide || undefined,
    targetHandle: edge.toSide || undefined,
    type: 'canvas-edge',
    markerEnd: edge.toEnd !== 'none' ? { type: 'arrowclosed' as const } : undefined,
    data: { ...edge },
    label: edge.label || undefined,
  }));

  return { nodes: flowNodes, edges: flowEdges };
}

/**
 * Convert Svelte Flow nodes and edges back to JSON Canvas data.
 * Strips parentId (not in spec) and converts positions back to absolute coordinates.
 */
export function flowToCanvas(nodes: FlowNode[], edges: FlowEdge[]): CanvasData {
  // Build a map for computing absolute positions (not reactive, local to function)
  // eslint-disable-next-line svelte/prefer-svelte-reactivity
  const nodeMap = new Map<string, FlowNode>();
  for (const node of nodes) {
    nodeMap.set(node.id, node);
  }

  function getAbsolutePosition(node: FlowNode): { x: number; y: number } {
    let x = node.position.x;
    let y = node.position.y;
    if (node.parentId) {
      const parent = nodeMap.get(node.parentId);
      if (parent) {
        const parentPos = getAbsolutePosition(parent);
        x += parentPos.x;
        y += parentPos.y;
      }
    }
    return { x, y };
  }

  const canvasNodes: CanvasNode[] = nodes.map((node) => {
    const abs = getAbsolutePosition(node);
    const w = node.measured?.width ?? node.width ?? 200;
    const h = node.measured?.height ?? node.height ?? 100;
    const nodeData = (node.data || {}) as Record<string, unknown>;

    const base = {
      id: node.id,
      x: Math.round(abs.x),
      y: Math.round(abs.y),
      width: Math.round(w),
      height: Math.round(h),
      ...(nodeData.color ? { color: nodeData.color as string } : {}),
    };

    const specType = (nodeData.type as string) || node.type?.replace('canvas-', '') || 'text';

    switch (specType) {
      case 'text':
        return { ...base, type: 'text' as const, text: (nodeData.text as string) || '' };
      case 'file':
        return {
          ...base,
          type: 'file' as const,
          file: (nodeData.file as string) || '',
          ...(nodeData.subpath ? { subpath: nodeData.subpath as string } : {}),
          ...(nodeData['x-xelanote-note-id']
            ? { 'x-xelanote-note-id': nodeData['x-xelanote-note-id'] as string }
            : {}),
        };
      case 'link':
        return { ...base, type: 'link' as const, url: (nodeData.url as string) || '' };
      case 'group':
        return {
          ...base,
          type: 'group' as const,
          ...(nodeData.label ? { label: nodeData.label as string } : {}),
        };
      default:
        return { ...base, type: 'text' as const, text: '' };
    }
  });

  const canvasEdges: CanvasEdge[] = edges.map((edge) => {
    const edgeData = (edge.data || {}) as Record<string, unknown>;
    return {
      id: edge.id,
      fromNode: edge.source,
      toNode: edge.target,
      ...(edgeData.fromSide ? { fromSide: edgeData.fromSide } : {}),
      ...(edgeData.toSide ? { toSide: edgeData.toSide } : {}),
      ...(edgeData.fromEnd ? { fromEnd: edgeData.fromEnd } : {}),
      ...(edgeData.toEnd ? { toEnd: edgeData.toEnd } : {}),
      ...(edgeData.color ? { color: edgeData.color } : {}),
      ...(edge.label ? { label: edge.label as string } : {}),
    } as CanvasEdge;
  });

  return { nodes: canvasNodes, edges: canvasEdges };
}

/**
 * Generate a unique node ID.
 */
export function generateNodeId(): string {
  return `node-${crypto.randomUUID().slice(0, 8)}`;
}

/**
 * Generate a unique edge ID.
 */
export function generateEdgeId(): string {
  return `edge-${crypto.randomUUID().slice(0, 8)}`;
}

/**
 * Create a default text node at the given position.
 */
export function createTextNode(x: number, y: number): CanvasTextNode {
  return {
    id: generateNodeId(),
    type: 'text',
    x: Math.round(x),
    y: Math.round(y),
    width: 300,
    height: 200,
    text: '',
  };
}

/**
 * Create a file node (note embed) at the given position.
 */
export function createFileNode(
  x: number,
  y: number,
  title: string,
  noteId?: string
): CanvasFileNode {
  return {
    id: generateNodeId(),
    type: 'file',
    x: Math.round(x),
    y: Math.round(y),
    width: 300,
    height: 200,
    file: title,
    ...(noteId ? { 'x-xelanote-note-id': noteId } : {}),
  };
}

/**
 * Create a link node at the given position.
 */
export function createLinkNode(x: number, y: number, url: string): CanvasLinkNode {
  return {
    id: generateNodeId(),
    type: 'link',
    x: Math.round(x),
    y: Math.round(y),
    width: 300,
    height: 200,
    url,
  };
}

/**
 * Create a group node at the given position.
 */
export function createGroupNode(x: number, y: number, label?: string): CanvasGroupNode {
  return {
    id: generateNodeId(),
    type: 'group',
    x: Math.round(x),
    y: Math.round(y),
    width: 500,
    height: 400,
    label: label || 'Group',
  };
}
