// Color syntax plugin for markdown-it: {color:VALUE}text{/color}

import type MarkdownIt from 'markdown-it';
import type StateInline from 'markdown-it/lib/rules_inline/state_inline.mjs';
import type Token from 'markdown-it/lib/token.mjs';

import { FEATURE_FLAGS } from '$lib/config';

// Named colors that map to CSS classes (design tokens)
const NAMED_COLORS = new Set(['primary', 'destructive', 'accent', 'muted', 'secondary']);

// Regex patterns for color validation
const HEX_COLOR_REGEX = /^#(?:[0-9a-f]{3}|[0-9a-f]{6})$/i;
const RGB_REGEX = /^rgb\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*\)$/i;
const RGBA_REGEX = /^rgba\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(0|1|0?\.\d+)\s*\)$/i;

// Maximum scan distance for color content (prevent ReDoS)
const COLOR_MAX_SCAN = 5000;

/**
 * Sanitize and validate a color value.
 * Returns the normalized color or null if invalid.
 *
 * Allowed formats:
 * - Named colors: primary, destructive, accent, muted, secondary
 * - Hex: #fff, #ffffff
 * - RGB: rgb(0, 0, 0) - rgb(255, 255, 255)
 * - RGBA: rgba(0, 0, 0, 0) - rgba(255, 255, 255, 1)
 */
export function sanitizeColor(color: string): string | null {
  const normalized = color.toLowerCase().trim();

  if (NAMED_COLORS.has(normalized)) {
    return normalized;
  }

  if (HEX_COLOR_REGEX.test(normalized)) {
    return normalized;
  }

  const rgbMatch = normalized.match(RGB_REGEX);
  if (rgbMatch) {
    const [, r, g, b] = rgbMatch;
    if (parseInt(r, 10) <= 255 && parseInt(g, 10) <= 255 && parseInt(b, 10) <= 255) {
      return `rgb(${r}, ${g}, ${b})`;
    }
    return null;
  }

  const rgbaMatch = normalized.match(RGBA_REGEX);
  if (rgbaMatch) {
    const [, r, g, b, a] = rgbaMatch;
    const alpha = parseFloat(a);
    if (
      parseInt(r, 10) <= 255 &&
      parseInt(g, 10) <= 255 &&
      parseInt(b, 10) <= 255 &&
      alpha >= 0 &&
      alpha <= 1
    ) {
      return `rgba(${r}, ${g}, ${b}, ${a})`;
    }
    return null;
  }

  return null;
}

// Color escape rule - handles \{color:...} to output literal text
function colorEscapeRule(state: StateInline, silent: boolean): boolean {
  if (!FEATURE_FLAGS.colorSyntax) return false;

  const start = state.pos;
  const max = state.posMax;

  if (state.src.charCodeAt(start) !== 0x5c /* \ */) return false;
  if (start + 7 >= max) return false;
  if (state.src.slice(start + 1, start + 8) !== '{color:') return false;

  let pos = start + 8;
  while (pos < max && state.src.charCodeAt(pos) !== 0x7d /* } */) {
    if (state.src.charCodeAt(pos) === 0x0a /* \n */) return false;
    pos++;
  }

  if (pos >= max) return false;

  if (!silent) {
    const token = state.push('text', '', 0);
    token.content = state.src.slice(start + 1, pos + 1);
  }

  state.pos = pos + 1;
  return true;
}

// Color close escape rule - handles \{/color}
function colorCloseEscapeRule(state: StateInline, silent: boolean): boolean {
  if (!FEATURE_FLAGS.colorSyntax) return false;

  const start = state.pos;
  const max = state.posMax;

  if (state.src.charCodeAt(start) !== 0x5c /* \ */) return false;
  if (start + 8 >= max) return false;
  if (state.src.slice(start + 1, start + 9) !== '{/color}') return false;

  if (!silent) {
    const token = state.push('text', '', 0);
    token.content = '{/color}';
  }

  state.pos = start + 9;
  return true;
}

// Color inline rule - parses {color:VALUE}text{/color}
function colorRule(state: StateInline, silent: boolean): boolean {
  if (!FEATURE_FLAGS.colorSyntax) return false;

  const start = state.pos;
  const max = state.posMax;

  if (start + 7 >= max) return false;
  if (state.src.charCodeAt(start) !== 0x7b /* { */) return false;
  if (state.src.slice(start, start + 7) !== '{color:') return false;

  let pos = start + 7;
  let colorEnd = -1;

  while (pos < max && pos < start + 100) {
    const char = state.src.charCodeAt(pos);
    if (char === 0x7d /* } */) {
      colorEnd = pos;
      break;
    }
    if (char === 0x0a /* \n */ || char === 0x7b /* { */) {
      return false;
    }
    pos++;
  }

  if (colorEnd === -1) return false;

  const colorValue = state.src.slice(start + 7, colorEnd);
  const sanitizedColor = sanitizeColor(colorValue);

  if (!sanitizedColor) {
    return false;
  }

  const contentStart = colorEnd + 1;
  let contentEnd = -1;
  const closingTag = '{/color}';
  const maxScan = Math.min(contentStart + COLOR_MAX_SCAN, max);

  pos = contentStart;
  while (pos < maxScan) {
    const char = state.src.charCodeAt(pos);

    if (char === 0x0a /* \n */) {
      return false;
    }

    if (char === 0x5c /* \ */ && state.src.slice(pos + 1, pos + 9) === '{/color}') {
      pos += 9;
      continue;
    }

    if (state.src.slice(pos, pos + closingTag.length) === closingTag) {
      contentEnd = pos;
      break;
    }
    pos++;
  }

  if (contentEnd === -1) return false;

  if (!silent) {
    const openToken = state.push('color_open', 'span', 1);
    openToken.meta = { color: sanitizedColor };

    const content = state.src.slice(contentStart, contentEnd);
    if (content.length > 0) {
      const oldPos = state.pos;
      const oldMax = state.posMax;

      state.pos = contentStart;
      state.posMax = contentEnd;
      state.md.inline.tokenize(state);

      state.pos = oldPos;
      state.posMax = oldMax;
    }

    state.push('color_close', 'span', -1);
  }

  state.pos = contentEnd + closingTag.length;
  return true;
}

/** Register color syntax plugin with a MarkdownIt instance. */
export function register(md: MarkdownIt, escapeHtml: (s: string) => string): void {
  if (!FEATURE_FLAGS.colorSyntax) return;

  md.inline.ruler.before('escape', 'color_escape', colorEscapeRule);
  md.inline.ruler.before('escape', 'color_close_escape', colorCloseEscapeRule);
  md.inline.ruler.before('link', 'color', colorRule);

  md.renderer.rules.color_open = (tokens: Token[], idx: number): string => {
    const token = tokens[idx];
    const color = token.meta?.color as string;

    if (NAMED_COLORS.has(color)) {
      return `<span class="text-color-${escapeHtml(color)}">`;
    } else {
      return `<span style="color: ${escapeHtml(color)};">`;
    }
  };

  md.renderer.rules.color_close = (): string => {
    return '</span>';
  };
}
