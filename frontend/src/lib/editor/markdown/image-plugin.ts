// Image renderer plugin for markdown-it with resize support

import type MarkdownIt from 'markdown-it';
import type { Options as MarkdownItOptions } from 'markdown-it';
import type Token from 'markdown-it/lib/token.mjs';

import { FEATURE_FLAGS } from '$lib/config';

type MarkdownItEnv = Record<string, unknown>;
type MarkdownItRenderer = unknown;

// Regex for validating image width values (XSS protection)
const WIDTH_VALUE_REGEX = /^\d+%?$/;

/**
 * Extract image widths from content and store in a map.
 * Returns the content with {width=...} suffixes removed.
 */
export function extractImageWidths(content: string): {
  cleanContent: string;
  widthMap: Map<number, string>;
} {
  const widthMap = new Map<number, string>();
  let imageIndex = 0;

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

  const cleanContent = content.replace(/(!\[[^\]]*\]\([^)]+\))\{width=\d+%?\}/g, '$1');

  return { cleanContent, widthMap };
}

/** Register image renderer plugin with a MarkdownIt instance. */
export function register(md: MarkdownIt, escapeHtml: (s: string) => string): void {
  if (!FEATURE_FLAGS.imageResize) return;

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

    const currentIndex = typeof env.imageIndex === 'number' ? env.imageIndex : 0;
    const imageIndex = currentIndex + 1;
    env.imageIndex = imageIndex;

    const widthMap = env.widthMap as Map<number, string> | undefined;
    const width = widthMap?.get(imageIndex);

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
