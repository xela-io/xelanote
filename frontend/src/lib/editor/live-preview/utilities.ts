// Shared utilities: profiling, localStorage persistence, active lines, set helpers

import { EditorView } from '@codemirror/view';

import { ApiError } from '$lib/api/client';
import { getNoteUserState, updateNoteUserCollapseState } from '$lib/api/notes';

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
const LIVE_TASK_KEY_PREFIX = 'tasks:';
const LIVE_TASK_SYNC_DEBOUNCE_MS = 500;

const loadedServerStateByNote = new Map<string, Record<string, boolean> | null>();
const pendingServerStateLoads = new Map<string, Promise<Record<string, boolean> | null>>();
const liveTaskSyncTimers = new Map<string, ReturnType<typeof setTimeout>>();
let liveTaskServerSyncSupported = true;

function isLiveTaskKey(key: string): boolean {
  return key.startsWith(LIVE_TASK_KEY_PREFIX);
}

function normalizeServerState(state: unknown): Record<string, boolean> | null {
  if (!state || typeof state !== 'object') return null;
  const result: Record<string, boolean> = {};
  for (const [key, value] of Object.entries(state as Record<string, unknown>)) {
    if (typeof value === 'boolean') result[key] = value;
  }
  return result;
}

async function ensureServerStateLoaded(noteId: string): Promise<Record<string, boolean> | null> {
  if (!noteId || !liveTaskServerSyncSupported) return null;
  if (loadedServerStateByNote.has(noteId)) {
    return loadedServerStateByNote.get(noteId) ?? null;
  }

  const pending = pendingServerStateLoads.get(noteId);
  if (pending) return pending;

  const requestPromise = getNoteUserState(noteId)
    .then((response) => {
      const normalized = normalizeServerState(response.collapse_state);
      loadedServerStateByNote.set(noteId, normalized);
      pendingServerStateLoads.delete(noteId);
      return normalized;
    })
    .catch((error: unknown) => {
      pendingServerStateLoads.delete(noteId);
      if (error instanceof ApiError && [404, 405].includes(error.status)) {
        liveTaskServerSyncSupported = false;
      }
      return null;
    });

  pendingServerStateLoads.set(noteId, requestPromise);
  return requestPromise;
}

export async function loadCollapsedTaskGroupsFromServer(
  noteId?: string
): Promise<Set<string> | null> {
  if (!noteId || !liveTaskServerSyncSupported) return null;
  const state = await ensureServerStateLoaded(noteId);
  if (state === null) return null;

  const keys = new Set<string>();
  for (const [key, value] of Object.entries(state)) {
    if (value && isLiveTaskKey(key)) keys.add(key);
  }
  return keys;
}

export function queueCollapsedTaskGroupsServerSync(
  noteId: string | undefined,
  keys: Set<string>
): void {
  if (!noteId || !liveTaskServerSyncSupported) return;

  const existing = liveTaskSyncTimers.get(noteId);
  if (existing !== undefined) clearTimeout(existing);

  const snapshot = new Set(keys);
  liveTaskSyncTimers.set(
    noteId,
    setTimeout(() => {
      liveTaskSyncTimers.delete(noteId);
      void (async () => {
        if (!liveTaskServerSyncSupported) return;

        const baseState = (await ensureServerStateLoaded(noteId)) ?? {};
        const nextState: Record<string, boolean> = { ...baseState };

        for (const key of Object.keys(nextState)) {
          if (isLiveTaskKey(key)) delete nextState[key];
        }
        for (const key of snapshot) {
          nextState[key] = true;
        }

        try {
          await updateNoteUserCollapseState(noteId, nextState);
          loadedServerStateByNote.set(noteId, nextState);
        } catch (error: unknown) {
          if (error instanceof ApiError && [404, 405].includes(error.status)) {
            liveTaskServerSyncSupported = false;
          }
        }
      })();
    }, LIVE_TASK_SYNC_DEBOUNCE_MS)
  );
}

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

export function _resetLivePreviewPersistenceForTest(): void {
  loadedServerStateByNote.clear();
  pendingServerStateLoads.clear();
  for (const timer of liveTaskSyncTimers.values()) clearTimeout(timer);
  liveTaskSyncTimers.clear();
  liveTaskServerSyncSupported = true;
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
