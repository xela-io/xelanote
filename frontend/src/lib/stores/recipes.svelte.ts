// Recipe store using Svelte 5 runes
// Manages recipes, collections, and ingredient scaling

import { goto } from '$app/navigation';
import type {
  CollectionShare,
  Note,
  RecipeCollection,
  RecipeDetail,
  RecipeIngredient,
  RecipeListItem,
  RecipeMetadata,
  ScaledIngredient,
  SharedCollection,
  SharedNote,
} from '$lib/api';
import * as api from '$lib/api';

import { getRecipeFeatureEnabled } from './features.svelte';
import * as notesStore from './notes.svelte';
import * as tree from './tree.svelte';

// Recipe folder path
const RECIPE_FOLDER = '/Rezepte';

// State
let recipes = $state<RecipeListItem[]>([]);
let currentRecipe = $state<RecipeDetail | null>(null);
let collections = $state<RecipeCollection[]>([]);
let targetServings = $state<number>(4);
let recipesLoading = $state(false);
let recipeDetailLoading = $state(false);
let collectionsLoading = $state(false);
let saving = $state(false);
let lastError = $state<string | null>(null);

// Collection filter state
let selectedCollectionId = $state<number | null>(null);
let collectionItems = $state<RecipeListItem[]>([]);
let collectionItemsLoading = $state(false);

// Shared state
let sharedRecipes = $state<SharedNote[]>([]);
let sharedCollections = $state<SharedCollection[]>([]);
let sharedCollectionItems = $state<Note[]>([]);
let sharedRecipesLoading = $state(false);
let sharedCollectionsLoading = $state(false);

// Getters
export function getRecipes() {
  return recipes;
}

export function getCurrentRecipe() {
  return currentRecipe;
}

export function getCollections() {
  return collections;
}

export function getTargetServings() {
  return targetServings;
}

export function getRecipesLoading() {
  return recipesLoading;
}

export function getRecipeDetailLoading() {
  return recipeDetailLoading;
}

export function getCollectionsLoading() {
  return collectionsLoading;
}

export function getSaving() {
  return saving;
}

export function getLastError() {
  return lastError;
}

export function getSelectedCollectionId() {
  return selectedCollectionId;
}

export function getCollectionItems() {
  return collectionItems;
}

export function getCollectionItemsLoading() {
  return collectionItemsLoading;
}

/**
 * Set target servings for scaling (UI-only state).
 */
export function setTargetServings(servings: number) {
  if (servings >= 1 && servings <= 999) {
    targetServings = servings;
  }
}

// === Client-Side Scaling (must match server logic exactly) ===

/**
 * Format a numeric amount for display.
 * Must match server's FormatDisplayAmount exactly.
 */
export function formatDisplayAmount(v: number): string {
  if (v === Math.trunc(v)) return v.toString();
  if (v * 10 === Math.trunc(v * 10)) return v.toFixed(1);
  return v.toFixed(2);
}

/**
 * Format an amount with optional amount_text fallback.
 * Must match server's FormatAmount exactly.
 */
export function formatAmount(
  amount: number | null | undefined,
  amountText: string | null | undefined
): string {
  if (amount == null) return '';
  if (amountText != null) return amountText;
  return formatDisplayAmount(amount);
}

/**
 * Scale ingredients client-side.
 * Must produce identical results to server's ScaleIngredients.
 */
export function scaleIngredients(
  ingredients: RecipeIngredient[],
  baseServings: number,
  servings: number
): ScaledIngredient[] {
  const factor = servings / baseServings;

  return ingredients.map((ing) => {
    if (!ing.scalable || ing.amount == null) {
      return {
        ...ing,
        scaled_amount: ing.amount,
        display_amount: formatAmount(ing.amount, ing.amount_text),
      };
    }

    const scaled = Math.round(ing.amount * factor * 100) / 100;
    return {
      ...ing,
      scaled_amount: scaled,
      display_amount: formatDisplayAmount(scaled),
    };
  });
}

/**
 * Get current scaled ingredients based on targetServings.
 */
export function getScaledIngredients(): ScaledIngredient[] {
  if (!currentRecipe) return [];
  const baseServings = currentRecipe.metadata?.servings ?? 4;
  return scaleIngredients(currentRecipe.ingredients, baseServings, targetServings);
}

// === Recipe CRUD ===

/**
 * Load all recipes for the current user.
 */
export async function loadRecipes() {
  if (!getRecipeFeatureEnabled()) return;

  recipesLoading = true;
  lastError = null;
  try {
    recipes = await api.getRecipes();
  } catch (error) {
    console.error('Failed to load recipes:', error);
    lastError = 'Rezepte konnten nicht geladen werden';
    recipes = [];
  } finally {
    recipesLoading = false;
  }
}

/**
 * Load recipe detail for a specific note.
 */
export async function loadRecipeDetail(noteId: string) {
  recipeDetailLoading = true;
  lastError = null;
  try {
    currentRecipe = await api.getRecipeDetail(noteId);
    // Set target servings to base servings
    targetServings = currentRecipe.metadata?.servings ?? 4;
  } catch (error) {
    console.error('Failed to load recipe detail:', error);
    if (error instanceof api.ApiError) {
      if (error.status === 403) {
        lastError = 'Kein Zugriff auf dieses Rezept';
      } else if (error.status === 404) {
        lastError = 'Rezept nicht gefunden';
      } else {
        lastError = error.message;
      }
    } else {
      lastError = 'Rezept konnte nicht geladen werden';
    }
    currentRecipe = null;
  } finally {
    recipeDetailLoading = false;
  }
}

/**
 * Create a new recipe note.
 */
export async function createRecipe(title: string, content = ''): Promise<string | null> {
  if (!getRecipeFeatureEnabled()) {
    lastError = 'Rezept-Feature ist nicht aktiviert';
    return null;
  }

  lastError = null;
  try {
    const note = await notesStore.createNote(title, content, RECIPE_FOLDER, {
      note_type: 'recipe',
    });

    goto(`/note/${note.id}`);
    await tree.loadTree();

    return note.id;
  } catch (error) {
    console.error('Failed to create recipe:', error);
    lastError = error instanceof Error ? error.message : 'Rezept konnte nicht erstellt werden';
    return null;
  }
}

/**
 * Update recipe metadata.
 */
export async function updateMetadata(
  noteId: string,
  metadata: {
    servings: number;
    prep_time_minutes?: number | null;
    cook_time_minutes?: number | null;
    source_url?: string | null;
    difficulty?: string | null;
  }
): Promise<RecipeMetadata | null> {
  if (!currentRecipe?.metadata) {
    lastError = 'Keine Rezept-Metadaten vorhanden';
    return null;
  }

  saving = true;
  lastError = null;
  try {
    const updated = await api.updateRecipeMetadata(
      noteId,
      metadata,
      currentRecipe.metadata.updated_at
    );
    // Update local state
    currentRecipe = {
      ...currentRecipe,
      metadata: updated,
    };
    targetServings = updated.servings;
    return updated;
  } catch (error) {
    console.error('Failed to update recipe metadata:', error);
    if (error instanceof api.ApiError && error.status === 409) {
      lastError = 'Rezept wurde zwischenzeitlich geändert. Bitte neu laden.';
    } else {
      lastError =
        error instanceof Error ? error.message : 'Metadaten konnten nicht aktualisiert werden';
    }
    return null;
  } finally {
    saving = false;
  }
}

/**
 * Update recipe ingredients (replace all).
 */
export async function updateIngredients(
  noteId: string,
  ingredients: RecipeIngredient[]
): Promise<boolean> {
  if (!currentRecipe?.metadata) {
    lastError = 'Keine Rezept-Metadaten vorhanden';
    return false;
  }

  saving = true;
  lastError = null;
  try {
    await api.setRecipeIngredients(noteId, ingredients, currentRecipe.metadata.updated_at);
    // Silently refresh to get the server's updated_at (needed for optimistic locking).
    // Don't use loadRecipeDetail() here — it sets recipeDetailLoading=true which
    // swaps the editor for a loading spinner, causing focus loss mid-typing.
    const updated = await api.getRecipeDetail(noteId);
    currentRecipe = updated;
    return true;
  } catch (error) {
    console.error('Failed to update recipe ingredients:', error);
    if (error instanceof api.ApiError && error.status === 409) {
      lastError = 'Rezept wurde zwischenzeitlich geändert. Bitte neu laden.';
    } else {
      lastError =
        error instanceof Error ? error.message : 'Zutaten konnten nicht aktualisiert werden';
    }
    return false;
  } finally {
    saving = false;
  }
}

// === Collections ===

/**
 * Load all recipe collections.
 */
export async function loadCollections() {
  if (!getRecipeFeatureEnabled()) return;

  collectionsLoading = true;
  try {
    collections = await api.getRecipeCollections();
  } catch (error) {
    console.error('Failed to load collections:', error);
    collections = [];
  } finally {
    collectionsLoading = false;
  }
}

/**
 * Create a new collection.
 */
export async function createCollection(
  name: string,
  description?: string | null,
  color?: string | null
): Promise<RecipeCollection | null> {
  try {
    const coll = await api.createRecipeCollection(name, description, color);
    collections = [...collections, coll];
    return coll;
  } catch (error) {
    console.error('Failed to create collection:', error);
    lastError = error instanceof Error ? error.message : 'Kochbuch konnte nicht erstellt werden';
    return null;
  }
}

/**
 * Update a collection.
 */
export async function updateCollection(
  collectionId: number,
  name: string,
  description?: string | null,
  color?: string | null
): Promise<boolean> {
  try {
    await api.updateRecipeCollection(collectionId, name, description, color);
    collections = collections.map((c) =>
      c.id === collectionId ? { ...c, name, description, color } : c
    );
    return true;
  } catch (error) {
    console.error('Failed to update collection:', error);
    lastError =
      error instanceof Error ? error.message : 'Kochbuch konnte nicht aktualisiert werden';
    return false;
  }
}

/**
 * Delete a collection.
 */
export async function deleteCollection(collectionId: number): Promise<boolean> {
  try {
    await api.deleteRecipeCollection(collectionId);
    collections = collections.filter((c) => c.id !== collectionId);
    return true;
  } catch (error) {
    console.error('Failed to delete collection:', error);
    lastError = error instanceof Error ? error.message : 'Kochbuch konnte nicht gelöscht werden';
    return false;
  }
}

/**
 * Select a collection and load its items.
 */
export async function selectCollection(collectionId: number) {
  selectedCollectionId = collectionId;
  collectionItemsLoading = true;
  try {
    collectionItems = await api.getCollectionItems(collectionId);
  } catch (error) {
    console.error('Failed to load collection items:', error);
    collectionItems = [];
    lastError =
      error instanceof Error ? error.message : 'Kochbuch-Rezepte konnten nicht geladen werden';
  } finally {
    collectionItemsLoading = false;
  }
}

/**
 * Clear collection filter, show all recipes again.
 */
export function clearCollectionFilter() {
  selectedCollectionId = null;
  collectionItems = [];
}

/**
 * Add a recipe to a collection.
 */
export async function addToCollection(collectionId: number, noteId: string): Promise<boolean> {
  try {
    await api.addRecipeToCollection(collectionId, noteId);
    // Update recipe count locally
    collections = collections.map((c) =>
      c.id === collectionId ? { ...c, recipe_count: (c.recipe_count ?? 0) + 1 } : c
    );
    // Refresh current recipe if viewing it
    if (currentRecipe?.note.id === noteId) {
      await loadRecipeDetail(noteId);
    }
    return true;
  } catch (error) {
    console.error('Failed to add to collection:', error);
    lastError = error instanceof Error ? error.message : 'Rezept konnte nicht hinzugefügt werden';
    return false;
  }
}

/**
 * Remove a recipe from a collection.
 */
export async function removeFromCollection(collectionId: number, noteId: string): Promise<boolean> {
  try {
    await api.removeRecipeFromCollection(collectionId, noteId);
    collections = collections.map((c) =>
      c.id === collectionId ? { ...c, recipe_count: Math.max(0, (c.recipe_count ?? 1) - 1) } : c
    );
    if (currentRecipe?.note.id === noteId) {
      await loadRecipeDetail(noteId);
    }
    return true;
  } catch (error) {
    console.error('Failed to remove from collection:', error);
    lastError = error instanceof Error ? error.message : 'Rezept konnte nicht entfernt werden';
    return false;
  }
}

// === Shared Recipes & Collections ===

export function getSharedRecipes() {
  return sharedRecipes;
}

export function getSharedCollections() {
  return sharedCollections;
}

export function getSharedCollectionItems() {
  return sharedCollectionItems;
}

export function getSharedRecipesLoading() {
  return sharedRecipesLoading;
}

export function getSharedCollectionsLoading() {
  return sharedCollectionsLoading;
}

/**
 * Load all shared recipes for the current user.
 */
export async function loadSharedRecipes() {
  sharedRecipesLoading = true;
  try {
    sharedRecipes = await api.getSharedRecipes();
  } catch (error) {
    console.error('Failed to load shared recipes:', error);
    sharedRecipes = [];
  } finally {
    sharedRecipesLoading = false;
  }
}

/**
 * Load all shared collections for the current user.
 */
export async function loadSharedCollections() {
  sharedCollectionsLoading = true;
  try {
    sharedCollections = await api.getSharedCollections();
  } catch (error) {
    console.error('Failed to load shared collections:', error);
    sharedCollections = [];
  } finally {
    sharedCollectionsLoading = false;
  }
}

/**
 * Load recipes in a shared collection.
 */
export async function loadSharedCollectionItems(collectionId: number) {
  try {
    sharedCollectionItems = await api.getSharedCollectionItems(collectionId);
  } catch (error) {
    console.error('Failed to load shared collection items:', error);
    sharedCollectionItems = [];
  }
}

/**
 * Share a collection with another user.
 */
export async function shareCollectionWithUser(
  collectionId: number,
  identifier: string,
  role: string
): Promise<CollectionShare | null> {
  try {
    const share = await api.shareCollection(collectionId, identifier, role);
    return share;
  } catch (error) {
    console.error('Failed to share collection:', error);
    lastError = error instanceof Error ? error.message : 'Kochbuch konnte nicht geteilt werden';
    return null;
  }
}

/**
 * Remove a collection share.
 */
export async function unshareCollection(collectionId: number, userId: number): Promise<boolean> {
  try {
    await api.removeCollectionShare(collectionId, userId);
    return true;
  } catch (error) {
    console.error('Failed to unshare collection:', error);
    lastError = error instanceof Error ? error.message : 'Teilen konnte nicht aufgehoben werden';
    return false;
  }
}

/**
 * Update the role of a collection share.
 */
export async function updateShareRole(
  collectionId: number,
  userId: number,
  role: string
): Promise<boolean> {
  try {
    await api.updateCollectionShareRole(collectionId, userId, role);
    return true;
  } catch (error) {
    console.error('Failed to update collection share role:', error);
    lastError = error instanceof Error ? error.message : 'Rolle konnte nicht aktualisiert werden';
    return false;
  }
}

/**
 * Get all shares for a collection (owner-only).
 */
export async function getCollectionSharesList(collectionId: number): Promise<CollectionShare[]> {
  try {
    return await api.getCollectionShares(collectionId);
  } catch (error) {
    console.error('Failed to get collection shares:', error);
    return [];
  }
}

/**
 * Add a recipe to a shared collection (editor only).
 */
export async function addToSharedCollection(
  collectionId: number,
  noteId: string
): Promise<boolean> {
  try {
    await api.addToSharedCollection(collectionId, noteId);
    await loadSharedCollectionItems(collectionId);
    return true;
  } catch (error) {
    console.error('Failed to add to shared collection:', error);
    lastError = error instanceof Error ? error.message : 'Rezept konnte nicht hinzugefügt werden';
    return false;
  }
}

/**
 * Remove a recipe from a shared collection (editor only).
 */
export async function removeFromSharedCollection(
  collectionId: number,
  noteId: string
): Promise<boolean> {
  try {
    await api.removeFromSharedCollection(collectionId, noteId);
    await loadSharedCollectionItems(collectionId);
    return true;
  } catch (error) {
    console.error('Failed to remove from shared collection:', error);
    lastError = error instanceof Error ? error.message : 'Rezept konnte nicht entfernt werden';
    return false;
  }
}

// === Recipe Images ===

/**
 * Upload and add an image to a recipe.
 * Uses the existing uploadImage() system, then stores the base URL reference.
 */
export async function addImage(
  noteId: string,
  file: File,
  caption?: string | null
): Promise<boolean> {
  saving = true;
  lastError = null;
  try {
    // Upload the file
    const uploadResult = await api.uploadImage(file);
    // Extract base URL (without query params / signature)
    const baseUrl = uploadResult.url.split('?')[0];
    // Add image reference to recipe
    await api.addRecipeImage(noteId, baseUrl, caption);
    // Reload detail to get signed URLs
    await loadRecipeDetail(noteId);
    return true;
  } catch (error) {
    console.error('Failed to add recipe image:', error);
    lastError = error instanceof Error ? error.message : 'Bild konnte nicht hinzugefügt werden';
    return false;
  } finally {
    saving = false;
  }
}

/**
 * Batch upload multiple images to a recipe.
 * Respects the 50 image limit. Uploads sequentially.
 */
export async function addImages(
  noteId: string,
  files: File[],
  onProgress?: (current: number, total: number) => void
): Promise<{ success: number; failed: number }> {
  const currentCount = currentRecipe?.images?.length ?? 0;
  const remaining = 50 - currentCount;
  const filesToUpload = files.slice(0, remaining);

  if (filesToUpload.length === 0) {
    lastError = 'Maximale Fotoanzahl erreicht';
    return { success: 0, failed: 0 };
  }

  let success = 0;
  let failed = 0;

  for (let i = 0; i < filesToUpload.length; i++) {
    onProgress?.(i + 1, filesToUpload.length);
    const ok = await addImage(noteId, filesToUpload[i]);
    if (ok) {
      success++;
    } else {
      failed++;
    }
  }

  return { success, failed };
}

/**
 * Update the caption of a recipe image.
 */
export async function updateImageCaption(
  noteId: string,
  imageId: number,
  caption: string | null
): Promise<boolean> {
  try {
    await api.updateRecipeImageCaption(noteId, imageId, caption);
    await loadRecipeDetail(noteId);
    return true;
  } catch (error) {
    console.error('Failed to update image caption:', error);
    lastError =
      error instanceof Error ? error.message : 'Bildunterschrift konnte nicht aktualisiert werden';
    return false;
  }
}

/**
 * Delete a recipe image.
 */
export async function deleteImage(noteId: string, imageId: number): Promise<boolean> {
  try {
    await api.deleteRecipeImage(noteId, imageId);
    await loadRecipeDetail(noteId);
    return true;
  } catch (error) {
    console.error('Failed to delete recipe image:', error);
    lastError = error instanceof Error ? error.message : 'Bild konnte nicht gelöscht werden';
    return false;
  }
}

/**
 * Reorder recipe images.
 */
export async function reorderImages(noteId: string, imageIds: number[]): Promise<boolean> {
  try {
    await api.reorderRecipeImages(noteId, imageIds);
    await loadRecipeDetail(noteId);
    return true;
  } catch (error) {
    console.error('Failed to reorder recipe images:', error);
    lastError = error instanceof Error ? error.message : 'Bilder konnten nicht umsortiert werden';
    return false;
  }
}

// === Remote Updates (for WebSocket) ===

/**
 * Handle remote recipe metadata update.
 */
export function handleRemoteMetadataUpdate(payload: { note_id: string }) {
  if (currentRecipe?.note.id === payload.note_id) {
    loadRecipeDetail(payload.note_id);
  }
}

/**
 * Handle remote recipe ingredients update.
 */
export function handleRemoteIngredientsUpdate(payload: { note_id: string }) {
  if (currentRecipe?.note.id === payload.note_id) {
    loadRecipeDetail(payload.note_id);
  }
}

/**
 * Reset recipe state (called on logout).
 */
export function resetRecipeState() {
  recipes = [];
  currentRecipe = null;
  collections = [];
  targetServings = 4;
  recipesLoading = false;
  recipeDetailLoading = false;
  collectionsLoading = false;
  saving = false;
  lastError = null;
  selectedCollectionId = null;
  collectionItems = [];
  collectionItemsLoading = false;
  sharedRecipes = [];
  sharedCollections = [];
  sharedCollectionItems = [];
  sharedRecipesLoading = false;
  sharedCollectionsLoading = false;
}
