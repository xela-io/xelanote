import { request } from './client';
import type { Note, QuickSearchFilters, SearchResult } from './types';

function validateSearchQuery(query: string): string {
  query = query.trim();

  if (query.length > 500) {
    throw new Error('Search query too long (max 500 characters)');
  }

  const terms = query.split(/\s+/).filter((t) => t.length > 0);
  if (terms.length > 20) {
    throw new Error('Too many search terms (max 20)');
  }

  for (const term of terms) {
    if (term.length > 100) {
      throw new Error('Search term too long (max 100 characters)');
    }
  }

  return query;
}

export async function search(query: string, limit = 20): Promise<{ results: SearchResult[] }> {
  query = validateSearchQuery(query);
  const params = new URLSearchParams({ q: query, limit: limit.toString() });
  return request(`/search?${params}`);
}

export async function quickSearch(
  query: string,
  limit = 10,
  filters?: QuickSearchFilters
): Promise<{ notes: Note[] }> {
  const params = new URLSearchParams({ q: query, limit: limit.toString() });

  // Add filter parameters if provided
  if (filters) {
    if (filters.folders && filters.folders.length > 0) {
      params.set('folders', filters.folders.join(','));
    }
    if (filters.tags && filters.tags.length > 0) {
      params.set('tags', filters.tags.join(','));
    }
    if (filters.created_after) {
      params.set('created_after', filters.created_after);
    }
    if (filters.created_before) {
      params.set('created_before', filters.created_before);
    }
    if (filters.updated_after) {
      params.set('updated_after', filters.updated_after);
    }
    if (filters.updated_before) {
      params.set('updated_before', filters.updated_before);
    }
  }

  return request(`/quick-search?${params}`);
}
