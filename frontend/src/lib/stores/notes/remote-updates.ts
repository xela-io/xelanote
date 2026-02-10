import type { Note } from '$lib/api';
import type { EncryptedPayload } from '$lib/crypto/e2e';

export interface RemoteUpdateDeps {
  getCurrentNote: () => Note | null;
  getIsSaving: () => boolean;
  getIsDirty: () => boolean;
  getLastSavedVersion: () => number | null;
  getLastSaveTimestamp: () => number | null;
  clearLastSaved: () => void;
  setCurrentNote: (note: Note | null) => void;
  updateNoteInList: (note: Note) => void;
  getNotes: () => Note[];
  setNotes: (notes: Note[]) => void;
  isEncryptionUnlocked: () => boolean;
  decryptNote: (
    encryptedTitle: string | null,
    payload: EncryptedPayload
  ) => { title: string | null; content: string };
  warn: (message: string, options: { label: string; handler: () => void }) => void;
  loadNote: (id: string) => void;
}

export interface RemoteCreateDeps {
  getNotes: () => Note[];
  setNotes: (notes: Note[]) => void;
  info: (message: string) => void;
}

export interface RemoteDeleteDeps {
  getNotes: () => Note[];
  setNotes: (notes: Note[]) => void;
  getCurrentNote: () => Note | null;
  setCurrentNote: (note: Note | null) => void;
  info: (message: string) => void;
}

export function handleRemoteUpdate(remoteNote: Note, deps: RemoteUpdateDeps) {
  const localNote = deps.getCurrentNote();

  if (deps.getIsSaving() && localNote && localNote.id === remoteNote.id) {
    console.log('[WebSocket] Update während Save ignoriert (potentielles Echo)', {
      remoteVersion: remoteNote.version,
      isSaving: deps.getIsSaving(),
    });
    return;
  }

  const isEcho =
    localNote &&
    localNote.id === remoteNote.id &&
    deps.getLastSavedVersion() !== null &&
    remoteNote.version === deps.getLastSavedVersion() &&
    deps.getLastSaveTimestamp() !== null &&
    Date.now() - (deps.getLastSaveTimestamp() || 0) < 2000;

  if (isEcho) {
    console.log('[WebSocket] Echo erkannt, ignoriere', {
      version: remoteNote.version,
      timeSinceSave: Date.now() - (deps.getLastSaveTimestamp() || 0),
    });
    deps.clearLastSaved();
    return;
  }

  let processedNote = remoteNote;
  if (remoteNote.content_encrypted && remoteNote.encrypted_content) {
    if (deps.isEncryptionUnlocked()) {
      try {
        const encryptedPayload: EncryptedPayload = {
          ciphertext: remoteNote.encrypted_content,
          metadata: JSON.parse(remoteNote.encryption_metadata || '{}'),
        };
        const decrypted = deps.decryptNote(remoteNote.encrypted_title || null, encryptedPayload);
        processedNote = {
          ...remoteNote,
          title: decrypted.title || remoteNote.title,
          content: decrypted.content,
        };
      } catch (err) {
        console.error('[WebSocket] Failed to decrypt remote note:', err);
        deps.updateNoteInList(remoteNote);
        deps.setNotes(
          deps.getNotes().map((note) => (note.id === remoteNote.id ? remoteNote : note))
        );
        return;
      }
    } else {
      console.log('[WebSocket] Encryption locked, skipping currentNote update');
      deps.updateNoteInList(remoteNote);
      deps.setNotes(deps.getNotes().map((note) => (note.id === remoteNote.id ? remoteNote : note)));
      return;
    }
  }

  if (localNote && localNote.id === processedNote.id) {
    const versionDiverged = processedNote.version !== localNote.version;
    if (deps.getIsDirty() && versionDiverged) {
      const localChanges = localNote.content.length - processedNote.content.length;
      const changeInfo = Math.abs(localChanges) > 0 ? ` (±${Math.abs(localChanges)} Zeichen)` : '';

      console.warn('[Konflikt] Remote Update mit lokalen Änderungen', {
        localVersion: localNote.version,
        remoteVersion: processedNote.version,
        localChanges,
      });

      deps.warn(
        `Remote-Update erkannt (Version ${processedNote.version}). Du hast lokale Änderungen${changeInfo}. Speichern überschreibt Remote-Version.`,
        {
          label: 'Remote laden',
          handler: () => deps.loadNote(processedNote.id),
        }
      );
      return;
    }

    if (!deps.getIsDirty() || !versionDiverged) {
      deps.setCurrentNote(processedNote);
      deps.updateNoteInList(processedNote);
      deps.setNotes(
        deps.getNotes().map((note) => (note.id === processedNote.id ? processedNote : note))
      );
      return;
    }
  }

  if (!localNote || localNote.id !== processedNote.id) {
    deps.updateNoteInList(processedNote);
    deps.setNotes(
      deps.getNotes().map((note) => (note.id === processedNote.id ? processedNote : note))
    );
  }
}

export function handleRemoteCreate(note: Note, deps: RemoteCreateDeps) {
  if (!deps.getNotes().find((n) => n.id === note.id)) {
    deps.setNotes([note, ...deps.getNotes()]);
    deps.info(`New note "${note.title}" created`);
  }
}

export function handleRemoteDelete(id: string, deps: RemoteDeleteDeps) {
  const remaining = deps.getNotes().filter((n) => n.id !== id);
  deps.setNotes(remaining);

  if (deps.getCurrentNote()?.id === id) {
    deps.setCurrentNote(null);
    deps.info('This note was deleted');
    return;
  }

  const deletedNote = remaining.find((n) => n.id === id);
  if (deletedNote) {
    deps.info(`Note "${deletedNote.title}" was deleted`);
  }
}
