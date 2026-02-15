// Unified Tree Store - Merges folders and notes into a single tree view
// Using Svelte 5 runes

import { SvelteMap, SvelteSet } from 'svelte/reactivity';

import type { Folder, Note } from '$lib/api';
import * as api from '$lib/api';

// Tree Node Types
export type TreeNode = FolderTreeNode | NoteTreeNode;

export interface FolderTreeNode {
  type: 'folder';
  id: number;
  path: string;
  name: string;
  children: TreeNode[];
  noteCount: number;
  isExpanded: boolean;
  displayOrder: number;
  color?: string | null;
}

export interface NoteTreeNode {
  type: 'note';
  id: string;
  displayOrder: number;
  title: string;
  folderPath: string;
  color?: string | null;
  aiEnabled?: boolean;
}

// State
let treeData = $state<FolderTreeNode | null>(null);
let selectedFolderPath = $state<string | null>(null);
let selectedNoteId = $state<string | null>(null);
let isLoading = $state(false);

// Expanded state (persisted to localStorage)
const EXPANDED_KEY = 'xelanote_tree_expanded';
let expandedFolders = $state<Record<string, boolean>>({ '/': true });

// Flattened tree cache (for virtual scrolling performance)
let flatTreeCache: FlatTreeItem[] | null = null;
let flatTreeCacheTimestamp = 0;

// Granular cache update feature flag (localStorage-backed)
const GRANULAR_CACHE_KEY = 'xelanote_granular_tree_cache';
let useGranularTreeCache = false;
try {
  useGranularTreeCache = localStorage.getItem(GRANULAR_CACHE_KEY) === 'true';
} catch {
  // localStorage unavailable
}

/** Enable/disable granular tree cache updates (for testing/rollout). */
export function setGranularTreeCache(enabled: boolean): void {
  useGranularTreeCache = enabled;
  try {
    localStorage.setItem(GRANULAR_CACHE_KEY, String(enabled));
  } catch {
    // silent
  }
}

/** Index from node key (folder:path or note:id) to position in flatTreeCache. */
let flatTreeIndex = new Map<string, number>();

function nodeKey(node: TreeNode): string {
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
function assertFlatTreeConsistency(): void {
  if (!import.meta.env.DEV) return;
  if (!flatTreeCache || !treeData) return;

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

// Getters
export function getTreeData() {
  return treeData;
}

export function getSelectedFolderPath() {
  return selectedFolderPath;
}

export function getSelectedNoteId() {
  return selectedNoteId;
}

export function getIsLoading() {
  return isLoading;
}

// Actions

/** Max pagination iterations to prevent infinite loops (500 notes/page × 100 = 50,000 notes) */
const MAX_PAGINATION_ITERATIONS = 100;

/**
 * Fetch all notes using cursor-based pagination.
 * Returns all regular notes (not journal/recipe).
 */
async function fetchAllNotes(): Promise<Note[]> {
  const allNotes: Note[] = [];
  let cursor: string | undefined;
  let iterations = 0;

  do {
    const result = await api.listNotes({ limit: 500, cursor });
    allNotes.push(...(result.notes ?? []));
    cursor = result.next_cursor;
    iterations++;

    if (iterations >= MAX_PAGINATION_ITERATIONS) {
      console.error(
        `Note pagination stopped after ${iterations} pages (${allNotes.length} notes loaded). ` +
          `Not all notes may be visible.`
      );
      break;
    }
  } while (cursor);

  return allNotes;
}

export async function loadTree() {
  isLoading = true;
  try {
    // Load folders, notes, and journal notes in parallel
    // Journal notes are excluded by the default listNotes query (note_type='note' filter),
    // so we fetch them separately via the folder parameter.
    // Journal/folder queries return all matching notes without pagination.
    const [foldersResult, regularNotes, journalNotesResult] = await Promise.all([
      api.getFolders(),
      fetchAllNotes(),
      api.listNotes({ folder: '/Journal' }),
    ]);

    const folders = foldersResult.folders ?? [];
    const journalNotes = journalNotesResult.notes ?? [];

    // Merge notes, deduplicating by ID in case of overlap
    const seenIds = new SvelteSet(regularNotes.map((n) => n.id));
    const uniqueJournalNotes = journalNotes.filter((n) => !seenIds.has(n.id));
    const notes = [...regularNotes, ...uniqueJournalNotes];
    treeData = buildTree(folders, notes);

    // Invalidate cache after full tree reload
    invalidateFlatTreeCache();
  } catch (e) {
    console.error('Failed to load tree:', e);
  } finally {
    isLoading = false;
  }
}

function buildTree(folders: Folder[], notes: Note[]): FolderTreeNode {
  // Create map of folder_id -> FolderTreeNode
  const folderMap = new SvelteMap<number, FolderTreeNode>();

  // Sort folders by path length to process parents before children
  // Hide /Rezepte folder from tree (recipes have dedicated UI via RecipeButton)
  const sortedFolders = [...folders]
    .filter((f) => f.path !== '/Rezepte')
    .sort((a, b) => a.path.localeCompare(b.path));

  // Create folder nodes
  for (const folder of sortedFolders) {
    const node: FolderTreeNode = {
      type: 'folder',
      id: folder.id,
      path: folder.path,
      name: folder.name,
      children: [],
      noteCount: folder.note_count,
      isExpanded: loadExpandedState(folder.path),
      displayOrder: folder.display_order || 0,
      color: folder.color,
    };
    folderMap.set(folder.id, node);
  }

  // Build folder hierarchy
  let root: FolderTreeNode | null = null;
  const topLevelFolders: FolderTreeNode[] = [];

  for (const folder of sortedFolders) {
    const node = folderMap.get(folder.id)!;

    if (folder.path === '/') {
      root = node;
    } else if (
      folder.parent_id === null ||
      folder.parent_id === undefined ||
      folder.parent_id === 1
    ) {
      // Top-level folders (virtual root)
      // - parent_id=null: After Migration 025 (with omitempty removed)
      // - parent_id=undefined: After Migration 025 (with omitempty, field missing)
      // - parent_id=1: Backward compatibility (before Migration 025)
      topLevelFolders.push(node);
    } else if (folder.parent_id != null) {
      // Nested folders - add to their parent normally
      const parent = folderMap.get(folder.parent_id);
      if (parent) {
        parent.children.push(node);
      }
    }
  }

  // Build path-based lookup for O(1) folder resolution (instead of O(n) find per note)
  const pathMap = new Map<string, FolderTreeNode>();
  for (const node of folderMap.values()) {
    pathMap.set(node.path, node);
  }

  // Collect orphan notes (notes without a matching folder)
  const orphanNotes: NoteTreeNode[] = [];

  // Add notes to their respective folders
  for (const note of notes) {
    const noteNode: NoteTreeNode = {
      type: 'note',
      id: note.id,
      title: note.title,
      folderPath: note.folder_path,
      displayOrder: note.display_order || 0,
      color: note.color,
      aiEnabled: note.ai_enabled,
    };

    // Find the folder node for this note — O(1) via pathMap
    const folder = pathMap.get(note.folder_path);
    if (folder) {
      folder.children.push(noteNode);
    } else {
      // No matching folder - add to orphans (will be shown at root level)
      orphanNotes.push(noteNode);
    }
  }

  // Sort children: by display_order, then folders before notes
  function sortChildren(node: FolderTreeNode) {
    node.children.sort((a, b) => {
      // Primary sort: by display_order
      const orderDiff = a.displayOrder - b.displayOrder;
      if (orderDiff !== 0) return orderDiff;

      // Secondary sort: folders before notes
      if (a.type === 'folder' && b.type === 'note') return -1;
      if (a.type === 'note' && b.type === 'folder') return 1;

      // Tertiary sort: alphabetically by name/title
      if (a.type === 'folder' && b.type === 'folder') {
        return a.name.localeCompare(b.name);
      } else {
        return (a as NoteTreeNode).title.localeCompare((b as NoteTreeNode).title);
      }
    });

    // Recursively sort children
    for (const child of node.children) {
      if (child.type === 'folder') {
        sortChildren(child);
      }
    }
  }

  // Sort root's children (notes only, since folders are in topLevelFolders)
  if (root) {
    sortChildren(root);
  }

  // Sort top-level folders and their children
  for (const folder of topLevelFolders) {
    sortChildren(folder);
  }

  // Sort top-level folders by display_order
  topLevelFolders.sort((a, b) => a.displayOrder - b.displayOrder);

  // Create virtual container with ONLY top-level folders (Root is hidden)
  const virtualRoot: FolderTreeNode = {
    type: 'folder',
    id: 0,
    path: '',
    name: '',
    children: topLevelFolders,
    noteCount: 0,
    isExpanded: true,
    displayOrder: 0,
  };

  // Store root separately for access to root notes if needed
  if (root && root.children.length > 0) {
    // Add root notes as a special section at the top
    const rootNotes = root.children.filter((child) => child.type === 'note');
    virtualRoot.children.unshift(...rootNotes);
  }

  // Sort orphan notes by display_order before adding to top level
  orphanNotes.sort((a, b) => {
    const orderDiff = a.displayOrder - b.displayOrder;
    if (orderDiff !== 0) return orderDiff;
    return a.title.localeCompare(b.title);
  });

  if (orphanNotes.length > 0) {
    virtualRoot.children.unshift(...orphanNotes);
  }

  return virtualRoot;
}

// Flattened tree for virtual scrolling
export interface FlatTreeItem {
  node: TreeNode;
  level: number;
  index: number;
}

/**
 * Invalidate flattened tree cache.
 * Should be called whenever the tree structure changes.
 */
function invalidateFlatTreeCache(): void {
  flatTreeCache = null;
  flatTreeCacheTimestamp = 0;
  flatTreeIndex.clear();
}

/**
 * Get flattened tree representation for virtual scrolling.
 * Uses caching for performance - cache is invalidated on tree changes.
 */
export function getFlattenedTree(): FlatTreeItem[] {
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

// Selection

export function selectFolder(path: string) {
  selectedFolderPath = path;
  selectedNoteId = null;
}

export function selectNote(id: string) {
  selectedNoteId = id;
  selectedFolderPath = null;
}

export function clearSelection() {
  selectedFolderPath = null;
  selectedNoteId = null;
}

// Expand/Collapse

export function toggleExpanded(path: string) {
  const node = findFolderNode(treeData, path);
  if (!node) return;

  const wasExpanded = node.isExpanded;
  node.isExpanded = !wasExpanded;
  saveExpandedState(path, node.isExpanded);

  // Try granular cache update (splice children in/out)
  if (useGranularTreeCache && flatTreeCache) {
    const folderIdx = flatTreeIndex.get(`folder:${path}`);
    if (folderIdx !== undefined) {
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

        assertFlatTreeConsistency();
      } catch {
        // Fallback to full invalidation on any error
        invalidateFlatTreeCache();
      }
    } else {
      // Node not in index — fallback
      invalidateFlatTreeCache();
    }
  } else {
    invalidateFlatTreeCache();
  }

  // Force reactivity
  treeData = { ...treeData! };
}

function findFolderNode(root: FolderTreeNode | null, path: string): FolderTreeNode | null {
  if (!root) return null;
  if (root.path === path) return root;

  for (const child of root.children) {
    if (child.type === 'folder') {
      const found = findFolderNode(child, path);
      if (found) return found;
    }
  }

  return null;
}

function findFolderNodeById(root: FolderTreeNode | null, id: number): FolderTreeNode | null {
  if (!root) return null;
  if (root.id === id) return root;

  for (const child of root.children) {
    if (child.type === 'folder') {
      const found = findFolderNodeById(child, id);
      if (found) return found;
    }
  }

  return null;
}

/**
 * Find a TreeNode by type and id in the tree.
 */
export function findNodeById(type: 'folder' | 'note', id: string | number): TreeNode | null {
  if (!treeData) return null;

  function search(node: FolderTreeNode): TreeNode | null {
    for (const child of node.children) {
      if (type === 'folder' && child.type === 'folder' && child.id === Number(id)) return child;
      if (type === 'note' && child.type === 'note' && child.id === String(id)) return child;
      if (child.type === 'folder') {
        const found = search(child);
        if (found) return found;
      }
    }
    return null;
  }

  return search(treeData);
}

/**
 * Find the parent FolderTreeNode of a node identified by type and id.
 */
export function findParentOfNodeById(
  type: 'folder' | 'note',
  id: string | number
): FolderTreeNode | null {
  if (!treeData) return null;

  function search(parent: FolderTreeNode): FolderTreeNode | null {
    for (const child of parent.children) {
      if (type === 'folder' && child.type === 'folder' && child.id === Number(id)) return parent;
      if (type === 'note' && child.type === 'note' && child.id === String(id)) return parent;
      if (child.type === 'folder') {
        const found = search(child);
        if (found) return found;
      }
    }
    return null;
  }

  return search(treeData);
}

// LocalStorage Persistence

function loadExpandedState(path: string): boolean {
  return path === '/' || expandedFolders[path] === true;
}

function saveExpandedState(path: string, expanded: boolean) {
  if (expanded) {
    expandedFolders[path] = true;
  } else {
    delete expandedFolders[path];
  }
  expandedFolders = { ...expandedFolders };

  try {
    const paths = Object.keys(expandedFolders).filter((k) => expandedFolders[k]);
    localStorage.setItem(EXPANDED_KEY, JSON.stringify(paths));
  } catch (_e) {
    // localStorage might not be available
  }
}

export function loadExpandedStateFromStorage() {
  try {
    const stored = localStorage.getItem(EXPANDED_KEY);
    if (stored) {
      const paths = parseExpandedPaths(stored);
      if (!paths) {
        expandedFolders = { '/': true };
        return;
      }
      const expanded: Record<string, boolean> = { '/': true };
      for (const p of paths) {
        expanded[p] = true;
      }
      expandedFolders = expanded;
    } else {
      // Initialize with root expanded if no stored state
      expandedFolders = { '/': true };
    }
  } catch (_e) {
    // localStorage might not be available - use default
    expandedFolders = { '/': true };
  }
}

function parseExpandedPaths(raw: string): string[] | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }

  if (!Array.isArray(parsed)) return null;
  if (!parsed.every((entry) => typeof entry === 'string')) return null;
  return parsed;
}

// Folder Operations

export async function createFolder(path: string) {
  // Snapshot for rollback (optimistic UI pattern)
  const snapshot = JSON.parse(JSON.stringify(treeData));

  // Parse path to get parent and name
  const lastSlash = path.lastIndexOf('/');
  const parentPath = path.substring(0, lastSlash) || '/';
  const folderName = path.substring(lastSlash + 1);

  // Optimistic UI update - add temporary folder node
  const parent = findFolderNode(treeData, parentPath);
  if (parent && treeData) {
    const tempNode: FolderTreeNode = {
      type: 'folder',
      id: -Date.now(), // Temporary negative ID
      path: path,
      name: folderName,
      children: [],
      noteCount: 0,
      isExpanded: false,
      displayOrder: 0,
    };
    parent.children.push(tempNode);
    treeData = { ...treeData }; // Force reactivity
  }

  try {
    // API call in background
    await api.createFolder(path);
    // Success - reload to get real folder with server-assigned ID
    await loadTree();
    // Note: loadTree() already invalidates cache
  } catch (e) {
    // Revert to snapshot on error
    treeData = snapshot;
    invalidateFlatTreeCache(); // Invalidate after revert
    throw e;
  }
}

export async function moveFolder(folderId: number, newParentPath: string) {
  // Snapshot for rollback (optimistic UI pattern)
  const snapshot = JSON.parse(JSON.stringify(treeData));

  // Optimistic UI update - move folder in tree
  const folderToMove = findFolderNodeById(treeData, folderId);
  if (folderToMove && treeData) {
    // Remove from current parent
    const removeFromParent = (node: FolderTreeNode): boolean => {
      const index = node.children.findIndex(
        (child) => child.type === 'folder' && child.id === folderId
      );
      if (index !== -1) {
        node.children.splice(index, 1);
        return true;
      }
      for (const child of node.children) {
        if (child.type === 'folder' && removeFromParent(child)) {
          return true;
        }
      }
      return false;
    };

    removeFromParent(treeData);

    // Add to new parent
    const newParent = findFolderNode(treeData, newParentPath);
    if (newParent) {
      newParent.children.push(folderToMove);
    }

    treeData = { ...treeData }; // Force reactivity
  }

  try {
    // API call in background
    await api.moveFolder(folderId, newParentPath);
    // Success - reload to get updated paths and server state
    await loadTree();
    // Note: loadTree() already invalidates cache
  } catch (e) {
    // Revert to snapshot on error
    treeData = snapshot;
    invalidateFlatTreeCache(); // Invalidate after revert
    throw e;
  }
}

export async function deleteFolder(folderId: number) {
  // Snapshot for rollback (optimistic UI pattern)
  const snapshot = JSON.parse(JSON.stringify(treeData));

  // Optimistic UI update - remove folder from tree
  const removeFolder = (node: FolderTreeNode): boolean => {
    const index = node.children.findIndex(
      (child) => child.type === 'folder' && child.id === folderId
    );
    if (index !== -1) {
      node.children.splice(index, 1);
      return true;
    }
    // Recursively search children
    for (const child of node.children) {
      if (child.type === 'folder' && removeFolder(child)) {
        return true;
      }
    }
    return false;
  };

  if (treeData) {
    removeFolder(treeData);
    treeData = { ...treeData }; // Force reactivity
  }

  try {
    // API call in background
    await api.deleteFolder(folderId);
    // Success - reload to ensure consistency
    await loadTree();
    // Note: loadTree() already invalidates cache
  } catch (e) {
    // Revert to snapshot on error
    treeData = snapshot;
    invalidateFlatTreeCache(); // Invalidate after revert
    throw e;
  }
}

export async function renameFolder(folderId: number, newName: string) {
  // Snapshot for rollback (optimistic UI pattern)
  const snapshot = JSON.parse(JSON.stringify(treeData));

  // Optimistic UI update
  const node = findFolderNodeById(treeData, folderId);
  if (node) {
    node.name = newName;
    treeData = { ...treeData! }; // Force reactivity
  }

  try {
    // API call in background
    await api.renameFolder(folderId, newName);
    // Success - reload to get any server-side updates
    await loadTree();
    // Note: loadTree() already invalidates cache
  } catch (e) {
    // Revert to snapshot on error
    treeData = snapshot;
    invalidateFlatTreeCache(); // Invalidate after revert
    throw e;
  }
}

export async function reorderFolders(parentID: number | null, folderIds: number[]) {
  await api.reorderFolders(parentID, folderIds);
  await loadTree(); // Invalidates cache
}

export async function reorderNotes(folderPath: string, noteIds: string[]) {
  await api.reorderNotes(folderPath, noteIds);
  await loadTree(); // Invalidates cache
}

export async function updateFolderColor(folderId: number, color: string | null) {
  await api.updateFolderColor(folderId, color);

  // Try granular update for color-only change (no structural change)
  if (useGranularTreeCache && flatTreeCache) {
    const folderNode = findFolderNodeById(treeData, folderId);
    if (folderNode) {
      folderNode.color = color;
      // Update the cached item in-place (no splice needed)
      const idx = flatTreeIndex.get(`folder:${folderNode.path}`);
      if (idx !== undefined && flatTreeCache[idx]) {
        flatTreeCache[idx] = { ...flatTreeCache[idx], node: folderNode };
        flatTreeCacheTimestamp = Date.now();
        treeData = { ...treeData! };
        return;
      }
    }
  }

  await loadTree(); // Fallback: full reload invalidates cache
}

export async function updateNoteColor(noteId: string, color: string | null) {
  await api.updateNoteColor(noteId, color);

  // Try granular update for color-only change (no structural change)
  if (useGranularTreeCache && flatTreeCache) {
    const idx = flatTreeIndex.get(`note:${noteId}`);
    if (idx !== undefined && flatTreeCache[idx]) {
      const item = flatTreeCache[idx];
      if (item.node.type === 'note') {
        item.node.color = color;
        flatTreeCache[idx] = { ...item, node: { ...item.node } };
        flatTreeCacheTimestamp = Date.now();
        treeData = { ...treeData! };
        return;
      }
    }
  }

  await loadTree(); // Fallback: full reload invalidates cache
}

/**
 * Update a note's title in the tree without full reload.
 * Used for granular cache updates when only the title changes.
 */
export function updateNoteInTree(noteId: string, updates: { title?: string; color?: string | null }): void {
  if (!treeData) return;

  // Find and update the note node in the tree
  function findAndUpdate(parent: FolderTreeNode): boolean {
    for (let i = 0; i < parent.children.length; i++) {
      const child = parent.children[i];
      if (child.type === 'note' && child.id === noteId) {
        if (updates.title !== undefined) child.title = updates.title;
        if (updates.color !== undefined) child.color = updates.color;
        return true;
      }
      if (child.type === 'folder' && findAndUpdate(child)) {
        return true;
      }
    }
    return false;
  }

  if (!findAndUpdate(treeData)) return;

  // Granular cache update: replace item in-place
  if (useGranularTreeCache && flatTreeCache) {
    const idx = flatTreeIndex.get(`note:${noteId}`);
    if (idx !== undefined && flatTreeCache[idx]) {
      flatTreeCache[idx] = { ...flatTreeCache[idx] };
      flatTreeCacheTimestamp = Date.now();
    } else {
      invalidateFlatTreeCache();
    }
  } else {
    invalidateFlatTreeCache();
  }

  treeData = { ...treeData };
}

// Select and expand a folder (for breadcrumb navigation)
export function setSelectedFolder(path: string) {
  let cacheInvalidated = false;

  // Expand all parent folders along the path
  if (path && path !== '/') {
    const parts = path.split('/').filter(Boolean);
    let currentPath = '';
    for (const part of parts) {
      currentPath += '/' + part;
      const node = findFolderNode(treeData, currentPath);
      if (node && !node.isExpanded) {
        node.isExpanded = true;
        saveExpandedState(currentPath, true);
        cacheInvalidated = true;
      }
    }
  }

  // Select the folder
  selectedFolderPath = path;
  selectedNoteId = null;

  // Force reactivity
  if (treeData) {
    treeData = { ...treeData };
  }

  // Invalidate cache if any folder was expanded
  if (cacheInvalidated) {
    invalidateFlatTreeCache();
  }
}
