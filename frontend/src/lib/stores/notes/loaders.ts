import type { Backlink, Note } from '$lib/api';
import type { EncryptedPayload } from '$lib/crypto/e2e';

export interface LoadNotesDeps {
  listNotes: (options: { limit: number }) => Promise<{ notes: Note[] }>;
  setNotes: (notes: Note[]) => void;
  setLoading: (value: boolean) => void;
}

export async function loadNotes(deps: LoadNotesDeps) {
  deps.setLoading(true);
  try {
    const NOTES_LIST_LIMIT = 1000;
    const result = await deps.listNotes({ limit: NOTES_LIST_LIMIT });
    deps.setNotes(result.notes);
  } catch (err) {
    console.error('Failed to load notes:', err);
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
          metadata: JSON.parse(note.encryption_metadata || '{}'),
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
    deps.setError(err instanceof Error ? err.message : deps.defaultErrorMessage);
    deps.setCurrentNote(null);
  } finally {
    deps.setIsLoading(false);
  }
}
