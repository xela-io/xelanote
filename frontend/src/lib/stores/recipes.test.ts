import { beforeEach, describe, expect, it, vi } from 'vitest';

import type {
  CollectionShare,
  Note,
  RecipeCollection,
  RecipeDetail,
  RecipeIngredient,
  RecipeListItem,
  RecipeMetadata,
  SharedCollection,
  SharedNote,
} from '$lib/api';

const getRecipes = vi.fn();
const getRecipeDetail = vi.fn();
const updateRecipeMetadata = vi.fn();
const setRecipeIngredients = vi.fn();
const getRecipeCollections = vi.fn();
const createRecipeCollection = vi.fn();
const updateRecipeCollection = vi.fn();
const deleteRecipeCollection = vi.fn();
const getCollectionItems = vi.fn();
const addRecipeToCollection = vi.fn();
const removeRecipeFromCollection = vi.fn();
const getSharedRecipes = vi.fn();
const getSharedCollectionsMock = vi.fn();
const getSharedCollectionItems = vi.fn();
const shareCollection = vi.fn();
const removeCollectionShare = vi.fn();
const updateCollectionShareRole = vi.fn();
const getCollectionShares = vi.fn();
const addToSharedCollection = vi.fn();
const removeFromSharedCollection = vi.fn();
const uploadImage = vi.fn();
const addRecipeImage = vi.fn();
const updateRecipeImageCaption = vi.fn();
const deleteRecipeImage = vi.fn();
const reorderRecipeImages = vi.fn();

vi.mock('$lib/api', () => ({
  ApiError: class ApiError extends Error {
    status: number;
    constructor(status = 500, msg = 'api error') {
      super(msg);
      this.status = status;
    }
  },
  getRecipes,
  getRecipeDetail,
  updateRecipeMetadata,
  setRecipeIngredients,
  getRecipeCollections,
  createRecipeCollection,
  updateRecipeCollection,
  deleteRecipeCollection,
  getCollectionItems,
  addRecipeToCollection,
  removeRecipeFromCollection,
  getSharedRecipes,
  getSharedCollections: getSharedCollectionsMock,
  getSharedCollectionItems,
  shareCollection,
  removeCollectionShare,
  updateCollectionShareRole,
  getCollectionShares,
  addToSharedCollection,
  removeFromSharedCollection,
  uploadImage,
  addRecipeImage,
  updateRecipeImageCaption,
  deleteRecipeImage,
  reorderRecipeImages,
}));

vi.mock('$app/navigation', () => ({
  goto: vi.fn(),
}));

const getRecipeFeatureEnabled = vi.fn().mockReturnValue(true);
vi.mock('./features.svelte', () => ({
  getRecipeFeatureEnabled,
}));

const createNote = vi.fn();
vi.mock('./notes.svelte', () => ({
  createNote,
}));

const loadTree = vi.fn();
vi.mock('./tree.svelte', () => ({
  loadTree,
}));

const mockRecipeItem = (id: string): RecipeListItem =>
  ({
    id,
    title: `Recipe ${id}`,
    note_type: 'recipe',
  }) as RecipeListItem;

const mockMetadata = (overrides: Partial<RecipeMetadata> = {}): RecipeMetadata =>
  ({
    servings: 4,
    prep_time_minutes: 15,
    cook_time_minutes: 30,
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }) as RecipeMetadata;

const mockRecipeDetail = (noteId: string, overrides: Partial<RecipeDetail> = {}): RecipeDetail =>
  ({
    note: { id: noteId, title: `Recipe ${noteId}` } as Note,
    metadata: mockMetadata(),
    ingredients: [],
    images: [],
    collections: [],
    ...overrides,
  }) as RecipeDetail;

const mockCollection = (id: number, name: string): RecipeCollection =>
  ({
    id,
    name,
    recipe_count: 0,
  }) as RecipeCollection;

describe('recipes store', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    getRecipeFeatureEnabled.mockReturnValue(true);
  });

  it('should start with empty state', async () => {
    const store = await import('$lib/stores/recipes.svelte');
    expect(store.getRecipes()).toEqual([]);
    expect(store.getCurrentRecipe()).toBeNull();
    expect(store.getCollections()).toEqual([]);
    expect(store.getTargetServings()).toBe(4);
    expect(store.getRecipesLoading()).toBe(false);
    expect(store.getSaving()).toBe(false);
    expect(store.getLastError()).toBeNull();
  });

  // === Pure Functions ===

  describe('formatDisplayAmount', () => {
    it('should format integers without decimals', async () => {
      const store = await import('$lib/stores/recipes.svelte');
      expect(store.formatDisplayAmount(3)).toBe('3');
      expect(store.formatDisplayAmount(100)).toBe('100');
    });

    it('should format single decimal', async () => {
      const store = await import('$lib/stores/recipes.svelte');
      expect(store.formatDisplayAmount(1.5)).toBe('1.5');
      expect(store.formatDisplayAmount(0.5)).toBe('0.5');
    });

    it('should format two decimals for finer amounts', async () => {
      const store = await import('$lib/stores/recipes.svelte');
      expect(store.formatDisplayAmount(1.25)).toBe('1.25');
      expect(store.formatDisplayAmount(0.33)).toBe('0.33');
    });
  });

  describe('formatAmount', () => {
    it('should return empty string for null amount', async () => {
      const store = await import('$lib/stores/recipes.svelte');
      expect(store.formatAmount(null, null)).toBe('');
      expect(store.formatAmount(undefined, null)).toBe('');
    });

    it('should use amount_text when provided', async () => {
      const store = await import('$lib/stores/recipes.svelte');
      expect(store.formatAmount(2, '2-3')).toBe('2-3');
    });

    it('should format numeric amount when no amount_text', async () => {
      const store = await import('$lib/stores/recipes.svelte');
      expect(store.formatAmount(2.5, null)).toBe('2.5');
      expect(store.formatAmount(2.5, undefined)).toBe('2.5');
    });
  });

  describe('scaleIngredients', () => {
    it('should scale scalable ingredients', async () => {
      const store = await import('$lib/stores/recipes.svelte');
      const ingredients: RecipeIngredient[] = [
        { name: 'Flour', amount: 200, unit: 'g', scalable: true } as RecipeIngredient,
        { name: 'Salt', amount: 1, unit: 'tsp', scalable: true } as RecipeIngredient,
      ];

      const result = store.scaleIngredients(ingredients, 4, 8);
      expect(result[0].scaled_amount).toBe(400);
      expect(result[0].display_amount).toBe('400');
      expect(result[1].scaled_amount).toBe(2);
    });

    it('should not scale non-scalable ingredients', async () => {
      const store = await import('$lib/stores/recipes.svelte');
      const ingredients: RecipeIngredient[] = [
        { name: 'Water', amount: 500, unit: 'ml', scalable: false } as RecipeIngredient,
      ];

      const result = store.scaleIngredients(ingredients, 4, 8);
      expect(result[0].scaled_amount).toBe(500); // unchanged
    });

    it('should handle null amount', async () => {
      const store = await import('$lib/stores/recipes.svelte');
      const ingredients: RecipeIngredient[] = [
        { name: 'Salt', amount: null, unit: null, scalable: true } as RecipeIngredient,
      ];

      const result = store.scaleIngredients(ingredients, 4, 8);
      expect(result[0].scaled_amount).toBeNull();
    });

    it('should round to 2 decimal places', async () => {
      const store = await import('$lib/stores/recipes.svelte');
      const ingredients: RecipeIngredient[] = [
        { name: 'Butter', amount: 100, unit: 'g', scalable: true } as RecipeIngredient,
      ];

      const result = store.scaleIngredients(ingredients, 3, 2);
      // 100 * (2/3) = 66.666... → rounded to 66.67
      expect(result[0].scaled_amount).toBe(66.67);
    });
  });

  describe('setTargetServings', () => {
    it('should set valid servings', async () => {
      const store = await import('$lib/stores/recipes.svelte');
      store.setTargetServings(8);
      expect(store.getTargetServings()).toBe(8);
    });

    it('should reject servings < 1', async () => {
      const store = await import('$lib/stores/recipes.svelte');
      store.setTargetServings(0);
      expect(store.getTargetServings()).toBe(4); // default unchanged
    });

    it('should reject servings > 999', async () => {
      const store = await import('$lib/stores/recipes.svelte');
      store.setTargetServings(1000);
      expect(store.getTargetServings()).toBe(4);
    });
  });

  // === CRUD ===

  describe('loadRecipes', () => {
    it('should load recipes', async () => {
      const recipes = [mockRecipeItem('1'), mockRecipeItem('2')];
      getRecipes.mockResolvedValue(recipes);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadRecipes();

      expect(store.getRecipes()).toEqual(recipes);
      expect(store.getRecipesLoading()).toBe(false);
    });

    it('should set error and empty array on failure', async () => {
      getRecipes.mockRejectedValue(new Error('fail'));

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadRecipes();

      expect(store.getRecipes()).toEqual([]);
      expect(store.getLastError()).toBe('Rezepte konnten nicht geladen werden');
    });

    it('should skip when feature disabled', async () => {
      getRecipeFeatureEnabled.mockReturnValue(false);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadRecipes();

      expect(getRecipes).not.toHaveBeenCalled();
    });
  });

  describe('loadRecipeDetail', () => {
    it('should load recipe detail and set target servings', async () => {
      const detail = mockRecipeDetail('note-1', {
        metadata: mockMetadata({ servings: 6 }),
      });
      getRecipeDetail.mockResolvedValue(detail);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadRecipeDetail('note-1');

      expect(store.getCurrentRecipe()).toEqual(detail);
      expect(store.getTargetServings()).toBe(6);
      expect(store.getRecipeDetailLoading()).toBe(false);
    });

    it('should handle 404 error with specific message', async () => {
      const { ApiError } = await import('$lib/api');
      getRecipeDetail.mockRejectedValue(new ApiError(404, 'not found'));

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadRecipeDetail('missing');

      expect(store.getLastError()).toBe('Rezept nicht gefunden');
      expect(store.getCurrentRecipe()).toBeNull();
    });

    it('should handle 403 error with specific message', async () => {
      const { ApiError } = await import('$lib/api');
      getRecipeDetail.mockRejectedValue(new ApiError(403, 'forbidden'));

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadRecipeDetail('secret');

      expect(store.getLastError()).toBe('Kein Zugriff auf dieses Rezept');
    });
  });

  describe('clearCurrentRecipe', () => {
    it('should clear current recipe and error', async () => {
      const detail = mockRecipeDetail('note-1');
      getRecipeDetail.mockResolvedValue(detail);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadRecipeDetail('note-1');
      expect(store.getCurrentRecipe()).not.toBeNull();

      store.clearCurrentRecipe();
      expect(store.getCurrentRecipe()).toBeNull();
      expect(store.getLastError()).toBeNull();
    });
  });

  describe('createRecipe', () => {
    it('should create recipe, navigate, and reload tree', async () => {
      createNote.mockResolvedValue({ id: 'new-recipe' });
      loadTree.mockResolvedValue(undefined);

      const { goto } = await import('$app/navigation');
      const store = await import('$lib/stores/recipes.svelte');
      const id = await store.createRecipe('My Recipe', 'content');

      expect(id).toBe('new-recipe');
      expect(createNote).toHaveBeenCalledWith('My Recipe', 'content', '/Rezepte', {
        note_type: 'recipe',
      });
      expect(goto).toHaveBeenCalledWith('/note/new-recipe');
      expect(loadTree).toHaveBeenCalled();
    });

    it('should return null when feature disabled', async () => {
      getRecipeFeatureEnabled.mockReturnValue(false);

      const store = await import('$lib/stores/recipes.svelte');
      const id = await store.createRecipe('Recipe');

      expect(id).toBeNull();
      expect(store.getLastError()).toBe('Rezept-Feature ist nicht aktiviert');
    });

    it('should set error on failure', async () => {
      createNote.mockRejectedValue(new Error('boom'));

      const store = await import('$lib/stores/recipes.svelte');
      const id = await store.createRecipe('Recipe');

      expect(id).toBeNull();
      expect(store.getLastError()).toBe('boom');
    });
  });

  describe('updateMetadata', () => {
    it('should update metadata and local state', async () => {
      const detail = mockRecipeDetail('note-1');
      getRecipeDetail.mockResolvedValue(detail);

      const updatedMeta = mockMetadata({ servings: 6 });
      updateRecipeMetadata.mockResolvedValue(updatedMeta);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadRecipeDetail('note-1');

      const result = await store.updateMetadata('note-1', { servings: 6 });
      expect(result).toEqual(updatedMeta);
      expect(store.getTargetServings()).toBe(6);
      expect(store.getSaving()).toBe(false);
    });

    it('should return null when no metadata present', async () => {
      const store = await import('$lib/stores/recipes.svelte');
      // No recipe loaded
      const result = await store.updateMetadata('note-1', { servings: 4 });
      expect(result).toBeNull();
    });

    it('should handle 409 conflict', async () => {
      const detail = mockRecipeDetail('note-1');
      getRecipeDetail.mockResolvedValue(detail);

      const { ApiError } = await import('$lib/api');
      updateRecipeMetadata.mockRejectedValue(new ApiError(409, 'conflict'));

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadRecipeDetail('note-1');

      const result = await store.updateMetadata('note-1', { servings: 4 });
      expect(result).toBeNull();
      expect(store.getLastError()).toBe('Rezept wurde zwischenzeitlich geändert. Bitte neu laden.');
    });
  });

  describe('updateIngredients', () => {
    it('should update ingredients and refresh detail', async () => {
      const detail = mockRecipeDetail('note-1');
      getRecipeDetail.mockResolvedValue(detail);
      setRecipeIngredients.mockResolvedValue(undefined);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadRecipeDetail('note-1');

      const ingredients: RecipeIngredient[] = [
        { name: 'Flour', amount: 200, unit: 'g', scalable: true } as RecipeIngredient,
      ];
      const ok = await store.updateIngredients('note-1', ingredients);

      expect(ok).toBe(true);
      expect(setRecipeIngredients).toHaveBeenCalled();
      // Should also re-fetch detail
      expect(getRecipeDetail).toHaveBeenCalledTimes(2);
    });

    it('should return false when no metadata present', async () => {
      const store = await import('$lib/stores/recipes.svelte');
      const ok = await store.updateIngredients('note-1', []);
      expect(ok).toBe(false);
    });
  });

  // === Collections ===

  describe('loadCollections', () => {
    it('should load collections', async () => {
      const colls = [mockCollection(1, 'Italian'), mockCollection(2, 'Asian')];
      getRecipeCollections.mockResolvedValue(colls);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadCollections();

      expect(store.getCollections()).toEqual(colls);
    });

    it('should set empty on error', async () => {
      getRecipeCollections.mockRejectedValue(new Error('fail'));

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadCollections();

      expect(store.getCollections()).toEqual([]);
    });
  });

  describe('createCollection', () => {
    it('should create and add to local list', async () => {
      const newColl = mockCollection(1, 'Italian');
      createRecipeCollection.mockResolvedValue(newColl);

      const store = await import('$lib/stores/recipes.svelte');
      const result = await store.createCollection('Italian', 'desc', '#red');

      expect(result).toEqual(newColl);
      expect(store.getCollections()).toContainEqual(newColl);
    });

    it('should return null on error', async () => {
      createRecipeCollection.mockRejectedValue(new Error('fail'));

      const store = await import('$lib/stores/recipes.svelte');
      const result = await store.createCollection('Bad');

      expect(result).toBeNull();
    });
  });

  describe('updateCollection', () => {
    it('should update collection in local list', async () => {
      const colls = [mockCollection(1, 'Italian')];
      getRecipeCollections.mockResolvedValue(colls);
      updateRecipeCollection.mockResolvedValue(undefined);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadCollections();

      const ok = await store.updateCollection(1, 'Mediterranean', 'updated', null);
      expect(ok).toBe(true);
      expect(store.getCollections()[0].name).toBe('Mediterranean');
    });
  });

  describe('deleteCollection', () => {
    it('should remove collection from local list', async () => {
      const colls = [mockCollection(1, 'Italian'), mockCollection(2, 'Asian')];
      getRecipeCollections.mockResolvedValue(colls);
      deleteRecipeCollection.mockResolvedValue(undefined);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadCollections();

      const ok = await store.deleteCollection(1);
      expect(ok).toBe(true);
      expect(store.getCollections()).toHaveLength(1);
      expect(store.getCollections()[0].id).toBe(2);
    });
  });

  describe('selectCollection / clearCollectionFilter', () => {
    it('should load collection items', async () => {
      const items = [mockRecipeItem('1')];
      getCollectionItems.mockResolvedValue(items);

      const store = await import('$lib/stores/recipes.svelte');
      await store.selectCollection(5);

      expect(store.getSelectedCollectionId()).toBe(5);
      expect(store.getCollectionItems()).toEqual(items);
    });

    it('should clear filter', async () => {
      getCollectionItems.mockResolvedValue([mockRecipeItem('1')]);

      const store = await import('$lib/stores/recipes.svelte');
      await store.selectCollection(5);
      store.clearCollectionFilter();

      expect(store.getSelectedCollectionId()).toBeNull();
      expect(store.getCollectionItems()).toEqual([]);
    });
  });

  describe('addToCollection / removeFromCollection', () => {
    it('should increment recipe count on add', async () => {
      const colls = [mockCollection(1, 'Italian')];
      getRecipeCollections.mockResolvedValue(colls);
      addRecipeToCollection.mockResolvedValue(undefined);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadCollections();

      const ok = await store.addToCollection(1, 'note-1');
      expect(ok).toBe(true);
      expect(store.getCollections()[0].recipe_count).toBe(1);
    });

    it('should decrement recipe count on remove', async () => {
      const colls = [{ ...mockCollection(1, 'Italian'), recipe_count: 3 }];
      getRecipeCollections.mockResolvedValue(colls);
      removeRecipeFromCollection.mockResolvedValue(undefined);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadCollections();

      const ok = await store.removeFromCollection(1, 'note-1');
      expect(ok).toBe(true);
      expect(store.getCollections()[0].recipe_count).toBe(2);
    });
  });

  // === Shared ===

  describe('loadSharedRecipes', () => {
    it('should load shared recipes', async () => {
      const shared = [{ id: '1', title: 'Shared' }] as SharedNote[];
      getSharedRecipes.mockResolvedValue(shared);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadSharedRecipes();

      expect(store.getSharedRecipes()).toEqual(shared);
      expect(store.getSharedRecipesLoading()).toBe(false);
    });

    it('should set empty on error', async () => {
      getSharedRecipes.mockRejectedValue(new Error('fail'));

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadSharedRecipes();

      expect(store.getSharedRecipes()).toEqual([]);
    });
  });

  describe('loadSharedCollections', () => {
    it('should load shared collections', async () => {
      const shared = [{ id: 1, name: 'Shared' }] as SharedCollection[];
      getSharedCollectionsMock.mockResolvedValue(shared);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadSharedCollections();

      expect(store.getSharedCollections()).toEqual(shared);
    });
  });

  describe('shareCollectionWithUser', () => {
    it('should share and return share object', async () => {
      const share = { id: 1, user_id: 2, role: 'viewer' } as CollectionShare;
      shareCollection.mockResolvedValue(share);

      const store = await import('$lib/stores/recipes.svelte');
      const result = await store.shareCollectionWithUser(1, 'user@test.com', 'viewer');

      expect(result).toEqual(share);
      expect(shareCollection).toHaveBeenCalledWith(1, 'user@test.com', 'viewer');
    });

    it('should return null on error', async () => {
      shareCollection.mockRejectedValue(new Error('fail'));

      const store = await import('$lib/stores/recipes.svelte');
      const result = await store.shareCollectionWithUser(1, 'user', 'viewer');

      expect(result).toBeNull();
    });
  });

  describe('unshareCollection', () => {
    it('should unshare and return true', async () => {
      removeCollectionShare.mockResolvedValue(undefined);

      const store = await import('$lib/stores/recipes.svelte');
      const ok = await store.unshareCollection(1, 2);

      expect(ok).toBe(true);
    });
  });

  describe('updateShareRole', () => {
    it('should update role and return true', async () => {
      updateCollectionShareRole.mockResolvedValue(undefined);

      const store = await import('$lib/stores/recipes.svelte');
      const ok = await store.updateShareRole(1, 2, 'editor');

      expect(ok).toBe(true);
    });
  });

  describe('getCollectionSharesList', () => {
    it('should return shares array', async () => {
      const shares = [{ id: 1, user_id: 2, role: 'viewer' }] as CollectionShare[];
      getCollectionShares.mockResolvedValue(shares);

      const store = await import('$lib/stores/recipes.svelte');
      const result = await store.getCollectionSharesList(1);

      expect(result).toEqual(shares);
    });

    it('should return empty array on error', async () => {
      getCollectionShares.mockRejectedValue(new Error('fail'));

      const store = await import('$lib/stores/recipes.svelte');
      const result = await store.getCollectionSharesList(1);

      expect(result).toEqual([]);
    });
  });

  // === Images ===

  describe('addImage', () => {
    it('should upload image and add to recipe', async () => {
      const detail = mockRecipeDetail('note-1');
      getRecipeDetail.mockResolvedValue(detail);
      uploadImage.mockResolvedValue({ url: 'https://cdn.example.com/img.jpg?sig=abc' });
      addRecipeImage.mockResolvedValue(undefined);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadRecipeDetail('note-1');

      const file = new File([''], 'photo.jpg', { type: 'image/jpeg' });
      const ok = await store.addImage('note-1', file, 'My photo');

      expect(ok).toBe(true);
      expect(uploadImage).toHaveBeenCalledWith(file);
      // Should strip query params from URL
      expect(addRecipeImage).toHaveBeenCalledWith(
        'note-1',
        'https://cdn.example.com/img.jpg',
        'My photo'
      );
    });
  });

  describe('deleteImage', () => {
    it('should delete image and reload detail', async () => {
      const detail = mockRecipeDetail('note-1');
      getRecipeDetail.mockResolvedValue(detail);
      deleteRecipeImage.mockResolvedValue(undefined);

      const store = await import('$lib/stores/recipes.svelte');
      const ok = await store.deleteImage('note-1', 42);

      expect(ok).toBe(true);
      expect(deleteRecipeImage).toHaveBeenCalledWith('note-1', 42);
    });
  });

  describe('reorderImages', () => {
    it('should reorder images and reload', async () => {
      const detail = mockRecipeDetail('note-1');
      getRecipeDetail.mockResolvedValue(detail);
      reorderRecipeImages.mockResolvedValue(undefined);

      const store = await import('$lib/stores/recipes.svelte');
      const ok = await store.reorderImages('note-1', [3, 1, 2]);

      expect(ok).toBe(true);
      expect(reorderRecipeImages).toHaveBeenCalledWith('note-1', [3, 1, 2]);
    });
  });

  // === WebSocket handlers ===

  describe('handleRemoteMetadataUpdate', () => {
    it('should reload detail if viewing the same recipe', async () => {
      const detail = mockRecipeDetail('note-1');
      getRecipeDetail.mockResolvedValue(detail);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadRecipeDetail('note-1');
      vi.clearAllMocks();

      getRecipeDetail.mockResolvedValue(detail);
      store.handleRemoteMetadataUpdate({ note_id: 'note-1' });

      expect(getRecipeDetail).toHaveBeenCalledWith('note-1');
    });

    it('should not reload for different recipe', async () => {
      const detail = mockRecipeDetail('note-1');
      getRecipeDetail.mockResolvedValue(detail);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadRecipeDetail('note-1');
      vi.clearAllMocks();

      store.handleRemoteMetadataUpdate({ note_id: 'note-other' });
      expect(getRecipeDetail).not.toHaveBeenCalled();
    });
  });

  // === Reset ===

  describe('resetRecipeState', () => {
    it('should reset all state to defaults', async () => {
      const recipes = [mockRecipeItem('1')];
      getRecipes.mockResolvedValue(recipes);

      const store = await import('$lib/stores/recipes.svelte');
      await store.loadRecipes();
      expect(store.getRecipes().length).toBe(1);

      store.resetRecipeState();

      expect(store.getRecipes()).toEqual([]);
      expect(store.getCurrentRecipe()).toBeNull();
      expect(store.getCollections()).toEqual([]);
      expect(store.getTargetServings()).toBe(4);
      expect(store.getRecipesLoading()).toBe(false);
      expect(store.getSaving()).toBe(false);
      expect(store.getLastError()).toBeNull();
      expect(store.getSelectedCollectionId()).toBeNull();
      expect(store.getSharedRecipes()).toEqual([]);
      expect(store.getSharedCollections()).toEqual([]);
    });
  });
});
