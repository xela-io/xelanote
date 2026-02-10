import { request } from './client';
import type { Note, RecipeIngredient, RecipeMetadata } from './types';

/**
 * Decrypt a note by clearing all encryption fields and setting plaintext content.
 * The client sends the already-decrypted title and content.
 */
export async function decryptNote(
  id: string,
  title: string,
  content: string,
  version: number,
  recipeData?: { recipe_metadata?: RecipeMetadata; recipe_ingredients?: RecipeIngredient[] }
): Promise<Note> {
  const body: Record<string, unknown> = { title, content };
  if (recipeData?.recipe_metadata) {
    body.recipe_metadata = recipeData.recipe_metadata;
  }
  if (recipeData?.recipe_ingredients) {
    body.recipe_ingredients = recipeData.recipe_ingredients;
  }
  return request(`/notes/${id}/decrypt`, {
    method: 'POST',
    headers: {
      'If-Match': version.toString(),
    },
    body: JSON.stringify(body),
  });
}

/**
 * Get the encryption_default status for a folder.
 */
export async function getFolderEncryptionDefault(folderId: number): Promise<boolean> {
  const response = await request<{ encrypted: boolean }>(`/folders/${folderId}/encryption-default`);
  return response.encrypted;
}

/**
 * Update the encryption_default status for a folder.
 */
export async function updateFolderEncryptionDefault(
  folderId: number,
  encrypted: boolean
): Promise<void> {
  await request(`/folders/${folderId}/encryption-default`, {
    method: 'PUT',
    body: JSON.stringify({ encrypted }),
  });
}
