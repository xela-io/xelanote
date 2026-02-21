import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { SharedCollection, SharedFolder, SharedNote } from '$lib/api';

const getSharedNotes = vi.fn();
const getSharedFolders = vi.fn();
const getSharedCollections = vi.fn();
const getSharedFolderNotes = vi.fn();
const getSharedNote = vi.fn();

vi.mock('$lib/api', () => ({
  getSharedNotes,
  getSharedFolders,
  getSharedCollections,
  getSharedFolderNotes,
  getSharedNote,
}));

const mockNote = (id: string): SharedNote =>
  ({
    id,
    title: `Note ${id}`,
    shared_by: 'alice',
    role: 'viewer',
  }) as SharedNote;

const mockFolder = (id: number): SharedFolder =>
  ({
    id,
    path: `/shared-${id}`,
    shared_by: 'alice',
    role: 'viewer',
  }) as SharedFolder;

const mockCollection = (id: number): SharedCollection =>
  ({
    id,
    name: `Collection ${id}`,
    shared_by: 'alice',
    role: 'viewer',
  }) as SharedCollection;

describe('sharing store', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
  });

  it('should start with empty state', async () => {
    const store = await import('$lib/stores/sharing.svelte');
    expect(store.getSharedNotes()).toEqual([]);
    expect(store.getSharedFolders()).toEqual([]);
    expect(store.getSharedCollections()).toEqual([]);
    expect(store.getCurrentSharedFolderNotes()).toEqual([]);
    expect(store.getCurrentSharedNote()).toBeNull();
    expect(store.getIsLoading()).toBe(false);
  });

  describe('loadAllShared', () => {
    it('should load notes, folders, and collections in parallel', async () => {
      const notes = [mockNote('1'), mockNote('2')];
      const folders = [mockFolder(1)];
      const collections = [mockCollection(1)];

      getSharedNotes.mockResolvedValue(notes);
      getSharedFolders.mockResolvedValue(folders);
      getSharedCollections.mockResolvedValue(collections);

      const store = await import('$lib/stores/sharing.svelte');
      await store.loadAllShared();

      expect(store.getSharedNotes()).toEqual(notes);
      expect(store.getSharedFolders()).toEqual(folders);
      expect(store.getSharedCollections()).toEqual(collections);
      expect(store.getIsLoading()).toBe(false);
    });

    it('should reset to empty arrays on error', async () => {
      getSharedNotes.mockRejectedValue(new Error('network'));
      getSharedFolders.mockRejectedValue(new Error('network'));
      getSharedCollections.mockRejectedValue(new Error('network'));

      const store = await import('$lib/stores/sharing.svelte');
      await store.loadAllShared();

      expect(store.getSharedNotes()).toEqual([]);
      expect(store.getSharedFolders()).toEqual([]);
      expect(store.getSharedCollections()).toEqual([]);
      expect(store.getIsLoading()).toBe(false);
    });

    it('should handle collection fetch failure gracefully while others succeed', async () => {
      const notes = [mockNote('1')];
      const folders = [mockFolder(1)];

      getSharedNotes.mockResolvedValue(notes);
      getSharedFolders.mockResolvedValue(folders);
      getSharedCollections.mockRejectedValue(new Error('collections not available'));

      const store = await import('$lib/stores/sharing.svelte');
      await store.loadAllShared();

      // Notes and folders should still be loaded (collections fallback to [])
      expect(store.getSharedNotes()).toEqual(notes);
      expect(store.getSharedFolders()).toEqual(folders);
      expect(store.getSharedCollections()).toEqual([]);
    });
  });

  describe('loadSharedNotes', () => {
    it('should load shared notes', async () => {
      const notes = [mockNote('1')];
      getSharedNotes.mockResolvedValue(notes);

      const store = await import('$lib/stores/sharing.svelte');
      await store.loadSharedNotes();

      expect(store.getSharedNotes()).toEqual(notes);
      expect(store.getIsLoading()).toBe(false);
    });

    it('should set empty array on error', async () => {
      getSharedNotes.mockRejectedValue(new Error('fail'));

      const store = await import('$lib/stores/sharing.svelte');
      await store.loadSharedNotes();

      expect(store.getSharedNotes()).toEqual([]);
      expect(store.getIsLoading()).toBe(false);
    });
  });

  describe('loadSharedFolders', () => {
    it('should load shared folders', async () => {
      const folders = [mockFolder(1), mockFolder(2)];
      getSharedFolders.mockResolvedValue(folders);

      const store = await import('$lib/stores/sharing.svelte');
      await store.loadSharedFolders();

      expect(store.getSharedFolders()).toEqual(folders);
      expect(store.getIsLoading()).toBe(false);
    });

    it('should set empty array on error', async () => {
      getSharedFolders.mockRejectedValue(new Error('fail'));

      const store = await import('$lib/stores/sharing.svelte');
      await store.loadSharedFolders();

      expect(store.getSharedFolders()).toEqual([]);
    });
  });

  describe('loadSharedFolderNotes', () => {
    it('should load notes for a shared folder', async () => {
      const notes = [mockNote('1'), mockNote('2')];
      getSharedFolderNotes.mockResolvedValue(notes);

      const store = await import('$lib/stores/sharing.svelte');
      await store.loadSharedFolderNotes(42);

      expect(getSharedFolderNotes).toHaveBeenCalledWith(42);
      expect(store.getCurrentSharedFolderNotes()).toEqual(notes);
      expect(store.getIsLoading()).toBe(false);
    });

    it('should set empty array and re-throw on error', async () => {
      getSharedFolderNotes.mockRejectedValue(new Error('not found'));

      const store = await import('$lib/stores/sharing.svelte');
      await expect(store.loadSharedFolderNotes(99)).rejects.toThrow('not found');

      expect(store.getCurrentSharedFolderNotes()).toEqual([]);
      expect(store.getIsLoading()).toBe(false);
    });
  });

  describe('loadSharedNote', () => {
    it('should load a single shared note', async () => {
      const note = mockNote('abc');
      getSharedNote.mockResolvedValue(note);

      const store = await import('$lib/stores/sharing.svelte');
      await store.loadSharedNote('abc');

      expect(getSharedNote).toHaveBeenCalledWith('abc');
      expect(store.getCurrentSharedNote()).toEqual(note);
      expect(store.getIsLoading()).toBe(false);
    });

    it('should set null and re-throw on error', async () => {
      getSharedNote.mockRejectedValue(new Error('forbidden'));

      const store = await import('$lib/stores/sharing.svelte');
      await expect(store.loadSharedNote('xyz')).rejects.toThrow('forbidden');

      expect(store.getCurrentSharedNote()).toBeNull();
      expect(store.getIsLoading()).toBe(false);
    });
  });

  describe('clear functions', () => {
    it('clearCurrentSharedNote should set current note to null', async () => {
      const note = mockNote('1');
      getSharedNote.mockResolvedValue(note);

      const store = await import('$lib/stores/sharing.svelte');
      await store.loadSharedNote('1');
      expect(store.getCurrentSharedNote()).toEqual(note);

      store.clearCurrentSharedNote();
      expect(store.getCurrentSharedNote()).toBeNull();
    });

    it('clearCurrentSharedFolderNotes should set folder notes to empty', async () => {
      const notes = [mockNote('1')];
      getSharedFolderNotes.mockResolvedValue(notes);

      const store = await import('$lib/stores/sharing.svelte');
      await store.loadSharedFolderNotes(1);
      expect(store.getCurrentSharedFolderNotes()).toEqual(notes);

      store.clearCurrentSharedFolderNotes();
      expect(store.getCurrentSharedFolderNotes()).toEqual([]);
    });
  });

  describe('count helpers', () => {
    it('getSharedNoteCount returns number of shared notes', async () => {
      getSharedNotes.mockResolvedValue([mockNote('1'), mockNote('2'), mockNote('3')]);
      getSharedFolders.mockResolvedValue([]);
      getSharedCollections.mockResolvedValue([]);

      const store = await import('$lib/stores/sharing.svelte');
      await store.loadAllShared();

      expect(store.getSharedNoteCount()).toBe(3);
    });

    it('getTotalSharedCount returns sum of notes + folders + collections', async () => {
      getSharedNotes.mockResolvedValue([mockNote('1'), mockNote('2')]);
      getSharedFolders.mockResolvedValue([mockFolder(1)]);
      getSharedCollections.mockResolvedValue([mockCollection(1), mockCollection(2)]);

      const store = await import('$lib/stores/sharing.svelte');
      await store.loadAllShared();

      expect(store.getTotalSharedCount()).toBe(5); // 2 + 1 + 2
    });
  });
});
