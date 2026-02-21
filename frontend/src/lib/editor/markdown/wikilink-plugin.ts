// Wikilink syntax plugin for markdown-it: [[title]] and [[title|alias]]

import type MarkdownIt from 'markdown-it';
import type { Options as MarkdownItOptions } from 'markdown-it';
import type StateInline from 'markdown-it/lib/rules_inline/state_inline.mjs';
import type Token from 'markdown-it/lib/token.mjs';

type MarkdownItEnv = Record<string, unknown>;

function wikilinkRule(state: StateInline, silent: boolean): boolean {
  const start = state.pos;
  const max = state.posMax;

  if (start + 3 >= max) return false;
  if (state.src.charCodeAt(start) !== 0x5b /* [ */) return false;
  if (state.src.charCodeAt(start + 1) !== 0x5b /* [ */) return false;

  let pos = start + 2;
  let title = '';
  let alias = '';
  let foundPipe = false;

  while (pos < max) {
    const char = state.src.charCodeAt(pos);

    if (char === 0x5d /* ] */ && pos + 1 < max && state.src.charCodeAt(pos + 1) === 0x5d /* ] */) {
      if (title.length === 0) {
        return false;
      }

      if (title.trim().length === 0) {
        return false;
      }

      if (!silent) {
        const token = state.push('wikilink', 'a', 0);
        token.content = title.trim();
        token.meta = { alias: alias.trim() || undefined };
      }

      state.pos = pos + 2;
      return true;
    }

    if (char === 0x7c /* | */ && !foundPipe) {
      foundPipe = true;
      pos++;
      continue;
    }

    if (char === 0x0a /* \n */ || char === 0x5d /* ] */) {
      return false;
    }

    if (foundPipe) {
      alias += state.src.charAt(pos);
    } else {
      title += state.src.charAt(pos);
    }

    pos++;
  }

  return false;
}

/** Register wikilink plugin with a MarkdownIt instance. */
export function register(md: MarkdownIt, escapeHtml: (s: string) => string): void {
  md.inline.ruler.before('link', 'wikilink', wikilinkRule);

  md.renderer.rules.wikilink = (
    tokens: Token[],
    idx: number,
    _options: MarkdownItOptions,
    env: MarkdownItEnv
  ): string => {
    const token = tokens[idx];
    const title = token.content;
    const alias = token.meta?.alias as string | undefined;
    const displayText = alias || title;

    const titleLower = title.toLowerCase().trim();
    const titleToIdMap = (env.titleToIdMap as Map<string, string>) ?? new Map<string, string>();
    const resolvedTitles = (env.resolvedTitles as Set<string>) ?? new Set<string>();
    const isResolved = resolvedTitles.has(titleLower);
    const className = isResolved ? 'wikilink-resolved' : 'wikilink-unresolved';

    const noteId = titleToIdMap.get(titleLower);
    const href = noteId ? `/note/${noteId}` : `/note/${encodeURIComponent(title.trim())}`;

    const escapedTitle = escapeHtml(title.trim());
    const escapedDisplayText = escapeHtml(displayText);

    return `<a href="${href}" class="wikilink ${className}" data-title="${escapedTitle}">${escapedDisplayText}</a>`;
  };
}
