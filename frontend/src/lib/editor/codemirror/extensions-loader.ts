// Lazy-loaded CodeMirror extensions (markdown language, history, keymaps, autocomplete)

import type { Extension } from '@codemirror/state';
import { keymap } from '@codemirror/view';

import type { EditorConfig } from '../codemirror';
import { markdownSyntaxStyle } from './theme';

let lazyExtensionsPromise: Promise<Extension[]> | null = null;

export async function loadEditorExtensions(_config: EditorConfig): Promise<Extension[]> {
  if (lazyExtensionsPromise) {
    return lazyExtensionsPromise;
  }

  lazyExtensionsPromise = (async () => {
    const [
      { defaultKeymap, history, historyKeymap, indentWithTab },
      { markdown, markdownLanguage },
      { syntaxHighlighting, defaultHighlightStyle },
      { createWikilinkAutocomplete },
    ] = await Promise.all([
      import('@codemirror/commands'),
      import('@codemirror/lang-markdown'),
      import('@codemirror/language'),
      import('../wikilink-autocomplete'),
    ]);

    const wikilinkExt = createWikilinkAutocomplete();

    return [
      history(),
      markdown({ base: markdownLanguage }),
      syntaxHighlighting(markdownSyntaxStyle),
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      wikilinkExt,
      keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
    ];
  })();

  return lazyExtensionsPromise;
}
