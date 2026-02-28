import type { Note, NotePayload, RecipeIngredient, RecipeMetadata } from '$lib/api';
import type { EncryptedPayload } from '$lib/crypto/e2e';
import { migrateLegacyEncryptedAttachmentLinks } from '$lib/editor/encrypted-attachment-markdown';
import { parseEncryptionMetadata } from '$lib/stores/encryption-metadata';

type RecipeDecryptPayload = {
  recipe_metadata?: RecipeMetadata;
  recipe_ingredients?: RecipeIngredient[];
};

type RecipeContentPayload = {
  content?: string;
  recipe_metadata?: RecipeMetadata;
  recipe_ingredients?: RecipeIngredient[];
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object';
}

function isRecipeMetadata(value: unknown): value is RecipeMetadata {
  if (!isRecord(value)) return false;
  return (
    typeof value.note_id === 'string' &&
    typeof value.user_id === 'number' &&
    typeof value.servings === 'number' &&
    typeof value.updated_at === 'string'
  );
}

function isRecipeIngredient(value: unknown): value is RecipeIngredient {
  if (!isRecord(value)) return false;
  return (
    typeof value.name === 'string' &&
    typeof value.display_order === 'number' &&
    typeof value.optional === 'boolean' &&
    typeof value.scalable === 'boolean'
  );
}

function isRecipeIngredientArray(value: unknown): value is RecipeIngredient[] {
  return Array.isArray(value) && value.every((entry) => isRecipeIngredient(entry));
}

function parseRecipeContentPayload(raw: string): RecipeContentPayload | null {
  try {
    const parsed = JSON.parse(raw);
    if (!isRecord(parsed)) return null;

    const content = typeof parsed.content === 'string' ? parsed.content : undefined;
    const recipeMetadata = isRecipeMetadata(parsed.recipe_metadata)
      ? parsed.recipe_metadata
      : undefined;
    const recipeIngredients = isRecipeIngredientArray(parsed.recipe_ingredients)
      ? parsed.recipe_ingredients
      : undefined;

    return {
      content,
      recipe_metadata: recipeMetadata,
      recipe_ingredients: recipeIngredients,
    };
  } catch {
    return null;
  }
}

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
    content: string,
    noteId: string
  ) => {
    encryptedTitle: string | null;
    encryptedContent: EncryptedPayload;
    keywords: string[];
  };
  decryptNote: (
    encryptedTitle: string | null,
    payload: EncryptedPayload,
    noteId?: string
  ) => { title: string | null; content: string };
  extractUniqueLinks: (content: string) => { title: string }[];
  updateNote: (id: string, payload: NotePayload, version: number) => Promise<Note>;
  decryptNoteApi: (
    id: string,
    title: string,
    content: string,
    version: number,
    recipeData?: RecipeDecryptPayload
  ) => Promise<Note>;
  getRecipeDetail: (id: string) => Promise<{
    metadata?: RecipeMetadata | null;
    ingredients?: RecipeIngredient[];
  }>;
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

      let recipeData: RecipeDecryptPayload | undefined;
      if (currentNote.note_type === 'recipe' && currentNote.content) {
        const parsed = parseRecipeContentPayload(currentNote.content);
        if (parsed?.recipe_metadata || parsed?.recipe_ingredients) {
          recipeData = {
            recipe_metadata: parsed.recipe_metadata,
            recipe_ingredients: parsed.recipe_ingredients,
          };
          deps.setCurrentNote({ ...currentNote, content: parsed.content ?? '' });
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
      contentToEncrypt,
      currentNote.id
    );

    const payload: NotePayload = {
      title: encryptedTitle ? '' : currentNote.title,
      encrypted_title: encryptedTitle,
      title_encrypted: !!encryptedTitle,
      encrypted_content: encryptedContent.ciphertext,
      wrapped_dek: encryptedContent.metadata.wrapped_dek,
      encryption_metadata: JSON.stringify(encryptedContent.metadata),
      keywords,
      folder_path: currentNote.folder_path,
    };

    const updated = await deps.updateNote(currentNote.id, payload, currentNote.version);

    let processedUpdate = updated;
    if (updated.content_encrypted && updated.encrypted_content) {
      const encryptedPayload: EncryptedPayload = {
        ciphertext: updated.encrypted_content,
        metadata: parseEncryptionMetadata(updated.encryption_metadata),
      };

      const dec = deps.decryptNote(updated.encrypted_title || null, encryptedPayload, updated.id);
      const migrated = migrateLegacyEncryptedAttachmentLinks(dec.content);

      processedUpdate = {
        ...updated,
        title: dec.title || updated.title,
        content: migrated.content,
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
