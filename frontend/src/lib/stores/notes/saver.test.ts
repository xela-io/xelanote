import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { Note } from '$lib/api';
import type { TaskEventQueue } from '$lib/stores/notes/task-events';

import { saveNote } from './saver';

const metadata = {
  version: 3 as const,
  algorithm: 'XChaCha20-Poly1305' as const,
  kdf: 'Argon2id' as const,
  kdf_strength: 'interactive' as const,
  nonce_bytes: 24 as const,
  wrapped_dek: 'wrapped-new',
};

const baseNote = (overrides: Partial<Note> = {}): Note => ({
  id: 'note-1',
  title: 'Encrypted Title',
  content: 'Encrypted content body',
  folder_path: '/secure',
  version: 7,
  created_at: '2026-02-28T10:00:00Z',
  updated_at: '2026-02-28T10:00:00Z',
  content_encrypted: true,
  encrypted_content: 'cipher-old',
  encrypted_title: '{"ciphertext":"old"}',
  title_encrypted: true,
  wrapped_dek: 'wrapped-old',
  encryption_version: 3,
  encryption_metadata: JSON.stringify(metadata),
  ...overrides,
});

describe('saveNote security behavior', () => {
  let logSpy: ReturnType<typeof vi.spyOn>;
  let _errorSpy: ReturnType<typeof vi.spyOn>;
  let _warnSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    _errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    _warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('omits links/due_dates for encrypted save payload and avoids ciphertext fragment logging', async () => {
    let currentNote: Note | null = baseNote();
    let isDirty = true;
    let isSaving = false;
    let autoSaveStatus: 'idle' | 'pending' | 'saving' | 'saved' | 'error' = 'idle';
    let autoSaveError: string | null = null;
    let storeError: string | null = null;
    let lastSavedVersion: number | null = null;
    let lastSaveTimestamp: number | null = null;
    let saveCounter = 0;
    let notes: Note[] = [currentNote];

    const encryptNote = vi.fn((_: string, __: string, ___: string) => ({
      encryptedTitle: '{"ciphertext":"enc-title"}',
      encryptedContent: {
        ciphertext: 'cipher-new',
        metadata,
      },
      keywords: [],
    }));
    const decryptNote = vi.fn((_: string | null, __: unknown, ___?: string) => ({
      title: 'Decrypted Title',
      content: 'Decrypted Body',
    }));
    const updateSearchIndex = vi.fn();
    const extractUniqueLinks = vi.fn(() => [{ title: 'LeakTarget' }]);
    const extractDueDates = vi.fn(() => [
      {
        due_date: '2026-03-01',
        line_text: '- [ ] task @due(2026-03-01)',
        line_index: 0,
        is_task_item: true,
        is_completed: false,
      },
    ]);
    const updateNote = vi.fn(async (_id: string, _payload: unknown, _version: number) =>
      baseNote({
        version: 8,
        encrypted_content: 'cipher-new',
        encrypted_title: '{"ciphertext":"enc-title"}',
        encryption_metadata: JSON.stringify(metadata),
      })
    );

    const taskEventQueue: TaskEventQueue = {
      add: () => {},
      getForNote: () => [],
      clearForNote: () => {},
    };

    await saveNote({
      getCurrentNote: () => currentNote,
      getIsDirty: () => isDirty,
      getIsSaving: () => isSaving,
      setIsSaving: (value) => {
        isSaving = value;
      },
      setError: (value) => {
        storeError = value;
      },
      setAutoSaveStatus: (status) => {
        autoSaveStatus = status;
      },
      setAutoSaveError: (value) => {
        autoSaveError = value;
      },
      getAutoSaveTimeout: () => null,
      setAutoSaveTimeout: () => {},
      incrementSaveCounter: () => {
        saveCounter += 1;
        return saveCounter;
      },
      getSaveCounter: () => saveCounter,
      setDirty: (dirty) => {
        isDirty = dirty;
      },
      setCurrentNote: (note) => {
        currentNote = note;
      },
      updateNotes: (updater) => {
        notes = updater(notes);
      },
      setLastSavedVersion: (version) => {
        lastSavedVersion = version;
      },
      setLastSaveTimestamp: (timestamp) => {
        lastSaveTimestamp = timestamp;
      },
      taskEventQueue,
      assertOnline: () => {},
      isEncryptionUnlocked: () => true,
      encryptNote,
      encryptFolderPath: (_folderPath: string, _noteID: string, _wrappedDEK: string) =>
        'encrypted-folder-path',
      decryptNote,
      encryptTaskText: (text) => ({ ciphertext: text, metadata: {} }),
      extractUniqueLinks,
      extractDueDates,
      updateNote,
      updateSearchIndex,
      recordTaskEvent: async () => {},
      isConflictError: () => false,
    });

    expect(updateNote).toHaveBeenCalledTimes(1);
    const payload = updateNote.mock.calls[0]?.[1] as Record<string, unknown>;
    expect(payload).not.toHaveProperty('links');
    expect(payload).not.toHaveProperty('due_dates');
    expect(payload.keywords).toEqual([]);

    expect(encryptNote).toHaveBeenCalledWith('Encrypted Title', 'Encrypted content body', 'note-1');
    expect(decryptNote).toHaveBeenCalled();
    expect(decryptNote.mock.calls[0]?.[2]).toBe('note-1');
    expect(extractDueDates).not.toHaveBeenCalled();
    expect(updateSearchIndex).toHaveBeenCalledWith('note-1', 'Decrypted Title', 'Decrypted Body');

    const flattenedLogs = logSpy.mock.calls.flatMap((args: unknown[]) =>
      args.map((entry: unknown) => String(entry))
    );
    expect(flattenedLogs.some((line: string) => line.includes('first 50 chars'))).toBe(false);

    expect(isDirty).toBe(false);
    expect(autoSaveStatus).toBe('saved');
    expect(autoSaveError).toBeNull();
    expect(storeError).toBeNull();
    expect(lastSavedVersion).toBe(8);
    expect(lastSaveTimestamp).not.toBeNull();
    expect(currentNote?.content).toBe('Decrypted Body');
    expect(notes[0]?.version).toBe(8);
  });
});
