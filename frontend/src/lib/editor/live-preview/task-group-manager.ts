// Completed task group collapse management

import { EditorView } from '@codemirror/view';

import { parseLinePrimitives } from './line-primitives';

export interface CompletedTaskGroup {
  key: string;
  startLine: number;
  endLine: number;
  count: number;
  collapsed: boolean;
}

export interface CompletedTaskGroupInfo {
  groups: CompletedTaskGroup[];
  groupByLine: Map<number, CompletedTaskGroup>;
  keys: Set<string>;
}

export function buildTaskGroupKey(view: EditorView, startLine: number, endLine: number): string {
  let content = '';
  const anchorLine = startLine > 1 ? view.state.doc.line(startLine - 1).text.trim() : '';
  if (anchorLine) content += `anchor:${anchorLine}\n`;
  for (let lineNo = startLine; lineNo <= endLine; lineNo++) {
    content += `${view.state.doc.line(lineNo).text.trim()}\n`;
  }
  let hash = 2166136261;
  for (let i = 0; i < content.length; i++) {
    hash ^= content.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return `tasks:${(hash >>> 0).toString(36)}`;
}

export function collectCompletedTaskGroups(
  view: EditorView,
  collapsedTaskGroups: Set<string>
): CompletedTaskGroupInfo {
  const groups: CompletedTaskGroup[] = [];
  const groupByLine = new Map<number, CompletedTaskGroup>();
  const keys = new Set<string>();

  let runStart = -1;
  let runCount = 0;

  const flushRun = (endLine: number) => {
    if (runCount >= 2) {
      const key = buildTaskGroupKey(view, runStart, endLine);
      keys.add(key);
      const group: CompletedTaskGroup = {
        key,
        startLine: runStart,
        endLine,
        count: runCount,
        collapsed: collapsedTaskGroups.has(key),
      };
      groups.push(group);
      for (let l = runStart; l <= endLine; l++) {
        groupByLine.set(l, group);
      }
    }
    runStart = -1;
    runCount = 0;
  };

  for (let lineNo = 1; lineNo <= view.state.doc.lines; lineNo++) {
    const text = view.state.doc.line(lineNo).text;
    const primitives = parseLinePrimitives(text);
    if (primitives.taskRegex?.checked) {
      if (runStart === -1) runStart = lineNo;
      runCount++;
    } else {
      if (runStart !== -1) flushRun(lineNo - 1);
    }
  }
  if (runStart !== -1) flushRun(view.state.doc.lines);

  return { groups, groupByLine, keys };
}

export function rangesOverlap(
  aStart: number,
  aEnd: number,
  bStart: number,
  bEnd: number,
  tolerance = 0
): boolean {
  return aStart <= bEnd + tolerance && bStart <= aEnd + tolerance;
}

export function remapCollapsedTaskGroups(
  previousInfo: CompletedTaskGroupInfo,
  nextInfo: CompletedTaskGroupInfo,
  collapsedTaskGroups: Set<string>
): Set<string> {
  const remapped = new Set<string>();
  const previousCollapsedGroups = previousInfo.groups.filter((group) =>
    collapsedTaskGroups.has(group.key)
  );

  if (previousCollapsedGroups.length === 0) return remapped;

  for (const group of nextInfo.groups) {
    if (collapsedTaskGroups.has(group.key)) {
      remapped.add(group.key);
      continue;
    }

    // Preserve collapse state when a completed run changes shape after edits
    // (toggle, insert/remove items) but still maps to the same visual group.
    const overlapsCollapsedGroup = previousCollapsedGroups.some((previousGroup) =>
      rangesOverlap(
        group.startLine,
        group.endLine,
        previousGroup.startLine,
        previousGroup.endLine,
        1
      )
    );

    if (overlapsCollapsedGroup) {
      remapped.add(group.key);
    }
  }

  return remapped;
}
