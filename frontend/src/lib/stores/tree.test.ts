import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import * as tree from '$lib/stores/tree.svelte';
import type { Folder, Note } from '$lib/api';

// Mock API
vi.mock('$lib/api', () => ({
  getFolders: vi.fn(),
  listNotes: vi.fn(),
  createFolder: vi.fn(),
  moveFolder: vi.fn(),
  deleteFolder: vi.fn(),
  renameFolder: vi.fn(),
  reorderFolders: vi.fn(),
  updateFolderColor: vi.fn(),
  updateNoteColor: vi.fn(),
}));

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      store = {};
    },
  };
})();

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

describe('tree store - getFlattenedTree()', () => {
  beforeEach(() => {
    localStorageMock.clear();
    vi.clearAllMocks();
  });

  afterEach(() => {
    localStorageMock.clear();
  });

  it('should flatten empty tree correctly', async () => {
    const { getFolders, listNotes } = await import('$lib/api');
    vi.mocked(getFolders).mockResolvedValue({ folders: [] });
    vi.mocked(listNotes).mockResolvedValue({ notes: [] });

    await tree.loadTree();
    const flatTree = tree.getFlattenedTree();

    expect(flatTree).toEqual([]);
  });

  it('should flatten single-level tree correctly', async () => {
    const { getFolders, listNotes } = await import('$lib/api');

    const folders: Folder[] = [
      {
        id: 1,
        path: '/',
        name: 'Root',
        parent_id: undefined,
        note_count: 2,
        display_order: 0,
        created_at: '',
        updated_at: '',
      },
      {
        id: 2,
        path: '/Projects',
        name: 'Projects',
        parent_id: 1,
        note_count: 1,
        display_order: 0,
        created_at: '',
        updated_at: '',
      },
    ];

    const notes: Note[] = [
      {
        id: 'note1',
        title: 'Note 1',
        version: 1,
        folder_path: '/Projects',
        display_order: 0,
        content: '',
        content_encrypted: false,
        encryption_version: 0,
        created_at: '',
        updated_at: '',
      },
    ];

    vi.mocked(getFolders).mockResolvedValue({ folders });
    vi.mocked(listNotes).mockResolvedValue({ notes });

    await tree.loadTree();

    // Expand the folder first
    tree.toggleExpanded('/Projects');

    const flatTree = tree.getFlattenedTree();

    // Should have: Projects folder + Note 1
    expect(flatTree.length).toBe(2);
    expect(flatTree[0].node.type).toBe('folder');
    expect(flatTree[0].level).toBe(0);
    expect(flatTree[1].node.type).toBe('note');
    expect(flatTree[1].level).toBe(1);
  });

  it('should respect expanded state in flattening', async () => {
    const { getFolders, listNotes } = await import('$lib/api');

    const folders: Folder[] = [
      {
        id: 1,
        path: '/',
        name: 'Root',
        parent_id: undefined,
        note_count: 0,
        display_order: 0,
        created_at: '',
        updated_at: '',
      },
      {
        id: 2,
        path: '/Projects',
        name: 'Projects',
        parent_id: 1,
        note_count: 2,
        display_order: 0,
        created_at: '',
        updated_at: '',
      },
    ];

    const notes: Note[] = [
      {
        id: 'note1',
        title: 'Note in Projects',
        version: 1,
        folder_path: '/Projects',
        display_order: 0,
        content: '',
        content_encrypted: false,
        encryption_version: 0,
        created_at: '',
        updated_at: '',
      },
    ];

    vi.mocked(getFolders).mockResolvedValue({ folders });
    vi.mocked(listNotes).mockResolvedValue({ notes });

    // Load expanded state BEFORE loadTree
    tree.loadExpandedStateFromStorage();
    await tree.loadTree();

    // Initially collapsed - should only show folder (no children visible)
    let flatTree = tree.getFlattenedTree();
    const initialLength = flatTree.length;
    expect(flatTree[0].node.type).toBe('folder');

    // Expand folder
    tree.toggleExpanded('/Projects');

    // Now should show folder + note (more items than before)
    flatTree = tree.getFlattenedTree();
    expect(flatTree.length).toBeGreaterThan(initialLength);
    expect(flatTree[0].node.type).toBe('folder');
    expect(flatTree[flatTree.length - 1].node.type).toBe('note');
  });

  it('should calculate correct depth levels', async () => {
    const { getFolders, listNotes } = await import('$lib/api');

    const folders: Folder[] = [
      {
        id: 1,
        path: '/',
        name: 'Root',
        parent_id: undefined,
        note_count: 0,
        display_order: 0,
        created_at: '',
        updated_at: '',
      },
      {
        id: 2,
        path: '/Level1',
        name: 'Level1',
        parent_id: 1,
        note_count: 0,
        display_order: 0,
        created_at: '',
        updated_at: '',
      },
      {
        id: 3,
        path: '/Level1/Level2',
        name: 'Level2',
        parent_id: 2,
        note_count: 1,
        display_order: 0,
        created_at: '',
        updated_at: '',
      },
    ];

    const notes: Note[] = [
      {
        id: 'note1',
        title: 'Deep Note',
        version: 1,
        folder_path: '/Level1/Level2',
        display_order: 0,
        content: '',
        content_encrypted: false,
        encryption_version: 0,
        created_at: '',
        updated_at: '',
      },
    ];

    vi.mocked(getFolders).mockResolvedValue({ folders });
    vi.mocked(listNotes).mockResolvedValue({ notes });

    await tree.loadTree();

    // Expand all folders
    tree.toggleExpanded('/Level1');
    tree.toggleExpanded('/Level1/Level2');

    const flatTree = tree.getFlattenedTree();

    // Level1 (depth 0), Level2 (depth 1), Note (depth 2)
    expect(flatTree[0].level).toBe(0); // Level1
    expect(flatTree[1].level).toBe(1); // Level2
    expect(flatTree[2].level).toBe(2); // Note
  });

  it('should cache flattened tree for performance', async () => {
    const { getFolders, listNotes } = await import('$lib/api');

    const folders: Folder[] = [
      {
        id: 1,
        path: '/',
        name: 'Root',
        parent_id: undefined,
        note_count: 0,
        display_order: 0,
        created_at: '',
        updated_at: '',
      },
      {
        id: 2,
        path: '/Folder',
        name: 'Folder',
        parent_id: 1,
        note_count: 0,
        display_order: 0,
        created_at: '',
        updated_at: '',
      },
    ];

    vi.mocked(getFolders).mockResolvedValue({ folders });
    vi.mocked(listNotes).mockResolvedValue({ notes: [] });

    await tree.loadTree();

    // First call - builds cache
    const flatTree1 = tree.getFlattenedTree();

    // Second call - should return same reference (cached)
    const flatTree2 = tree.getFlattenedTree();

    expect(flatTree1).toBe(flatTree2); // Same reference = cached
  });

  it('should invalidate cache on toggleExpanded', async () => {
    const { getFolders, listNotes } = await import('$lib/api');

    const folders: Folder[] = [
      {
        id: 1,
        path: '/',
        name: 'Root',
        parent_id: undefined,
        note_count: 0,
        display_order: 0,
        created_at: '',
        updated_at: '',
      },
      {
        id: 2,
        path: '/Folder',
        name: 'Folder',
        parent_id: 1,
        note_count: 1,
        display_order: 0,
        created_at: '',
        updated_at: '',
      },
    ];

    const notes: Note[] = [
      {
        id: 'note1',
        title: 'Note',
        version: 1,
        folder_path: '/Folder',
        display_order: 0,
        content: '',
        content_encrypted: false,
        encryption_version: 0,
        created_at: '',
        updated_at: '',
      },
    ];

    vi.mocked(getFolders).mockResolvedValue({ folders });
    vi.mocked(listNotes).mockResolvedValue({ notes });

    await tree.loadTree();

    const flatTree1 = tree.getFlattenedTree();
    const length1 = flatTree1.length;

    // Toggle should invalidate cache
    tree.toggleExpanded('/Folder');

    const flatTree2 = tree.getFlattenedTree();
    const length2 = flatTree2.length;

    // Different reference (cache invalidated)
    expect(flatTree1).not.toBe(flatTree2);

    // Different length (expanded shows more items)
    expect(length2).toBeGreaterThan(length1);
  });

  it('should invalidate cache on loadTree', async () => {
    const { getFolders, listNotes } = await import('$lib/api');

    const folders: Folder[] = [
      {
        id: 1,
        path: '/',
        name: 'Root',
        parent_id: undefined,
        note_count: 0,
        display_order: 0,
        created_at: '',
        updated_at: '',
      },
    ];

    vi.mocked(getFolders).mockResolvedValue({ folders });
    vi.mocked(listNotes).mockResolvedValue({ notes: [] });

    await tree.loadTree();
    const flatTree1 = tree.getFlattenedTree();

    // Reload tree
    await tree.loadTree();
    const flatTree2 = tree.getFlattenedTree();

    // Cache should be invalidated
    expect(flatTree1).not.toBe(flatTree2);
  });

  it('should correctly index items in flattened tree', async () => {
    const { getFolders, listNotes } = await import('$lib/api');

    const folders: Folder[] = [
      {
        id: 1,
        path: '/',
        name: 'Root',
        parent_id: undefined,
        note_count: 0,
        display_order: 0,
        created_at: '',
        updated_at: '',
      },
      {
        id: 2,
        path: '/Folder1',
        name: 'Folder1',
        parent_id: 1,
        note_count: 1,
        display_order: 0,
        created_at: '',
        updated_at: '',
      },
      {
        id: 3,
        path: '/Folder2',
        name: 'Folder2',
        parent_id: 1,
        note_count: 1,
        display_order: 1,
        created_at: '',
        updated_at: '',
      },
    ];

    const notes: Note[] = [
      {
        id: 'note1',
        title: 'Note 1',
        version: 1,
        folder_path: '/Folder1',
        display_order: 0,
        content: '',
        content_encrypted: false,
        encryption_version: 0,
        created_at: '',
        updated_at: '',
      },
      {
        id: 'note2',
        title: 'Note 2',
        version: 1,
        folder_path: '/Folder2',
        display_order: 0,
        content: '',
        content_encrypted: false,
        encryption_version: 0,
        created_at: '',
        updated_at: '',
      },
    ];

    vi.mocked(getFolders).mockResolvedValue({ folders });
    vi.mocked(listNotes).mockResolvedValue({ notes });

    await tree.loadTree();

    // Expand both folders
    tree.toggleExpanded('/Folder1');
    tree.toggleExpanded('/Folder2');

    const flatTree = tree.getFlattenedTree();

    // Should have correct sequential indexes
    flatTree.forEach((item, idx) => {
      expect(item.index).toBe(idx);
    });

    // Verify structure: Folder1, Note1, Folder2, Note2
    expect(flatTree.length).toBe(4);
    expect(flatTree[0].node.type).toBe('folder');
    expect(flatTree[1].node.type).toBe('note');
    expect(flatTree[2].node.type).toBe('folder');
    expect(flatTree[3].node.type).toBe('note');
  });
});
