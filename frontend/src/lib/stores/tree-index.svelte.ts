// Tree Index - Types, flat tree item interface, and tree search/lookup functions
// Part of the tree store module split.

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
  noteType?: string;
  updatedAt?: string;
  createdAt?: string;
}

// Sort mode for notes in the sidebar
export type SortMode = 'manual' | 'updated' | 'title' | 'created';

// Flattened tree for virtual scrolling
export interface FlatTreeItem {
  node: TreeNode;
  level: number;
  index: number;
}

export function findFolderNode(
  root: FolderTreeNode | null,
  path: string
): FolderTreeNode | null {
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

export function findFolderNodeById(
  root: FolderTreeNode | null,
  id: number
): FolderTreeNode | null {
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
export function findNodeById(
  treeData: FolderTreeNode | null,
  type: 'folder' | 'note',
  id: string | number
): TreeNode | null {
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
  treeData: FolderTreeNode | null,
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
