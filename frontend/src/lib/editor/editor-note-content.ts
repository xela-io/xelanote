import type { EditorView } from '@codemirror/view';

import { insertWikiLink, insertWikiLinkInContent } from '$lib/editor/codemirror';

interface InsertLinkActionParams {
  editorView: EditorView | undefined;
  term: string;
  targetTitle: string;
  getFallbackContent: () => string;
  setFallbackContent: (content: string) => void;
  scheduleAutoSave: () => void;
}

export function handleInsertLinkAction(params: InsertLinkActionParams): void {
  const {
    editorView,
    term,
    targetTitle,
    getFallbackContent,
    setFallbackContent,
    scheduleAutoSave,
  } = params;

  if (editorView) {
    insertWikiLink(editorView, term, targetTitle);
    scheduleAutoSave();
    return;
  }

  const content = getFallbackContent();
  const { newContent, found } = insertWikiLinkInContent(content, term, targetTitle);
  if (!found) return;

  setFallbackContent(newContent);
  scheduleAutoSave();
}

interface NoteSummaryTarget {
  summary?: string | null;
  summary_generated_at?: string | null;
}

export function updateCurrentNoteSummary(
  currentNote: NoteSummaryTarget | null,
  summary: string
): void {
  if (!currentNote) return;
  currentNote.summary = summary;
  currentNote.summary_generated_at = new Date().toISOString();
}
