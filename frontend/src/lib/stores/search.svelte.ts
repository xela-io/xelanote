// Search filter store using Svelte 5 runes

import { SvelteDate } from 'svelte/reactivity';

export type DateFilter = {
  type: 'absolute' | 'relative';
  // Absolute
  after?: string; // ISO8601 string
  before?: string; // ISO8601 string
  // Relative
  preset?: 'today' | 'yesterday' | 'last7days' | 'last30days' | 'thisWeek' | 'thisMonth';
};

export type SearchFilters = {
  query: string;
  folders: string[];
  tags: string[];
  createdDate?: DateFilter;
  updatedDate?: DateFilter;
};

// State
let filters = $state<SearchFilters>({
  query: '',
  folders: [],
  tags: [],
  createdDate: undefined,
  updatedDate: undefined,
});

// Export functions to access and modify state

export function getFilters() {
  return filters;
}

export function getQuery() {
  return filters.query;
}

export function setQuery(query: string) {
  filters.query = query;
}

export function getFolders() {
  return filters.folders;
}

export function getTags() {
  return filters.tags;
}

export function getCreatedDate() {
  return filters.createdDate;
}

export function getUpdatedDate() {
  return filters.updatedDate;
}

export function addFolderFilter(folderPath: string) {
  if (!filters.folders.includes(folderPath)) {
    filters.folders = [...filters.folders, folderPath];
  }
}

export function removeFolderFilter(folderPath: string) {
  filters.folders = filters.folders.filter((f) => f !== folderPath);
}

export function addTagFilter(tag: string) {
  if (!filters.tags.includes(tag)) {
    filters.tags = [...filters.tags, tag];
  }
}

export function removeTagFilter(tag: string) {
  filters.tags = filters.tags.filter((t) => t !== tag);
}

export function setCreatedDateFilter(dateFilter: DateFilter | undefined) {
  filters.createdDate = dateFilter;
}

export function setUpdatedDateFilter(dateFilter: DateFilter | undefined) {
  filters.updatedDate = dateFilter;
}

export function clearAllFilters() {
  filters = {
    query: filters.query, // Keep query
    folders: [],
    tags: [],
    createdDate: undefined,
    updatedDate: undefined,
  };
}

export function clearAll() {
  filters = {
    query: '',
    folders: [],
    tags: [],
    createdDate: undefined,
    updatedDate: undefined,
  };
}

export function hasActiveFilters(): boolean {
  return (
    filters.folders.length > 0 ||
    filters.tags.length > 0 ||
    filters.createdDate !== undefined ||
    filters.updatedDate !== undefined
  );
}

export function getActiveFilterCount(): number {
  let count = 0;
  count += filters.folders.length;
  count += filters.tags.length;
  if (filters.createdDate) count++;
  if (filters.updatedDate) count++;
  return count;
}

// Convert relative date presets to absolute ISO8601 timestamps
export function getAbsoluteDateRange(
  dateFilter: DateFilter | undefined
): { after?: string; before?: string } | undefined {
  if (!dateFilter) return undefined;

  if (dateFilter.type === 'absolute') {
    return {
      after: dateFilter.after,
      before: dateFilter.before,
    };
  }

  // Convert relative to absolute
  const now = new SvelteDate();
  const today = new SvelteDate(now.getFullYear(), now.getMonth(), now.getDate());

  switch (dateFilter.preset) {
    case 'today': {
      const start = new SvelteDate(today);
      const end = new SvelteDate(today);
      end.setDate(end.getDate() + 1);
      return {
        after: start.toISOString(),
        before: end.toISOString(),
      };
    }
    case 'yesterday': {
      const start = new SvelteDate(today);
      start.setDate(start.getDate() - 1);
      const end = new SvelteDate(today);
      return {
        after: start.toISOString(),
        before: end.toISOString(),
      };
    }
    case 'last7days': {
      const start = new SvelteDate(today);
      start.setDate(start.getDate() - 7);
      return {
        after: start.toISOString(),
        before: undefined,
      };
    }
    case 'last30days': {
      const start = new SvelteDate(today);
      start.setDate(start.getDate() - 30);
      return {
        after: start.toISOString(),
        before: undefined,
      };
    }
    case 'thisWeek': {
      const start = new SvelteDate(today);
      const dayOfWeek = start.getDay();
      const diff = dayOfWeek === 0 ? 6 : dayOfWeek - 1; // Monday is first day
      start.setDate(start.getDate() - diff);
      return {
        after: start.toISOString(),
        before: undefined,
      };
    }
    case 'thisMonth': {
      const start = new SvelteDate(now.getFullYear(), now.getMonth(), 1);
      return {
        after: start.toISOString(),
        before: undefined,
      };
    }
    default:
      return undefined;
  }
}
