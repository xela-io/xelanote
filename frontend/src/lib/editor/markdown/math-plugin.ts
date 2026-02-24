// markdown-it plugin for math expressions using KaTeX.
// Supports $inline$ and $$display$$ syntax.
// Uses a two-pass approach: marks math spans in first render,
// then a Svelte action replaces them with KaTeX output.

import type MarkdownIt from 'markdown-it';
import type StateBlock from 'markdown-it/lib/rules_block/state_block.mjs';
import type StateInline from 'markdown-it/lib/rules_inline/state_inline.mjs';

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

/**
 * Inline math rule: $...$
 * Escaped \$ is not treated as math delimiter.
 */
function mathInline(state: StateInline, silent: boolean): boolean {
  if (state.src.charCodeAt(state.pos) !== 0x24 /* $ */) return false;

  // Skip escaped dollar signs
  if (state.pos > 0 && state.src.charCodeAt(state.pos - 1) === 0x5c /* \ */) return false;

  // Don't match $$ (that's display math)
  if (state.src.charCodeAt(state.pos + 1) === 0x24) return false;

  const start = state.pos + 1;
  let end = start;

  // Find closing $
  while (end < state.posMax) {
    if (state.src.charCodeAt(end) === 0x24 /* $ */ && state.src.charCodeAt(end - 1) !== 0x5c) {
      break;
    }
    end++;
  }

  if (end >= state.posMax) return false;

  const content = state.src.slice(start, end).trim();
  if (!content) return false;

  if (!silent) {
    const token = state.push('math_inline', 'span', 0);
    token.content = content;
    token.markup = '$';
  }

  state.pos = end + 1;
  return true;
}

/**
 * Block math rule: $$...$$
 */
function mathBlock(
  state: StateBlock,
  startLine: number,
  endLine: number,
  silent: boolean
): boolean {
  const startPos = state.bMarks[startLine] + state.tShift[startLine];

  if (state.src.charCodeAt(startPos) !== 0x24 || state.src.charCodeAt(startPos + 1) !== 0x24) {
    return false;
  }

  if (silent) return true;

  let nextLine = startLine;
  let hasEnding = false;

  // Find closing $$
  while (nextLine < endLine) {
    nextLine++;
    if (nextLine >= endLine) break;

    const lineStart = state.bMarks[nextLine] + state.tShift[nextLine];
    const lineEnd = state.eMarks[nextLine];
    const lineText = state.src.slice(lineStart, lineEnd).trim();

    if (lineText === '$$') {
      hasEnding = true;
      break;
    }
  }

  // Collect content between $$ markers
  const contentLines: string[] = [];
  for (let i = startLine + 1; i < (hasEnding ? nextLine : endLine); i++) {
    const lineStart = state.bMarks[i];
    const lineEnd = state.eMarks[i];
    contentLines.push(state.src.slice(lineStart, lineEnd));
  }

  const content = contentLines.join('\n').trim();

  const token = state.push('math_block', 'div', 0);
  token.content = content;
  token.markup = '$$';
  token.map = [startLine, hasEnding ? nextLine + 1 : nextLine];

  state.line = hasEnding ? nextLine + 1 : nextLine;

  return true;
}

/**
 * Register math plugin with a MarkdownIt instance.
 * Math expressions are rendered as placeholders that the math action processes.
 */
export function register(md: MarkdownIt): void {
  // Inline: $...$
  md.inline.ruler.before('escape', 'math_inline', mathInline);
  md.renderer.rules.math_inline = (tokens, idx): string => {
    const content = tokens[idx].content;
    return `<span class="math-inline math-pending" data-math="${escapeHtml(content)}" data-display="false">$${escapeHtml(content)}$</span>`;
  };

  // Block: $$...$$
  md.block.ruler.before('fence', 'math_block', mathBlock);
  md.renderer.rules.math_block = (tokens, idx): string => {
    const content = tokens[idx].content;
    return `<div class="math-block math-pending" data-math="${escapeHtml(content)}" data-display="true">$$${escapeHtml(content)}$$</div>\n`;
  };
}
