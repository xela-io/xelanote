// Markdown rendering with wikilink and color syntax support

import DOMPurify from 'isomorphic-dompurify';
import type { Options as MarkdownItOptions } from 'markdown-it';
import MarkdownIt from 'markdown-it';
import type StateInline from 'markdown-it/lib/rules_inline/state_inline.mjs';
import type Token from 'markdown-it/lib/token.mjs';
import taskLists from 'markdown-it-task-lists';

import { FEATURE_FLAGS } from '$lib/config';

type MarkdownItEnv = Record<string, unknown>;
type MarkdownItRenderer = unknown;

// Named colors that map to CSS classes (design tokens)
const NAMED_COLORS = new Set(['primary', 'destructive', 'accent', 'muted', 'secondary']);

// HTML escape function to prevent XSS
function escapeHtml(text: string): string {
  const map: { [key: string]: string } = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#039;',
  };
  return text.replace(/[&<>"']/g, (m) => map[m]);
}

// Regex patterns for color validation
const HEX_COLOR_REGEX = /^#(?:[0-9a-f]{3}|[0-9a-f]{6})$/i;
const RGB_REGEX = /^rgb\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*\)$/i;
const RGBA_REGEX = /^rgba\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(0|1|0?\.\d+)\s*\)$/i;

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

  // Check named colors
  if (NAMED_COLORS.has(normalized)) {
    return normalized;
  }

  // Check hex colors
  if (HEX_COLOR_REGEX.test(normalized)) {
    return normalized;
  }

  // Check RGB
  const rgbMatch = normalized.match(RGB_REGEX);
  if (rgbMatch) {
    const [, r, g, b] = rgbMatch;
    if (parseInt(r, 10) <= 255 && parseInt(g, 10) <= 255 && parseInt(b, 10) <= 255) {
      return `rgb(${r}, ${g}, ${b})`;
    }
    return null;
  }

  // Check RGBA
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

// Due date validation
const DUE_DATE_REGEX = /^\d{4}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01])$/;

/**
 * Validate a date string in YYYY-MM-DD format.
 * Checks both format and overflow (e.g. Feb 30 is invalid).
 */
export function isValidDueDate(dateStr: string): boolean {
  if (!DUE_DATE_REGEX.test(dateStr)) return false;
  const d = new Date(dateStr + 'T00:00:00');
  if (isNaN(d.getTime())) return false;
  const [y, m, day] = dateStr.split('-').map(Number);
  return d.getFullYear() === y && d.getMonth() + 1 === m && d.getDate() === day;
}

/**
 * Determine the status of a due date relative to today.
 */
export function getDueDateStatus(dateStr: string): 'overdue' | 'today' | 'soon' | 'future' {
  const now = new Date();
  now.setHours(0, 0, 0, 0);
  const due = new Date(dateStr + 'T00:00:00');
  const diffMs = due.getTime() - now.getTime();
  const diffDays = Math.round(diffMs / (1000 * 60 * 60 * 24));
  if (diffDays < 0) return 'overdue';
  if (diffDays === 0) return 'today';
  if (diffDays <= 3) return 'soon';
  return 'future';
}

// Maximum scan distance for color content (prevent ReDoS)
const COLOR_MAX_SCAN = 5000;

// Regex for validating image width values (XSS protection)
const WIDTH_VALUE_REGEX = /^\d+%?$/;

// Color escape rule - handles \{color:...} to output literal text
function colorEscapeRule(state: StateInline, silent: boolean): boolean {
  if (!FEATURE_FLAGS.colorSyntax) return false;

  const start = state.pos;
  const max = state.posMax;

  // Check for backslash followed by {color:
  if (state.src.charCodeAt(start) !== 0x5c /* \ */) return false;
  if (start + 7 >= max) return false;
  if (state.src.slice(start + 1, start + 8) !== '{color:') return false;

  // Find the closing }
  let pos = start + 8;
  while (pos < max && state.src.charCodeAt(pos) !== 0x7d /* } */) {
    if (state.src.charCodeAt(pos) === 0x0a /* \n */) return false;
    pos++;
  }

  if (pos >= max) return false;

  if (!silent) {
    const token = state.push('text', '', 0);
    token.content = state.src.slice(start + 1, pos + 1); // Remove the backslash
  }

  state.pos = pos + 1;
  return true;
}

// Color close escape rule - handles \{/color}
function colorCloseEscapeRule(state: StateInline, silent: boolean): boolean {
  if (!FEATURE_FLAGS.colorSyntax) return false;

  const start = state.pos;
  const max = state.posMax;

  // Check for \{/color}
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

  // Quick fail if not starting with {color:
  if (start + 7 >= max) return false;
  if (state.src.charCodeAt(start) !== 0x7b /* { */) return false;
  if (state.src.slice(start, start + 7) !== '{color:') return false;

  // Find the closing } of the opening tag
  let pos = start + 7;
  let colorEnd = -1;

  while (pos < max && pos < start + 100) {
    // Limit search for color value
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

  // Extract and validate the color
  const colorValue = state.src.slice(start + 7, colorEnd);
  const sanitizedColor = sanitizeColor(colorValue);

  if (!sanitizedColor) {
    return false; // Invalid color, don't match
  }

  // Find the closing {/color}
  const contentStart = colorEnd + 1;
  let contentEnd = -1;
  const closingTag = '{/color}';
  const maxScan = Math.min(contentStart + COLOR_MAX_SCAN, max);

  pos = contentStart;
  while (pos < maxScan) {
    const char = state.src.charCodeAt(pos);

    // Don't allow newlines in color content (inline element only)
    if (char === 0x0a /* \n */) {
      return false;
    }

    // Skip escaped closing tags (\{/color})
    if (char === 0x5c /* \ */ && state.src.slice(pos + 1, pos + 9) === '{/color}') {
      pos += 9; // Skip the entire \{/color}
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
    // Create opening token
    const openToken = state.push('color_open', 'span', 1);
    openToken.meta = { color: sanitizedColor };

    // Parse inner content
    const content = state.src.slice(contentStart, contentEnd);
    if (content.length > 0) {
      // Parse inner content recursively
      const oldPos = state.pos;
      const oldMax = state.posMax;

      state.pos = contentStart;
      state.posMax = contentEnd;
      state.md.inline.tokenize(state);

      state.pos = oldPos;
      state.posMax = oldMax;
    }

    // Create closing token
    state.push('color_close', 'span', -1);
  }

  state.pos = contentEnd + closingTag.length;
  return true;
}

// Due date inline rule - parses @due(YYYY-MM-DD)
function dueDateRule(state: StateInline, silent: boolean): boolean {
  if (!FEATURE_FLAGS.dueDateSyntax) return false;

  const start = state.pos;
  const max = state.posMax;

  // Quick fail: must start with @
  if (state.src.charCodeAt(start) !== 0x40 /* @ */) return false;
  if (start + 5 >= max) return false;
  if (state.src.slice(start, start + 5) !== '@due(') return false;

  // Find closing paren (max 16 chars: @due( = 5, YYYY-MM-DD = 10, ) = 1)
  const closePos = state.src.indexOf(')', start + 5);
  if (closePos === -1 || closePos > start + 15) return false;

  const dateStr = state.src.slice(start + 5, closePos);
  if (!isValidDueDate(dateStr)) return false;

  if (!silent) {
    const token = state.push('due_date', 'span', 0);
    token.content = dateStr;
  }

  state.pos = closePos + 1;
  return true;
}

export interface RenderOptions {
  resolvedTitles?: Set<string>;
  titleToIdMap?: Map<string, string>;
}

// Wikilink inline rule - parses [[title]] and [[title|alias]]
function wikilinkRule(state: StateInline, silent: boolean): boolean {
  const start = state.pos;
  const max = state.posMax;

  // Quick fail if not starting with [[
  if (start + 3 >= max) return false;
  if (state.src.charCodeAt(start) !== 0x5b /* [ */) return false;
  if (state.src.charCodeAt(start + 1) !== 0x5b /* [ */) return false;

  // Find the closing ]]
  let pos = start + 2;
  let title = '';
  let alias = '';
  let foundPipe = false;

  while (pos < max) {
    const char = state.src.charCodeAt(pos);

    // Check for closing ]]
    if (char === 0x5d /* ] */ && pos + 1 < max && state.src.charCodeAt(pos + 1) === 0x5d /* ] */) {
      // Empty wikilinks not allowed
      if (title.length === 0) {
        return false;
      }

      // Validate - title should not be just whitespace
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

    // Check for pipe (alias separator)
    if (char === 0x7c /* | */ && !foundPipe) {
      foundPipe = true;
      pos++;
      continue;
    }

    // Don't allow newlines or ] inside the wikilink content
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

// Cached MarkdownIt instance — rules and plugins are static,
// dynamic data (titleToIdMap, widthMap) flows through the env parameter.
let cachedMd: MarkdownIt | null = null;

function getMarkdownInstance(): MarkdownIt {
  if (cachedMd) return cachedMd;

  const md = new MarkdownIt({
    html: false,
    linkify: true,
    typographer: true,
  });

  // Add task lists plugin (generates checkboxes with data-line attributes)
  if (FEATURE_FLAGS.taskLists) {
    md.use(taskLists, {
      enabled: true,
      label: true,
      lineNumber: true,
    });
  }

  // Add escape rules first (highest priority)
  if (FEATURE_FLAGS.colorSyntax) {
    md.inline.ruler.before('escape', 'color_escape', colorEscapeRule);
    md.inline.ruler.before('escape', 'color_close_escape', colorCloseEscapeRule);
  }

  // Add wikilink inline rule
  md.inline.ruler.before('link', 'wikilink', wikilinkRule);

  // Add due date inline rule (after wikilinks)
  if (FEATURE_FLAGS.dueDateSyntax) {
    md.inline.ruler.before('link', 'due_date', dueDateRule);
  }

  // Add color inline rule (after escape rules)
  if (FEATURE_FLAGS.colorSyntax) {
    md.inline.ruler.before('link', 'color', colorRule);
  }

  // Wikilink renderer — reads titleToIdMap from env for each render
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

    // Use note ID if available, otherwise fall back to title
    const noteId = titleToIdMap.get(titleLower);
    const href = noteId ? `/note/${noteId}` : `/note/${encodeURIComponent(title.trim())}`;

    // Escape HTML to prevent XSS
    const escapedTitle = escapeHtml(title.trim());
    const escapedDisplayText = escapeHtml(displayText);

    return `<a href="${href}" class="wikilink ${className}" data-title="${escapedTitle}">${escapedDisplayText}</a>`;
  };

  // Add color renderers
  if (FEATURE_FLAGS.colorSyntax) {
    md.renderer.rules.color_open = (tokens: Token[], idx: number): string => {
      const token = tokens[idx];
      const color = token.meta?.color as string;

      // Named colors use CSS classes, custom colors use inline styles
      if (NAMED_COLORS.has(color)) {
        return `<span class="text-color-${escapeHtml(color)}">`;
      } else {
        // Inline style for hex/rgb/rgba
        return `<span style="color: ${escapeHtml(color)};">`;
      }
    };

    md.renderer.rules.color_close = (): string => {
      return '</span>';
    };
  }

  // Add due date renderer
  if (FEATURE_FLAGS.dueDateSyntax) {
    md.renderer.rules.due_date = (tokens: Token[], idx: number): string => {
      const dateStr = tokens[idx].content;
      const status = getDueDateStatus(dateStr);
      return `<span class="due-date due-date-${status}" data-due-date="${escapeHtml(dateStr)}">${escapeHtml(dateStr)}</span>`;
    };
  }

  // Add custom image renderer with resizable wrapper
  if (FEATURE_FLAGS.imageResize) {
    md.renderer.rules.image = (
      tokens: Token[],
      idx: number,
      _options: MarkdownItOptions,
      env: MarkdownItEnv,
      _self: MarkdownItRenderer
    ): string => {
      const token = tokens[idx];
      const src = token.attrGet('src') || '';
      const alt = token.content || '';

      // Track image index for resize targeting
      const currentIndex = typeof env.imageIndex === 'number' ? env.imageIndex : 0;
      const imageIndex = currentIndex + 1;
      env.imageIndex = imageIndex;

      // Get width from widthMap (populated by extractImageWidths)
      const widthMap = env.widthMap as Map<number, string> | undefined;
      const width = widthMap?.get(imageIndex);

      // Build the HTML with wrapper
      let html = `<span class="resizable-image-wrapper" data-image-index="${imageIndex}">`;
      html += `<img src="${escapeHtml(src)}" alt="${escapeHtml(alt)}"`;
      html += ` data-original-src="${escapeHtml(src)}"`;
      html += ` data-image-index="${imageIndex}"`;

      if (width && WIDTH_VALUE_REGEX.test(width)) {
        const widthValue = width.includes('%') ? width : `${width}px`;
        html += ` width="${escapeHtml(width)}"`;
        html += ` style="width: ${escapeHtml(widthValue)}"`;
      }

      html += `>`;
      html += `<span class="resize-handle"></span>`;
      html += `</span>`;

      return html;
    };
  }

  // Heading renderer with IDs for TOC anchor links.
  // slugCounts is reset per render via env to avoid cross-render collisions.
  md.renderer.rules.heading_open = (
    tokens: Token[],
    idx: number,
    _options: MarkdownItOptions,
    env: MarkdownItEnv
  ): string => {
    const token = tokens[idx];
    const level = token.tag; // h1, h2, etc.

    // Get the heading text from the next inline token
    const inlineToken = tokens[idx + 1];
    let text = '';
    if (inlineToken && inlineToken.children) {
      text = inlineToken.children
        .filter((t) => t.type === 'text' || t.type === 'code_inline')
        .map((t) => t.content)
        .join('');
    }

    // Generate slug
    let slug = text
      .toLowerCase()
      .replace(/[^\w\s-]/g, '')
      .replace(/\s+/g, '-');

    // Handle duplicate slugs (per-render counter stored in env)
    const slugCounts = ((env._slugCounts as Map<string, number>) ??= new Map<string, number>());
    const count = slugCounts.get(slug) || 0;
    slugCounts.set(slug, count + 1);
    if (count > 0) {
      slug = `${slug}-${count}`;
    }

    return `<${level} id="${escapeHtml(slug)}">`;
  };

  cachedMd = md;
  return md;
}

/**
 * Sanitizes rendered HTML to prevent XSS attacks
 * Defense-in-depth: markdown-it already has html:false, but this adds extra layer
 */
function sanitizeRenderedHtml(html: string): string {
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: [
      // Text formatting
      'p',
      'br',
      'strong',
      'em',
      'u',
      's',
      'del',
      'code',
      'pre',
      // Headings
      'h1',
      'h2',
      'h3',
      'h4',
      'h5',
      'h6',
      // Lists
      'ul',
      'ol',
      'li',
      // Other
      'blockquote',
      'hr',
      // Links and spans
      'a',
      'span',
      // Task lists
      'input',
      'label',
      // Tables
      'table',
      'thead',
      'tbody',
      'tr',
      'th',
      'td',
      // Images
      'img',
      // SVG for drag handles
      'svg',
      'circle',
      'path',
      'line',
      'rect',
      'polyline',
      'polygon',
      'g',
    ],
    ALLOWED_ATTR: [
      'href',
      'title',
      'class',
      'data-*',
      'type',
      'checked',
      'disabled',
      'data-line',
      // Table attributes
      'align',
      'colspan',
      'rowspan',
      // Image attributes
      'src',
      'alt',
      'width',
      'height',
      // Span attributes (for color syntax)
      'style',
      'id',
      // SVG attributes for drag handle icons
      'xmlns',
      'viewBox',
      'fill',
      'stroke',
      'stroke-width',
      'stroke-linecap',
      'stroke-linejoin',
      'cx',
      'cy',
      'r',
      'd',
      'x',
      'y',
      'x1',
      'x2',
      'y1',
      'y2',
      'points',
    ],
    ALLOWED_URI_REGEXP:
      /^(?:(?:(?:f|ht)tps?|mailto|tel|callto|sms|cid|xmpp|#):|[^a-z]|[a-z+.-]+(?:[^a-z+.:-]|$))/i,
    ALLOW_DATA_ATTR: true,
    KEEP_CONTENT: true,
  });
}

// SVG icon for drag handle (GripVertical from lucide)
const DRAG_HANDLE_SVG = `<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="12" r="1"/><circle cx="9" cy="5" r="1"/><circle cx="9" cy="19" r="1"/><circle cx="15" cy="12" r="1"/><circle cx="15" cy="5" r="1"/><circle cx="15" cy="19" r="1"/></svg>`;

/**
 * Extract image widths from content and store in a map.
 * Returns the content with {width=...} suffixes removed.
 */
function extractImageWidths(content: string): {
  cleanContent: string;
  widthMap: Map<number, string>;
} {
  const widthMap = new Map<number, string>();
  let imageIndex = 0;

  // First pass: count images to build index map
  const imagePattern = /!\[[^\]]*\]\([^)]+\)(?:\{width=\d+%?\})?/g;
  let match;

  while ((match = imagePattern.exec(content)) !== null) {
    imageIndex++;
    const fullMatch = match[0];
    const widthMatch = fullMatch.match(/\{width=(\d+%?)\}$/);
    if (widthMatch && WIDTH_VALUE_REGEX.test(widthMatch[1])) {
      widthMap.set(imageIndex, widthMatch[1]);
    }
  }

  // Remove {width=...} from content for clean markdown parsing
  const cleanContent = content.replace(/(!\[[^\]]*\]\([^)]+\))\{width=\d+%?\}/g, '$1');

  return { cleanContent, widthMap };
}

/**
 * Add drag handles and data-task-index attributes to task list items.
 * This post-processes the HTML after markdown-it rendering.
 */
function getRenderedTaskLineNumbers(content: string): number[] {
  const lines = content.split('\n');
  const taskLines: number[] = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const match = /^(\s*(?:[-*+]|\d+[.)]) )\[([xX ])\]/.exec(line);
    if (!match) continue;

    // Match markdown-it-task-lists behavior: no checkbox for empty tasks.
    const taskBody = line.substring(match[0].length).trim();
    if (!taskBody) continue;
    taskLines.push(i + 1); // 1-based line number
  }

  return taskLines;
}

function addDragHandlesToTasks(html: string, taskLines: number[]): string {
  let taskIndex = 0;

  // Match task list items and add drag handle + data-task-index
  // Pattern: <li class="task-list-item...">
  return html.replace(/<li class="task-list-item([^"]*)">/g, (match, existingClasses) => {
    const index = taskIndex++;
    const handle = `<span class="drag-handle" aria-hidden="true">${DRAG_HANDLE_SVG}</span>`;
    const line = taskLines[index];
    const lineAttr = Number.isInteger(line) ? ` data-task-line="${line}"` : '';
    return `<li class="task-list-item${existingClasses}" data-task-index="${index}"${lineAttr}>${handle}`;
  });
}

export function renderMarkdown(content: string, options: RenderOptions = {}): string {
  const md = getMarkdownInstance();

  // Extract image widths and clean content for parsing
  let processedContent = content;
  let widthMap = new Map<number, string>();

  if (FEATURE_FLAGS.imageResize) {
    const extracted = extractImageWidths(content);
    processedContent = extracted.cleanContent;
    widthMap = extracted.widthMap;
  }

  // Pass dynamic data through env (titleToIdMap, resolvedTitles, widthMap)
  const env: MarkdownItEnv = {
    widthMap,
    titleToIdMap: options.titleToIdMap,
    resolvedTitles: options.resolvedTitles,
  };
  let rendered = md.render(processedContent, env);

  // Add drag handles to task items for drag & drop support
  if (FEATURE_FLAGS.taskLists) {
    rendered = addDragHandlesToTasks(rendered, getRenderedTaskLineNumbers(processedContent));
  }

  // Apply DOMPurify layer for defense-in-depth
  // (markdown-it already has html:false, but this protects against future bugs)
  return sanitizeRenderedHtml(rendered);
}

// Table of Contents types and extraction
export interface TocEntry {
  level: number;
  text: string;
  slug: string;
}

export function extractHeadings(content: string): TocEntry[] {
  const headings: TocEntry[] = [];
  const lines = content.split('\n');
  const slugCounts = new Map<string, number>();

  for (const line of lines) {
    const match = line.match(/^(#{1,6})\s+(.+)$/);
    if (match) {
      const level = match[1].length;
      const text = match[2].trim();
      // Generate slug: lowercase, remove special chars, replace spaces with hyphens
      let slug = text
        .toLowerCase()
        .replace(/[^\w\s-]/g, '')
        .replace(/\s+/g, '-');

      // Handle duplicate slugs by appending a number
      const count = slugCounts.get(slug) || 0;
      slugCounts.set(slug, count + 1);
      if (count > 0) {
        slug = `${slug}-${count}`;
      }

      headings.push({ level, text, slug });
    }
  }
  return headings;
}

// Extract all valid @due(YYYY-MM-DD) dates from content (ignoring code blocks)
export function extractDueDates(content: string): string[] {
  const dates: string[] = [];
  let inCodeBlock = false;
  let inInlineCode = false;

  const lines = content.split('\n');
  for (const line of lines) {
    if (line.trimStart().startsWith('```')) {
      inCodeBlock = !inCodeBlock;
      continue;
    }
    if (inCodeBlock) continue;

    // Process line character by character to skip inline code
    let i = 0;
    while (i < line.length) {
      if (line[i] === '`') {
        inInlineCode = !inInlineCode;
        i++;
        continue;
      }
      if (inInlineCode) {
        i++;
        continue;
      }
      if (line[i] === '@' && line.slice(i, i + 5) === '@due(') {
        const closeIdx = line.indexOf(')', i + 5);
        if (closeIdx !== -1 && closeIdx <= i + 15) {
          const dateStr = line.slice(i + 5, closeIdx);
          if (isValidDueDate(dateStr)) {
            dates.push(dateStr);
          }
        }
      }
      i++;
    }
    inInlineCode = false; // Reset at end of line
  }

  return dates;
}

// Detailed due date info for server sync (matches backend parser.DueDate struct)
export interface DueDateInfo {
  due_date: string;
  line_text: string;
  line_index: number;
  is_task_item: boolean;
  is_completed: boolean;
}

const dueDateCleanupRegex = /@due\([^)]*\)/g;
const checkboxRegex = /^\s*(?:[-*+]|\d+[.)]) \[([xX ])\]\s*/;
const listPrefixRegex = /^\s*(?:[-*+]|\d+[.)]) (?:\[[xX ]\]\s*)?/;

// Extract due dates with full metadata for server sync (used by encrypted notes)
export function extractDueDatesDetailed(content: string): DueDateInfo[] {
  const results: DueDateInfo[] = [];
  const lines = content.split('\n');
  let inCodeBlock = false;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const trimmed = line.trimStart();

    if (trimmed.startsWith('```')) {
      inCodeBlock = !inCodeBlock;
      continue;
    }
    if (inCodeBlock) continue;

    const dateRegex = /@due\((\d{4}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01]))\)/g;
    let match;
    const matches: string[] = [];
    while ((match = dateRegex.exec(line)) !== null) {
      matches.push(match[1]);
    }
    if (matches.length === 0) continue;

    const cbMatch = checkboxRegex.exec(line);
    const isTask = cbMatch !== null;
    const isCompleted = isTask && (cbMatch![1] === 'x' || cbMatch![1] === 'X');

    for (const dateStr of matches) {
      if (!isValidDueDate(dateStr)) continue;

      let cleanText = line.replace(dueDateCleanupRegex, '');
      if (isTask) {
        cleanText = cleanText.replace(listPrefixRegex, '');
      }
      cleanText = cleanText.trim();

      results.push({
        due_date: dateStr,
        line_text: cleanText,
        line_index: i,
        is_task_item: isTask,
        is_completed: isCompleted,
      });
    }
  }

  return results;
}

// Parse wikilinks from content (for extracting link targets)
export function extractWikilinks(content: string): Array<{ title: string; alias?: string }> {
  const links: Array<{ title: string; alias?: string }> = [];
  const regex = /\[\[([^\]|]+)(?:\|([^\]]+))?\]\]/g;
  let match;

  while ((match = regex.exec(content)) !== null) {
    links.push({
      title: match[1].trim(),
      alias: match[2]?.trim(),
    });
  }

  return links;
}
