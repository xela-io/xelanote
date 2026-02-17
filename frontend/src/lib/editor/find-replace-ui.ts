import type { EditorView } from '@codemirror/view';

import { clearSearch, sanitizeSearchQuery } from '$lib/editor/find-replace';

export interface FindReplaceState {
  show: boolean;
  query: string;
  showReplace: boolean;
  caseSensitive: boolean;
  pendingHighlightQuery: string | null;
  editorExtensionsReady: boolean;
  prevNoteId: string | null;
}

export interface FindReplaceHandlers {
  getEditorView: () => EditorView | undefined;
  getEditorMode: () => 'edit' | 'split' | 'preview' | 'live';
  getNoteId: () => string | null;
  getUrlHighlight: () => string | null;
  setUrlHighlight: (value: string | null) => void;
  setState: (partial: Partial<FindReplaceState>) => void;
}

export function openFindReplace(
  state: FindReplaceState,
  handlers: FindReplaceHandlers,
  query?: string,
  options?: { replace?: boolean }
): FindReplaceState {
  const editorView = handlers.getEditorView();
  let nextQuery = query ?? '';
  const nextShowReplace = options?.replace ?? false;

  if (!query && editorView) {
    const selection = editorView.state.selection.main;
    if (selection.from !== selection.to) {
      nextQuery = editorView.state.doc.sliceString(selection.from, selection.to);
    }
  }

  return {
    ...state,
    show: true,
    query: nextQuery,
    showReplace: nextShowReplace,
  };
}

export function closeFindReplace(
  state: FindReplaceState,
  handlers: FindReplaceHandlers
): FindReplaceState {
  const editorView = handlers.getEditorView();
  if (state.show) {
    if (editorView) {
      clearSearch(editorView);
    }
    handlers.setUrlHighlight(null);
  }

  return {
    ...state,
    show: false,
    query: '',
    showReplace: false,
  };
}

export function handleNoteChange(
  state: FindReplaceState,
  handlers: FindReplaceHandlers
): FindReplaceState {
  const noteId = handlers.getNoteId();
  if (state.prevNoteId !== null && state.prevNoteId !== noteId) {
    const editorView = handlers.getEditorView();
    if (editorView) {
      clearSearch(editorView);
    }
    return {
      ...state,
      show: false,
      query: '',
      showReplace: false,
      prevNoteId: noteId,
    };
  }

  return {
    ...state,
    prevNoteId: noteId,
  };
}

export function handleUrlHighlight(
  state: FindReplaceState,
  handlers: FindReplaceHandlers
): FindReplaceState {
  const query = handlers.getUrlHighlight();
  if (!query) {
    return state;
  }

  const sanitized = sanitizeSearchQuery(query);
  const isPreviewOnly = handlers.getEditorMode() === 'preview';

  if (isPreviewOnly || (state.editorExtensionsReady && handlers.getEditorView())) {
    return openFindReplace(state, handlers, sanitized);
  }

  return {
    ...state,
    pendingHighlightQuery: sanitized,
  };
}

export function handleExtensionsReady(
  state: FindReplaceState,
  handlers: FindReplaceHandlers
): FindReplaceState {
  if (!state.pendingHighlightQuery) {
    return state;
  }

  const nextState = openFindReplace(state, handlers, state.pendingHighlightQuery);
  return {
    ...nextState,
    pendingHighlightQuery: null,
  };
}
