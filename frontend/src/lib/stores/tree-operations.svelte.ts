// Tree Operations - CRUD operations on tree nodes (add, remove, move, rename, reorder, color)
// Part of the tree store module split.

import * as api from '$lib/api';

import {
  granularFolderColorUpdate,
  granularNoteColorUpdate,
  granularNoteUpdate,
  invalidateFlatTreeCache,
} from './tree-cache.svelte';
import type { FolderTreeNode } from './tree-index.svelte';
import { findFolderNode, findFolderNodeById } from './tree-index.svelte';

/**
 * Internal interface for accessing and mutating the shared tree state.
 * This avoids circular dependencies by having the main module inject its state accessors.
 */
export interface TreeStateAccessors {
  getTreeData: () => FolderTreeNode | null;
  setTreeData: (data: FolderTreeNode | null) => void;
  loadTree: () => Promise<void>;
}

let stateAccessors: TreeStateAccessors | null = null;

/** Called once from tree.svelte.ts to wire up shared state. */
export function initOperations(accessors: TreeStateAccessors): void {
  stateAccessors = accessors;
}

function getTreeData(): FolderTreeNode | null {
  return stateAccessors!.getTreeData();
}

function setTreeData(data: FolderTreeNode | null): void {
  stateAccessors!.setTreeData(data);
}

async function loadTree(): Promise<void> {
  return stateAccessors!.loadTree();
}

// Folder Operations

export async function createFolder(path: string) {
  const treeData = getTreeData();

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
    setTreeData({ ...treeData }); // Force reactivity
  }

  try {
    // API call in background
    await api.createFolder(path);
    // Success - reload to get real folder with server-assigned ID
    await loadTree();
    // Note: loadTree() already invalidates cache
  } catch (e) {
    // Revert to snapshot on error
    setTreeData(snapshot);
    invalidateFlatTreeCache(); // Invalidate after revert
    throw e;
  }
}

export async function moveFolder(folderId: number, newParentPath: string) {
  const treeData = getTreeData();

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

    setTreeData({ ...treeData }); // Force reactivity
  }

  try {
    // API call in background
    await api.moveFolder(folderId, newParentPath);
    // Success - reload to get updated paths and server state
    await loadTree();
    // Note: loadTree() already invalidates cache
  } catch (e) {
    // Revert to snapshot on error
    setTreeData(snapshot);
    invalidateFlatTreeCache(); // Invalidate after revert
    throw e;
  }
}

export async function deleteFolder(folderId: number) {
  const treeData = getTreeData();

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
    setTreeData({ ...treeData }); // Force reactivity
  }

  try {
    // API call in background
    await api.deleteFolder(folderId);
    // Success - reload to ensure consistency
    await loadTree();
    // Note: loadTree() already invalidates cache
  } catch (e) {
    // Revert to snapshot on error
    setTreeData(snapshot);
    invalidateFlatTreeCache(); // Invalidate after revert
    throw e;
  }
}

export async function renameFolder(folderId: number, newName: string) {
  const treeData = getTreeData();

  // Snapshot for rollback (optimistic UI pattern)
  const snapshot = JSON.parse(JSON.stringify(treeData));

  // Optimistic UI update
  const node = findFolderNodeById(treeData, folderId);
  if (node) {
    node.name = newName;
    setTreeData({ ...treeData! }); // Force reactivity
  }

  try {
    // API call in background
    await api.renameFolder(folderId, newName);
    // Success - reload to get any server-side updates
    await loadTree();
    // Note: loadTree() already invalidates cache
  } catch (e) {
    // Revert to snapshot on error
    setTreeData(snapshot);
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
  const treeData = getTreeData();
  await api.updateFolderColor(folderId, color);

  // Try granular update for color-only change (no structural change)
  const folderNode = findFolderNodeById(treeData, folderId);
  if (folderNode && granularFolderColorUpdate(folderNode.path, folderNode, color)) {
    setTreeData({ ...treeData! });
    return;
  }

  await loadTree(); // Fallback: full reload invalidates cache
}

export async function updateNoteColor(noteId: string, color: string | null) {
  const treeData = getTreeData();
  await api.updateNoteColor(noteId, color);

  // Try granular update for color-only change (no structural change)
  if (granularNoteColorUpdate(noteId, color)) {
    setTreeData({ ...treeData! });
    return;
  }

  await loadTree(); // Fallback: full reload invalidates cache
}

/**
 * Update a note's title in the tree without full reload.
 * Used for granular cache updates when only the title changes.
 */
export function updateNoteInTree(
  noteId: string,
  updates: { title?: string; color?: string | null }
): void {
  const treeData = getTreeData();
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
  if (!granularNoteUpdate(noteId)) {
    invalidateFlatTreeCache();
  }

  setTreeData({ ...treeData });
}
