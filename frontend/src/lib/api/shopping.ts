import { AI_REQUEST_TIMEOUT_MS, request } from './client';
import type {
  ShoppingFavorite,
  ShoppingItem,
  ShoppingList,
  ShoppingListDetail,
  ShoppingListShare,
  ShoppingListSummary,
} from './types';

// --- Lists ---

export async function getShoppingLists(): Promise<ShoppingListSummary[]> {
  const result = await request<{ lists: ShoppingListSummary[] }>('/shopping/lists');
  return result.lists || [];
}

export async function createShoppingList(
  name: string,
  color?: string | null
): Promise<ShoppingList> {
  return request<ShoppingList>('/shopping/lists', {
    method: 'POST',
    body: JSON.stringify({ name, color }),
  });
}

export async function getShoppingList(listId: number): Promise<ShoppingListDetail> {
  return request<ShoppingListDetail>(`/shopping/lists/${listId}`);
}

export async function updateShoppingList(
  listId: number,
  updates: { name?: string; color?: string | null },
  expectedVersion: number
): Promise<ShoppingList> {
  return request<ShoppingList>(`/shopping/lists/${listId}`, {
    method: 'PUT',
    body: JSON.stringify({ ...updates, expected_version: expectedVersion }),
  });
}

export async function deleteShoppingList(listId: number): Promise<void> {
  await request(`/shopping/lists/${listId}`, { method: 'DELETE' });
}

export async function archiveShoppingList(listId: number, expectedVersion: number): Promise<void> {
  await request(`/shopping/lists/${listId}/archive`, {
    method: 'POST',
    body: JSON.stringify({ expected_version: expectedVersion }),
  });
}

// --- Items ---

export async function addShoppingItem(
  listId: number,
  item: Partial<ShoppingItem>
): Promise<ShoppingItem> {
  return request<ShoppingItem>(`/shopping/lists/${listId}/items`, {
    method: 'POST',
    body: JSON.stringify(item),
  });
}

export async function addShoppingItems(
  listId: number,
  items: Partial<ShoppingItem>[]
): Promise<ShoppingItem[]> {
  const result = await request<{ items: ShoppingItem[] }>(`/shopping/lists/${listId}/items/batch`, {
    method: 'POST',
    body: JSON.stringify({ items }),
  });
  return result.items || [];
}

export async function updateShoppingItem(
  listId: number,
  itemId: number,
  updates: Partial<ShoppingItem>,
  expectedVersion: number
): Promise<ShoppingItem> {
  return request<ShoppingItem>(`/shopping/lists/${listId}/items/${itemId}`, {
    method: 'PUT',
    body: JSON.stringify({ ...updates, expected_version: expectedVersion }),
  });
}

export async function deleteShoppingItem(listId: number, itemId: number): Promise<void> {
  await request(`/shopping/lists/${listId}/items/${itemId}`, { method: 'DELETE' });
}

export async function setShoppingItemChecked(
  listId: number,
  itemId: number,
  isChecked: boolean
): Promise<ShoppingItem> {
  return request<ShoppingItem>(`/shopping/lists/${listId}/items/${itemId}/checked`, {
    method: 'PUT',
    body: JSON.stringify({ is_checked: isChecked }),
  });
}

export async function setShoppingItemsChecked(
  listId: number,
  itemIds: number[],
  isChecked: boolean
): Promise<void> {
  await request(`/shopping/lists/${listId}/items/checked`, {
    method: 'PUT',
    body: JSON.stringify({ item_ids: itemIds, is_checked: isChecked }),
  });
}

export async function clearCheckedItems(listId: number): Promise<{ cleared_count: number }> {
  return request<{ cleared_count: number }>(`/shopping/lists/${listId}/items/checked`, {
    method: 'DELETE',
  });
}

export async function reorderShoppingItems(listId: number, itemIds: number[]): Promise<void> {
  await request(`/shopping/lists/${listId}/items/reorder`, {
    method: 'PUT',
    body: JSON.stringify({ item_ids: itemIds }),
  });
}

// --- Favorites ---

export async function getShoppingFavorites(): Promise<ShoppingFavorite[]> {
  const result = await request<{ favorites: ShoppingFavorite[] }>('/shopping/favorites');
  return result.favorites || [];
}

export async function addShoppingFavorite(
  name: string,
  defaultQuantity?: number | null,
  defaultUnit?: string | null,
  category?: string | null
): Promise<ShoppingFavorite> {
  return request<ShoppingFavorite>('/shopping/favorites', {
    method: 'POST',
    body: JSON.stringify({
      name,
      default_quantity: defaultQuantity,
      default_unit: defaultUnit,
      category,
    }),
  });
}

export async function removeShoppingFavorite(favoriteId: number): Promise<void> {
  await request(`/shopping/favorites/${favoriteId}`, { method: 'DELETE' });
}

// --- AI Sort ---

export async function sortShoppingList(listId: number): Promise<void> {
  await request(`/shopping/lists/${listId}/sort`, {
    method: 'POST',
    _timeout: AI_REQUEST_TIMEOUT_MS,
  });
}

// --- Recipe Import ---

export async function importRecipeToShoppingList(
  listId: number,
  recipeNoteId: string
): Promise<ShoppingItem[]> {
  const result = await request<{ items: ShoppingItem[] }>(
    `/shopping/lists/${listId}/import-recipe`,
    {
      method: 'POST',
      body: JSON.stringify({ recipe_note_id: recipeNoteId }),
    }
  );
  return result.items || [];
}

// --- Sharing ---

export async function shareShoppingList(
  listId: number,
  userId: number,
  role: 'viewer' | 'editor' = 'editor'
): Promise<ShoppingListShare> {
  return request<ShoppingListShare>(`/shopping/lists/${listId}/shares`, {
    method: 'POST',
    body: JSON.stringify({ user_id: userId, role }),
  });
}

export async function getShoppingListShares(listId: number): Promise<ShoppingListShare[]> {
  const result = await request<{ shares: ShoppingListShare[] }>(`/shopping/lists/${listId}/shares`);
  return result.shares || [];
}

export async function updateShoppingListShareRole(
  listId: number,
  userId: number,
  role: 'viewer' | 'editor'
): Promise<void> {
  await request(`/shopping/lists/${listId}/shares/${userId}`, {
    method: 'PUT',
    body: JSON.stringify({ role }),
  });
}

export async function removeShoppingListShare(listId: number, userId: number): Promise<void> {
  await request(`/shopping/lists/${listId}/shares/${userId}`, { method: 'DELETE' });
}
