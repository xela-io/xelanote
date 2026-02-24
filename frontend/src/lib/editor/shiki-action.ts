// Svelte action that highlights code blocks marked with .shiki-pending.
// Lazily loads Shiki + language grammars on demand.

import type { ActionReturn } from 'svelte/action';

import { highlightCode } from './shiki-loader';

export interface ShikiHighlighterOptions {
  revision?: string | number;
}

/**
 * Svelte action for the preview container.
 * Scans for pre.shiki-pending elements and replaces them with Shiki output.
 */
export function shikiHighlighter(
  node: HTMLElement,
  _options?: ShikiHighlighterOptions
): ActionReturn<ShikiHighlighterOptions | undefined> {
  function highlight() {
    const pendingBlocks = node.querySelectorAll<HTMLPreElement>('pre.shiki-pending');
    for (const block of pendingBlocks) {
      const lang = block.dataset.lang;
      if (!lang) continue;

      const codeEl = block.querySelector('code');
      if (!codeEl) continue;

      const code = codeEl.textContent || '';

      // Mark as processing to avoid double-highlighting
      block.classList.remove('shiki-pending');
      block.classList.add('shiki-processing');

      highlightCode(code, lang).then((html) => {
        if (html) {
          // Shiki returns a full <pre><code>...</code></pre> structure
          const temp = document.createElement('div');
          temp.innerHTML = html;
          const newPre = temp.querySelector('pre');
          if (newPre && block.parentNode) {
            // Preserve data-lang for CSS styling
            newPre.dataset.lang = lang;
            block.parentNode.replaceChild(newPre, block);
          }
        } else {
          // Language not available — remove processing class, show plain
          block.classList.remove('shiki-processing');
        }
      });
    }
  }

  highlight();

  return {
    update() {
      highlight();
    },
  };
}
