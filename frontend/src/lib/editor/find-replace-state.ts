import type { FindReplaceState } from '$lib/editor/find-replace-ui';

export interface FindReplaceStateSource {
  show: boolean;
  query: string;
  showReplace: boolean;
  caseSensitive: boolean;
  pendingHighlightQuery: string | null;
  editorExtensionsReady: boolean;
  prevNoteId: string | null;
}

export function readFindReplaceState(source: FindReplaceStateSource): FindReplaceState {
  return {
    show: source.show,
    query: source.query,
    showReplace: source.showReplace,
    caseSensitive: source.caseSensitive,
    pendingHighlightQuery: source.pendingHighlightQuery,
    editorExtensionsReady: source.editorExtensionsReady,
    prevNoteId: source.prevNoteId,
  };
}

export function writeFindReplaceState(
  nextState: FindReplaceState,
  sink: {
    setShow: (value: boolean) => void;
    setQuery: (value: string) => void;
    setShowReplace: (value: boolean) => void;
    setCaseSensitive: (value: boolean) => void;
    setPendingHighlightQuery: (value: string | null) => void;
    setEditorExtensionsReady: (value: boolean) => void;
    setPrevNoteId: (value: string | null) => void;
  }
): void {
  sink.setShow(nextState.show);
  sink.setQuery(nextState.query);
  sink.setShowReplace(nextState.showReplace);
  sink.setCaseSensitive(nextState.caseSensitive);
  sink.setPendingHighlightQuery(nextState.pendingHighlightQuery);
  sink.setEditorExtensionsReady(nextState.editorExtensionsReady);
  sink.setPrevNoteId(nextState.prevNoteId);
}
