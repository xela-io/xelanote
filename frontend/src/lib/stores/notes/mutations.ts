import type { Backlink, Note, OfflineNoteContext } from '$lib/api';
import type { EncryptedPayload } from '$lib/crypto/e2e';

export interface DeleteNoteDeps {
  getCurrentNote: () => Note | null;
  assertOnline: () => void;
  setIsLoading: (value: boolean) => void;
  setError: (value: string | null) => void;
  deleteNote: (id: string, soft: boolean) => Promise<void>;
  setNotes: (notes: Note[]) => void;
  getNotes: () => Note[];
  removeFromSearchIndex: (id: string) => void;
  setCurrentNote: (note: Note | null) => void;
  setDirty: (dirty: boolean) => void;
  setBacklinks: (backlinks: Backlink[]) => void;
}

export async function deleteCurrentNote(deps: DeleteNoteDeps) {
  const currentNote = deps.getCurrentNote();
  if (!currentNote) return;

  deps.assertOnline();
  deps.setIsLoading(true);
  deps.setError(null);

  try {
    const deletedId = currentNote.id;
    await deps.deleteNote(deletedId, true);
    deps.setNotes(deps.getNotes().filter((n) => n.id !== deletedId));
    deps.removeFromSearchIndex(deletedId);
    deps.setCurrentNote(null);
    deps.setDirty(false);
    deps.setBacklinks([]);
  } catch (err) {
    deps.setError(err instanceof Error ? err.message : 'Failed to delete note');
    throw err;
  } finally {
    deps.setIsLoading(false);
  }
}

export interface MoveNoteDeps {
  id: string;
  folderPath: string;
  assertOnline: () => void;
  getNotes: () => Note[];
  setNotes: (notes: Note[]) => void;
  getCurrentNote: () => Note | null;
  setCurrentNote: (note: Note | null) => void;
  setError: (value: string | null) => void;
  getNote: (id: string) => Promise<Note>;
  decryptNote: (
    encryptedTitle: string | null,
    payload: EncryptedPayload
  ) => { title: string | null; content: string };
  isEncryptionUnlocked: () => boolean;
  encryptNote: (
    title: string,
    content: string
  ) => {
    encryptedTitle: string | null;
    encryptedContent: { ciphertext: string; metadata: any };
    keywords: string[];
  };
  extractUniqueLinks: (content: string) => { title: string }[];
  extractDueDates: (content: string) => unknown[];
  updateNote: (
    id: string,
    payload: any,
    version: number,
    offlineContext?: OfflineNoteContext
  ) => Promise<Note>;
}

export async function moveNote(deps: MoveNoteDeps) {
  deps.assertOnline();

  let note = deps.getNotes().find((n) => n.id === deps.id);

  if (!note) {
    console.log('[NOTES] Note not in local list, loading from backend for move');
    try {
      note = await deps.getNote(deps.id);
      if (note.content_encrypted && note.encrypted_content) {
        const encryptedPayload: EncryptedPayload = {
          ciphertext: note.encrypted_content,
          metadata: JSON.parse(note.encryption_metadata || '{}'),
        };
        const decrypted = deps.decryptNote(note.encrypted_title || null, encryptedPayload);
        note.content = decrypted.content;
        note.title = decrypted.title || note.title;
      }
    } catch (err) {
      console.error('[NOTES] Failed to load note for move:', err);
      throw new Error('Note not found');
    }
  }

  if (!deps.isEncryptionUnlocked()) {
    deps.setError('Encryption locked - please re-login');
    throw new Error('Encryption locked');
  }

  const snapshot = {
    notesList: [...deps.getNotes()],
    currentNote: deps.getCurrentNote() ? { ...deps.getCurrentNote()! } : null,
  };

  const optimisticUpdate = { ...note, folder_path: deps.folderPath };
  deps.setNotes(deps.getNotes().map((n) => (n.id === deps.id ? optimisticUpdate : n)));
  if (deps.getCurrentNote()?.id === deps.id) {
    deps.setCurrentNote(optimisticUpdate);
  }
  deps.setError(null);

  try {
    const { encryptedTitle, encryptedContent, keywords } = deps.encryptNote(
      note.title,
      note.content
    );
    const uniqueLinks = deps.extractUniqueLinks(note.content);

    const payload = {
      title: encryptedTitle ? '' : note.title,
      encrypted_title: encryptedTitle,
      title_encrypted: !!encryptedTitle,
      encrypted_content: encryptedContent.ciphertext,
      wrapped_dek: encryptedContent.metadata.wrapped_dek,
      encryption_metadata: JSON.stringify(encryptedContent.metadata),
      keywords,
      folder_path: deps.folderPath,
      links: uniqueLinks.map((l) => ({ target_title: l.title })),
      due_dates: deps.extractDueDates(note.content),
    };

    const offlineContext: OfflineNoteContext = {
      created_at: note.created_at,
      folder_path: deps.folderPath,
      note_type: note.note_type,
      journal_date: note.journal_date,
      ai_enabled: note.ai_enabled,
      encryption_version: note.encryption_version,
    };

    const updated = await deps.updateNote(deps.id, payload, note.version, offlineContext);

    let processedUpdate = updated;
    if (updated.content_encrypted && updated.encrypted_content) {
      const encryptedPayload: EncryptedPayload = {
        ciphertext: updated.encrypted_content,
        metadata: JSON.parse(updated.encryption_metadata || '{}'),
      };
      const decrypted = deps.decryptNote(updated.encrypted_title || null, encryptedPayload);
      processedUpdate = {
        ...updated,
        title: decrypted.title || updated.title,
        content: decrypted.content,
      };
    }

    deps.setNotes(deps.getNotes().map((n) => (n.id === processedUpdate.id ? processedUpdate : n)));
    if (deps.getCurrentNote()?.id === deps.id) {
      deps.setCurrentNote(processedUpdate);
    }
    return processedUpdate;
  } catch (err) {
    deps.setNotes(snapshot.notesList);
    deps.setCurrentNote(snapshot.currentNote);
    deps.setError(err instanceof Error ? err.message : 'Failed to move note');
    throw err;
  }
}
