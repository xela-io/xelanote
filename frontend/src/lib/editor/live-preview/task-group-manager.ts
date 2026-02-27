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

export function buildTaskGroupByLineForViewport(
  groups: CompletedTaskGroup[],
  visibleFrom: number,
  visibleTo: number
): Map<number, CompletedTaskGroup> {
  const groupByLine = new Map<number, CompletedTaskGroup>();
  for (const group of groups) {
    const populateStart = Math.max(group.startLine, visibleFrom);
    const populateEnd = Math.min(group.endLine, visibleTo);
    for (let line = populateStart; line <= populateEnd; line++) {
      groupByLine.set(line, group);
    }
  }
  return groupByLine;
}

function hashBase36(input: string): string {
  let hash = 2166136261;
  for (let i = 0; i < input.length; i++) {
    hash ^= input.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return (hash >>> 0).toString(36);
}

function buildTaskGroupContent(view: EditorView, startLine: number, endLine: number): string {
  let content = '';
  const anchorLine = startLine > 1 ? view.state.doc.line(startLine - 1).text.trim() : '';
  if (anchorLine) content += `anchor:${anchorLine}\n`;
  for (let lineNo = startLine; lineNo <= endLine; lineNo++) {
    content += `${view.state.doc.line(lineNo).text.trim()}\n`;
  }
  return content;
}

function buildTaskGroupKeyFromHash(hash: string, occurrence: number): string {
  if (occurrence <= 1) return `tasks:${hash}`;
  // Ensure duplicate groups in the same note receive distinct keys while
  // keeping key shape compatible with backend validation (tasks:<base36>).
  return `tasks:${hashBase36(`${hash}#${occurrence}`)}`;
}

export function buildTaskGroupKey(
  view: EditorView,
  startLine: number,
  endLine: number,
  occurrence = 1
): string {
  const hash = hashBase36(buildTaskGroupContent(view, startLine, endLine));
  return buildTaskGroupKeyFromHash(hash, occurrence);
}

/**
 * Collect completed task groups with viewport-optimized groupByLine population.
 * Full document is scanned for task runs, but groupByLine is only populated
 * for lines within the visible range.
 */
export function collectCompletedTaskGroups(
  view: EditorView,
  collapsedTaskGroups: Set<string>,
  visibleFrom?: number,
  visibleTo?: number
): CompletedTaskGroupInfo {
  const groups: CompletedTaskGroup[] = [];
  const groupByLine = new Map<number, CompletedTaskGroup>();
  const keys = new Set<string>();
  const groupHashCounts = new Map<string, number>();

  const viewportStartLine = visibleFrom ?? 1;
  const viewportEndLine = visibleTo ?? view.state.doc.lines;

  let runStart = -1;
  let runCount = 0;

  const flushRun = (endLine: number) => {
    if (runCount >= 2) {
      const hash = hashBase36(buildTaskGroupContent(view, runStart, endLine));
      const occurrence = (groupHashCounts.get(hash) ?? 0) + 1;
      groupHashCounts.set(hash, occurrence);
      const key = buildTaskGroupKeyFromHash(hash, occurrence);
      keys.add(key);
      const group: CompletedTaskGroup = {
        key,
        startLine: runStart,
        endLine,
        count: runCount,
        collapsed: collapsedTaskGroups.has(key),
      };
      groups.push(group);
    }
    runStart = -1;
    runCount = 0;
  };

  // Track checked state of the most recent top-level task so that checked
  // children within an unchecked parent don't form their own completed groups.
  let topLevelParentChecked = false;

  for (let lineNo = 1; lineNo <= view.state.doc.lines; lineNo++) {
    const text = view.state.doc.line(lineNo).text;
    const primitives = parseLinePrimitives(text);

    // Update top-level task state when we encounter one
    if (primitives.nestLevel === 0 && primitives.taskRegex) {
      topLevelParentChecked = primitives.taskRegex.checked;
    }

    // A checked task counts towards a completed run only if it is a top-level
    // task OR its nearest top-level ancestor is also checked.  This prevents
    // checked children within an unchecked parent from splitting the completed
    // group into multiple fragments.
    const countsAsCompleted =
      primitives.taskRegex?.checked === true &&
      (primitives.nestLevel === 0 || topLevelParentChecked);

    if (countsAsCompleted) {
      if (runStart === -1) runStart = lineNo;
      runCount++;
    } else {
      if (runStart !== -1) flushRun(lineNo - 1);
    }
  }
  if (runStart !== -1) flushRun(view.state.doc.lines);

  return {
    groups,
    groupByLine:
      groups.length === 0
        ? groupByLine
        : buildTaskGroupByLineForViewport(groups, viewportStartLine, viewportEndLine),
    keys,
  };
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
  collapsedTaskGroups: Set<string>,
  mapPreviousGroupToNextRange?: (previousGroup: CompletedTaskGroup) => {
    startLine: number;
    endLine: number;
  }
): Set<string> {
  const remapped = new Set<string>();
  const previousCollapsedGroups = previousInfo.groups.filter((group) =>
    collapsedTaskGroups.has(group.key)
  );

  if (previousCollapsedGroups.length === 0) return remapped;

  const mappedPreviousCollapsedGroups = previousCollapsedGroups.map((group) => {
    const mapped = mapPreviousGroupToNextRange?.(group);
    if (!mapped) return { startLine: group.startLine, endLine: group.endLine };
    return mapped;
  });

  for (const group of nextInfo.groups) {
    // Preserve collapse state when a completed run changes shape after edits
    // (toggle, insert/remove items) but still maps to the same visual group.
    const overlapsCollapsedGroup = mappedPreviousCollapsedGroups.some((previousGroup) =>
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
