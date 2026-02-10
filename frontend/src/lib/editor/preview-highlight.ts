/**
 * Svelte Action for highlighting search terms in the markdown preview pane.
 *
 * Uses TreeWalker on text nodes (no innerHTML manipulation) to safely
 * wrap matches in <mark class="search-highlight"> elements.
 */

export interface HighlightOptions {
  query: string;
  caseSensitive?: boolean;
}

const SKIP_ELEMENTS = new Set(['PRE', 'SCRIPT', 'STYLE', 'CODE']);

function escapeRegExp(str: string): string {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function removeHighlights(container: HTMLElement) {
  const marks = container.querySelectorAll('mark.search-highlight');
  marks.forEach((mark) => {
    const parent = mark.parentNode;
    if (parent) {
      // Replace <mark> with its text content
      const text = document.createTextNode(mark.textContent ?? '');
      parent.replaceChild(text, mark);
      // Normalize to merge adjacent text nodes
      parent.normalize();
    }
  });
}

function applyHighlights(container: HTMLElement, query: string, caseSensitive: boolean) {
  if (!query) return;

  const flags = caseSensitive ? 'g' : 'gi';
  const regex = new RegExp(escapeRegExp(query), flags);

  // Collect text nodes using TreeWalker
  const walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT, {
    acceptNode: (node) => {
      // Skip nodes inside pre, script, style, code elements
      let parent = node.parentElement;
      while (parent && parent !== container) {
        if (SKIP_ELEMENTS.has(parent.tagName)) {
          return NodeFilter.FILTER_REJECT;
        }
        parent = parent.parentElement;
      }
      return NodeFilter.FILTER_ACCEPT;
    },
  });

  const textNodes: Text[] = [];
  let current: Node | null;
  while ((current = walker.nextNode())) {
    textNodes.push(current as Text);
  }

  // Process text nodes in reverse to avoid invalidating positions
  for (let i = textNodes.length - 1; i >= 0; i--) {
    const textNode = textNodes[i];
    const text = textNode.textContent ?? '';

    const matches: { index: number; length: number }[] = [];
    let match: RegExpExecArray | null;
    while ((match = regex.exec(text)) !== null) {
      matches.push({ index: match.index, length: match[0].length });
      if (match[0].length === 0) break; // Prevent infinite loops on zero-length matches
    }

    if (matches.length === 0) continue;

    // Split text node and wrap matches
    const parent = textNode.parentNode;
    if (!parent) continue;

    const fragment = document.createDocumentFragment();
    let lastIndex = 0;

    for (const m of matches) {
      // Text before match
      if (m.index > lastIndex) {
        fragment.appendChild(document.createTextNode(text.slice(lastIndex, m.index)));
      }

      // Wrapped match
      const mark = document.createElement('mark');
      mark.className = 'search-highlight';
      mark.textContent = text.slice(m.index, m.index + m.length);
      fragment.appendChild(mark);

      lastIndex = m.index + m.length;
    }

    // Remaining text after last match
    if (lastIndex < text.length) {
      fragment.appendChild(document.createTextNode(text.slice(lastIndex)));
    }

    parent.replaceChild(fragment, textNode);
  }
}

/**
 * Svelte Action: Highlights search terms in the container's text nodes.
 */
export function highlightSearchTerms(container: HTMLElement, options: HighlightOptions) {
  function apply() {
    removeHighlights(container);
    if (options.query) {
      applyHighlights(container, options.query, options.caseSensitive ?? false);
    }
  }

  // Initial apply
  apply();

  return {
    update(newOptions: HighlightOptions) {
      options = newOptions;
      apply();
    },
    destroy() {
      removeHighlights(container);
    },
  };
}
