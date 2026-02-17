import { syntaxTree } from '@codemirror/language';
import type { EditorView } from '@codemirror/view';

import { scanDueDatesFromLine, scanWikilinksFromLine } from './scanners';
import type {
  TreeDueDateFeature,
  TreeFeatureCollection,
  TreeInlineFeature,
  TreeLinkFeature,
  TreeTaskFeature,
  TreeWikilinkFeature,
} from './types';

function parseLinkFromRange(text: string): { label: string; href: string } | null {
  if (!text.startsWith('[') || !text.endsWith(')')) return null;
  const split = text.indexOf('](');
  if (split <= 0) return null;
  const label = text.slice(1, split);
  const href = text.slice(split + 2, -1).trim();
  if (!href) return null;
  return { label, href };
}

function stripDelimitedText(text: string, kind: 'code' | 'strong' | 'em'): string {
  if (kind === 'code') {
    const match = /^(`+)([\s\S]*?)\1$/.exec(text);
    return match ? match[2] : text;
  }
  if (kind === 'strong') {
    if (
      (text.startsWith('**') && text.endsWith('**')) ||
      (text.startsWith('__') && text.endsWith('__'))
    ) {
      return text.slice(2, -2);
    }
    return text;
  }
  if (
    (text.startsWith('*') && text.endsWith('*')) ||
    (text.startsWith('_') && text.endsWith('_'))
  ) {
    return text.slice(1, -1);
  }
  return text;
}

export function collectTreeFeatures(view: EditorView): TreeFeatureCollection {
  const tasksByLine = new Map<number, TreeTaskFeature>();
  const linksByLine = new Map<number, TreeLinkFeature[]>();
  const inlineByLine = new Map<number, TreeInlineFeature[]>();
  const wikilinksByLine = new Map<number, TreeWikilinkFeature[]>();
  const dueDatesByLine = new Map<number, TreeDueDateFeature[]>();
  const tree = syntaxTree(view.state);
  const doc = view.state.doc;

  for (const { from, to } of view.visibleRanges) {
    tree.iterate({
      from,
      to,
      enter: (node) => {
        if (node.type.name === 'TaskMarker') {
          const line = doc.lineAt(node.from);
          const taskBody = line.text.slice(node.to - line.from).trim();
          if (taskBody.length === 0 || tasksByLine.has(line.number)) return;
          const marker = doc.sliceString(node.from, node.to).toLowerCase();
          tasksByLine.set(line.number, {
            line: line.number,
            from: node.from,
            to: node.to,
            checked: marker === '[x]',
          });
          return;
        }

        if (node.type.name === 'Link') {
          const full = doc.sliceString(node.from, node.to);
          const parsed = parseLinkFromRange(full);
          if (!parsed) return;
          const line = doc.lineAt(node.from);
          const list = linksByLine.get(line.number) ?? [];
          list.push({
            line: line.number,
            from: node.from,
            to: node.to,
            label: parsed.label,
            href: parsed.href,
          });
          linksByLine.set(line.number, list);
          return;
        }

        if (
          node.type.name === 'InlineCode' ||
          node.type.name === 'StrongEmphasis' ||
          node.type.name === 'Emphasis'
        ) {
          const startLine = doc.lineAt(node.from);
          const endLine = doc.lineAt(node.to);
          if (startLine.number !== endLine.number) return;

          const full = doc.sliceString(node.from, node.to);
          if (!full) return;

          const kind =
            node.type.name === 'InlineCode'
              ? 'code'
              : node.type.name === 'StrongEmphasis'
                ? 'strong'
                : 'em';
          const parsedText = stripDelimitedText(full, kind);
          if (!parsedText) return;

          const className =
            kind === 'code'
              ? 'cm-live-preview-code'
              : kind === 'strong'
                ? 'cm-live-preview-strong'
                : 'cm-live-preview-em';
          const list = inlineByLine.get(startLine.number) ?? [];
          list.push({
            line: startLine.number,
            from: node.from,
            to: node.to,
            text: parsedText,
            className,
          });
          inlineByLine.set(startLine.number, list);
        }
      },
    });

    let pos = from;
    while (pos <= to && pos <= doc.length) {
      const line = doc.lineAt(pos);
      pos = line.to + 1;

      const wikilinks = scanWikilinksFromLine(line.text, line.number, line.from);
      if (wikilinks.length > 0) {
        wikilinksByLine.set(line.number, wikilinks);
      }

      const dueDates = scanDueDatesFromLine(line.text, line.number, line.from);
      if (dueDates.length > 0) {
        dueDatesByLine.set(line.number, dueDates);
      }
    }
  }

  return {
    tasksByLine,
    linksByLine,
    inlineByLine,
    wikilinksByLine,
    dueDatesByLine,
  };
}
