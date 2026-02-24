// Performance baseline benchmarks for the preview rendering pipeline.
// Run with: npx vitest bench src/lib/editor/__benchmarks__/

// @vitest-environment jsdom

import { bench, describe } from 'vitest';

import { renderMarkdown, renderMarkdownUnsanitized } from '../markdown';
import { sanitizeRenderedHtml } from '../markdown/html-sanitizer';

function generateDoc(lineCount: number): string {
  const lines: string[] = ['# Benchmark Document'];

  for (let i = 0; i < lineCount; i++) {
    const mod = i % 10;
    if (mod === 0) lines.push(`## Section ${i / 10}`);
    else if (mod === 1) lines.push(`- [ ] Open task ${i}`);
    else if (mod === 2) lines.push(`- [x] Done task ${i} @due(2026-03-15)`);
    else if (mod === 3)
      lines.push(`Paragraph with [[WikiLink${i}]] and [link](https://example.com/${i})`);
    else if (mod === 4) lines.push(`{color:primary}colored text ${i}{/color}`);
    else if (mod === 5) lines.push(`![image](https://example.com/${i}.png){width=300}`);
    else if (mod === 6) {
      lines.push('| Col A | Col B |');
      lines.push('| --- | --- |');
      lines.push(`| ${i} | data |`);
    } else if (mod === 7) {
      lines.push('```typescript');
      lines.push(`const x${i} = ${i};`);
      lines.push('```');
    } else lines.push(`Regular paragraph ${i} with **bold** and *italic* text.`);
  }

  return lines.join('\n');
}

const doc100 = generateDoc(100);
const doc500 = generateDoc(500);
const doc2000 = generateDoc(2000);

// Pre-render for sanitize-only benchmarks
const html100 = renderMarkdownUnsanitized(doc100);
const html500 = renderMarkdownUnsanitized(doc500);
const html2000 = renderMarkdownUnsanitized(doc2000);

describe('renderMarkdown (total)', () => {
  bench('100 lines', () => {
    renderMarkdown(doc100);
  });

  bench('500 lines', () => {
    renderMarkdown(doc500);
  });

  bench('2000 lines', () => {
    renderMarkdown(doc2000);
  });
});

describe('markdown-it parse+render (without sanitize)', () => {
  bench('100 lines', () => {
    renderMarkdownUnsanitized(doc100);
  });

  bench('500 lines', () => {
    renderMarkdownUnsanitized(doc500);
  });

  bench('2000 lines', () => {
    renderMarkdownUnsanitized(doc2000);
  });
});

describe('DOMPurify sanitize only', () => {
  bench('100 lines', () => {
    sanitizeRenderedHtml(html100);
  });

  bench('500 lines', () => {
    sanitizeRenderedHtml(html500);
  });

  bench('2000 lines', () => {
    sanitizeRenderedHtml(html2000);
  });
});
