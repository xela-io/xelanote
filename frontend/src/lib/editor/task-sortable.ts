/**
 * Svelte Action for drag & drop task reordering using SortableJS.
 * Supports both mouse (desktop) and touch (mobile with long-press).
 */

import Sortable from 'sortablejs';

export interface TaskSortableOptions {
  onReorder: (fromIndex: number, toIndex: number) => void;
  revision?: string | number;
}

/**
 * Svelte action that initializes SortableJS on task lists within a container.
 * Handles cleanup on unmount and re-initialization when content changes.
 */
export function taskSortable(container: HTMLElement, options: TaskSortableOptions) {
  const instancesByList = new Map<HTMLElement, Sortable>();
  let rafId: number | null = null;

  function refresh() {
    const lists = Array.from(container.querySelectorAll('ul.contains-task-list')) as HTMLElement[];
    const activeLists = new Set(lists);

    // Remove instances for lists that no longer exist
    for (const [list, instance] of instancesByList) {
      if (!activeLists.has(list)) {
        instance.destroy();
        instancesByList.delete(list);
      }
    }

    lists.forEach((list) => {
      if (instancesByList.has(list)) return;

      const sortable = Sortable.create(list as HTMLElement, {
        animation: 150,
        handle: '.drag-handle',
        draggable: '.task-list-item',
        // Long-press delay for touch devices (prevents accidental drags)
        delay: 200,
        delayOnTouchOnly: true,
        // Visual feedback classes
        ghostClass: 'task-drag-ghost',
        chosenClass: 'task-drag-chosen',
        dragClass: 'task-drag-active',
        // Prevent text selection during drag
        preventOnFilter: true,
        // Disable sorting between different lists
        group: {
          name: 'tasks-' + Math.random().toString(36).substr(2, 9),
          pull: false,
          put: false,
        },
        onEnd: (evt) => {
          // No actual movement happened
          if (evt.oldIndex === evt.newIndex) return;
          if (evt.oldIndex === undefined || evt.newIndex === undefined) return;

          // Get task indices from data attributes
          const items = evt.to.querySelectorAll('.task-list-item');
          const movedItem = items[evt.newIndex];

          if (!movedItem) return;

          const fromIndex = parseInt(movedItem.getAttribute('data-task-index') || '-1', 10);

          // Calculate target index: we need to find which task is at the position
          // where we want to insert
          // After the DOM move, the item at newIndex is our moved item
          // We need to determine where it should go in the source

          // The target task is the one that was at newIndex before the move
          // Since SortableJS already moved the DOM, we need to calculate based on
          // the new positions

          // Find the task that should be at the target position
          let toIndex: number;

          if (evt.oldIndex < evt.newIndex) {
            // Moving down: target is the item now before us (which moved up)
            const targetItem = items[evt.newIndex - 1];
            if (targetItem) {
              toIndex = parseInt(targetItem.getAttribute('data-task-index') || '-1', 10);
            } else {
              toIndex = fromIndex;
            }
          } else {
            // Moving up: target is the item now after us (which moved down)
            const targetItem = items[evt.newIndex + 1];
            if (targetItem) {
              toIndex = parseInt(targetItem.getAttribute('data-task-index') || '-1', 10);
            } else {
              toIndex = fromIndex;
            }
          }

          if (fromIndex === -1 || toIndex === -1 || fromIndex === toIndex) return;

          options.onReorder(fromIndex, toIndex);
        },
      });

      instancesByList.set(list, sortable);
    });
  }

  function scheduleRefresh() {
    if (rafId !== null) cancelAnimationFrame(rafId);
    rafId = requestAnimationFrame(() => {
      rafId = null;
      refresh();
    });
  }

  function destroy() {
    if (rafId !== null) {
      cancelAnimationFrame(rafId);
      rafId = null;
    }
    instancesByList.forEach((s) => s.destroy());
    instancesByList.clear();
  }

  // Initial setup
  scheduleRefresh();

  return {
    // Refresh when preview HTML changes (controlled via options.revision)
    update: (newOptions: TaskSortableOptions) => {
      options = newOptions;
      scheduleRefresh();
    },
    // Cleanup on unmount
    destroy,
  };
}

/**
 * Trigger re-initialization of sortable instances.
 * Call this after the preview HTML is re-rendered.
 */
export function reinitTaskSortable(container: HTMLElement, options: TaskSortableOptions) {
  // The action's update function handles this, but this can be called directly
  // if needed from outside the action context
  return taskSortable(container, options);
}
