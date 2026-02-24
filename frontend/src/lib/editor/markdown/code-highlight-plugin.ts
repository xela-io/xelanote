// markdown-it plugin: marks code blocks for deferred Shiki highlighting.
// Adds data-lang and shiki-pending class for the Svelte action to pick up.

import type MarkdownIt from 'markdown-it';
import type Token from 'markdown-it/lib/token.mjs';

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

/**
 * Register a fence renderer that marks code blocks for Shiki highlighting.
 * The actual highlighting is done asynchronously by the shikiHighlighter action.
 */
export function register(md: MarkdownIt): void {
  md.renderer.rules.fence = (tokens: Token[], idx: number): string => {
    const token = tokens[idx];
    const info = token.info.trim();
    const lang = info.split(/\s+/)[0] || '';
    const code = token.content;

    if (lang) {
      return `<pre class="shiki-pending" data-lang="${escapeHtml(lang)}"><code>${escapeHtml(code)}</code></pre>\n`;
    }

    // No language specified — plain code block
    return `<pre><code>${escapeHtml(code)}</code></pre>\n`;
  };
}
