import type { TreeDueDateFeature, TreeWikilinkFeature } from './types';

export function scanWikilinksFromLine(
  text: string,
  lineNumber: number,
  baseOffset: number
): TreeWikilinkFeature[] {
  const result: TreeWikilinkFeature[] = [];
  let i = 0;
  while (i < text.length) {
    const start = text.indexOf('[[', i);
    if (start === -1) break;
    const end = text.indexOf(']]', start + 2);
    if (end === -1) break;

    const inner = text.slice(start + 2, end).trim();
    if (inner.length > 0) {
      const separator = inner.indexOf('|');
      const title = (separator === -1 ? inner : inner.slice(0, separator)).trim();
      const label = (separator === -1 ? inner : inner.slice(separator + 1)).trim();
      if (title.length > 0) {
        result.push({
          line: lineNumber,
          from: baseOffset + start,
          to: baseOffset + end + 2,
          title,
          label: label.length > 0 ? label : title,
        });
      }
    }

    i = end + 2;
  }
  return result;
}

function isIsoDateLiteral(value: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(value);
}

export function scanDueDatesFromLine(
  text: string,
  lineNumber: number,
  baseOffset: number
): TreeDueDateFeature[] {
  const result: TreeDueDateFeature[] = [];
  let i = 0;
  while (i < text.length) {
    const start = text.indexOf('@due(', i);
    if (start === -1) break;

    const openEnd = start + 5;
    const close = text.indexOf(')', openEnd);
    if (close === -1) break;

    const date = text.slice(openEnd, close);
    if (isIsoDateLiteral(date)) {
      result.push({
        line: lineNumber,
        from: baseOffset + start,
        to: baseOffset + close + 1,
        date,
      });
    }
    i = close + 1;
  }
  return result;
}
