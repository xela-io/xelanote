// Markdown rendering with wikilink, color, due-date, and image resize support.
// This module orchestrates plugin registration and re-exports the public API.

import MarkdownIt from 'markdown-it';

import { FEATURE_FLAGS } from '$lib/config';

import { sanitizeRenderedHtml } from './markdown/html-sanitizer';
import * as imagePlugin from './markdown/image-plugin';
import { addDragHandlesToTasks, getRenderedTaskLineNumbers } from './markdown/task-processor';
import { createConfiguredMarkdownIt } from './markdown-config';

// Re-export public API from sub-modules
export { migrateLegacyEncryptedAttachmentLinks } from './encrypted-attachment-markdown';
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

// Cached MarkdownIt instance — rules and plugins are static,
// dynamic data (titleToIdMap, widthMap) flows through the env parameter.
let cachedMd: MarkdownIt | null = null;

function getMarkdownInstance(): MarkdownIt {
  if (cachedMd) return cachedMd;
  cachedMd = createConfiguredMarkdownIt();
  return cachedMd;
}

/**
 * Render markdown without DOMPurify sanitization.
 * Exported for benchmarking only — do NOT use in production code.
 */
export function renderMarkdownUnsanitized(content: string, options: RenderOptions = {}): string {
  const md = getMarkdownInstance();

  let processedContent = content;
  let widthMap = new Map<number, string>();

  if (FEATURE_FLAGS.imageResize) {
    const extracted = imagePlugin.extractImageWidths(content);
    processedContent = extracted.cleanContent;
    widthMap = extracted.widthMap;
  }

  const env: MarkdownItEnv = {
    widthMap,
    titleToIdMap: options.titleToIdMap,
    resolvedTitles: options.resolvedTitles,
  };
  let rendered = md.render(processedContent, env);

  if (FEATURE_FLAGS.taskLists) {
    rendered = addDragHandlesToTasks(rendered, getRenderedTaskLineNumbers(processedContent));
  }

  return rendered;
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
