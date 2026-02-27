/**
 * Svelte Action for drag & drop task reordering using SortableJS.
 * Supports both mouse (desktop) and touch (mobile with long-press).
 */

import type { EditorView } from '@codemirror/view';
import Sortable from 'sortablejs';

export interface TaskSortableOptions {
  onReorder: (fromIndex: number, toIndex: number) => void;
  onReorderByLine?: (fromLine: number, toLine: number) => void;
  editorView?: EditorView;
  mode?: 'preview' | 'live';
  enabled?: boolean;
  revision?: string | number;
}

interface MutationObserverControl {
  stop(): void;
  start(): void;
}

type EditorViewWithObserver = EditorView & {
  observer?: MutationObserverControl;
};

/**
 * Stop CodeMirror's internal MutationObserver. This prevents CM from
 * interpreting SortableJS DOM mutations as user input during a drag.
 * Uses the internal `observer` property (not public API, but stable
 * and used by CM itself via `ignore()`).
 */
export function stopCMObserver(view: EditorView): void {
  (view as unknown as EditorViewWithObserver).observer?.stop();
}

/**
 * Restart CodeMirror's MutationObserver. Idempotent — safe to call
 * even if the observer is already active.
 */
export function startCMObserver(view: EditorView): void {
  (view as unknown as EditorViewWithObserver).observer?.start();
}

interface SortableIndexEventLike {
  oldDraggableIndex?: number;
  newDraggableIndex?: number;
  oldIndex?: number;
  newIndex?: number;
}

/**
 * Determine the target line for a live-mode task reorder by comparing the
 * dragged item's original line number with its new DOM neighbors.
 *
 * Returns the target line number, or -1 if no effective movement occurred.
 *
 * @param fromLine - The dragged item's original 1-based line number
 * @param prevLine - Line number of the item now above the dragged item (-1 if none)
 * @param nextLine - Line number of the item now below the dragged item (-1 if none)
 */
export function determineMoveTarget(fromLine: number, prevLine: number, nextLine: number): number {
  // If the item's original line is smaller than its new predecessor,
  // it was dragged downward past that predecessor.
  const movedDown = prevLine !== -1 && fromLine < prevLine;
  // If the item's original line is larger than its new successor,
  // it was dragged upward past that successor.
  const movedUp = nextLine !== -1 && fromLine > nextLine;

  if (movedDown) return prevLine;
  if (movedUp) return nextLine;
  return -1;
}

export function getSortableIndices(
  evt: SortableIndexEventLike,
  options: { draggableOnly?: boolean } = {}
): {
  oldIndex: number;
  newIndex: number;
} | null {
  if (options.draggableOnly) {
    if (evt.oldDraggableIndex === undefined || evt.newDraggableIndex === undefined) {
      return null;
    }
    return { oldIndex: evt.oldDraggableIndex, newIndex: evt.newDraggableIndex };
  }

  const oldIndex = evt.oldDraggableIndex ?? evt.oldIndex;
  const newIndex = evt.newDraggableIndex ?? evt.newIndex;
  if (oldIndex === undefined || newIndex === undefined) {
    return null;
  }
  return { oldIndex, newIndex };
}

/**
 * Svelte action that initializes SortableJS on task lists within a container.
 * Handles cleanup on unmount and re-initialization when content changes.
 */
export function taskSortable(container: HTMLElement, options: TaskSortableOptions) {
  const instancesByContainer = new Map<HTMLElement, Sortable>();
  let rafId: number | null = null;
  const liveDraggableSelector =
    '.cm-line.cm-live-task-line:not(.cm-live-collapsed-line):not([class*="cm-live-nest-"])';

  function getLineNumberFromLiveTaskItem(item: Element | null): number {
    if (!item) return -1;
    const lineSource = item.querySelector(
      '.cm-live-task-drag-handle, .cm-live-task-checkbox'
    ) as HTMLElement | null;
    const line = parseInt(lineSource?.dataset.line ?? '-1', 10);
    return Number.isInteger(line) && line > 0 ? line : -1;
  }

  function refresh() {
    const mode = options.mode ?? 'preview';
    const enabled = options.enabled ?? true;
    const isTouch = window.matchMedia('(pointer: coarse)').matches;

    // Live-mode drag explicitly disabled on touch: long-press conflicts
    // with text selection in contenteditable.
    const effectiveEnabled = enabled && !(mode === 'live' && isTouch);

    const containers =
      effectiveEnabled && mode === 'live'
        ? (Array.from(container.querySelectorAll('.cm-content')) as HTMLElement[])
        : effectiveEnabled
          ? (Array.from(container.querySelectorAll('ul.contains-task-list')) as HTMLElement[])
          : [];
    const _activeContainers = new Set(containers);

    // Destroy all existing instances and recreate from scratch.
    // This handles recycled containers (e.g. after DOM morphing) where
    // the element reference persists but content has changed.
    for (const [, instance] of instancesByContainer) {
      instance.destroy();
    }
    instancesByContainer.clear();

    containers.forEach((sortableContainer) => {
      // Track boundary listeners added during drag so we can clean up.
      let boundaryCleanup: (() => void) | null = null;

      const sortable = Sortable.create(sortableContainer, {
        animation: 150,
        handle: isTouch
          ? undefined
          : mode === 'live'
            ? '.cm-live-task-drag-handle'
            : '.drag-handle',
        draggable: mode === 'live' ? liveDraggableSelector : '.task-list-item',
        // Long-press delay for touch devices (prevents accidental drags)
        delay: 200,
        delayOnTouchOnly: true,
        // Live preview uses contenteditable DOM; force fallback DnD is more stable there.
        forceFallback: mode === 'live',
        fallbackOnBody: mode === 'live',
        // Improve drop precision and avoid viewport autoscroll jumps in live mode.
        scroll: mode === 'live' ? false : undefined,
        swapThreshold: mode === 'live' ? 0.65 : undefined,
        invertSwap: mode === 'live' ? true : undefined,
        invertedSwapThreshold: mode === 'live' ? 0.65 : undefined,
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
        onStart: () => {
          // Stop CodeMirror's MutationObserver so it doesn't interpret
          // SortableJS DOM mutations (moving .cm-line elements) as user
          // input. Without this, CM processes each intermediate move
          // between pointer events and corrupts the document state.
          if (mode === 'live' && options.editorView) {
            stopCMObserver(options.editorView);
          }

          // Freeze sorting when the pointer leaves the container bounds.
          // mouseleave/mouseenter are unreliable during forceFallback drags
          // (SortableJS may capture pointer events). Instead, track the
          // pointer position directly on document and toggle `sort`.
          if (mode === 'live') {
            const onPointerMove = (e: PointerEvent) => {
              const rect = sortableContainer.getBoundingClientRect();
              const inside =
                e.clientX >= rect.left &&
                e.clientX <= rect.right &&
                e.clientY >= rect.top &&
                e.clientY <= rect.bottom;
              sortable.option('sort', inside);
            };
            document.addEventListener('pointermove', onPointerMove);
            boundaryCleanup = () => {
              document.removeEventListener('pointermove', onPointerMove);
              sortable.option('sort', true);
            };
          }
        },
        onUnchoose: () => {
          // Clean up boundary listeners and re-enable sorting.
          boundaryCleanup?.();
          boundaryCleanup = null;

          // Safety net: ensure observer is restarted even if onEnd
          // bails out early (e.g. oldIndex === newIndex).
          // start() is idempotent — no-op if already active.
          if (mode === 'live' && options.editorView) {
            startCMObserver(options.editorView);
          }
        },
        onMove: (evt, originalEvent) => {
          if (mode !== 'live') return undefined;

          const target = evt.related as HTMLElement | null;
          if (!target || !target.classList.contains('cm-live-task-line')) return false;

          const getClientY = (event: Event): number | null => {
            const mouseLike = event as MouseEvent;
            if (typeof mouseLike.clientY === 'number') {
              return mouseLike.clientY;
            }
            const touchLike = event as TouchEvent;
            if (touchLike.touches && touchLike.touches.length > 0) {
              return touchLike.touches[0].clientY;
            }
            if (touchLike.changedTouches && touchLike.changedTouches.length > 0) {
              return touchLike.changedTouches[0].clientY;
            }
            return null;
          };

          // Cursor-precise insert direction:
          // upper half => before target, lower half => after target.
          const rect = target.getBoundingClientRect();
          const clientY = getClientY(originalEvent);
          if (clientY === null) return undefined;
          const midpoint = rect.top + rect.height / 2;
          return clientY < midpoint ? -1 : 1;
        },
        onEnd: (evt) => {
          if (mode === 'live') {
            // Determine the move target from the actual DOM position of the
            // dragged element rather than SortableJS's reported draggable
            // indices, which can desynchronize from the real DOM order when
            // onMove overrides (-1/1) interact with invertSwap and non-task
            // elements are interspersed between task lines.
            const fromLine = getLineNumberFromLiveTaskItem(evt.item);
            if (fromLine === -1) return;

            // Safety: reject moves for nested tasks
            const fromLineEl = evt.item as HTMLElement;
            if (fromLineEl.className && fromLineEl.className.includes('cm-live-nest-')) return;

            const items = Array.from(evt.to.querySelectorAll(liveDraggableSelector));
            const actualIndex = items.indexOf(evt.item as HTMLElement);
            if (actualIndex === -1) return;

            const prevLine =
              actualIndex > 0 ? getLineNumberFromLiveTaskItem(items[actualIndex - 1]) : -1;
            const nextLine =
              actualIndex < items.length - 1
                ? getLineNumberFromLiveTaskItem(items[actualIndex + 1])
                : -1;

            const toLine = determineMoveTarget(fromLine, prevLine, nextLine);

            if (toLine === -1 || fromLine === toLine) return;

            // Restart observer BEFORE dispatching so that dispatch() →
            // ignore() correctly calls clear() to discard stale mutations.
            // (ignore() short-circuits with just f() when observer is inactive.)
            if (options.editorView) {
              startCMObserver(options.editorView);
            }

            options.onReorderByLine?.(fromLine, toLine);
            return;
          }

          // Preview mode: use precomputed task indices from rendered HTML.
          const indices = getSortableIndices(evt);
          if (!indices) return;
          const { oldIndex, newIndex } = indices;
          if (oldIndex === newIndex) return;

          const items = evt.to.querySelectorAll('.task-list-item');
          const movedItem = items[newIndex];
          if (!movedItem) return;

          const fromIndex = parseInt(movedItem.getAttribute('data-task-index') || '-1', 10);
          let toIndex: number;

          if (oldIndex < newIndex) {
            const targetItem = items[newIndex - 1];
            toIndex = targetItem
              ? parseInt(targetItem.getAttribute('data-task-index') || '-1', 10)
              : fromIndex;
          } else {
            const targetItem = items[newIndex + 1];
            toIndex = targetItem
              ? parseInt(targetItem.getAttribute('data-task-index') || '-1', 10)
              : fromIndex;
          }

          if (fromIndex === -1 || toIndex === -1 || fromIndex === toIndex) return;
          options.onReorder(fromIndex, toIndex);
        },
      });

      instancesByContainer.set(sortableContainer, sortable);
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
    instancesByContainer.forEach((s) => s.destroy());
    instancesByContainer.clear();
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
