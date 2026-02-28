// Shopping list store using Svelte 5 runes
import { SvelteMap } from 'svelte/reactivity';

import {
  addShoppingFavorite as apiAddFavorite,
  addShoppingItem as apiAddItem,
  addShoppingItems as apiAddItems,
  archiveShoppingList as apiArchiveList,
  clearCheckedItems as apiClearChecked,
  createShoppingList,
  deleteShoppingItem as apiDeleteItem,
  deleteShoppingList as apiDeleteList,
  getShoppingFavorites,
  getShoppingList,
  getShoppingLists,
  importRecipeToShoppingList as apiImportRecipe,
  removeShoppingFavorite as apiRemoveFavorite,
  reorderShoppingItems as apiReorderItems,
  setShoppingItemChecked as apiSetChecked,
  sortShoppingList as apiSortList,
  updateShoppingItem as apiUpdateItem,
  updateShoppingList as apiUpdateList,
} from '$lib/api/shopping';
import type {
  ShoppingFavorite,
  ShoppingItem,
  ShoppingListDetail,
  ShoppingListSummary,
} from '$lib/api/types';
import { getShoppingFeatureEnabled } from '$lib/stores/features.svelte';

// --- State ---
let lists = $state<ShoppingListSummary[]>([]);
let currentList = $state<ShoppingListDetail | null>(null);
let favorites = $state<ShoppingFavorite[]>([]);
let listsLoading = $state(false);
let listLoading = $state(false);
let saving = $state(false);
let sorting = $state(false);
let lastError = $state<string | null>(null);

// Echo-detection for WebSocket events
const pendingWrites = new SvelteMap<string, { version: number; timestamp: number }>();

// --- Getters ---
export function getLists() {
  return lists;
}

export function getCurrentList() {
  return currentList;
}

export function getFavorites() {
  return favorites;
}

export function getListsLoading() {
  return listsLoading;
}

export function getListLoading() {
  return listLoading;
}

export function getSaving() {
  return saving;
}

export function getSorting() {
  return sorting;
}

export function getLastError() {
  return lastError;
}

// --- List operations ---

export async function loadLists() {
  if (!getShoppingFeatureEnabled()) return;
  listsLoading = true;
  lastError = null;
  try {
    lists = await getShoppingLists();
  } catch (error) {
    console.error('Failed to load shopping lists:', error);
    lastError = 'Einkaufslisten konnten nicht geladen werden';
    lists = [];
  } finally {
    listsLoading = false;
  }
}

export async function loadList(listId: number) {
  if (!getShoppingFeatureEnabled()) return;
  listLoading = true;
  lastError = null;
  try {
    const detail = await getShoppingList(listId);
    // Ensure items is always an array (backend may return null for empty lists)
    if (!detail.items) detail.items = [];
    currentList = detail;
  } catch (error) {
    console.error('Failed to load shopping list:', error);
    lastError = 'Einkaufsliste konnte nicht geladen werden';
  } finally {
    listLoading = false;
  }
}

export async function addList(name: string, color?: string | null) {
  saving = true;
  lastError = null;
  try {
    const newList = await createShoppingList(name, color);
    lists = [
      ...lists,
      {
        ...newList,
        item_count: 0,
        checked_count: 0,
        role: 'owner' as const,
      },
    ];
    return newList;
  } catch (error: unknown) {
    console.error('Failed to create shopping list:', error);
    lastError = 'Liste konnte nicht erstellt werden';
    throw error;
  } finally {
    saving = false;
  }
}

export async function updateList(
  listId: number,
  updates: { name?: string; color?: string | null }
) {
  if (!currentList) return;
  saving = true;
  lastError = null;
  try {
    const updated = await apiUpdateList(listId, updates, currentList.version);
    currentList = { ...currentList, ...updated };
    trackPendingWrite(`list:${listId}`, updated.version);
    // Update in lists array too
    lists = lists.map((l) => (l.id === listId ? { ...l, ...updated } : l));
  } catch (error: unknown) {
    if (isConflictError(error)) {
      await loadList(listId);
      lastError = 'Liste wurde zwischenzeitlich geändert — neu geladen';
    } else {
      lastError = 'Liste konnte nicht aktualisiert werden';
    }
    throw error;
  } finally {
    saving = false;
  }
}

export async function deleteList(listId: number) {
  saving = true;
  lastError = null;
  try {
    await apiDeleteList(listId);
    lists = lists.filter((l) => l.id !== listId);
    if (currentList?.id === listId) currentList = null;
  } catch (error) {
    console.error('Failed to delete shopping list:', error);
    lastError = 'Liste konnte nicht gelöscht werden';
    throw error;
  } finally {
    saving = false;
  }
}

export async function archiveList(listId: number) {
  if (!currentList) return;
  saving = true;
  lastError = null;
  try {
    await apiArchiveList(listId, currentList.version);
    lists = lists.filter((l) => l.id !== listId);
    if (currentList?.id === listId) currentList = null;
  } catch (error) {
    console.error('Failed to archive shopping list:', error);
    lastError = 'Liste konnte nicht archiviert werden';
    throw error;
  } finally {
    saving = false;
  }
}

// --- Item operations ---

export async function addItem(listId: number, item: Partial<ShoppingItem>) {
  saving = true;
  lastError = null;
  try {
    const newItem = await apiAddItem(listId, item);
    if (currentList?.id === listId) {
      currentList = {
        ...currentList,
        items: [...(currentList.items ?? []), newItem],
        item_count: currentList.item_count + 1,
      };
    }
    trackPendingWrite(`item:${newItem.id}`, newItem.version);
    return newItem;
  } catch (error) {
    console.error('Failed to add shopping item:', error);
    lastError = 'Artikel konnte nicht hinzugefügt werden';
    throw error;
  } finally {
    saving = false;
  }
}

export async function addItems(listId: number, items: Partial<ShoppingItem>[]) {
  saving = true;
  lastError = null;
  try {
    const newItems = await apiAddItems(listId, items);
    if (currentList?.id === listId) {
      currentList = {
        ...currentList,
        items: [...(currentList.items ?? []), ...newItems],
        item_count: currentList.item_count + newItems.length,
      };
    }
    for (const item of newItems) {
      trackPendingWrite(`item:${item.id}`, item.version);
    }
    return newItems;
  } catch (error) {
    console.error('Failed to add shopping items:', error);
    lastError = 'Artikel konnten nicht hinzugefügt werden';
    throw error;
  } finally {
    saving = false;
  }
}

export async function updateItem(listId: number, itemId: number, updates: Partial<ShoppingItem>) {
  if (!currentList) return;
  const item = currentList.items.find((i) => i.id === itemId);
  if (!item) return;

  saving = true;
  lastError = null;
  try {
    const updated = await apiUpdateItem(listId, itemId, updates, item.version);
    if (currentList?.id === listId) {
      currentList = {
        ...currentList,
        items: currentList.items.map((i) => (i.id === itemId ? updated : i)),
      };
    }
    trackPendingWrite(`item:${itemId}`, updated.version);
    return updated;
  } catch (error: unknown) {
    if (isConflictError(error)) {
      await loadList(listId);
      lastError = 'Artikel wurde zwischenzeitlich geändert — neu geladen';
    } else {
      lastError = 'Artikel konnte nicht aktualisiert werden';
    }
    throw error;
  } finally {
    saving = false;
  }
}

export async function setItemChecked(listId: number, itemId: number, isChecked: boolean) {
  lastError = null;
  try {
    const updated = await apiSetChecked(listId, itemId, isChecked);
    if (currentList?.id === listId) {
      currentList = {
        ...currentList,
        items: currentList.items.map((i) => (i.id === itemId ? updated : i)),
      };
    }
    return updated;
  } catch (error) {
    console.error('Failed to check shopping item:', error);
    lastError = 'Artikel konnte nicht aktualisiert werden';
    throw error;
  }
}

export async function deleteItem(listId: number, itemId: number) {
  lastError = null;
  try {
    await apiDeleteItem(listId, itemId);
    if (currentList?.id === listId) {
      currentList = {
        ...currentList,
        items: currentList.items.filter((i) => i.id !== itemId),
        item_count: currentList.item_count - 1,
      };
    }
  } catch (error) {
    console.error('Failed to delete shopping item:', error);
    lastError = 'Artikel konnte nicht gelöscht werden';
    throw error;
  }
}

export async function clearChecked(listId: number) {
  lastError = null;
  try {
    const { cleared_count } = await apiClearChecked(listId);
    if (currentList?.id === listId) {
      currentList = {
        ...currentList,
        items: currentList.items.filter((i) => !i.is_checked),
        item_count: currentList.item_count - cleared_count,
      };
    }
    return cleared_count;
  } catch (error) {
    console.error('Failed to clear checked items:', error);
    lastError = 'Erledigte Artikel konnten nicht gelöscht werden';
    throw error;
  }
}

export async function reorderItems(listId: number, itemIds: number[]) {
  try {
    await apiReorderItems(listId, itemIds);
  } catch (error) {
    console.error('Failed to reorder items:', error);
  }
}

// --- AI Sort ---

export async function sortByCategory(listId: number) {
  sorting = true;
  lastError = null;
  try {
    await apiSortList(listId);
    // Refetch the list to get sorted items
    await loadList(listId);
  } catch (error) {
    console.error('Failed to sort shopping list:', error);
    lastError = 'KI-Sortierung fehlgeschlagen';
    throw error;
  } finally {
    sorting = false;
  }
}

// --- Recipe Import ---

export async function importRecipe(listId: number, recipeNoteId: string) {
  saving = true;
  lastError = null;
  try {
    const items = await apiImportRecipe(listId, recipeNoteId);
    if (currentList?.id === listId) {
      currentList = {
        ...currentList,
        items: [...(currentList.items ?? []), ...items],
        item_count: currentList.item_count + items.length,
      };
    }
    return items;
  } catch (error) {
    console.error('Failed to import recipe:', error);
    lastError = 'Rezept-Import fehlgeschlagen';
    throw error;
  } finally {
    saving = false;
  }
}

// --- Favorites ---

export async function loadFavorites() {
  try {
    favorites = await getShoppingFavorites();
  } catch (error) {
    console.error('Failed to load shopping favorites:', error);
  }
}

export async function addFavorite(
  name: string,
  quantity?: number | null,
  unit?: string | null,
  category?: string | null
) {
  try {
    const fav = await apiAddFavorite(name, quantity, unit, category);
    favorites = [...favorites, fav];
    return fav;
  } catch (error) {
    console.error('Failed to add favorite:', error);
    throw error;
  }
}

export async function removeFavorite(favoriteId: number) {
  try {
    await apiRemoveFavorite(favoriteId);
    favorites = favorites.filter((f) => f.id !== favoriteId);
  } catch (error) {
    console.error('Failed to remove favorite:', error);
    throw error;
  }
}

// --- WebSocket handlers ---

export function handleRemoteItemAdded(payload: { list_id: number; item: ShoppingItem }) {
  if (isEcho(`item:${payload.item.id}`, payload.item.version)) return;
  if (currentList?.id === payload.list_id) {
    const items = currentList.items ?? [];
    const exists = items.some((i) => i.id === payload.item.id);
    if (!exists) {
      currentList = {
        ...currentList,
        items: [...items, payload.item],
        item_count: currentList.item_count + 1,
      };
    }
  }
}

export function handleRemoteItemUpdated(payload: { list_id: number; item: ShoppingItem }) {
  if (isEcho(`item:${payload.item.id}`, payload.item.version)) return;
  if (currentList?.id === payload.list_id) {
    currentList = {
      ...currentList,
      items: currentList.items.map((i) => (i.id === payload.item.id ? payload.item : i)),
    };
  }
}

export function handleRemoteItemChecked(payload: {
  list_id: number;
  item_id: number;
  is_checked: boolean;
  checked_at: string | null;
}) {
  if (currentList?.id === payload.list_id) {
    currentList = {
      ...currentList,
      items: currentList.items.map((i) =>
        i.id === payload.item_id
          ? { ...i, is_checked: payload.is_checked, checked_at: payload.checked_at }
          : i
      ),
    };
  }
}

export function handleRemoteItemRemoved(payload: { list_id: number; item_id: number }) {
  if (currentList?.id === payload.list_id) {
    currentList = {
      ...currentList,
      items: currentList.items.filter((i) => i.id !== payload.item_id),
      item_count: currentList.item_count - 1,
    };
  }
}

export function handleRemoteItemsCleared(payload: { list_id: number; cleared_count: number }) {
  if (currentList?.id === payload.list_id) {
    currentList = {
      ...currentList,
      items: currentList.items.filter((i) => !i.is_checked),
      item_count: currentList.item_count - payload.cleared_count,
    };
  }
}

export function handleRemoteItemsSorted(payload: { list_id: number }) {
  if (currentList?.id === payload.list_id) {
    loadList(payload.list_id);
  }
}

export function handleRemoteListUpdated(payload: { list_id: number; list: ShoppingListSummary }) {
  if (isEcho(`list:${payload.list_id}`, payload.list.version)) return;
  lists = lists.map((l) => (l.id === payload.list_id ? { ...l, ...payload.list } : l));
  if (currentList?.id === payload.list_id) {
    currentList = { ...currentList, ...payload.list };
  }
}

// --- Echo detection ---

function trackPendingWrite(key: string, version: number) {
  pendingWrites.set(key, { version, timestamp: Date.now() });
}

function isEcho(key: string, eventVersion: number): boolean {
  const pending = pendingWrites.get(key);
  if (!pending) return false;
  if (pending.version === eventVersion) {
    pendingWrites.delete(key);
    return true;
  }
  if (Date.now() - pending.timestamp > 5000) {
    pendingWrites.delete(key);
  }
  return false;
}

// --- Reset ---

export function resetShoppingState() {
  lists = [];
  currentList = null;
  favorites = [];
  listsLoading = false;
  listLoading = false;
  saving = false;
  sorting = false;
  lastError = null;
  pendingWrites.clear();
}

// --- Helpers ---

function isConflictError(error: unknown): boolean {
  return (
    error instanceof Error && (error.message.includes('409') || error.message.includes('conflict'))
  );
}
