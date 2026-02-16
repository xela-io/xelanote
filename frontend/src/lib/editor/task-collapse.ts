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
}

// Persist collapse state across re-renders (noteId-listIndex -> open)
const collapseState = new Map<string, boolean>();

const CHEVRON_SVG = `<svg class="chevron-icon" aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>`;

export function taskCollapse(container: HTMLElement, options: TaskCollapseOptions) {
  let cleanups: (() => void)[] = [];

  function init() {
    const lists = container.querySelectorAll('ul.contains-task-list');

    lists.forEach((list, listIndex) => {
      // Defensive: skip if already wrapped
      if (list.querySelector(':scope > li.completed-tasks-wrapper')) return;

      // Only consider direct children that are task items
      const items = Array.from(list.querySelectorAll(':scope > li.task-list-item'));

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
      const stateKey = `${options.noteId}-${listIndex}`;
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
      summary.innerHTML = `${CHEVRON_SVG} ${options.completedLabel(checkedCount)}`;

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
      };
      details.addEventListener('toggle', onToggle);
      cleanups.push(() => details.removeEventListener('toggle', onToggle));
    });
  }

  function cleanup() {
    cleanups.forEach((fn) => fn());
    cleanups = [];
  }

  // Initial setup
  init();

  return {
    update(newOptions: TaskCollapseOptions) {
      options = newOptions;
      cleanup();
      init();
    },
    destroy() {
      cleanup();
    },
  };
}
