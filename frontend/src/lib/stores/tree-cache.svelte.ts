// Tree Cache - Granular cache, flat tree cache, and invalidation
// Part of the tree store module split.

import { SvelteMap } from 'svelte/reactivity';

import type { FlatTreeItem, FolderTreeNode, TreeNode } from './tree-index.svelte';

// Flattened tree cache (for virtual scrolling performance)
let flatTreeCache: FlatTreeItem[] | null = null;
let flatTreeCacheTimestamp = 0;

/** Index from node key (folder:path or note:id) to position in flatTreeCache. */
const flatTreeIndex = new SvelteMap<string, number>();

export function nodeKey(node: TreeNode): string {
  return node.type === 'folder' ? `folder:${node.path}` : `note:${node.id}`;
}

/** Rebuild the flat tree index from the current cache. */
function rebuildFlatTreeIndex(): void {
  flatTreeIndex.clear();
  if (!flatTreeCache) return;
  for (let i = 0; i < flatTreeCache.length; i++) {
    flatTreeIndex.set(nodeKey(flatTreeCache[i].node), i);
  }
}

/**
 * Dev-mode invariant check: verify flat cache matches a fresh flatten.
 * Only runs in dev builds to catch granular-update bugs.
 */
function assertFlatTreeConsistency(treeData: FolderTreeNode): void {
  if (!import.meta.env.DEV) return;
  if (!flatTreeCache) return;

  const fresh = buildFreshFlatTree(treeData);
  if (fresh.length !== flatTreeCache.length) {
    console.warn(
      `[Tree] Granular cache inconsistency: length ${flatTreeCache.length} vs expected ${fresh.length}. Falling back.`
    );
    invalidateFlatTreeCache();
    return;
  }

  for (let i = 0; i < fresh.length; i++) {
    if (nodeKey(fresh[i].node) !== nodeKey(flatTreeCache[i].node)) {
      console.warn(
        `[Tree] Granular cache inconsistency at index ${i}: ` +
          `got ${nodeKey(flatTreeCache[i].node)}, expected ${nodeKey(fresh[i].node)}. Falling back.`
      );
      invalidateFlatTreeCache();
      return;
    }
  }
}

/** Build a fresh flat tree without mutating the cache. */
function buildFreshFlatTree(root: FolderTreeNode): FlatTreeItem[] {
  const items: FlatTreeItem[] = [];
  let idx = 0;
  function flatten(node: TreeNode, level: number) {
    items.push({ node, level, index: idx++ });
    if (node.type === 'folder' && node.isExpanded) {
      for (const child of node.children) {
        flatten(child, level + 1);
      }
    }
  }
  for (const child of root.children) {
    flatten(child, 0);
  }
  return items;
}

/**
 * Invalidate flattened tree cache.
 * Should be called whenever the tree structure changes.
 */
export function invalidateFlatTreeCache(): void {
  flatTreeCache = null;
  flatTreeCacheTimestamp = 0;
  flatTreeIndex.clear();
}

/**
 * Get flattened tree representation for virtual scrolling.
 * Uses caching for performance - cache is invalidated on tree changes.
 */
export function getFlattenedTree(treeData: FolderTreeNode | null): FlatTreeItem[] {
  if (!treeData) return [];

  // Return cached result if available
  if (flatTreeCache && flatTreeCacheTimestamp > 0) {
    return flatTreeCache;
  }

  // Build flattened tree
  const items: FlatTreeItem[] = [];
  let index = 0;

  function flatten(node: TreeNode, level: number) {
    items.push({ node, level, index: index++ });

    // Only recurse into expanded folders
    if (node.type === 'folder' && node.isExpanded) {
      for (const child of node.children) {
        flatten(child, level + 1);
      }
    }
  }

  // Flatten all children of virtual root (skip root itself)
  for (const child of treeData.children) {
    flatten(child, 0);
  }

  // Cache result
  flatTreeCache = items;
  flatTreeCacheTimestamp = Date.now();
  rebuildFlatTreeIndex();

  return items;
}

/**
 * Perform a granular cache update for a folder toggle (expand/collapse).
 * Returns true if the granular update succeeded, false if a full invalidation is needed.
 */
export function granularToggleUpdate(
  path: string,
  wasExpanded: boolean,
  node: FolderTreeNode
): boolean {
  if (!flatTreeCache) return false;

  const folderIdx = flatTreeIndex.get(`folder:${path}`);
  if (folderIdx === undefined) return false;

  try {
    if (wasExpanded) {
      // Collapsing: remove all descendants after this folder
      const folderLevel = flatTreeCache[folderIdx].level;
      let removeCount = 0;
      for (let i = folderIdx + 1; i < flatTreeCache.length; i++) {
        if (flatTreeCache[i].level <= folderLevel) break;
        removeCount++;
      }
      if (removeCount > 0) {
        flatTreeCache.splice(folderIdx + 1, removeCount);
      }
    } else {
      // Expanding: insert flattened children after this folder
      const folderLevel = flatTreeCache[folderIdx].level;
      const childItems: FlatTreeItem[] = [];
      let idx = 0;
      function flattenChildren(parent: FolderTreeNode, level: number) {
        for (const child of parent.children) {
          childItems.push({ node: child, level, index: idx++ });
          if (child.type === 'folder' && child.isExpanded) {
            flattenChildren(child, level + 1);
          }
        }
      }
      flattenChildren(node, folderLevel + 1);
      if (childItems.length > 0) {
        flatTreeCache.splice(folderIdx + 1, 0, ...childItems);
      }
    }

    // Re-index after splice
    for (let i = 0; i < flatTreeCache.length; i++) {
      flatTreeCache[i].index = i;
    }
    rebuildFlatTreeIndex();
    flatTreeCacheTimestamp = Date.now();

    assertFlatTreeConsistency(node);
    return true;
  } catch {
    // Fallback to full invalidation on any error
    invalidateFlatTreeCache();
    return false;
  }
}

/**
 * Try a granular update for a folder color change.
 * Returns true if the update was applied in-place.
 */
export function granularFolderColorUpdate(
  folderPath: string,
  folderNode: FolderTreeNode,
  color: string | null
): boolean {
  if (!flatTreeCache) return false;

  const idx = flatTreeIndex.get(`folder:${folderPath}`);
  if (idx !== undefined && flatTreeCache[idx]) {
    folderNode.color = color;
    flatTreeCache[idx] = { ...flatTreeCache[idx], node: folderNode };
    flatTreeCacheTimestamp = Date.now();
    return true;
  }
  return false;
}

/**
 * Try a granular update for a note color change.
 * Returns true if the update was applied in-place.
 */
export function granularNoteColorUpdate(noteId: string, color: string | null): boolean {
  if (!flatTreeCache) return false;

  const idx = flatTreeIndex.get(`note:${noteId}`);
  if (idx !== undefined && flatTreeCache[idx]) {
    const item = flatTreeCache[idx];
    if (item.node.type === 'note') {
      item.node.color = color;
      flatTreeCache[idx] = { ...item, node: { ...item.node } };
      flatTreeCacheTimestamp = Date.now();
      return true;
    }
  }
  return false;
}

/**
 * Try a granular update for a note in the flat tree cache.
 * Returns true if the update was applied in-place, false if full invalidation is needed.
 */
export function granularNoteUpdate(noteId: string): boolean {
  if (!flatTreeCache) return false;

  const idx = flatTreeIndex.get(`note:${noteId}`);
  if (idx !== undefined && flatTreeCache[idx]) {
    flatTreeCache[idx] = { ...flatTreeCache[idx] };
    flatTreeCacheTimestamp = Date.now();
    return true;
  }
  return false;
}
