// CodeMirror utility functions: content update, live preview mode, focus mode, spell check, wikilink insertion

import { EditorView } from '@codemirror/view';

import { emptyExtension, setDimInactiveLines, setTypewriterMode } from '../focus-mode-extensions';
import { createLivePreviewExtension } from '../live-preview';
import {
  getSpellCheckState,
  setSpellLanguage as setSpellLanguageInternal,
  toggleSpellCheck as toggleSpellCheckInternal,
} from '../spell-check';
import { livePreviewCompartment } from './decoration-plugins';

// Re-export so the main module can access these
export { emptyExtension, setDimInactiveLines, setTypewriterMode } from '../focus-mode-extensions';
export { createSpellCheckExtension } from '../spell-check';

export function updateEditorContent(view: EditorView, content: string) {
  const currentContent = view.state.doc.toString();
  if (currentContent === content) {
    return;
  }

  view.dispatch({
    changes: {
      from: 0,
      to: view.state.doc.length,
      insert: content,
    },
  });
}

export function setLivePreviewMode(
  view: EditorView,
  enabled: boolean,
  options: { noteId?: string } = {}
) {
  view.dispatch({
    effects: livePreviewCompartment.reconfigure(
      enabled ? createLivePreviewExtension({ noteId: options.noteId }) : emptyExtension
    ),
  });
}

export function updateFocusMode(
  view: EditorView,
  options: { typewriter?: boolean; dimLines?: boolean }
) {
  if (options.typewriter !== undefined) {
    setTypewriterMode(view, options.typewriter);
  }
  if (options.dimLines !== undefined) {
    setDimInactiveLines(view, options.dimLines);
  }
}

export function insertWikiLink(view: EditorView, term: string, targetTitle: string): boolean {
  const doc = view.state.doc.toString();

  const lowerDoc = doc.toLowerCase();
  const lowerTerm = term.toLowerCase();
  const idx = lowerDoc.indexOf(lowerTerm);

  if (idx === -1) {
    return false;
  }

  const actualTerm = doc.substring(idx, idx + term.length);

  let wikilink: string;
  if (actualTerm.toLowerCase() === targetTitle.toLowerCase()) {
    wikilink = `[[${targetTitle}]]`;
  } else {
    wikilink = `[[${targetTitle}|${actualTerm}]]`;
  }

  view.dispatch({
    changes: {
      from: idx,
      to: idx + term.length,
      insert: wikilink,
    },
  });

  return true;
}

export function insertWikiLinkInContent(
  content: string,
  term: string,
  targetTitle: string
): { newContent: string; found: boolean } {
  const lowerContent = content.toLowerCase();
  const lowerTerm = term.toLowerCase();
  const idx = lowerContent.indexOf(lowerTerm);

  if (idx === -1) {
    return { newContent: content, found: false };
  }

  const actualTerm = content.substring(idx, idx + term.length);

  let wikilink: string;
  if (actualTerm.toLowerCase() === targetTitle.toLowerCase()) {
    wikilink = `[[${targetTitle}]]`;
  } else {
    wikilink = `[[${targetTitle}|${actualTerm}]]`;
  }

  const newContent = content.substring(0, idx) + wikilink + content.substring(idx + term.length);

  return { newContent, found: true };
}

export function toggleSpellCheck(view: EditorView, enabled: boolean) {
  toggleSpellCheckInternal(view, enabled);
}

export function setSpellCheckLanguage(view: EditorView, language: 'de' | 'en') {
  setSpellLanguageInternal(view, language);
}

export function isSpellCheckEnabled(view: EditorView): boolean {
  return getSpellCheckState(view).enabled;
}

export function getSpellCheckLanguage(view: EditorView): 'de' | 'en' {
  return getSpellCheckState(view).language;
}
