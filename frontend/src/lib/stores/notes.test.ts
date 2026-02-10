import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { Backlink, Note } from '$lib/api';

const createNoteHelper = vi.fn();

vi.mock('$lib/api', () => ({
  ApiError: class ApiError extends Error {
    status: number;
    constructor(status = 500) {
      super('api error');
      this.status = status;
    }
  },
  listNotes: vi.fn(),
  getNote: vi.fn(),
  getBacklinks: vi.fn(),
  createNote: vi.fn(),
  updateNote: vi.fn(),
  recordTaskEvent: vi.fn(),
  decryptNote: vi.fn(),
  getRecipeDetail: vi.fn(),
  deleteNote: vi.fn(),
  renameNote: vi.fn(),
  renameNoteAsync: vi.fn(),
  getJobStatus: vi.fn(),
}));

vi.mock('$lib/editor/markdown', () => ({
  extractDueDatesDetailed: vi.fn(() => []),
}));

vi.mock('$lib/offline/offline-queue', () => ({
  hasPendingForNote: vi.fn(async () => false),
}));

vi.mock('$lib/stores/autosave.svelte', () => ({
  getAutoSaveEnabled: vi.fn(() => true),
  getAutoSaveDelay: vi.fn(() => 10),
}));

vi.mock('$lib/stores/encryption.svelte', () => ({
  isEncryptionUnlocked: vi.fn(() => true),
  encryptNote: vi.fn((title: string, content: string) => ({
    encryptedTitle: title,
    encryptedContent: { ciphertext: content, metadata: { wrapped_dek: 'dek', version: 1 } },
    keywords: [],
  })),
  decryptNote: vi.fn((encryptedTitle: string | null, payload: { ciphertext: string }) => ({
    title: encryptedTitle,
    content: payload.ciphertext,
  })),
  encryptTaskText: vi.fn((text: string) => text),
}));

vi.mock('$lib/stores/folders.svelte', () => ({
  getFolders: vi.fn(() => []),
}));

vi.mock('$lib/stores/notes/creator', () => ({
  createNote: createNoteHelper,
}));

vi.mock('$lib/stores/notes/loaders', () => ({
  loadNote: vi.fn(),
  loadNotes: vi.fn(),
}));

vi.mock('$lib/stores/notes/mutations', () => ({
  deleteCurrentNote: vi.fn(),
  moveNote: vi.fn(),
}));

vi.mock('$lib/stores/notes/encryption-toggle', () => ({
  toggleEncryption: vi.fn(),
}));

vi.mock('$lib/stores/notes/rename', () => ({
  renameCurrentNote: vi.fn(),
}));

vi.mock('$lib/stores/notes/remote-update-gate', () => ({
  handleRemoteUpdateWithPendingCheck: vi.fn(),
}));

vi.mock('$lib/stores/notes/remote-updates', () => ({
  handleRemoteCreate: vi.fn(),
  handleRemoteDelete: vi.fn(),
  handleRemoteUpdate: vi.fn(),
}));

vi.mock('$lib/stores/notes/saver', () => ({
  saveNote: vi.fn(),
}));

vi.mock('$lib/stores/notes/helpers', () => ({
  assertOnlineForParanoidMode: vi.fn(),
  extractUniqueWikilinks: vi.fn(() => []),
}));

vi.mock('$lib/stores/search-index.svelte', () => ({
  addToIndex: vi.fn(),
  updateInIndex: vi.fn(),
  removeFromIndex: vi.fn(),
}));

vi.mock('$lib/stores/toast.svelte', () => ({
  info: vi.fn(),
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
}));

const baseNote = (overrides: Partial<Note> = {}): Note => ({
  id: 'note-1',
  title: 'Title',
  content: 'Body',
  folder_path: '/',
  version: 1,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  ...overrides,
});

describe('notes store', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
  });

  it('createNote wires helper and updates store state', async () => {
    const note = baseNote();
    const backlinks: Backlink[] = [{ id: 'b1', title: 'Ref' }];

    createNoteHelper.mockImplementation(async (deps) => {
      deps.setCurrentNote(note);
      deps.setNotes([note]);
      deps.setBacklinks(backlinks);
      deps.setDirty(false);
      return note;
    });

    const notesStore = await import('$lib/stores/notes.svelte');
    const created = await notesStore.createNote('Title', 'Body');

    expect(created).toEqual(note);
    expect(notesStore.getCurrentNote()).toEqual(note);
    expect(notesStore.getNotes()).toEqual([note]);
    expect(notesStore.getBacklinks()).toEqual(backlinks);

    const [deps] = createNoteHelper.mock.calls[0];
    expect(deps.title).toBe('Title');
    expect(deps.content).toBe('Body');
    expect(deps.folderPath).toBe('/');
  });

  it('updateCurrentNoteContent marks dirty and updates content', async () => {
    const note = baseNote();
    createNoteHelper.mockImplementation(async (deps) => {
      deps.setCurrentNote(note);
      deps.setNotes([note]);
      return note;
    });

    const notesStore = await import('$lib/stores/notes.svelte');
    await notesStore.createNote('Title', 'Body');

    notesStore.updateCurrentNoteContent('Updated');

    expect(notesStore.getCurrentNote()?.content).toBe('Updated');
    expect(notesStore.getIsDirty()).toBe(true);
  });

  it('updateCurrentNoteAIEnabled updates current note and list', async () => {
    const note = baseNote();
    createNoteHelper.mockImplementation(async (deps) => {
      deps.setCurrentNote(note);
      deps.setNotes([note]);
      return note;
    });

    const notesStore = await import('$lib/stores/notes.svelte');
    await notesStore.createNote('Title', 'Body');

    notesStore.updateCurrentNoteAIEnabled(true);

    expect(notesStore.getCurrentNote()?.ai_enabled).toBe(true);
    expect(notesStore.getNotes()[0]?.ai_enabled).toBe(true);
  });

  it('clearCurrentNote resets dirty state and auto-save status', async () => {
    vi.useFakeTimers();

    const note = baseNote();
    createNoteHelper.mockImplementation(async (deps) => {
      deps.setCurrentNote(note);
      deps.setNotes([note]);
      deps.setBacklinks([{ id: 'b1', title: 'Ref' }]);
      deps.setError('boom');
      return note;
    });

    const notesStore = await import('$lib/stores/notes.svelte');
    await notesStore.createNote('Title', 'Body');
    notesStore.updateCurrentNoteContent('Updated');

    notesStore.scheduleAutoSave();
    expect(notesStore.getAutoSaveStatus()).toBe('pending');

    notesStore.clearCurrentNote();

    expect(notesStore.getCurrentNote()).toBeNull();
    expect(notesStore.getIsDirty()).toBe(false);
    expect(notesStore.getBacklinks()).toEqual([]);
    expect(notesStore.getAutoSaveStatus()).toBe('idle');
    expect(notesStore.getError()).toBeNull();

    vi.useRealTimers();
  });

  it('replaceTempId swaps temp note in list and current selection', async () => {
    const tempNote = baseNote({ id: 'temp-1' });
    createNoteHelper.mockImplementation(async (deps) => {
      deps.setCurrentNote(tempNote);
      deps.setNotes([tempNote]);
      return tempNote;
    });

    const notesStore = await import('$lib/stores/notes.svelte');
    await notesStore.createNote('Temp', 'Body');

    const realNote = baseNote({ id: 'real-1', title: 'Real' });
    notesStore.replaceTempId('temp-1', realNote);

    expect(notesStore.getCurrentNote()).toEqual(realNote);
    expect(notesStore.getNotes()).toEqual([realNote]);
  });
});
