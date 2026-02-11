import type { Backlink, Note, NotePayload, OfflineNoteContext } from '$lib/api';
import type { EncryptedPayload } from '$lib/crypto/e2e';
import { parseEncryptionMetadata } from '$lib/stores/notes/helpers';

export interface CreateNoteDeps {
  title: string;
  content: string;
  folderPath: string;
  journalOptions?: { note_type: string; journal_date?: string };
  assertOnline: () => void;
  getFolders: () => Array<{ path: string; encryption_default?: boolean }>;
  isEncryptionUnlocked: () => boolean;
  encryptNote: (
    title: string,
    content: string
  ) => {
    encryptedTitle: string | null;
    encryptedContent: EncryptedPayload;
    keywords: string[];
  };
  decryptNote: (
    encryptedTitle: string | null,
    payload: EncryptedPayload
  ) => { title: string | null; content: string };
  extractUniqueLinks: (content: string) => { title: string }[];
  extractDueDates: (content: string) => NotePayload['due_dates'];
  createNote: (payload: NotePayload, offlineContext?: OfflineNoteContext) => Promise<Note>;
  addToSearchIndex: (id: string, title: string, content: string) => void;
  setCurrentNote: (note: Note | null) => void;
  setNotes: (notes: Note[]) => void;
  getNotes: () => Note[];
  setBacklinks: (backlinks: Backlink[]) => void;
  setDirty: (dirty: boolean) => void;
  setError: (value: string | null) => void;
  setIsLoading: (value: boolean) => void;
}

export async function createNote(deps: CreateNoteDeps) {
  deps.setIsLoading(true);
  deps.setError(null);

  try {
    deps.assertOnline();

    const allFolders = deps.getFolders();
    const targetFolder = allFolders.find((f) => f.path === deps.folderPath);
    const isRecipe = deps.journalOptions?.note_type === 'recipe';
    const shouldEncrypt = isRecipe ? false : targetFolder?.encryption_default !== false;

    let note: Note;
    let processedNote: Note;

    if (!shouldEncrypt) {
      console.log('[NOTES] Creating plaintext note (folder encryption_default=false)');
      const uniqueLinks = deps.extractUniqueLinks(deps.content);
      const payload: NotePayload = {
        title: deps.title,
        content: deps.content,
        folder_path: deps.folderPath,
        links: uniqueLinks.map((l) => ({ target_title: l.title })),
        ...(deps.journalOptions && {
          note_type: deps.journalOptions.note_type,
          journal_date: deps.journalOptions.journal_date,
        }),
      };

      note = await deps.createNote(payload);
      processedNote = note;
    } else {
      const isUnlocked = deps.isEncryptionUnlocked();
      console.log('[NOTES] Creating note, encryption unlocked:', isUnlocked);

      if (!isUnlocked) {
        deps.setError('ENCRYPTION_LOCKED');
        deps.setCurrentNote(null);
        deps.setIsLoading(false);
        throw new Error('ENCRYPTION_LOCKED');
      }

      const { encryptedTitle, encryptedContent, keywords } = deps.encryptNote(
        deps.title,
        deps.content
      );
      const uniqueLinks = deps.extractUniqueLinks(deps.content);

      const payload: NotePayload = {
        title: encryptedTitle ? '' : deps.title,
        encrypted_title: encryptedTitle,
        title_encrypted: !!encryptedTitle,
        encrypted_content: encryptedContent.ciphertext,
        wrapped_dek: encryptedContent.metadata.wrapped_dek,
        encryption_metadata: JSON.stringify(encryptedContent.metadata),
        keywords,
        folder_path: deps.folderPath,
        links: uniqueLinks.map((l) => ({ target_title: l.title })),
        due_dates: deps.extractDueDates(deps.content),
        ...(deps.journalOptions && {
          note_type: deps.journalOptions.note_type,
          journal_date: deps.journalOptions.journal_date,
        }),
      };

      const offlineContext: OfflineNoteContext = {
        note_type: deps.journalOptions?.note_type || 'note',
        journal_date: deps.journalOptions?.journal_date,
        encryption_version: encryptedContent.metadata.version as number | undefined,
      };

      note = await deps.createNote(payload, offlineContext);
      console.log('[NOTES] Note created, version:', note.version, 'id:', note.id);

      processedNote = note;
      if (note.content_encrypted && note.encrypted_content) {
        try {
          const encryptedPayload: EncryptedPayload = {
            ciphertext: note.encrypted_content,
            metadata: parseEncryptionMetadata(note.encryption_metadata),
          };

          const decrypted = deps.decryptNote(note.encrypted_title || null, encryptedPayload);

          processedNote = {
            ...note,
            title: decrypted.title || note.title,
            content: decrypted.content,
          };
          console.log('[NOTES] Note decrypted, content length:', decrypted.content.length);
        } catch (err) {
          console.error('[NOTES] Failed to decrypt created note:', err);
          throw new Error('Failed to decrypt created note');
        }
      }
    }

    deps.setNotes([processedNote, ...deps.getNotes()]);
    deps.setCurrentNote(processedNote);
    console.log(
      '[NOTES] currentNote set, version:',
      processedNote.version,
      'content length:',
      processedNote.content.length
    );
    deps.setDirty(false);
    deps.setBacklinks([]);

    if (processedNote.content_encrypted) {
      deps.addToSearchIndex(processedNote.id, processedNote.title, processedNote.content);
    }

    return processedNote;
  } catch (err) {
    deps.setError(err instanceof Error ? err.message : 'Failed to create note');
    throw err;
  } finally {
    deps.setIsLoading(false);
  }
}
