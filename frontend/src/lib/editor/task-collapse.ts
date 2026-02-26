/**
 * Svelte Action for collapsing completed task items into a toggleable group.
 * Completed (checked) items at the end of each task list are wrapped in a
 * <details> element that defaults to collapsed.
 *
 * State (open/closed) is preserved across re-renders via a module-level Map
 * and synced to the server for cross-device persistence.
 */

import { ApiError } from '$lib/api/client';
import { getNoteUserState, updateNoteUserCollapseState } from '$lib/api/notes';

export interface TaskCollapseOptions {
  completedLabel: (count: number) => string;
  completedAriaLabel: (count: number) => string;
  noteId: string;
  revision?: string | number;
}

const STORAGE_KEY_V1 = 'xelanote-task-collapse-v1';
const STORAGE_KEY_V2 = 'xelanote-task-collapse-v2';

// Nested state: noteId -> groupHash -> open
const collapseState = new Map<string, Map<string, boolean>>();

let persistedStateLoaded = false;

// Server sync tracking
const loadedNoteIds = new Set<string>();
const syncTimers = new Map<string, ReturnType<typeof setTimeout>>();
let serverSyncSupported = true;

const SYNC_DEBOUNCE_MS = 500;

export function normalizeText(value: string): string {
  return value.replace(/\s+/g, ' ').trim();
}

/**
 * Returns a base36 FNV hash string for a group of checked items.
 * Used as the key in collapse state maps.
 */
export function buildGroupHash(checkedItems: HTMLLIElement[], anchorText: string): string {
  const parts = checkedItems.map((item) => normalizeText(item.textContent ?? ''));
  const signature = `${normalizeText(anchorText)}|${parts.join('|')}`;
  let hash = 2166136261;
  for (let i = 0; i < signature.length; i++) {
    hash ^= signature.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return (hash >>> 0).toString(36);
}

function getNoteState(noteId: string): Map<string, boolean> {
  let noteState = collapseState.get(noteId);
  if (!noteState) {
    noteState = new Map();
    collapseState.set(noteId, noteState);
  }
  return noteState;
}

// --- localStorage migration (v1 → v2) and persistence ---

function migrateV1ToV2(): void {
  if (typeof localStorage === 'undefined') return;
  try {
    const v1Raw = localStorage.getItem(STORAGE_KEY_V1);
    if (!v1Raw) return;
    const v1 = JSON.parse(v1Raw);
    if (!v1 || typeof v1 !== 'object') return;

    const v2: Record<string, Record<string, boolean>> = {};
    for (const [key, value] of Object.entries(v1)) {
      if (typeof value !== 'boolean') continue;
      // Key format: "noteId-hash" — split at last dash (noteId may contain dashes like UUIDs)
      const lastDash = key.lastIndexOf('-');
      if (lastDash <= 0) continue;
      const noteId = key.substring(0, lastDash);
      const hash = key.substring(lastDash + 1);
      if (!v2[noteId]) v2[noteId] = {};
      v2[noteId][hash] = value;
    }

    localStorage.setItem(STORAGE_KEY_V2, JSON.stringify(v2));
    localStorage.removeItem(STORAGE_KEY_V1);
  } catch {
    // ignore migration errors
  }
}

function loadPersistedState(): void {
  if (persistedStateLoaded || typeof localStorage === 'undefined') return;
  persistedStateLoaded = true;

  // Try v2 first, migrate from v1 if needed
  try {
    let raw = localStorage.getItem(STORAGE_KEY_V2);
    if (!raw) {
      migrateV1ToV2();
      raw = localStorage.getItem(STORAGE_KEY_V2);
    }
    if (!raw) return;
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== 'object') return;

    for (const [noteId, hashes] of Object.entries(parsed)) {
      if (!hashes || typeof hashes !== 'object') continue;
      const noteState = getNoteState(noteId);
      for (const [hash, value] of Object.entries(hashes as Record<string, unknown>)) {
        if (typeof value === 'boolean') {
          noteState.set(hash, value);
        }
      }
    }
  } catch {
    // localStorage unavailable or invalid JSON
  }
}

function persistState(): void {
  if (typeof localStorage === 'undefined') return;
  try {
    const data: Record<string, Record<string, boolean>> = {};
    for (const [noteId, noteState] of collapseState.entries()) {
      if (noteState.size > 0) {
        data[noteId] = Object.fromEntries(noteState.entries());
      }
    }
    localStorage.setItem(STORAGE_KEY_V2, JSON.stringify(data));
  } catch {
    // localStorage unavailable or quota exceeded
  }
}

// --- Server sync ---

function queueServerSync(noteId: string): void {
  if (!serverSyncSupported || !noteId) return;

  const existing = syncTimers.get(noteId);
  if (existing !== undefined) clearTimeout(existing);

  syncTimers.set(
    noteId,
    setTimeout(() => {
      syncTimers.delete(noteId);
      const noteState = collapseState.get(noteId);
      const stateObj: Record<string, boolean> = noteState
        ? Object.fromEntries(noteState.entries())
        : {};

      updateNoteUserCollapseState(noteId, stateObj).catch((error: unknown) => {
        if (error instanceof ApiError) {
          if ([404, 405].includes(error.status)) {
            serverSyncSupported = false;
          }
          // 400, 422, 500 = transient, don't disable sync
        }
        // Network errors = transient, don't disable sync
      });
    }, SYNC_DEBOUNCE_MS)
  );
}

function loadFromServer(noteId: string): void {
  if (!serverSyncSupported || !noteId || loadedNoteIds.has(noteId)) return;
  loadedNoteIds.add(noteId);

  getNoteUserState(noteId)
    .then((response) => {
      if (!response.collapse_state) return;

      const noteState = getNoteState(noteId);
      let changed = false;

      for (const [hash, value] of Object.entries(response.collapse_state)) {
        if (typeof value === 'boolean' && noteState.get(hash) !== value) {
          noteState.set(hash, value);
          changed = true;
        }
      }

      if (changed) {
        persistState();
        // Targeted DOM update: find <details> elements with matching data-group-hash
        const allDetails = document.querySelectorAll<HTMLDetailsElement>(
          'details.completed-tasks-group[data-group-hash]'
        );
        for (const details of allDetails) {
          const hash = details.getAttribute('data-group-hash');
          if (hash && noteState.has(hash)) {
            details.open = noteState.get(hash)!;
          }
        }
      }
    })
    .catch((error: unknown) => {
      if (error instanceof ApiError) {
        if ([404, 405].includes(error.status)) {
          serverSyncSupported = false;
        }
      }
    });
}

// --- DOM rendering ---

const CHEVRON_SVG = `<svg class="chevron-icon" aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>`;

export function taskCollapse(container: HTMLElement, options: TaskCollapseOptions) {
  let cleanups: (() => void)[] = [];
  let rafId: number | null = null;
  function init() {
    loadPersistedState();

    // Skip wrapping when noteId is not yet set (e.g., initial mount before $effect
    // sets the real noteId). Without this guard, wrappers would be created under the
    // '' key and toggle state would be persisted under the wrong noteId.
    if (!options.noteId) {
      return;
    }

    // Trigger server load for this note (first time only)
    loadFromServer(options.noteId);

    const lists = container.querySelectorAll('ul.contains-task-list');
    lists.forEach((list) => {
      // Defensive: skip if already wrapped
      if (list.querySelector(':scope > li.completed-tasks-wrapper')) return;

      // Only consider direct children that are task items
      const items = Array.from(list.querySelectorAll<HTMLLIElement>(':scope > li.task-list-item'));

      if (items.length === 0) return;

      // Scan from bottom: find trailing consecutive checked items.
      // markdown-it-task-lists does NOT set a "task-list-item-checked" class,
      // so we check the HTML checked attribute (not the DOM .checked property).
      // The attribute reflects the markdown-rendered state and is immune to
      // browser-side toggling from label clicks or form restoration.
      let checkedCount = 0;
      for (let i = items.length - 1; i >= 0; i--) {
        const checkbox = items[i].querySelector(
          'input.task-list-item-checkbox'
        ) as HTMLInputElement | null;
        if (checkbox?.hasAttribute('checked')) {
          checkedCount++;
        } else {
          break;
        }
      }

      if (checkedCount === 0) return;

      const checkedItems = items.slice(items.length - checkedCount);
      const anchorItem = items[items.length - checkedCount - 1];
      const anchorText = anchorItem ? (anchorItem.textContent ?? '') : '';
      const groupHash = buildGroupHash(checkedItems, anchorText);
      const noteState = getNoteState(options.noteId);
      const isOpen = noteState.get(groupHash) ?? false;

      // Build DOM structure
      const wrapper = document.createElement('li');
      wrapper.className = 'completed-tasks-wrapper';

      const details = document.createElement('details');
      details.className = 'completed-tasks-group';
      details.setAttribute('data-group-hash', groupHash);
      if (isOpen) details.open = true;

      const summary = document.createElement('summary');
      summary.className = 'completed-tasks-summary';
      summary.setAttribute('aria-label', options.completedAriaLabel(checkedCount));
      // F2-04: Use DOM methods instead of innerHTML to prevent XSS if completedLabel
      // ever includes user data. SVG is inserted via a template, text via textContent.
      const chevronContainer = document.createElement('span');
      chevronContainer.innerHTML = CHEVRON_SVG; // hardcoded constant, safe
      summary.appendChild(chevronContainer.firstChild!);
      summary.appendChild(document.createTextNode(` ${options.completedLabel(checkedCount)}`));

      const innerList = document.createElement('ul');
      innerList.className = 'completed-tasks-inner';

      // Move (not clone) checked items into inner list
      for (const item of checkedItems) {
        innerList.appendChild(item);
      }

      details.appendChild(summary);
      details.appendChild(innerList);
      wrapper.appendChild(details);
      list.appendChild(wrapper);

      // Toggle listener to persist state
      const onToggle = () => {
        noteState.set(groupHash, details.open);
        persistState();
        queueServerSync(options.noteId);
      };
      details.addEventListener('toggle', onToggle);
      cleanups.push(() => details.removeEventListener('toggle', onToggle));
    });
  }

  function cleanup() {
    cleanups.forEach((fn) => fn());
    cleanups = [];
    // Remove wrapper DOM elements so init() can re-process lists.
    // Without this, init()'s defensive check finds existing wrappers and skips
    // re-processing, leaving <details> elements without event listeners.
    const wrappers = container.querySelectorAll('li.completed-tasks-wrapper');
    for (const wrapper of wrappers) {
      const innerList = wrapper.querySelector('ul.completed-tasks-inner');
      if (innerList) {
        const parentList = wrapper.parentElement;
        if (parentList) {
          // Move checked items back into the parent list before the wrapper
          while (innerList.firstChild) {
            parentList.insertBefore(innerList.firstChild, wrapper);
          }
        }
      }
      wrapper.remove();
    }
  }

  function scheduleInit() {
    if (rafId !== null) cancelAnimationFrame(rafId);
    rafId = requestAnimationFrame(() => {
      rafId = null;
      cleanup();
      init();
    });
  }

  // Initial setup
  scheduleInit();

  return {
    update(newOptions: TaskCollapseOptions) {
      options = newOptions;
      scheduleInit();
    },
    destroy() {
      if (rafId !== null) {
        cancelAnimationFrame(rafId);
        rafId = null;
      }
      cleanup();
    },
  };
}

/**
 * Reset module state — for testing only.
 */
export function _resetForTest(): void {
  collapseState.clear();
  loadedNoteIds.clear();
  for (const timer of syncTimers.values()) clearTimeout(timer);
  syncTimers.clear();
  persistedStateLoaded = false;
  serverSyncSupported = true;
}
