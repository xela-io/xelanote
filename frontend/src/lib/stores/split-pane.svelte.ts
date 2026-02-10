/**
 * Split Pane Store for Desktop App
 *
 * Manages split pane layout for multi-editor views.
 * Uses Svelte 5 runes for reactive state.
 */

export type SplitDirection = 'horizontal' | 'vertical';

export interface SplitPane {
  id: string;
  groupId: string; // Reference to tab group
  size: number; // Percentage or pixels
}

export interface SplitNode {
  id: string;
  type: 'pane' | 'split';
  direction?: SplitDirection;
  children?: SplitNode[];
  pane?: SplitPane;
}

interface SplitState {
  root: SplitNode;
  activePaneId: string;
}

// Generate unique IDs
function generateId(): string {
  return `pane-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}

// Initial state with single pane
const initialPane: SplitPane = {
  id: 'main-pane',
  groupId: 'main',
  size: 100,
};

const initialRoot: SplitNode = {
  id: 'root',
  type: 'pane',
  pane: initialPane,
};

// State
const state = $state<SplitState>({
  root: initialRoot,
  activePaneId: 'main-pane',
});

// Getters
export function getState(): SplitState {
  return state;
}

export function getRoot(): SplitNode {
  return state.root;
}

export function getActivePaneId(): string {
  return state.activePaneId;
}

export function isSplit(): boolean {
  return state.root.type === 'split';
}

export function getPaneCount(): number {
  return countPanes(state.root);
}

function countPanes(node: SplitNode): number {
  if (node.type === 'pane') return 1;
  return node.children?.reduce((sum, child) => sum + countPanes(child), 0) ?? 0;
}

// Find a pane by ID
function findPane(node: SplitNode, paneId: string): SplitPane | undefined {
  if (node.type === 'pane' && node.pane?.id === paneId) {
    return node.pane;
  }
  if (node.children) {
    for (const child of node.children) {
      const found = findPane(child, paneId);
      if (found) return found;
    }
  }
  return undefined;
}

// Find parent of a node
function findParent(
  root: SplitNode,
  nodeId: string
): { parent: SplitNode; index: number } | undefined {
  if (!root.children) return undefined;

  const index = root.children.findIndex(
    (c) => c.id === nodeId || (c.type === 'pane' && c.pane?.id === nodeId)
  );
  if (index !== -1) {
    return { parent: root, index };
  }

  for (const child of root.children) {
    const found = findParent(child, nodeId);
    if (found) return found;
  }

  return undefined;
}

// Find node by pane ID
function findNodeByPaneId(node: SplitNode, paneId: string): SplitNode | undefined {
  if (node.type === 'pane' && node.pane?.id === paneId) {
    return node;
  }
  if (node.children) {
    for (const child of node.children) {
      const found = findNodeByPaneId(child, paneId);
      if (found) return found;
    }
  }
  return undefined;
}

// Actions
export function split(direction: SplitDirection, paneId?: string): string {
  const targetPaneId = paneId ?? state.activePaneId;

  // Create new pane
  const newPaneId = generateId();
  const newPane: SplitPane = {
    id: newPaneId,
    groupId: newPaneId, // Create new tab group
    size: 50,
  };

  const newPaneNode: SplitNode = {
    id: `node-${newPaneId}`,
    type: 'pane',
    pane: newPane,
  };

  // If only one pane exists, convert root to split
  if (state.root.type === 'pane') {
    const existingPane = state.root.pane!;
    existingPane.size = 50;

    const existingPaneNode: SplitNode = {
      id: `node-${existingPane.id}`,
      type: 'pane',
      pane: existingPane,
    };

    state.root = {
      id: 'root',
      type: 'split',
      direction,
      children: [existingPaneNode, newPaneNode],
    };
  } else {
    // Find the target pane and split it
    const targetNode = findNodeByPaneId(state.root, targetPaneId);
    const parentInfo = findParent(state.root, targetPaneId);

    if (targetNode && parentInfo) {
      const { parent, index } = parentInfo;

      // If parent has same direction, just add to children
      if (parent.direction === direction) {
        // Resize existing panes
        parent.children?.forEach((c) => {
          if (c.type === 'pane' && c.pane) {
            c.pane.size = 100 / (parent.children!.length + 1);
          }
        });
        newPane.size = 100 / (parent.children!.length + 1);
        parent.children?.splice(index + 1, 0, newPaneNode);
      } else {
        // Create new split node
        const existingPane = targetNode.pane!;
        existingPane.size = 50;

        const newSplitNode: SplitNode = {
          id: generateId(),
          type: 'split',
          direction,
          children: [{ ...targetNode }, newPaneNode],
        };

        parent.children![index] = newSplitNode;
      }
    }
  }

  state.activePaneId = newPaneId;
  return newPaneId;
}

export function closePane(paneId: string): void {
  if (getPaneCount() <= 1) {
    // Don't close last pane
    return;
  }

  const parentInfo = findParent(state.root, paneId);
  if (!parentInfo) return;

  const { parent, index } = parentInfo;

  // Remove the pane
  parent.children?.splice(index, 1);

  // If only one child remains, flatten the structure
  if (parent.children?.length === 1) {
    const remainingChild = parent.children[0];

    if (parent === state.root || parent.id === 'root') {
      // Replace root
      state.root = remainingChild;
      state.root.id = 'root';
    } else {
      // Replace parent with remaining child in grandparent
      const grandparentInfo = findParent(state.root, parent.id);
      if (grandparentInfo) {
        grandparentInfo.parent.children![grandparentInfo.index] = remainingChild;
      }
    }
  }

  // Update active pane if closed pane was active
  if (state.activePaneId === paneId) {
    const firstPane = findFirstPane(state.root);
    if (firstPane) {
      state.activePaneId = firstPane.id;
    }
  }

  // Recalculate sizes
  normalizeSizes(state.root);
}

function findFirstPane(node: SplitNode): SplitPane | undefined {
  if (node.type === 'pane') return node.pane;
  if (node.children?.length) {
    return findFirstPane(node.children[0]);
  }
  return undefined;
}

function normalizeSizes(node: SplitNode): void {
  if (node.type === 'split' && node.children) {
    const size = 100 / node.children.length;
    for (const child of node.children) {
      if (child.type === 'pane' && child.pane) {
        child.pane.size = size;
      }
      normalizeSizes(child);
    }
  }
}

export function activatePane(paneId: string): void {
  const pane = findPane(state.root, paneId);
  if (pane) {
    state.activePaneId = paneId;
  }
}

export function resizePane(paneId: string, newSize: number): void {
  const pane = findPane(state.root, paneId);
  if (pane) {
    pane.size = Math.max(10, Math.min(90, newSize)); // Clamp between 10-90%
  }
}

export function resizePanes(sizes: { paneId: string; size: number }[]): void {
  for (const { paneId, size } of sizes) {
    const pane = findPane(state.root, paneId);
    if (pane) {
      pane.size = Math.max(10, Math.min(90, size));
    }
  }
}

// Reset to single pane
export function resetLayout(): void {
  state.root = { ...initialRoot, pane: { ...initialPane } };
  state.activePaneId = 'main-pane';
}

// Toggle split (keyboard shortcut)
export function toggleSplit(direction: SplitDirection = 'horizontal'): void {
  if (isSplit()) {
    // If multiple panes, close active pane
    if (getPaneCount() > 1) {
      closePane(state.activePaneId);
    }
  } else {
    // Create split
    split(direction);
  }
}
