import type { Note, NotePayload } from '$lib/api';
import type { EncryptedPayload } from '$lib/crypto/e2e';

export interface ToggleEncryptionDeps {
  getCurrentNote: () => Note | null;
  getIsSaving: () => boolean;
  setIsSaving: (value: boolean) => void;
  setError: (value: string | null) => void;
  setDirty: (dirty: boolean) => void;
  setCurrentNote: (note: Note | null) => void;
  setNotes: (notes: Note[]) => void;
  getNotes: () => Note[];
  setLastSavedVersion: (version: number | null) => void;
  setLastSaveTimestamp: (timestamp: number | null) => void;
  cancelAutoSave: () => void;
  isEncryptionUnlocked: () => boolean;
  encryptNote: (
    title: string,
    content: string
  ) => { encryptedTitle: string | null; encryptedContent: { ciphertext: string; metadata: any }; keywords: string[] };
  decryptNote: (
    encryptedTitle: string | null,
    payload: EncryptedPayload
  ) => { title: string | null; content: string };
  extractUniqueLinks: (content: string) => { title: string }[];
  updateNote: (id: string, payload: NotePayload, version: number) => Promise<Note>;
  decryptNoteApi: (
    id: string,
    title: string,
    content: string,
    version: number,
    recipeData?: { recipe_metadata?: any; recipe_ingredients?: any[] }
  ) => Promise<Note>;
  getRecipeDetail: (id: string) => Promise<{ metadata?: any; ingredients?: any[] }>;
  removeFromSearchIndex: (id: string) => void;
  addToSearchIndex: (id: string, title: string, content: string) => void;
}

export async function toggleEncryption(deps: ToggleEncryptionDeps) {
  const currentNote = deps.getCurrentNote();
  if (!currentNote) return;

  if (deps.getIsSaving()) {
    console.log('[NOTES] Save in progress, skipping encryption toggle');
    return;
  }

  deps.cancelAutoSave();

  deps.setIsSaving(true);
  deps.setError(null);

  try {
    if (currentNote.content_encrypted !== false) {
      console.log('[NOTES] Decrypting note:', currentNote.id);

      let recipeData:
        | { recipe_metadata?: any; recipe_ingredients?: any[] }
        | undefined;
      if (currentNote.note_type === 'recipe' && currentNote.content) {
        try {
          const parsed = JSON.parse(currentNote.content);
          if (parsed.recipe_metadata || parsed.recipe_ingredients) {
            recipeData = {
              recipe_metadata: parsed.recipe_metadata,
              recipe_ingredients: parsed.recipe_ingredients,
            };
            deps.setCurrentNote({ ...currentNote, content: parsed.content ?? '' });
          }
        } catch {
          // ignore legacy content
        }
      }

      const decrypted = await deps.decryptNoteApi(
        currentNote.id,
        currentNote.title,
        currentNote.content,
        currentNote.version,
        recipeData
      );

      deps.setCurrentNote(decrypted);
      deps.setLastSavedVersion(decrypted.version);
      deps.setLastSaveTimestamp(Date.now());
      deps.setDirty(false);

      deps.removeFromSearchIndex(decrypted.id);
      deps.setNotes(deps.getNotes().map((n) => (n.id === decrypted.id ? decrypted : n)));
      return;
    }

    console.log('[NOTES] Encrypting note:', currentNote.id);

    if (!deps.isEncryptionUnlocked()) {
      throw new Error('Encryption locked - please re-login');
    }

    let contentToEncrypt = currentNote.content;
    if (currentNote.note_type === 'recipe') {
      try {
        const detail = await deps.getRecipeDetail(currentNote.id);
        if (detail.metadata || (detail.ingredients && detail.ingredients.length > 0)) {
          contentToEncrypt = JSON.stringify({
            content: currentNote.content,
            recipe_metadata: detail.metadata,
            recipe_ingredients: detail.ingredients,
          });
        }
      } catch (err) {
        console.warn(
          '[NOTES] Failed to load recipe data for encryption, encrypting content only:',
          err
        );
      }
    }

    const { encryptedTitle, encryptedContent, keywords } = deps.encryptNote(
      currentNote.title,
      contentToEncrypt
    );
    const uniqueLinks = deps.extractUniqueLinks(currentNote.content);

    const payload: NotePayload = {
      title: encryptedTitle ? '' : currentNote.title,
      encrypted_title: encryptedTitle,
      title_encrypted: !!encryptedTitle,
      encrypted_content: encryptedContent.ciphertext,
      wrapped_dek: encryptedContent.metadata.wrapped_dek,
      encryption_metadata: JSON.stringify(encryptedContent.metadata),
      keywords,
      folder_path: currentNote.folder_path,
      links: uniqueLinks.map((l) => ({ target_title: l.title })),
    };

    const updated = await deps.updateNote(currentNote.id, payload, currentNote.version);

    let processedUpdate = updated;
    if (updated.content_encrypted && updated.encrypted_content) {
      const encryptedPayload: EncryptedPayload = {
        ciphertext: updated.encrypted_content,
        metadata: JSON.parse(updated.encryption_metadata || '{}'),
      };

      const dec = deps.decryptNote(updated.encrypted_title || null, encryptedPayload);

      processedUpdate = {
        ...updated,
        title: dec.title || updated.title,
        content: dec.content,
      };
    }

    deps.setCurrentNote(processedUpdate);
    deps.setLastSavedVersion(processedUpdate.version);
    deps.setLastSaveTimestamp(Date.now());
    deps.setDirty(false);

    deps.addToSearchIndex(processedUpdate.id, processedUpdate.title, processedUpdate.content);
    deps.setNotes(deps.getNotes().map((n) => (n.id === processedUpdate.id ? processedUpdate : n)));
  } catch (err) {
    deps.setError(err instanceof Error ? err.message : 'Failed to toggle encryption');
    throw err;
  } finally {
    deps.setIsSaving(false);
  }
}
