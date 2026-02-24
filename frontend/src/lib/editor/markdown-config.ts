// Shared markdown-it configuration used by both main thread and web worker.
// This module sets up plugins and renderers without importing DOMPurify.

import type { Options as MarkdownItOptions } from 'markdown-it';
import MarkdownIt from 'markdown-it';
import type Token from 'markdown-it/lib/token.mjs';
import taskLists from 'markdown-it-task-lists';

import { FEATURE_FLAGS } from '$lib/config';

import * as codeHighlightPlugin from './markdown/code-highlight-plugin';
import * as colorPlugin from './markdown/color-plugin';
import * as duedatePlugin from './markdown/duedate-plugin';
import * as imagePlugin from './markdown/image-plugin';
import * as mathPlugin from './markdown/math-plugin';
import * as wikilinkPlugin from './markdown/wikilink-plugin';

type MarkdownItEnv = Record<string, unknown>;

// HTML escape function to prevent XSS
export function escapeHtml(text: string): string {
  const map: { [key: string]: string } = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#039;',
  };
  return text.replace(/[&<>"']/g, (m) => map[m]);
}

/**
 * Create and configure a MarkdownIt instance with all plugins and renderers.
 * Shared between main thread and web worker.
 */
export function createConfiguredMarkdownIt(): MarkdownIt {
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

  // Code block highlighting (marks blocks for deferred Shiki processing)
  if (FEATURE_FLAGS.shikiHighlight) {
    codeHighlightPlugin.register(md);
  }

  // Math rendering (marks expressions for deferred KaTeX processing)
  if (FEATURE_FLAGS.mathRendering) {
    mathPlugin.register(md);
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

    const sourceLine = token.map ? token.map[0] + 1 : '';
    return `<${level} id="${escapeHtml(slug)}" data-source-line="${sourceLine}">`;
  };

  // Paragraph renderer with source line for scroll sync
  md.renderer.rules.paragraph_open = (tokens: Token[], idx: number): string => {
    const token = tokens[idx];
    const sourceLine = token.map ? token.map[0] + 1 : '';
    return `<p data-source-line="${sourceLine}">`;
  };

  return md;
}
