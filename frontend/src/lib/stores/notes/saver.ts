import type { Note, NotePayload, OfflineNoteContext, TaskEventPayload } from '$lib/api';
import type { EncryptedPayload } from '$lib/crypto/e2e';
import { migrateLegacyEncryptedAttachmentLinks } from '$lib/editor/encrypted-attachment-markdown';
import { parseEncryptionMetadata } from '$lib/stores/encryption-metadata';
import type { TaskEventQueue } from '$lib/stores/notes/task-events';

export interface SaveNoteDeps {
  getCurrentNote: () => Note | null;
  getIsDirty: () => boolean;
  getIsSaving: () => boolean;
  setIsSaving: (value: boolean) => void;
  setError: (value: string | null) => void;
  setAutoSaveStatus: (status: 'idle' | 'pending' | 'saving' | 'saved' | 'error') => void;
  setAutoSaveError: (value: string | null) => void;
  getAutoSaveTimeout: () => ReturnType<typeof setTimeout> | null;
  setAutoSaveTimeout: (handle: ReturnType<typeof setTimeout> | null) => void;
  incrementSaveCounter: () => number;
  getSaveCounter: () => number;
  setDirty: (dirty: boolean) => void;
  setCurrentNote: (note: Note | null) => void;
  updateNotes: (updater: (notes: Note[]) => Note[]) => void;
  setLastSavedVersion: (version: number | null) => void;
  setLastSaveTimestamp: (timestamp: number | null) => void;
  taskEventQueue: TaskEventQueue;
  assertOnline: () => void;
  isEncryptionUnlocked: () => boolean;
  encryptNote: (
    title: string,
    content: string,
    noteId: string
  ) => {
    encryptedTitle: string | null;
    encryptedContent: EncryptedPayload;
    keywords: string[];
  };
  encryptFolderPath: (folderPath: string, noteID: string, wrappedDEK: string) => string;
  decryptNote: (
    encryptedTitle: string | null,
    payload: EncryptedPayload,
    noteId?: string
  ) => { title: string | null; content: string };
  encryptTaskText: (text: string) => { ciphertext: string; metadata: { wrapped_dek?: string } };
  extractUniqueLinks: (content: string) => { title: string }[];
  extractDueDates: (content: string) => NotePayload['due_dates'];
  updateNote: (
    id: string,
    payload: NotePayload,
    version: number,
    offlineContext?: OfflineNoteContext
  ) => Promise<Note>;
  updateSearchIndex: (id: string, title: string, content: string) => void;
  recordTaskEvent: (noteId: string, payload: TaskEventPayload) => Promise<void>;
  isConflictError: (err: unknown) => boolean;
}

export async function saveNote(deps: SaveNoteDeps) {
  const currentNote = deps.getCurrentNote();
  if (!currentNote || !deps.getIsDirty()) return;

  if (deps.getIsSaving()) {
    console.log('Save already in progress, skipping...');
    return;
  }

  const pendingTimeout = deps.getAutoSaveTimeout();
  if (pendingTimeout) {
    clearTimeout(pendingTimeout);
    deps.setAutoSaveTimeout(null);
    deps.setAutoSaveStatus('idle');
  }

  deps.setIsSaving(true);
  deps.setError(null);

  const saveStartCounter = deps.incrementSaveCounter();

  try {
    deps.assertOnline();

    const uniqueLinks = deps.extractUniqueLinks(currentNote.content);

    let updated: Note;
    let processedUpdate: Note;

    if (currentNote.content_encrypted === false) {
      const payload = {
        title: currentNote.title,
        content: currentNote.content,
        folder_path: currentNote.folder_path,
        links: uniqueLinks.map((l) => ({ target_title: l.title })),
      };

      console.log(
        '[NOTES] Saving plaintext note, current version:',
        currentNote.version,
        'id:',
        currentNote.id,
        'content length:',
        currentNote.content.length,
        'links:',
        uniqueLinks.length
      );
      updated = await deps.updateNote(currentNote.id, payload, currentNote.version);
      processedUpdate = updated;
    } else {
      if (!deps.isEncryptionUnlocked()) {
        deps.setError('Encryption locked - please re-login');
        deps.setAutoSaveStatus('error');
        deps.setAutoSaveError('Encryption locked - please re-login');
        throw new Error('Encryption locked');
      }

      const { encryptedTitle, encryptedContent, keywords } = deps.encryptNote(
        currentNote.title,
        currentNote.content,
        currentNote.id
      );

      const encryptedFolderPath = deps.encryptFolderPath(
        currentNote.folder_path,
        currentNote.id,
        encryptedContent.metadata.wrapped_dek
      );

      const payload = {
        title: encryptedTitle ? '' : currentNote.title,
        encrypted_title: encryptedTitle,
        title_encrypted: !!encryptedTitle,
        encrypted_content: encryptedContent.ciphertext,
        wrapped_dek: encryptedContent.metadata.wrapped_dek,
        encryption_metadata: JSON.stringify(encryptedContent.metadata),
        encrypted_folder_path: encryptedFolderPath,
        keywords,
        folder_path: currentNote.folder_path,
      };

      const offlineContext: OfflineNoteContext = {
        created_at: currentNote.created_at,
        folder_path: currentNote.folder_path,
        note_type: currentNote.note_type,
        journal_date: currentNote.journal_date,
        ai_enabled: currentNote.ai_enabled,
        encryption_version: currentNote.encryption_version,
      };

      console.log(
        '[NOTES] Saving note, current version:',
        currentNote.version,
        'id:',
        currentNote.id,
        'content length:',
        currentNote.content.length,
        'links:',
        uniqueLinks.length
      );
      console.log(
        '[NOTES] Payload encrypted_content (base64) length:',
        encryptedContent.ciphertext.length
      );
      updated = await deps.updateNote(currentNote.id, payload, currentNote.version, offlineContext);
      console.log(
        '[NOTES] Save successful, backend returned version:',
        updated.version,
        'encrypted_content from backend length:',
        updated.encrypted_content?.length || 0
      );

      processedUpdate = updated;
      if (updated.content_encrypted && updated.encrypted_content) {
        try {
          const encryptedPayload: EncryptedPayload = {
            ciphertext: updated.encrypted_content,
            metadata: parseEncryptionMetadata(updated.encryption_metadata),
          };

          const decrypted = deps.decryptNote(
            updated.encrypted_title || null,
            encryptedPayload,
            updated.id
          );
          const migrated = migrateLegacyEncryptedAttachmentLinks(decrypted.content);

          processedUpdate = {
            ...updated,
            title: decrypted.title || updated.title,
            content: migrated.content,
          };
          console.log('[NOTES] Update decrypted, content length:', decrypted.content.length);
        } catch (err) {
          console.error('[NOTES] Failed to decrypt updated note:', err);
          throw new Error('Failed to decrypt updated note');
        }
      }
    }

    deps.setCurrentNote(processedUpdate);
    deps.setLastSavedVersion(processedUpdate.version);
    deps.setLastSaveTimestamp(Date.now());

    if (processedUpdate.content_encrypted) {
      deps.updateSearchIndex(processedUpdate.id, processedUpdate.title, processedUpdate.content);
    }

    if (deps.getSaveCounter() === saveStartCounter) {
      deps.setDirty(false);
    } else {
      console.log('[Save] Weitere Änderungen während Save erkannt, isDirty bleibt true');
    }

    deps.updateNotes((notes) =>
      notes.map((n) => (n.id === processedUpdate.id ? processedUpdate : n))
    );

    const noteEvents = deps.taskEventQueue.getForNote(processedUpdate.id);
    if (noteEvents.length > 0) {
      deps.taskEventQueue.clearForNote(processedUpdate.id);
      for (const evt of noteEvents) {
        const payload: TaskEventPayload = {
          event_type: evt.eventType,
          task_index: evt.taskIndex,
        };
        if (processedUpdate.content_encrypted) {
          const encrypted = deps.encryptTaskText(evt.taskText);
          payload.encrypted_task_text = encrypted.ciphertext;
          payload.wrapped_dek = encrypted.metadata.wrapped_dek;
          payload.encryption_metadata = JSON.stringify(encrypted.metadata);
        } else {
          payload.task_text = evt.taskText.substring(0, 500);
        }
        deps
          .recordTaskEvent(processedUpdate.id, payload)
          .catch((err) => console.warn('[TASK-EVENTS] Failed to record:', err));
      }
    }

    deps.setAutoSaveStatus('saved');
    setTimeout(() => {
      deps.setAutoSaveStatus('idle');
    }, 3000);

    return processedUpdate;
  } catch (err) {
    if (deps.isConflictError(err)) {
      console.error(
        '[NOTES] Save failed: Version conflict! Local version:',
        currentNote.version,
        'Error:',
        err instanceof Error ? err.message : String(err)
      );
    }
    deps.setError(err instanceof Error ? err.message : 'Failed to save note');
    deps.setAutoSaveStatus('error');
    throw err;
  } finally {
    deps.setIsSaving(false);
  }
}
