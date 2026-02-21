// Markdown rendering with wikilink, color, due-date, and image resize support.
// This module orchestrates plugin registration and re-exports the public API.

import type { Options as MarkdownItOptions } from 'markdown-it';
import MarkdownIt from 'markdown-it';
import type Token from 'markdown-it/lib/token.mjs';
import taskLists from 'markdown-it-task-lists';

import { FEATURE_FLAGS } from '$lib/config';

import * as colorPlugin from './markdown/color-plugin';
import * as duedatePlugin from './markdown/duedate-plugin';
import { sanitizeRenderedHtml } from './markdown/html-sanitizer';
import * as imagePlugin from './markdown/image-plugin';
import { addDragHandlesToTasks, getRenderedTaskLineNumbers } from './markdown/task-processor';
import * as wikilinkPlugin from './markdown/wikilink-plugin';

// Re-export public API from sub-modules
export { sanitizeColor } from './markdown/color-plugin';
export { getDueDateStatus, isValidDueDate } from './markdown/duedate-plugin';
export type { DueDateInfo, TocEntry } from './markdown/extractors';
export {
  extractDueDates,
  extractDueDatesDetailed,
  extractHeadings,
  extractWikilinks,
} from './markdown/extractors';
export { extractImageWidths } from './markdown/image-plugin';

type MarkdownItEnv = Record<string, unknown>;

export interface RenderOptions {
  resolvedTitles?: Set<string>;
  titleToIdMap?: Map<string, string>;
}

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

  // Register plugins (order matters: escape rules first, then inline rules)
  colorPlugin.register(md, escapeHtml);
  wikilinkPlugin.register(md, escapeHtml);
  duedatePlugin.register(md, escapeHtml);
  imagePlugin.register(md, escapeHtml);

  // Heading renderer with IDs for TOC anchor links.
  // slugCounts is reset per render via env to avoid cross-render collisions.
  md.renderer.rules.heading_open = (
    tokens: Token[],
    idx: number,
    _options: MarkdownItOptions,
    env: MarkdownItEnv
  ): string => {
    const token = tokens[idx];
    const level = token.tag;

    const inlineToken = tokens[idx + 1];
    let text = '';
    if (inlineToken && inlineToken.children) {
      text = inlineToken.children
        .filter((t) => t.type === 'text' || t.type === 'code_inline')
        .map((t) => t.content)
        .join('');
    }

    let slug = text
      .toLowerCase()
      .replace(/[^\w\s-]/g, '')
      .replace(/\s+/g, '-');

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

export function renderMarkdown(content: string, options: RenderOptions = {}): string {
  const md = getMarkdownInstance();

  // Extract image widths and clean content for parsing
  let processedContent = content;
  let widthMap = new Map<number, string>();

  if (FEATURE_FLAGS.imageResize) {
    const extracted = imagePlugin.extractImageWidths(content);
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
  return sanitizeRenderedHtml(rendered);
}
