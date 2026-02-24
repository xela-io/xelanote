// HTML sanitization for rendered markdown (defense-in-depth)

import DOMPurify from 'isomorphic-dompurify';

/**
 * Sanitizes rendered HTML to prevent XSS attacks.
 * Defense-in-depth: markdown-it already has html:false, but this adds extra layer.
 */
export function sanitizeRenderedHtml(html: string): string {
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
      // Div (for math blocks)
      'div',
      // KaTeX / MathML
      'math',
      'mrow',
      'mi',
      'mo',
      'mn',
      'msup',
      'msub',
      'mfrac',
      'msqrt',
      'mtext',
      'mspace',
      'mover',
      'munder',
      'munderover',
      'mtable',
      'mtr',
      'mtd',
      'annotation',
      'semantics',
      // SVG for drag handles and Mermaid diagrams
      'svg',
      'circle',
      'path',
      'line',
      'rect',
      'polyline',
      'polygon',
      'g',
      'text',
      'tspan',
      'defs',
      'clipPath',
      'marker',
      'foreignObject',
      'use',
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
      'loading',
      'decoding',
      // Span attributes (for color syntax)
      'style',
      'id',
      // MathML attributes
      'encoding',
      'mathvariant',
      'stretchy',
      'fence',
      'separator',
      'lspace',
      'rspace',
      'accent',
      'accentunder',
      'columnalign',
      'columnspacing',
      'rowspacing',
      'depth',
      'minsize',
      'maxsize',
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

/**
 * Sanitize SVG output from Mermaid diagrams.
 * Mermaid renders user input, so we must sanitize the SVG to prevent XSS.
 */
export function sanitizeSvg(svg: string): string {
  return DOMPurify.sanitize(svg, {
    USE_PROFILES: { svg: true, svgFilters: true },
    ADD_TAGS: ['foreignObject', 'style'],
    ADD_ATTR: [
      'dominant-baseline',
      'text-anchor',
      'transform',
      'marker-end',
      'marker-start',
      'font-size',
      'font-family',
      'font-weight',
      'alignment-baseline',
    ],
    ALLOW_DATA_ATTR: true,
  });
}
