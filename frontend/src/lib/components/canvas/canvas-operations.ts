/**
 * Pure logic functions for canvas node/edge manipulation.
 * These are stateless helpers used by CanvasEditor.svelte.
 */
import type { Edge as FlowEdge, Node as FlowNode } from '@xyflow/svelte';

import { generateEdgeId } from '$lib/stores/canvas.svelte';

// --- Coordinate conversion ---

export interface Viewport {
  x: number;
  y: number;
  zoom: number;
}

export function clientToFlowPosition(
  clientX: number,
  clientY: number,
  containerRect: DOMRect | undefined,
  viewport: Viewport
): { x: number; y: number } {
  if (!containerRect) return { x: clientX, y: clientY };
  return {
    x: (clientX - containerRect.left - viewport.x) / viewport.zoom,
    y: (clientY - containerRect.top - viewport.y) / viewport.zoom,
  };
}

export function flowToContainerPosition(
  flowX: number,
  flowY: number,
  viewport: Viewport
): { x: number; y: number } {
  return {
    x: flowX * viewport.zoom + viewport.x,
    y: flowY * viewport.zoom + viewport.y,
  };
}

// --- Node operations ---

function generateNodeId(): string {
  return `node-${crypto.randomUUID().slice(0, 8)}`;
}

export function deleteNodes(
  flowNodes: FlowNode[],
  flowEdges: FlowEdge[],
  nodeIds: Set<string>
): { nodes: FlowNode[]; edges: FlowEdge[] } {
  return {
    nodes: flowNodes.filter((n) => !nodeIds.has(n.id)),
    edges: flowEdges.filter((e) => !nodeIds.has(e.source) && !nodeIds.has(e.target)),
  };
}

export function deleteSelection(
  flowNodes: FlowNode[],
  flowEdges: FlowEdge[]
): { nodes: FlowNode[]; edges: FlowEdge[]; changed: boolean } {
  const selectedNodes = flowNodes.filter((n) => n.selected);
  const selectedEdges = flowEdges.filter((e) => e.selected);
  if (selectedNodes.length === 0 && selectedEdges.length === 0) {
    return { nodes: flowNodes, edges: flowEdges, changed: false };
  }
  const selectedNodeIds = new Set(selectedNodes.map((n) => n.id));
  const selectedEdgeIds = new Set(selectedEdges.map((e) => e.id));
  return {
    nodes: flowNodes.filter((n) => !selectedNodeIds.has(n.id)),
    edges: flowEdges.filter(
      (e) =>
        !selectedEdgeIds.has(e.id) &&
        !selectedNodeIds.has(e.source) &&
        !selectedNodeIds.has(e.target)
    ),
    changed: true,
  };
}

export function setNodeColor(flowNodes: FlowNode[], nodeId: string, color: string): FlowNode[] {
  return flowNodes.map((n) => {
    if (n.id === nodeId) {
      return { ...n, data: { ...n.data, color: color || undefined } };
    }
    return n;
  });
}

export function renameGroupNode(
  flowNodes: FlowNode[],
  nodeId: string,
  newLabel: string
): FlowNode[] {
  return flowNodes.map((n) => {
    if (n.id === nodeId) {
      return { ...n, data: { ...n.data, label: newLabel } };
    }
    return n;
  });
}

// --- Duplicate / Copy / Paste ---

export function duplicateNodes(flowNodes: FlowNode[], nodeIds: string[]): FlowNode[] {
  const idSet = new Set(nodeIds);
  const originals = flowNodes.filter((n) => idSet.has(n.id));
  if (originals.length === 0) return flowNodes;

  const duplicates = originals.map((original, index) => {
    const offset = 40 * (index + 1);
    return {
      ...original,
      id: generateNodeId(),
      position: { x: original.position.x + offset, y: original.position.y + offset },
      selected: false,
      data: { ...original.data },
    } satisfies FlowNode;
  });

  return [...flowNodes, ...duplicates];
}

export interface ClipboardState {
  nodes: FlowNode[];
  edges: FlowEdge[];
  pasteCount: number;
}

export function copySelection(flowNodes: FlowNode[], flowEdges: FlowEdge[]): ClipboardState | null {
  const selectedNodes = flowNodes.filter((n) => n.selected);
  if (selectedNodes.length === 0) return null;

  const selectedNodeIds = new Set(selectedNodes.map((n) => n.id));
  const connectedEdges = flowEdges.filter(
    (e) => selectedNodeIds.has(e.source) && selectedNodeIds.has(e.target)
  );

  return {
    nodes: selectedNodes.map((node) => ({
      ...node,
      selected: false,
      data: { ...node.data },
      position: { ...node.position },
    })),
    edges: connectedEdges.map((edge) => ({
      ...edge,
      selected: false,
      data: edge.data ? { ...edge.data } : edge.data,
    })),
    pasteCount: 0,
  };
}

export function pasteClipboard(
  flowNodes: FlowNode[],
  flowEdges: FlowEdge[],
  clipboard: ClipboardState
): { nodes: FlowNode[]; edges: FlowEdge[]; pasteCount: number } {
  const pasteCount = clipboard.pasteCount + 1;
  const offset = 40 * pasteCount;
  const idMap: Record<string, string> = {};

  const newNodes: FlowNode[] = clipboard.nodes.map((node) => {
    const newId = generateNodeId();
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
    return { ...node, parentId: newParentId };
  });

  const newEdges: FlowEdge[] = clipboard.edges
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
        data: edge.data ? { ...edge.data, fromNode: source, toNode: target } : edge.data,
      } as FlowEdge;
    })
    .filter((edge): edge is FlowEdge => edge !== null);

  return {
    nodes: flowNodes.map((node): FlowNode => ({ ...node, selected: false })).concat(remappedNodes),
    edges: flowEdges.map((edge): FlowEdge => ({ ...edge, selected: false })).concat(newEdges),
    pasteCount,
  };
}

// --- Link URL normalization ---

export function normalizeLinkUrl(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed) return '';
  if (/^[a-zA-Z][a-zA-Z\d+\-.]*:/.test(trimmed)) return trimmed;
  return `https://${trimmed}`;
}

export function validateLinkUrl(normalized: string): boolean {
  try {
    const parsed = new URL(normalized);
    return parsed.protocol === 'http:' || parsed.protocol === 'https:';
  } catch {
    return false;
  }
}

// --- Keyboard helpers ---

export function isEditingText(target: HTMLElement): boolean {
  return (
    target.tagName === 'INPUT' ||
    target.tagName === 'TEXTAREA' ||
    target.isContentEditable ||
    !!target.closest('.cm-editor')
  );
}

export function isImagePasteTarget(target: HTMLElement): boolean {
  return target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable;
}
