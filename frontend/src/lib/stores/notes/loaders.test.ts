import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { Note } from '$lib/api';
import type { EncryptedPayload } from '$lib/crypto/e2e';
import {
  _resetEncryptedAttachmentMigrationAuditForTests,
  getEncryptedAttachmentMigrationAuditCounters,
} from '$lib/stores/encrypted-attachment-migration-audit.svelte';

import { loadNote } from './loaders';

const metadata = {
  version: 3 as const,
  algorithm: 'XChaCha20-Poly1305' as const,
  kdf: 'Argon2id' as const,
  kdf_strength: 'interactive' as const,
  nonce_bytes: 24 as const,
  wrapped_dek: 'wrapped-new',
};

const baseEncryptedNote = (overrides: Partial<Note> = {}): Note => ({
  id: 'note-1',
  title: '',
  content: '',
  folder_path: '/secure',
  version: 7,
  created_at: '2026-02-28T10:00:00Z',
  updated_at: '2026-02-28T10:00:00Z',
  content_encrypted: true,
  encrypted_content: 'cipher-old',
  encrypted_title: null,
  title_encrypted: false,
  wrapped_dek: 'wrapped-old',
  encryption_version: 3,
  encryption_metadata: JSON.stringify(metadata),
  ...overrides,
});

describe('loadNote encrypted attachment migration persistence', () => {
  beforeEach(() => {
    _resetEncryptedAttachmentMigrationAuditForTests();
    vi.restoreAllMocks();
    vi.spyOn(console, 'log').mockImplementation(() => {});
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.spyOn(console, 'info').mockImplementation(() => {});
  });

  it('persists migrated legacy encrypted attachment markdown on load', async () => {
    const legacyContent =
      '[Encrypted attachment: photo.png](/uploads/1/photo.xenc#type=image/png "xenc:image/png")';

    const setCurrentNote = vi.fn();
    let storeError: string | null = null;
    const updateNote = vi.fn(async (_id: string, _payload: unknown, _version: number) =>
      baseEncryptedNote({
        version: 8,
        encrypted_content: 'cipher-new',
        wrapped_dek: 'wrapped-new',
        encryption_metadata: JSON.stringify(metadata),
      })
    );
    const decryptNote = vi.fn(
      (_encryptedTitle: string | null, _payload: EncryptedPayload, _noteId?: string) => ({
        title: 'Secure title',
        content: legacyContent,
      })
    );
    const encryptNote = vi.fn(() => ({
      encryptedTitle: null,
      encryptedContent: {
        ciphertext: 'cipher-updated',
        metadata,
      },
      keywords: [],
    }));

    await loadNote({
      id: 'note-1',
      isOnline: () => true,
      getLocalNotes: () => [],
      getNote: async () => baseEncryptedNote(),
      getBacklinks: async () => ({ backlinks: [] }),
      isEncryptionUnlocked: () => true,
      decryptNote,
      encryptNote,
      updateNote,
      isConflictError: () => false,
      setIsLoading: () => {},
      setError: (value) => {
        storeError = value;
      },
      setAutoSaveStatus: () => {},
      setAutoSaveError: () => {},
      resetLastSaved: () => {},
      setCurrentNote,
      setBacklinks: () => {},
      setDirty: () => {},
      offlineUnavailableMessage: 'offline unavailable',
      decryptFailedMessage: 'decrypt failed',
      defaultErrorMessage: 'default error',
    });

    expect(updateNote).toHaveBeenCalledTimes(1);
    expect(updateNote).toHaveBeenCalledWith('note-1', expect.any(Object), 7);
    expect(encryptNote).toHaveBeenCalledWith(
      'Secure title',
      expect.stringContaining('xela-enc-v1:'),
      'note-1'
    );

    expect(storeError).toBeNull();
    const loadedNote = setCurrentNote.mock.calls.at(-1)?.[0] as Note | null;
    expect(loadedNote).not.toBeNull();
    expect(loadedNote?.version).toBe(8);
    expect(loadedNote?.content).toContain('xela-enc-v1:');
    expect(loadedNote?.content).not.toContain('#type=');

    expect(getEncryptedAttachmentMigrationAuditCounters()).toEqual({
      detectedNotes: 1,
      detectedLinks: 1,
      persistedNotes: 1,
      persistedLinks: 1,
      failedNotes: 0,
      failedLinks: 0,
    });
  });

  it('records failed persistence but still loads migrated content', async () => {
    const legacyContent =
      '[Encrypted attachment: report.pdf](/uploads/1/report.xenc#type=application/pdf)';
    const conflictErr = new Error('conflict');

    const setCurrentNote = vi.fn();
    let storeError: string | null = null;
    const updateNote = vi.fn(async (_id: string, _payload: unknown, _version: number) => {
      throw conflictErr;
    });

    await loadNote({
      id: 'note-1',
      isOnline: () => true,
      getLocalNotes: () => [],
      getNote: async () => baseEncryptedNote(),
      getBacklinks: async () => ({ backlinks: [] }),
      isEncryptionUnlocked: () => true,
      decryptNote: () => ({
        title: 'Secure title',
        content: legacyContent,
      }),
      encryptNote: () => ({
        encryptedTitle: null,
        encryptedContent: {
          ciphertext: 'cipher-updated',
          metadata,
        },
        keywords: [],
      }),
      updateNote,
      isConflictError: (err) => err === conflictErr,
      setIsLoading: () => {},
      setError: (value) => {
        storeError = value;
      },
      setAutoSaveStatus: () => {},
      setAutoSaveError: () => {},
      resetLastSaved: () => {},
      setCurrentNote,
      setBacklinks: () => {},
      setDirty: () => {},
      offlineUnavailableMessage: 'offline unavailable',
      decryptFailedMessage: 'decrypt failed',
      defaultErrorMessage: 'default error',
    });

    expect(updateNote).toHaveBeenCalledTimes(1);
    const loadedNote = setCurrentNote.mock.calls.at(-1)?.[0] as Note | null;
    expect(loadedNote).not.toBeNull();
    expect(loadedNote?.version).toBe(7);
    expect(loadedNote?.content).toContain('xela-enc-v1:');
    expect(storeError).toBeNull();
    expect(getEncryptedAttachmentMigrationAuditCounters()).toEqual({
      detectedNotes: 1,
      detectedLinks: 1,
      persistedNotes: 0,
      persistedLinks: 0,
      failedNotes: 1,
      failedLinks: 1,
    });
  });
});
