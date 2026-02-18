import { request } from './client';
import type {
  CollectionShare,
  GeneratedIngredient,
  GeneratedRecipe,
  IngredientSuggestionResult,
  Note,
  RecipeCollection,
  RecipeDetail,
  RecipeImage,
  RecipeIngredient,
  RecipeListItem,
  RecipeMetadata,
  ScaledIngredient,
  SharedCollection,
  SharedNote,
  SimilarRecipeResult,
} from './types';

/**
 * List all recipes for the current user (owner-only).
 */
export async function getRecipes(): Promise<RecipeListItem[]> {
  const result = await request<{ recipes: RecipeListItem[] }>('/recipes');
  return result.recipes || [];
}

/**
 * Get full recipe detail including metadata, ingredients, and collections.
 */
export async function getRecipeDetail(noteId: string): Promise<RecipeDetail> {
  return request<RecipeDetail>(`/recipes/${noteId}`);
}

/**
 * Update recipe metadata (servings, prep/cook time, difficulty, source URL).
 * Uses optimistic locking via expected_updated_at.
 */
export async function updateRecipeMetadata(
  noteId: string,
  metadata: {
    servings: number;
    prep_time_minutes?: number | null;
    cook_time_minutes?: number | null;
    source_url?: string | null;
    difficulty?: string | null;
  },
  expectedUpdatedAt: string
): Promise<RecipeMetadata> {
  return request<RecipeMetadata>(`/recipes/${noteId}/metadata`, {
    method: 'PUT',
    body: JSON.stringify({
      ...metadata,
      expected_updated_at: expectedUpdatedAt,
    }),
  });
}

/**
 * Replace all ingredients for a recipe (atomic operation).
 * Uses optimistic locking via expected_updated_at from recipe_metadata.
 */
export async function setRecipeIngredients(
  noteId: string,
  ingredients: RecipeIngredient[],
  expectedUpdatedAt: string
): Promise<void> {
  return request(`/recipes/${noteId}/ingredients`, {
    method: 'PUT',
    body: JSON.stringify({
      ingredients,
      expected_updated_at: expectedUpdatedAt,
    }),
  });
}

/**
 * Get scaled ingredients for a target number of servings.
 */
export async function getScaledIngredients(
  noteId: string,
  targetServings: number
): Promise<ScaledIngredient[]> {
  const result = await request<{ ingredients: ScaledIngredient[] }>(
    `/recipes/${noteId}/scaled?servings=${targetServings}`
  );
  return result.ingredients || [];
}

// Recipe Collections API

/**
 * List all recipe collections (cookbooks) for the current user.
 */
export async function getRecipeCollections(): Promise<RecipeCollection[]> {
  const result = await request<{ collections: RecipeCollection[] }>('/recipes/collections');
  return result.collections || [];
}

/**
 * Create a new recipe collection.
 */
export async function createRecipeCollection(
  name: string,
  description?: string | null,
  color?: string | null
): Promise<RecipeCollection> {
  return request<RecipeCollection>('/recipes/collections', {
    method: 'POST',
    body: JSON.stringify({ name, description, color }),
  });
}

/**
 * Update a recipe collection.
 */
export async function updateRecipeCollection(
  collectionId: number,
  name: string,
  description?: string | null,
  color?: string | null
): Promise<void> {
  return request(`/recipes/collections/${collectionId}`, {
    method: 'PUT',
    body: JSON.stringify({ name, description, color }),
  });
}

/**
 * Delete a recipe collection.
 */
export async function deleteRecipeCollection(collectionId: number): Promise<void> {
  return request(`/recipes/collections/${collectionId}`, {
    method: 'DELETE',
  });
}

/**
 * Add a recipe to a collection.
 */
export async function addRecipeToCollection(collectionId: number, noteId: string): Promise<void> {
  return request(`/recipes/collections/${collectionId}/items`, {
    method: 'POST',
    body: JSON.stringify({ note_id: noteId }),
  });
}

/**
 * Remove a recipe from a collection.
 */
export async function removeRecipeFromCollection(
  collectionId: number,
  noteId: string
): Promise<void> {
  return request(`/recipes/collections/${collectionId}/items/${noteId}`, {
    method: 'DELETE',
  });
}

/**
 * List recipes in a collection.
 */
export async function getCollectionItems(collectionId: number): Promise<RecipeListItem[]> {
  const result = await request<{ recipes: RecipeListItem[] }>(
    `/recipes/collections/${collectionId}/items`
  );
  return result.recipes || [];
}

// --- Recipe Images ---

/**
 * Add an image to a recipe.
 */
export async function addRecipeImage(
  noteId: string,
  imageUrl: string,
  caption?: string | null
): Promise<RecipeImage> {
  return request<RecipeImage>(`/recipes/${noteId}/images`, {
    method: 'POST',
    body: JSON.stringify({ image_url: imageUrl, caption }),
  });
}

/**
 * Update the caption of a recipe image.
 */
export async function updateRecipeImageCaption(
  noteId: string,
  imageId: number,
  caption: string | null
): Promise<void> {
  return request(`/recipes/${noteId}/images/${imageId}`, {
    method: 'PUT',
    body: JSON.stringify({ caption }),
  });
}

/**
 * Delete a recipe image.
 */
export async function deleteRecipeImage(noteId: string, imageId: number): Promise<void> {
  return request(`/recipes/${noteId}/images/${imageId}`, {
    method: 'DELETE',
  });
}

/**
 * Reorder recipe images.
 */
export async function reorderRecipeImages(noteId: string, imageIds: number[]): Promise<void> {
  return request(`/recipes/${noteId}/images/order`, {
    method: 'PUT',
    body: JSON.stringify({ image_ids: imageIds }),
  });
}

// --- Collection Sharing ---

/**
 * Share a collection with another user.
 */
export async function shareCollection(
  collectionId: number,
  identifier: string,
  role: string
): Promise<CollectionShare> {
  return request<CollectionShare>(`/recipes/collections/${collectionId}/shares`, {
    method: 'POST',
    body: JSON.stringify({ identifier, role }),
  });
}

/**
 * Get all shares for a collection (owner-only).
 */
export async function getCollectionShares(collectionId: number): Promise<CollectionShare[]> {
  const result = await request<{ shares: CollectionShare[] }>(
    `/recipes/collections/${collectionId}/shares`
  );
  return result.shares || [];
}

/**
 * Update the role of a collection share.
 */
export async function updateCollectionShareRole(
  collectionId: number,
  userId: number,
  role: string
): Promise<void> {
  return request(`/recipes/collections/${collectionId}/shares/${userId}`, {
    method: 'PUT',
    body: JSON.stringify({ role }),
  });
}

/**
 * Remove a collection share.
 */
export async function removeCollectionShare(collectionId: number, userId: number): Promise<void> {
  return request(`/recipes/collections/${collectionId}/shares/${userId}`, {
    method: 'DELETE',
  });
}

/**
 * Get all shared recipes for the current user.
 */
export async function getSharedRecipes(): Promise<SharedNote[]> {
  const result = await request<{ recipes: SharedNote[] }>('/shared/recipes');
  return result.recipes || [];
}

/**
 * Get all shared collections for the current user.
 */
export async function getSharedCollections(): Promise<SharedCollection[]> {
  const result = await request<{ collections: SharedCollection[] }>('/shared/collections');
  return result.collections || [];
}

/**
 * Get recipes in a shared collection.
 */
export async function getSharedCollectionItems(collectionId: number): Promise<Note[]> {
  const result = await request<{ recipes: Note[] }>(`/shared/collections/${collectionId}/items`);
  return result.recipes || [];
}

/**
 * Add a recipe to a shared collection (editor only).
 */
export async function addToSharedCollection(collectionId: number, noteId: string): Promise<void> {
  return request(`/shared/collections/${collectionId}/items`, {
    method: 'POST',
    body: JSON.stringify({ note_id: noteId }),
  });
}

/**
 * Remove a recipe from a shared collection (editor only).
 */
export async function removeFromSharedCollection(
  collectionId: number,
  noteId: string
): Promise<void> {
  return request(`/shared/collections/${collectionId}/items/${noteId}`, {
    method: 'DELETE',
  });
}

// --- Recipe Suggestions (AI) ---

/**
 * Find recipes similar to the given recipe using AI.
 */
export async function findSimilarRecipes(
  noteId: string,
  locale: string,
  collectionId?: number | null
): Promise<SimilarRecipeResult[]> {
  const result = await request<{ results: SimilarRecipeResult[] }>('/recipes/suggestions/similar', {
    method: 'POST',
    body: JSON.stringify({
      note_id: noteId,
      collection_id: collectionId || undefined,
      locale,
    }),
  });
  return result.results || [];
}

/**
 * Suggest recipes based on available ingredients.
 */
export async function suggestByIngredients(
  ingredients: string[],
  locale: string,
  collectionId?: number | null
): Promise<IngredientSuggestionResult> {
  return request<IngredientSuggestionResult>('/recipes/suggestions/by-ingredients', {
    method: 'POST',
    body: JSON.stringify({
      ingredients,
      collection_id: collectionId || undefined,
      locale,
    }),
  });
}

/**
 * Save an AI-generated recipe as a new note.
 */
export async function saveGeneratedRecipe(data: {
  title: string;
  instructions: string;
  servings: number;
  prep_time_minutes?: number | null;
  cook_time_minutes?: number | null;
  difficulty?: string | null;
  source_url?: string | null;
  ingredients: GeneratedIngredient[];
  folder_path?: string;
}): Promise<{ note_id: string; title: string }> {
  return request('/recipes/suggestions/save-generated', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

/**
 * Extract ingredients from a photo using AI vision.
 */
export async function extractIngredientsFromPhoto(file: File, locale: string): Promise<string[]> {
  const formData = new FormData();
  formData.append('image', file);
  formData.append('locale', locale);

  const result = await request<{ ingredients: string[] }>(
    '/recipes/suggestions/extract-ingredients',
    {
      method: 'POST',
      body: formData,
    }
  );
  return result.ingredients || [];
}

/**
 * Extract a full recipe from an uploaded image using AI vision.
 */
export async function importRecipeFromImage(file: File, locale: string): Promise<GeneratedRecipe> {
  const formData = new FormData();
  formData.append('image', file);
  formData.append('locale', locale);

  return request<GeneratedRecipe>('/recipes/suggestions/import-from-image', {
    method: 'POST',
    body: formData,
  });
}

/**
 * Extract a full recipe from a recipe webpage URL.
 */
export async function importRecipeFromURL(url: string, locale: string): Promise<GeneratedRecipe> {
  return request<GeneratedRecipe>('/recipes/suggestions/import-from-url', {
    method: 'POST',
    body: JSON.stringify({ url, locale }),
  });
}
