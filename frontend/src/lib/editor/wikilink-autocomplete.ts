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
  console.log(
    '[Wikilink Autocomplete] Function called at pos:',
    context.pos,
    'explicit:',
    context.explicit
  );

  const line = context.state.doc.lineAt(context.pos);
  const textBefore = line.text.slice(0, context.pos - line.from);
  const match = textBefore.match(WIKILINK_TRIGGER);

  console.log('[Wikilink Autocomplete] textBefore:', textBefore, 'match:', match);

  if (!match) return null;

  const query = match[1];
  const from = context.pos - query.length;

  console.log('[Wikilink Autocomplete] query:', query, 'from:', from, 'pos:', context.pos);

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
        console.log('[Wikilink Autocomplete] Searching for:', query || '(empty)');
        const response = await quickSearch(query || '', 10);
        lastResults = response.notes || [];
        console.log('[Wikilink Autocomplete] Results:', lastResults.length);
      } catch (error) {
        console.error('Wikilink autocomplete failed:', error);
        lastResults = [];
      }
    })();

    await pendingPromise;
    pendingPromise = null;
  }

  if (lastResults.length === 0) {
    console.log('[Wikilink Autocomplete] No results to show');
    return null;
  }

  console.log('[Wikilink Autocomplete] Showing', lastResults.length, 'results');

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
  console.log('[Wikilink Autocomplete] Extension created');
  return autocompletion({
    override: [wikilinkCompletionSource],
    activateOnTyping: true,
    defaultKeymap: true,
    closeOnBlur: true,
    // Trigger autocomplete when typing '[' character
    activateOnCompletion: () => true,
  });
}
