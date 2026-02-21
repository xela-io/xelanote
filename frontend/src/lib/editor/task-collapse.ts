/**
 * Svelte Action for collapsing completed task items into a toggleable group.
 * Completed (checked) items at the end of each task list are wrapped in a
 * <details> element that defaults to collapsed.
 *
 * State (open/closed) is preserved across re-renders via a module-level Map.
 */

export interface TaskCollapseOptions {
  completedLabel: (count: number) => string;
  completedAriaLabel: (count: number) => string;
  noteId: string;
  revision?: string | number;
}

const TASK_COLLAPSE_STORAGE_KEY = 'xelanote-task-collapse-v1';

// Persist collapse state across re-renders (noteId-groupSignature -> open)
const collapseState = new Map<string, boolean>();

let persistedStateLoaded = false;

function normalizeText(value: string): string {
  return value.replace(/\s+/g, ' ').trim();
}

function buildStateKey(noteId: string, checkedItems: HTMLLIElement[], anchorText: string): string {
  const parts = checkedItems.map((item) => normalizeText(item.textContent ?? ''));
  const signature = `${normalizeText(anchorText)}|${parts.join('|')}`;
  let hash = 2166136261;
  for (let i = 0; i < signature.length; i++) {
    hash ^= signature.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return `${noteId}-${(hash >>> 0).toString(36)}`;
}

function loadPersistedState() {
  if (persistedStateLoaded || typeof localStorage === 'undefined') return;
  persistedStateLoaded = true;
  try {
    const raw = localStorage.getItem(TASK_COLLAPSE_STORAGE_KEY);
    if (!raw) return;
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== 'object') return;
    for (const [key, value] of Object.entries(parsed)) {
      if (typeof value === 'boolean') {
        collapseState.set(key, value);
      }
    }
  } catch {
    // localStorage unavailable or invalid JSON
  }
}

function persistState() {
  if (typeof localStorage === 'undefined') return;
  try {
    localStorage.setItem(
      TASK_COLLAPSE_STORAGE_KEY,
      JSON.stringify(Object.fromEntries(collapseState.entries()))
    );
  } catch {
    // localStorage unavailable or quota exceeded
  }
}

const CHEVRON_SVG = `<svg class="chevron-icon" aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>`;

export function taskCollapse(container: HTMLElement, options: TaskCollapseOptions) {
  let cleanups: (() => void)[] = [];
  let rafId: number | null = null;

  function init() {
    loadPersistedState();
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
      const stateKey = buildStateKey(options.noteId, checkedItems, anchorText);
      const isOpen = collapseState.get(stateKey) ?? false;

      // Build DOM structure
      const wrapper = document.createElement('li');
      wrapper.className = 'completed-tasks-wrapper';

      const details = document.createElement('details');
      details.className = 'completed-tasks-group';
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
        collapseState.set(stateKey, details.open);
        persistState();
      };
      details.addEventListener('toggle', onToggle);
      cleanups.push(() => details.removeEventListener('toggle', onToggle));
    });
  }

  function cleanup() {
    cleanups.forEach((fn) => fn());
    cleanups = [];
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
