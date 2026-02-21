// Shared utilities: profiling, localStorage persistence, active lines, set helpers

import { EditorView } from '@codemirror/view';

// Profiling types and functions

export type LivePreviewProfilePhase = 'build' | 'tree' | 'structured';

export interface LivePreviewProfileSample {
  phase: LivePreviewProfilePhase;
  reason: string;
  ms: number;
}

export type LivePreviewProfilerSink = (sample: LivePreviewProfileSample) => void;

let livePreviewProfilerSink: LivePreviewProfilerSink | null = null;

export function setLivePreviewProfilerSink(sink: LivePreviewProfilerSink | null): void {
  livePreviewProfilerSink = sink;
}

function nowMs(): number {
  return typeof performance !== 'undefined' ? performance.now() : Date.now();
}

export function profile<T>(phase: LivePreviewProfilePhase, reason: string, fn: () => T): T {
  const start = nowMs();
  const result = fn();
  const end = nowMs();
  livePreviewProfilerSink?.({
    phase,
    reason,
    ms: end - start,
  });
  return result;
}

// localStorage persistence for collapsed task groups

const TASK_COLLAPSE_STORAGE_PREFIX = 'xelanote-live-task-collapse-v1:';

export function loadCollapsedTaskGroups(noteId?: string): Set<string> {
  if (!noteId || typeof localStorage === 'undefined') return new Set<string>();
  try {
    const raw = localStorage.getItem(`${TASK_COLLAPSE_STORAGE_PREFIX}${noteId}`);
    if (!raw) return new Set<string>();
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return new Set<string>();
    return new Set<string>(parsed.filter((value): value is string => typeof value === 'string'));
  } catch {
    return new Set<string>();
  }
}

export function persistCollapsedTaskGroups(noteId: string | undefined, keys: Set<string>): void {
  if (!noteId || typeof localStorage === 'undefined') return;
  try {
    if (keys.size === 0) {
      localStorage.removeItem(`${TASK_COLLAPSE_STORAGE_PREFIX}${noteId}`);
      return;
    }
    localStorage.setItem(`${TASK_COLLAPSE_STORAGE_PREFIX}${noteId}`, JSON.stringify([...keys]));
  } catch {
    // localStorage unavailable or quota exceeded
  }
}

// Set and line helpers

export function setsEqual<T>(left: Set<T>, right: Set<T>): boolean {
  if (left.size !== right.size) return false;
  for (const value of left) {
    if (!right.has(value)) return false;
  }
  return true;
}

export function getActiveLines(view: EditorView): Set<number> {
  const lines = new Set<number>();

  // When the editor doesn't have focus, no line should be considered active.
  // This prevents the first line from showing raw markdown on load/note switch.
  if (!view.hasFocus) return lines;

  for (const range of view.state.selection.ranges) {
    let currentLine = view.state.doc.lineAt(range.from);
    lines.add(currentLine.number);

    if (range.empty) continue;

    while (currentLine.to < range.to) {
      const nextFrom = currentLine.to + 1;
      if (nextFrom > view.state.doc.length) break;
      currentLine = view.state.doc.lineAt(nextFrom);
      lines.add(currentLine.number);
    }
  }

  return lines;
}

export function activeLinesKey(lines: Set<number>): string {
  return [...lines].sort((a, b) => a - b).join(',');
}

export function isInsideRanges(
  position: number,
  ranges: Array<{ from: number; to: number }>
): boolean {
  return ranges.some((range) => position >= range.from && position < range.to);
}
