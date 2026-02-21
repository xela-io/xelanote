// Unified Tree Store - Merges folders and notes into a single tree view
// Using Svelte 5 runes
//
// This is the main barrel file. State declarations, getters, selection,
// expand/collapse, tree loading/building, and localStorage persistence live here.
// Sub-modules:
//   tree-index.svelte.ts  - Types & tree search/lookup helpers
//   tree-cache.svelte.ts  - Flat tree cache, granular cache updates
//   tree-operations.svelte.ts - CRUD operations (create/move/delete/rename/reorder/color)

import { SvelteMap, SvelteSet } from 'svelte/reactivity';

import type { Folder, Note } from '$lib/api';
import * as api from '$lib/api';

// Re-export types from tree-index
export type {
  FlatTreeItem,
  FolderTreeNode,
  NoteTreeNode,
  SortMode,
  TreeNode,
} from './tree-index.svelte';

import type { FolderTreeNode, NoteTreeNode, SortMode } from './tree-index.svelte';
import { findFolderNode } from './tree-index.svelte';

// Re-export cache functions
export { invalidateFlatTreeCache } from './tree-cache.svelte';
import {
  getFlattenedTree as getFlattenedTreeFromCache,
  granularToggleUpdate,
  invalidateFlatTreeCache,
} from './tree-cache.svelte';

// Re-export operations
export {
  createFolder,
  deleteFolder,
  moveFolder,
  renameFolder,
  reorderFolders,
  reorderNotes,
  updateFolderColor,
  updateNoteColor,
  updateNoteInTree,
} from './tree-operations.svelte';
// Re-export index lookup functions (these take treeData as parameter)
import {
  findNodeById as findNodeByIdRaw,
  findParentOfNodeById as findParentOfNodeByIdRaw,
} from './tree-index.svelte';
import { initOperations } from './tree-operations.svelte';

// State
let treeData = $state<FolderTreeNode | null>(null);
let selectedFolderPath = $state<string | null>(null);
let selectedNoteId = $state<string | null>(null);
let isLoading = $state(false);

// Cached data for sort-mode rebuild (avoid re-fetching from API)
let lastFolders: Folder[] | null = null;
let lastNotes: Note[] | null = null;

// Sort mode (persisted to localStorage)
const SORT_MODE_KEY = 'xelanote_sort_mode';
let sortMode = $state<SortMode>('manual');
try {
  const stored = localStorage.getItem(SORT_MODE_KEY);
  if (stored && ['manual', 'updated', 'title', 'created'].includes(stored)) {
    sortMode = stored as SortMode;
  }
} catch {
  // localStorage unavailable
}

// Expanded state (persisted to localStorage)
const EXPANDED_KEY = 'xelanote_tree_expanded';
let expandedFolders = $state<Record<string, boolean>>({ '/': true });

// Wire up the operations module with state accessors
initOperations({
  getTreeData: () => treeData,
  setTreeData: (data) => {
    treeData = data;
  },
  loadTree,
});

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

export function getSortMode(): SortMode {
  return sortMode;
}

export function isManualSortMode(): boolean {
  return sortMode === 'manual';
}

export function setSortMode(mode: SortMode): void {
  sortMode = mode;
  try {
    localStorage.setItem(SORT_MODE_KEY, mode);
  } catch {
    // localStorage unavailable
  }
  // Rebuild tree with new sort order
  if (treeData && lastFolders && lastNotes) {
    treeData = buildTree(lastFolders, lastNotes);
    invalidateFlatTreeCache();
  }
}

/**
 * Get flattened tree representation for virtual scrolling.
 * Wrapper that passes current treeData to the cache module.
 */
export function getFlattenedTree() {
  return getFlattenedTreeFromCache(treeData);
}

/**
 * Find a TreeNode by type and id in the tree.
 * Wrapper that closes over the module-level treeData.
 */
export function findNodeById(type: 'folder' | 'note', id: string | number) {
  return findNodeByIdRaw(treeData, type, id);
}

/**
 * Find the parent FolderTreeNode of a node identified by type and id.
 * Wrapper that closes over the module-level treeData.
 */
export function findParentOfNodeById(type: 'folder' | 'note', id: string | number) {
  return findParentOfNodeByIdRaw(treeData, type, id);
}

// Actions

/** Max pagination iterations to prevent infinite loops (500 notes/page x 100 = 50,000 notes) */
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
    const result = await api.listNotes({ limit: 500, cursor, fields: 'slim' });
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
      api.listNotes({ folder: '/Journal', fields: 'slim' }),
    ]);

    const folders = foldersResult.folders ?? [];
    const journalNotes = journalNotesResult.notes ?? [];

    // Merge notes, deduplicating by ID in case of overlap
    const seenIds = new SvelteSet(regularNotes.map((n) => n.id));
    const uniqueJournalNotes = journalNotes.filter((n) => !seenIds.has(n.id));
    const notes = [...regularNotes, ...uniqueJournalNotes];

    // Cache for sort-mode rebuild
    lastFolders = folders;
    lastNotes = notes;

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
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- local variable, not reactive state
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
      noteType: note.note_type,
      updatedAt: note.updated_at,
      createdAt: note.created_at,
    };

    // Find the folder node for this note -- O(1) via pathMap
    const folder = pathMap.get(note.folder_path);
    if (folder) {
      folder.children.push(noteNode);
    } else {
      // No matching folder - add to orphans (will be shown at root level)
      orphanNotes.push(noteNode);
    }
  }

  // Sort children: folders always by display_order, notes by current sortMode
  function sortChildren(node: FolderTreeNode) {
    node.children.sort((a, b) => {
      // Folders always come before notes
      if (a.type === 'folder' && b.type === 'note') return -1;
      if (a.type === 'note' && b.type === 'folder') return 1;

      // Folder-to-folder: always by displayOrder, then name
      if (a.type === 'folder' && b.type === 'folder') {
        const orderDiff = a.displayOrder - b.displayOrder;
        if (orderDiff !== 0) return orderDiff;
        return a.name.localeCompare(b.name);
      }

      // Note-to-note: sort by current mode
      const noteA = a as NoteTreeNode;
      const noteB = b as NoteTreeNode;

      switch (sortMode) {
        case 'updated': {
          // Newest first (DESC)
          const timeA = noteA.updatedAt || '';
          const timeB = noteB.updatedAt || '';
          const cmp = timeB.localeCompare(timeA);
          if (cmp !== 0) return cmp;
          return noteA.title.localeCompare(noteB.title);
        }
        case 'title':
          return noteA.title.localeCompare(noteB.title);
        case 'created': {
          // Newest first (DESC)
          const timeA = noteA.createdAt || '';
          const timeB = noteB.createdAt || '';
          const cmp = timeB.localeCompare(timeA);
          if (cmp !== 0) return cmp;
          return noteA.title.localeCompare(noteB.title);
        }
        case 'manual':
        default: {
          // Original behavior: displayOrder, then alphabetical
          const orderDiff = noteA.displayOrder - noteB.displayOrder;
          if (orderDiff !== 0) return orderDiff;
          return noteA.title.localeCompare(noteB.title);
        }
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

  // Sort orphan notes using the same sort mode as folder children
  orphanNotes.sort((a, b) => {
    switch (sortMode) {
      case 'updated': {
        const cmp = (b.updatedAt || '').localeCompare(a.updatedAt || '');
        if (cmp !== 0) return cmp;
        return a.title.localeCompare(b.title);
      }
      case 'title':
        return a.title.localeCompare(b.title);
      case 'created': {
        const cmp = (b.createdAt || '').localeCompare(a.createdAt || '');
        if (cmp !== 0) return cmp;
        return a.title.localeCompare(b.title);
      }
      case 'manual':
      default: {
        const orderDiff = a.displayOrder - b.displayOrder;
        if (orderDiff !== 0) return orderDiff;
        return a.title.localeCompare(b.title);
      }
    }
  });

  if (orphanNotes.length > 0) {
    virtualRoot.children.unshift(...orphanNotes);
  }

  return virtualRoot;
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
  if (!granularToggleUpdate(path, wasExpanded, node)) {
    invalidateFlatTreeCache();
  }

  // Force reactivity
  treeData = { ...treeData! };
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
