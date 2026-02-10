// Folders store using Svelte 5 runes

import { SvelteMap } from 'svelte/reactivity';
import * as api from '$lib/api';
import type { Folder } from '$lib/api';

// State
let folders = $state<Folder[]>([]);
let selectedFolder = $state<string | null>(null);
// Use object instead of Set for better reactivity tracking
let expandedFolders = $state<Record<string, boolean>>({ '/': true });
let isLoading = $state(false);
let cachedFolderTree: FolderNode | null = null;
let cachedFolderTreeKey = '';

// FolderNode for tree structure
export interface FolderNode {
  path: string;
  name: string;
  children: FolderNode[];
  noteCount: number;
}

// Getters
export function getFolders() {
  return folders;
}

export function getSelectedFolder() {
  return selectedFolder;
}

export function isExpanded(path: string) {
  return expandedFolders[path] === true;
}

export function getIsLoading() {
  return isLoading;
}

// Actions
export async function loadFolders() {
  isLoading = true;
  try {
    const result = await api.getFolders();
    folders = result.folders;
    // Ensure root folder is always expanded
    expandedFolders['/'] = true;
  } catch (e) {
    console.error('Failed to load folders:', e);
  } finally {
    isLoading = false;
  }
}

export function selectFolder(path: string | null) {
  selectedFolder = path;
}

export function toggleExpanded(path: string) {
  if (expandedFolders[path]) {
    delete expandedFolders[path];
  } else {
    expandedFolders[path] = true;
  }
  // Force reactivity by creating new object
  expandedFolders = { ...expandedFolders };
  // Persist to localStorage
  saveExpandedState();
}

// Build folder tree from flat list
export function getFolderTree(): FolderNode {
  const keyParts = folders.map((f) => `${f.path}:${f.note_count}`);
  keyParts.sort();
  const cacheKey = keyParts.join('|');

  if (cachedFolderTree && cacheKey === cachedFolderTreeKey) {
    return cachedFolderTree;
  }

  const root: FolderNode = {
    path: '/',
    name: 'Alle Notizen',
    children: [],
    noteCount: 0,
  };

  // Calculate total note count for root
  let totalNotes = 0;
  for (const f of folders) {
    totalNotes += f.note_count;
  }
  root.noteCount = totalNotes;

  // Build tree from paths
  const nodeMap = new SvelteMap<string, FolderNode>();
  nodeMap.set('/', root);

  // Sort folders by path to process parents before children
  const sortedFolders = [...folders].sort((a, b) => a.path.localeCompare(b.path));

  for (const folder of sortedFolders) {
    if (folder.path === '/') {
      // Update root note count (notes directly in root)
      const rootInfo = folders.find((f) => f.path === '/');
      if (rootInfo) {
        // Root's noteCount shows total, individual folder shows its count
      }
      continue;
    }

    // Split path and create nodes
    const parts = folder.path.split('/').filter(Boolean);
    let currentPath = '';
    let parentNode = root;

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      currentPath = currentPath + '/' + part;

      let node = nodeMap.get(currentPath);
      if (!node) {
        node = {
          path: currentPath,
          name: part,
          children: [],
          noteCount: 0,
        };
        nodeMap.set(currentPath, node);
        parentNode.children.push(node);
      }

      // If this is the final part, set the note count
      if (i === parts.length - 1) {
        node.noteCount = folder.note_count;
      }

      parentNode = node;
    }
  }

  // Sort children alphabetically at each level
  function sortChildren(node: FolderNode) {
    node.children.sort((a, b) => a.name.localeCompare(b.name));
    for (const child of node.children) {
      sortChildren(child);
    }
  }
  sortChildren(root);

  cachedFolderTree = root;
  cachedFolderTreeKey = cacheKey;

  return root;
}

// LocalStorage persistence
const EXPANDED_KEY = 'xelanote_expanded_folders';

function saveExpandedState() {
  try {
    const paths = Object.keys(expandedFolders).filter((k) => expandedFolders[k]);
    localStorage.setItem(EXPANDED_KEY, JSON.stringify(paths));
  } catch (_e) {
    // localStorage might not be available
  }
}

export function loadExpandedState() {
  try {
    const stored = localStorage.getItem(EXPANDED_KEY);
    if (stored) {
      const paths = JSON.parse(stored) as string[];
      const expanded: Record<string, boolean> = {};
      for (const p of paths) {
        expanded[p] = true;
      }
      expandedFolders = expanded;
    }
  } catch (_e) {
    // localStorage might not be available
  }
  // Ensure root is always expanded
  expandedFolders['/'] = true;
}
