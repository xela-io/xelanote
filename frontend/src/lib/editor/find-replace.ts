/**
 * CodeMirror 6 Search Extension Wrapper for xelanote.
 *
 * Uses @codemirror/search for match decorations and navigation,
 * but suppresses the default panel UI in favor of a custom Svelte component.
 */

import {
  findNext,
  findPrevious,
  getSearchQuery,
  replaceAll,
  replaceNext,
  search,
  searchKeymap,
  SearchQuery,
  setSearchQuery,
} from '@codemirror/search';
import { EditorView } from '@codemirror/view';

/**
 * Escape regex special characters for safe use in RegExp.
 * Applied to URL-based queries (?highlight=) to prevent regex injection.
 * NOT applied to manual input in FindReplaceBar (literal search, no regex).
 */
export function sanitizeSearchQuery(query: string): string {
  return query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/**
 * Create the search extension with a hidden panel.
 * The hidden panel is necessary to keep @codemirror/search's decorations
 * and state active without showing the default UI.
 */
export function createFindReplaceExtension() {
  return [
    search({
      createPanel: () => {
        // Return a minimal hidden DOM element to suppress default panel
        const dom = document.createElement('div');
        dom.style.display = 'none';
        return {
          dom,
          top: true,
        };
      },
    }),
    // Theme overrides for search match highlights
    EditorView.baseTheme({
      '.cm-searchMatch': {
        backgroundColor: 'color-mix(in srgb, var(--color-primary) 25%, transparent)',
        borderRadius: '2px',
      },
      '.cm-searchMatch-selected': {
        backgroundColor: 'color-mix(in srgb, var(--color-primary) 50%, transparent)',
        outline: '1px solid var(--color-primary)',
      },
    }),
  ];
}

/**
 * Programmatically set a search query on the editor.
 */
export function performSearch(
  view: EditorView,
  query: string,
  options: { caseSensitive?: boolean } = {}
) {
  if (!query) {
    clearSearch(view);
    return;
  }

  const searchQuery = new SearchQuery({
    search: query,
    caseSensitive: options.caseSensitive ?? false,
    literal: true,
  });

  view.dispatch({
    effects: setSearchQuery.of(searchQuery),
  });
}

/**
 * Navigate to the next match.
 */
export function goToNextMatch(view: EditorView): boolean {
  return findNext(view);
}

/**
 * Navigate to the previous match.
 */
export function goToPreviousMatch(view: EditorView): boolean {
  return findPrevious(view);
}

/**
 * Clear the search query and remove all decorations.
 */
export function clearSearch(view: EditorView) {
  const emptyQuery = new SearchQuery({ search: '' });
  view.dispatch({
    effects: setSearchQuery.of(emptyQuery),
  });
}

/**
 * Get the current match count (total and current index).
 * Iterates through all matches using the SearchQuery's cursor.
 */
export function getMatchCount(view: EditorView): { current: number; total: number } {
  const query = getSearchQuery(view.state);
  if (!query.valid) {
    return { current: 0, total: 0 };
  }

  const cursor = query.getCursor(view.state.doc);
  let total = 0;
  let current = 0;
  const selection = view.state.selection.main;
  let result = cursor.next();

  while (!result.done) {
    total++;
    // Check if this match contains the cursor position
    if (result.value.from <= selection.from && result.value.to >= selection.from) {
      current = total;
    }
    result = cursor.next();
  }

  return { current, total };
}

/**
 * Replace the current match with the given replacement text.
 */
export function performReplace(
  view: EditorView,
  query: string,
  replacement: string,
  options: { caseSensitive?: boolean } = {}
) {
  // Set the query with replacement
  const searchQuery = new SearchQuery({
    search: query,
    replace: replacement,
    caseSensitive: options.caseSensitive ?? false,
    literal: true,
  });

  view.dispatch({
    effects: setSearchQuery.of(searchQuery),
  });

  // Execute replace next
  replaceNext(view);
}

/**
 * Replace all matches with the given replacement text.
 */
export function performReplaceAll(
  view: EditorView,
  query: string,
  replacement: string,
  options: { caseSensitive?: boolean } = {}
) {
  // Set the query with replacement
  const searchQuery = new SearchQuery({
    search: query,
    replace: replacement,
    caseSensitive: options.caseSensitive ?? false,
    literal: true,
  });

  view.dispatch({
    effects: setSearchQuery.of(searchQuery),
  });

  // Execute replace all
  replaceAll(view);
}

/**
 * Check if the search extension is loaded and ready.
 */
export function isSearchReady(view: EditorView): boolean {
  try {
    getSearchQuery(view.state);
    return true;
  } catch {
    return false;
  }
}

// Re-export searchKeymap for potential use
export { searchKeymap };
