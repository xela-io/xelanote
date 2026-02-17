// Wikilink auto-completion for CodeMirror
import {
  autocompletion,
  type CompletionContext,
  type CompletionResult,
} from '@codemirror/autocomplete';

import { quickSearch } from '$lib/api';

// Pattern to detect wikilink context: [[...
const WIKILINK_TRIGGER = /\[\[([^\]|]*)$/;

// Cache for API results
let lastQuery = '';
let lastResults: { title: string; folder_path: string }[] = [];
let pendingPromise: Promise<void> | null = null;

export async function wikilinkCompletionSource(
  context: CompletionContext
): Promise<CompletionResult | null> {
  const line = context.state.doc.lineAt(context.pos);
  const textBefore = line.text.slice(0, context.pos - line.from);
  const match = textBefore.match(WIKILINK_TRIGGER);

  if (!match) return null;

  const query = match[1];
  const from = context.pos - query.length;

  // Use cache if query is the same
  if (query !== lastQuery || !lastResults.length) {
    lastQuery = query;

    // Wait for any pending request to complete
    if (pendingPromise) {
      await pendingPromise;
    }

    // Make new request
    pendingPromise = (async () => {
      try {
        const response = await quickSearch(query || '', 10);
        lastResults = response.notes || [];
      } catch {
        lastResults = [];
      }
    })();

    await pendingPromise;
    pendingPromise = null;
  }

  if (lastResults.length === 0) {
    return null;
  }

  return {
    from,
    options: lastResults.map((note) => ({
      label: note.title,
      detail: note.folder_path || '/',
      apply: `${note.title}]]`,
      type: 'variable',
    })),
    validFor: /^[^\]|]*$/,
  };
}

export function createWikilinkAutocomplete() {
  return autocompletion({
    override: [wikilinkCompletionSource],
    activateOnTyping: true,
    defaultKeymap: true,
    closeOnBlur: true,
    // Trigger autocomplete when typing '[' character
    activateOnCompletion: () => true,
  });
}
