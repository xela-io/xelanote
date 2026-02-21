import { beforeEach, describe, expect, it, vi } from 'vitest';

const createFolder = vi.fn();
const moveFolder = vi.fn();
const deleteFolder = vi.fn();
const renameFolder = vi.fn();
const reorderFolders = vi.fn();
const reorderNotes = vi.fn();
const updateFolderColor = vi.fn();
const updateNoteColor = vi.fn();

vi.mock('$lib/api', () => ({
  createFolder,
  moveFolder,
  deleteFolder,
  renameFolder,
  reorderFolders,
  reorderNotes,
  updateFolderColor,
  updateNoteColor,
}));

const granularFolderColorUpdate = vi.fn();
const granularNoteColorUpdate = vi.fn();
const granularNoteUpdate = vi.fn();
const invalidateFlatTreeCache = vi.fn();

vi.mock('./tree-cache.svelte', () => ({
  granularFolderColorUpdate,
  granularNoteColorUpdate,
  granularNoteUpdate,
  invalidateFlatTreeCache,
}));

const findFolderNode = vi.fn();
const findFolderNodeById = vi.fn();

vi.mock('./tree-index.svelte', () => ({
  findFolderNode,
  findFolderNodeById,
}));

import type { FolderTreeNode } from './tree-index.svelte';

function makeTree(): FolderTreeNode {
  return {
    type: 'folder',
    id: 0,
    path: '/',
    name: 'Root',
    children: [
      {
        type: 'folder',
        id: 1,
        path: '/docs',
        name: 'docs',
        children: [
          { type: 'note', id: 'note-1', title: 'Note 1', displayOrder: 0, folderPath: '/docs' },
        ],
        noteCount: 1,
        isExpanded: false,
        displayOrder: 0,
      },
      {
        type: 'folder',
        id: 2,
        path: '/archive',
        name: 'archive',
        children: [],
        noteCount: 0,
        isExpanded: false,
        displayOrder: 1,
      },
    ],
    noteCount: 0,
    isExpanded: true,
    displayOrder: 0,
  };
}

describe('tree-operations store', () => {
  let treeData: FolderTreeNode | null;
  let setTreeDataCalls: (FolderTreeNode | null)[];
  let loadTreeCalls: number;

  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();

    treeData = makeTree();
    setTreeDataCalls = [];
    loadTreeCalls = 0;
  });

  async function getStore() {
    const store = await import('$lib/stores/tree-operations.svelte');
    store.initOperations({
      getTreeData: () => treeData,
      setTreeData: (data) => {
        treeData = data;
        setTreeDataCalls.push(data);
      },
      loadTree: async () => {
        loadTreeCalls++;
      },
    });
    return store;
  }

  describe('initOperations', () => {
    it('should accept state accessors', async () => {
      const store = await getStore();
      // No error means init succeeded
      expect(store.initOperations).toBeDefined();
    });
  });

  describe('createFolder', () => {
    it('should call API and reload tree on success', async () => {
      findFolderNode.mockReturnValue(treeData!.children[0]); // parent = /docs
      createFolder.mockResolvedValue(undefined);

      const store = await getStore();
      await store.createFolder('/docs/subfolder');

      expect(createFolder).toHaveBeenCalledWith('/docs/subfolder');
      expect(loadTreeCalls).toBe(1);
    });

    it('should add optimistic temp node before API call', async () => {
      const parent = treeData!.children[0] as FolderTreeNode;
      findFolderNode.mockReturnValue(parent);
      createFolder.mockResolvedValue(undefined);

      const store = await getStore();
      await store.createFolder('/docs/subfolder');

      // setTreeData should have been called at least once for optimistic update
      expect(setTreeDataCalls.length).toBeGreaterThanOrEqual(1);
    });

    it('should rollback on API error', async () => {
      findFolderNode.mockReturnValue(treeData!.children[0]);
      createFolder.mockRejectedValue(new Error('forbidden'));

      const store = await getStore();
      await expect(store.createFolder('/docs/subfolder')).rejects.toThrow('forbidden');

      // Should have called invalidateFlatTreeCache on rollback
      expect(invalidateFlatTreeCache).toHaveBeenCalled();
    });
  });

  describe('moveFolder', () => {
    it('should call API and reload tree on success', async () => {
      const docsFolder = treeData!.children[0] as FolderTreeNode;
      findFolderNodeById.mockReturnValue(docsFolder);
      findFolderNode.mockReturnValue(treeData!.children[1]); // target parent
      moveFolder.mockResolvedValue(undefined);

      const store = await getStore();
      await store.moveFolder(1, '/archive');

      expect(moveFolder).toHaveBeenCalledWith(1, '/archive');
      expect(loadTreeCalls).toBe(1);
    });

    it('should rollback on API error', async () => {
      const docsFolder = treeData!.children[0] as FolderTreeNode;
      findFolderNodeById.mockReturnValue(docsFolder);
      findFolderNode.mockReturnValue(treeData!.children[1]);
      moveFolder.mockRejectedValue(new Error('conflict'));

      const store = await getStore();
      await expect(store.moveFolder(1, '/archive')).rejects.toThrow('conflict');

      expect(invalidateFlatTreeCache).toHaveBeenCalled();
    });
  });

  describe('deleteFolder', () => {
    it('should call API and reload tree on success', async () => {
      deleteFolder.mockResolvedValue(undefined);

      const store = await getStore();
      await store.deleteFolder(1);

      expect(deleteFolder).toHaveBeenCalledWith(1);
      expect(loadTreeCalls).toBe(1);
    });

    it('should rollback on API error', async () => {
      deleteFolder.mockRejectedValue(new Error('not empty'));

      const store = await getStore();
      await expect(store.deleteFolder(1)).rejects.toThrow('not empty');

      expect(invalidateFlatTreeCache).toHaveBeenCalled();
    });
  });

  describe('renameFolder', () => {
    it('should call API and reload tree on success', async () => {
      const docsFolder = treeData!.children[0] as FolderTreeNode;
      findFolderNodeById.mockReturnValue(docsFolder);
      renameFolder.mockResolvedValue(undefined);

      const store = await getStore();
      await store.renameFolder(1, 'documents');

      expect(renameFolder).toHaveBeenCalledWith(1, 'documents');
      expect(loadTreeCalls).toBe(1);
    });

    it('should optimistically update name before API call', async () => {
      const docsFolder = treeData!.children[0] as FolderTreeNode;
      findFolderNodeById.mockReturnValue(docsFolder);
      renameFolder.mockResolvedValue(undefined);

      const store = await getStore();
      await store.renameFolder(1, 'documents');

      // The optimistic update should have set docsFolder.name
      expect(docsFolder.name).toBe('documents');
    });

    it('should rollback on API error', async () => {
      const docsFolder = treeData!.children[0] as FolderTreeNode;
      findFolderNodeById.mockReturnValue(docsFolder);
      renameFolder.mockRejectedValue(new Error('duplicate'));

      const store = await getStore();
      await expect(store.renameFolder(1, 'archive')).rejects.toThrow('duplicate');

      expect(invalidateFlatTreeCache).toHaveBeenCalled();
    });
  });

  describe('reorderFolders', () => {
    it('should call API and reload tree', async () => {
      reorderFolders.mockResolvedValue(undefined);

      const store = await getStore();
      await store.reorderFolders(null, [2, 1]);

      expect(reorderFolders).toHaveBeenCalledWith(null, [2, 1]);
      expect(loadTreeCalls).toBe(1);
    });
  });

  describe('reorderNotes', () => {
    it('should call API and reload tree', async () => {
      reorderNotes.mockResolvedValue(undefined);

      const store = await getStore();
      await store.reorderNotes('/docs', ['note-2', 'note-1']);

      expect(reorderNotes).toHaveBeenCalledWith('/docs', ['note-2', 'note-1']);
      expect(loadTreeCalls).toBe(1);
    });
  });

  describe('updateFolderColor', () => {
    it('should use granular update when available', async () => {
      const docsFolder = treeData!.children[0] as FolderTreeNode;
      findFolderNodeById.mockReturnValue(docsFolder);
      updateFolderColor.mockResolvedValue(undefined);
      granularFolderColorUpdate.mockReturnValue(true);

      const store = await getStore();
      await store.updateFolderColor(1, '#ff0000');

      expect(updateFolderColor).toHaveBeenCalledWith(1, '#ff0000');
      expect(granularFolderColorUpdate).toHaveBeenCalledWith('/docs', docsFolder, '#ff0000');
      expect(loadTreeCalls).toBe(0); // No full reload needed
    });

    it('should fallback to full reload when granular update fails', async () => {
      findFolderNodeById.mockReturnValue(null);
      updateFolderColor.mockResolvedValue(undefined);

      const store = await getStore();
      await store.updateFolderColor(99, '#ff0000');

      expect(loadTreeCalls).toBe(1); // Fallback to full reload
    });
  });

  describe('updateNoteColor', () => {
    it('should use granular update when available', async () => {
      updateNoteColor.mockResolvedValue(undefined);
      granularNoteColorUpdate.mockReturnValue(true);

      const store = await getStore();
      await store.updateNoteColor('note-1', '#00ff00');

      expect(updateNoteColor).toHaveBeenCalledWith('note-1', '#00ff00');
      expect(granularNoteColorUpdate).toHaveBeenCalledWith('note-1', '#00ff00');
      expect(loadTreeCalls).toBe(0);
    });

    it('should fallback to full reload when granular update fails', async () => {
      updateNoteColor.mockResolvedValue(undefined);
      granularNoteColorUpdate.mockReturnValue(false);

      const store = await getStore();
      await store.updateNoteColor('note-1', '#00ff00');

      expect(loadTreeCalls).toBe(1);
    });
  });

  describe('updateNoteInTree', () => {
    it('should update note title in tree', async () => {
      granularNoteUpdate.mockReturnValue(true);

      const store = await getStore();
      store.updateNoteInTree('note-1', { title: 'Updated Title' });

      // Should have set tree data (reactivity trigger)
      expect(setTreeDataCalls.length).toBeGreaterThanOrEqual(1);
    });

    it('should invalidate flat tree cache when granular update fails', async () => {
      granularNoteUpdate.mockReturnValue(false);

      const store = await getStore();
      store.updateNoteInTree('note-1', { title: 'Updated Title' });

      expect(invalidateFlatTreeCache).toHaveBeenCalled();
    });

    it('should do nothing when treeData is null', async () => {
      treeData = null;

      const store = await getStore();
      store.updateNoteInTree('note-1', { title: 'Updated Title' });

      expect(setTreeDataCalls.length).toBe(0);
    });
  });
});
