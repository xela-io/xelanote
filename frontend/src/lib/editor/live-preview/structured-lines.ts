// Structured line detection: code fences, code content, and table lines

import type { ViewUpdate } from '@codemirror/view';
import { EditorView } from '@codemirror/view';

import {
  collectTableBlocks,
  isCodeFence,
  isTableCandidateLine,
  isTableSeparatorLine,
  type TableBlock,
} from './table-parser';

export interface StructuredLines {
  codeFenceLines: Set<number>;
  codeContentLines: Set<number>;
  tableLines: Set<number>;
  tableBlocks: TableBlock[];
}

const STRUCTURED_LINE_TRIGGER_RE = /[`|:-]/;

export function textMayAffectStructuredLines(text: string): boolean {
  return STRUCTURED_LINE_TRIGGER_RE.test(text);
}

function hasAnyLineInRange(lines: Set<number>, fromLine: number, toLine: number): boolean {
  for (let line = fromLine; line <= toLine; line++) {
    if (lines.has(line)) return true;
  }
  return false;
}

export function shouldRecomputeStructuredLines(
  update: ViewUpdate,
  previous: StructuredLines
): boolean {
  if (!update.docChanged) return false;
  if (update.startState.doc.lines !== update.state.doc.lines) return true;

  let needsRecompute = false;
  update.changes.iterChanges((fromA, toA, _fromB, _toB, inserted) => {
    if (needsRecompute) return;

    const removedText = update.startState.doc.sliceString(fromA, toA);
    const insertedText = inserted.toString();
    if (textMayAffectStructuredLines(removedText) || textMayAffectStructuredLines(insertedText)) {
      needsRecompute = true;
      return;
    }

    const oldDoc = update.startState.doc;
    const fromProbe = Math.min(fromA, oldDoc.length);
    const toProbe = Math.min(Math.max(toA - 1, fromA), oldDoc.length);
    const fromLine = oldDoc.lineAt(fromProbe).number;
    const toLine = oldDoc.lineAt(toProbe).number;
    const nearbyFrom = Math.max(1, fromLine - 1);
    const nearbyTo = Math.min(oldDoc.lines, toLine + 1);

    if (
      hasAnyLineInRange(previous.codeFenceLines, nearbyFrom, nearbyTo) ||
      hasAnyLineInRange(previous.tableLines, nearbyFrom, nearbyTo)
    ) {
      needsRecompute = true;
    }
  });

  return needsRecompute;
}

export function collectStructuredLines(view: EditorView): StructuredLines {
  const codeFenceLines = new Set<number>();
  const codeContentLines = new Set<number>();
  const tableLines = new Set<number>();
  const lines: string[] = [];

  let inCodeBlock = false;
  for (let i = 1; i <= view.state.doc.lines; i++) {
    const text = view.state.doc.line(i).text;
    lines.push(text);

    if (isCodeFence(text)) {
      codeFenceLines.add(i);
      inCodeBlock = !inCodeBlock;
      continue;
    }
    if (inCodeBlock) {
      codeContentLines.add(i);
    }
  }

  for (let i = 0; i < lines.length; i++) {
    const lineNo = i + 1;
    if (codeFenceLines.has(lineNo) || codeContentLines.has(lineNo)) continue;

    const text = lines[i];
    if (!isTableCandidateLine(text) && !isTableSeparatorLine(text)) continue;

    const prev = i > 0 ? lines[i - 1] : '';
    const next = i < lines.length - 1 ? lines[i + 1] : '';
    const nearSeparator =
      isTableSeparatorLine(text) || isTableSeparatorLine(prev) || isTableSeparatorLine(next);

    if (nearSeparator) {
      tableLines.add(lineNo);
      if (i > 0 && isTableCandidateLine(prev)) tableLines.add(lineNo - 1);
      if (i < lines.length - 1 && isTableCandidateLine(next)) tableLines.add(lineNo + 1);
    }
  }

  const tableBlocks = collectTableBlocks(tableLines, lines);

  return { codeFenceLines, codeContentLines, tableLines, tableBlocks };
}
