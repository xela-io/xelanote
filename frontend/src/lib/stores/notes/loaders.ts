import { ApiError, type Backlink, type Note } from '$lib/api';
import type { EncryptedPayload } from '$lib/crypto/e2e';
import { parseEncryptionMetadata } from '$lib/stores/encryption-metadata';

/** Max pagination iterations to prevent infinite loops (500 notes/page × 100 = 50,000 notes) */
const MAX_PAGINATION_ITERATIONS = 100;

/** Default page size for list requests */
const PAGE_SIZE = 500;

/** Deterministic sort: updated_at DESC, then id DESC (for stable UI order) */
function byUpdatedAtDescThenIdDesc(a: Note, b: Note): number {
  const timeCompare = b.updated_at.localeCompare(a.updated_at);
  if (timeCompare !== 0) return timeCompare;
  return b.id.localeCompare(a.id);
}

export interface LoadNotesDeps {
  listNotes: (options: {
    limit: number;
    cursor?: string;
    fields?: 'slim';
    updated_since?: string;
  }) => Promise<{ notes: Note[]; next_cursor?: string; sync_token?: string }>;
  setNotes: (notes: Note[]) => void;
  getNotes: () => Note[];
  setLoading: (value: boolean) => void;
  getSyncToken: () => string | null;
  setSyncToken: (token: string | null) => void;
}

export async function loadNotes(deps: LoadNotesDeps, mode: 'full' | 'delta' = 'full') {
  if (mode === 'delta') {
    const syncToken = deps.getSyncToken();
    if (!syncToken) {
      // No sync token → fallback to full load
      return loadNotesFull(deps);
    }
    try {
      return await loadNotesDelta(deps, syncToken);
    } catch (err) {
      console.error('[NOTES] Delta-sync failed, falling back to full load:', err);
      return loadNotesFull(deps);
    }
  }
  return loadNotesFull(deps);
}

async function loadNotesFull(deps: LoadNotesDeps) {
  deps.setLoading(true);
  try {
    const allNotes: Note[] = [];
    let maxSyncToken: string | null = null;
    let firstPageSyncToken: string | null = null;
    let cursor: string | undefined;
    let iterations = 0;

    // Cursor-pagination loop
    do {
      const result = await deps.listNotes({ limit: PAGE_SIZE, cursor, fields: 'slim' });
      allNotes.push(...(result.notes ?? []));

      // Track high-watermark across all pages
      if (result.sync_token && (!maxSyncToken || result.sync_token > maxSyncToken)) {
        maxSyncToken = result.sync_token;
      }
      if (!firstPageSyncToken && result.sync_token) {
        firstPageSyncToken = result.sync_token;
      }

      cursor = result.next_cursor;
      iterations++;

      if (iterations >= MAX_PAGINATION_ITERATIONS) {
        console.error(
          `[NOTES] Pagination stopped after ${iterations} pages (${allNotes.length} notes). Not all notes may be loaded.`
        );
        break;
      }
    } while (cursor);

    // Race-protection: if pagination took multiple pages, do a delta-pass
    // to catch changes that occurred during pagination
    if (firstPageSyncToken && iterations > 1) {
      try {
        const deltaResult = await deps.listNotes({
          limit: PAGE_SIZE,
          fields: 'slim',
          updated_since: firstPageSyncToken,
        });
        if (deltaResult.notes?.length) {
          // Deduplicate via Map
          const noteMap = new Map(allNotes.map((n) => [n.id, n]));
          for (const note of deltaResult.notes) {
            if (!note.is_deleted) {
              noteMap.set(note.id, note);
            } else {
              noteMap.delete(note.id);
            }
          }
          allNotes.length = 0;
          allNotes.push(...noteMap.values());
        }
        if (deltaResult.sync_token && (!maxSyncToken || deltaResult.sync_token > maxSyncToken)) {
          maxSyncToken = deltaResult.sync_token;
        }
      } catch (err) {
        console.warn('[NOTES] Race-protection delta-pass failed:', err);
      }
    }

    // Stable sort before setting
    allNotes.sort(byUpdatedAtDescThenIdDesc);
    deps.setNotes(allNotes);
    deps.setSyncToken(maxSyncToken);
  } catch (err) {
    console.error('Failed to load notes:', err);
  } finally {
    deps.setLoading(false);
  }
}

async function loadNotesDelta(deps: LoadNotesDeps, syncToken: string) {
  deps.setLoading(true);
  try {
    const changedNotes: Note[] = [];
    let cursor: string | undefined;
    let maxSyncToken: string | null = null;
    let iterations = 0;

    // Cursor-pagination loop for delta
    do {
      const result = await deps.listNotes({
        limit: PAGE_SIZE,
        cursor,
        fields: 'slim',
        updated_since: syncToken,
      });
      changedNotes.push(...(result.notes ?? []));

      if (result.sync_token && (!maxSyncToken || result.sync_token > maxSyncToken)) {
        maxSyncToken = result.sync_token;
      }

      cursor = result.next_cursor;
      iterations++;

      if (iterations >= MAX_PAGINATION_ITERATIONS) {
        console.error(`[NOTES] Delta pagination stopped after ${iterations} pages.`);
        break;
      }
    } while (cursor);

    if (changedNotes.length === 0) {
      // Nothing changed — just update sync token if present
      if (maxSyncToken) {
        deps.setSyncToken(maxSyncToken);
      }
      return;
    }

    // Merge into existing notes
    const existing = deps.getNotes();
    const existingMap = new Map(existing.map((n) => [n.id, n]));

    for (const note of changedNotes) {
      if (note.is_deleted) {
        existingMap.delete(note.id);
      } else {
        existingMap.set(note.id, note);
      }
    }

    // Deterministic sort after merge (prevents UI jumping)
    const merged = [...existingMap.values()].sort(byUpdatedAtDescThenIdDesc);
    deps.setNotes(merged);

    if (maxSyncToken) {
      deps.setSyncToken(maxSyncToken);
    }
  } catch (err) {
    console.error('Failed to delta-sync notes:', err);
    throw err; // Re-throw to trigger fallback in caller
  } finally {
    deps.setLoading(false);
  }
}

export interface LoadNoteDeps {
  id: string;
  isOnline: () => boolean;
  getLocalNotes: () => Note[];
  getNote: (id: string) => Promise<Note>;
  getBacklinks: (id: string) => Promise<{ backlinks: Backlink[] }>;
  isEncryptionUnlocked: () => boolean;
  decryptNote: (
    encryptedTitle: string | null,
    payload: EncryptedPayload
  ) => { title: string | null; content: string };
  setIsLoading: (value: boolean) => void;
  setError: (value: string | null) => void;
  setAutoSaveStatus: (status: 'idle' | 'pending' | 'saving' | 'saved' | 'error') => void;
  setAutoSaveError: (value: string | null) => void;
  resetLastSaved: () => void;
  setCurrentNote: (note: Note | null) => void;
  setBacklinks: (backlinks: Backlink[]) => void;
  setDirty: (dirty: boolean) => void;
  offlineUnavailableMessage: string;
  decryptFailedMessage: string;
  defaultErrorMessage: string;
}

export async function loadNote(deps: LoadNoteDeps) {
  deps.setIsLoading(true);
  deps.setError(null);
  deps.setAutoSaveStatus('idle');
  deps.setAutoSaveError(null);
  deps.resetLastSaved();

  try {
    let note: Note;

    if (!deps.isOnline()) {
      const localNote = deps.getLocalNotes().find((n) => n.id === deps.id);
      if (localNote) {
        note = { ...localNote };
        console.log('[NOTES] Using local note (offline), id:', note.id);
      } else {
        throw new Error(deps.offlineUnavailableMessage);
      }
    } else {
      note = await deps.getNote(deps.id);
      console.log(
        '[NOTES] Loaded note from API, id:',
        note.id,
        'content_encrypted:',
        note.content_encrypted,
        'encrypted_content (base64) length:',
        note.encrypted_content?.length || 0,
        'first 50 chars:',
        note.encrypted_content?.substring(0, 50) || ''
      );
    }

    if (note.content_encrypted && note.encrypted_content) {
      if (!deps.isEncryptionUnlocked()) {
        deps.setError('ENCRYPTION_LOCKED');
        deps.setCurrentNote(null);
        deps.setIsLoading(false);
        throw new Error('ENCRYPTION_LOCKED');
      }

      try {
        const encryptedPayload: EncryptedPayload = {
          ciphertext: note.encrypted_content,
          metadata: parseEncryptionMetadata(note.encryption_metadata),
        };
        console.log(
          '[NOTES] Decrypting loaded note, wrapped_dek length:',
          encryptedPayload.metadata.wrapped_dek?.length || 0
        );

        const { title, content } = deps.decryptNote(note.encrypted_title || null, encryptedPayload);
        console.log('[NOTES] Note decrypted after load, content length:', content.length);
        note.title = title || note.title;
        note.content = content;
      } catch (decryptError) {
        console.error('[NOTES] Failed to decrypt note:', decryptError);
        deps.setError(deps.decryptFailedMessage);
        deps.setCurrentNote(null);
        deps.setIsLoading(false);
        return;
      }
    }

    deps.setCurrentNote(note);
    console.log('[NOTES] currentNote set after load, content length:', note.content.length);
    deps.setDirty(false);

    if (deps.isOnline()) {
      try {
        const result = await deps.getBacklinks(deps.id);
        deps.setBacklinks(result.backlinks);
      } catch {
        // Backlinks not available offline
      }
    } else {
      deps.setBacklinks([]);
    }
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) {
      deps.setError('NOT_FOUND');
    } else {
      deps.setError(err instanceof Error ? err.message : deps.defaultErrorMessage);
    }
    deps.setCurrentNote(null);
  } finally {
    deps.setIsLoading(false);
  }
}
