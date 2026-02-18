import { describe, expect, it, vi } from 'vitest';

import {
  determineMoveTarget,
  getSortableIndices,
  startCMObserver,
  stopCMObserver,
} from './task-sortable';

describe('task-sortable', () => {
  describe('determineMoveTarget', () => {
    // Moved DOWN: fromLine is smaller than prevLine (the item above in the new DOM)
    it('detects downward move when fromLine < prevLine', () => {
      // Task at line 3 dragged down past task at line 8
      // New DOM order: [..., line 8, line 3, line 10, ...]
      expect(determineMoveTarget(3, 8, 10)).toBe(8);
    });

    it('detects downward move to last position', () => {
      // Task at line 3 dragged to the end (no next item)
      // New DOM order: [..., line 10, line 3]
      expect(determineMoveTarget(3, 10, -1)).toBe(10);
    });

    // Moved UP: fromLine is larger than nextLine (the item below in the new DOM)
    it('detects upward move when fromLine > nextLine', () => {
      // Task at line 10 dragged up past task at line 5
      // New DOM order: [..., line 3, line 10, line 5, ...]
      expect(determineMoveTarget(10, 3, 5)).toBe(5);
    });

    it('detects upward move to first position', () => {
      // Task at line 10 dragged to the very top (no prev item)
      // New DOM order: [line 10, line 3, ...]
      expect(determineMoveTarget(10, -1, 3)).toBe(3);
    });

    // No movement: item is still between its original neighbors
    it('returns -1 when item did not move (between original neighbors)', () => {
      // Task at line 5 is still between line 3 and line 8
      expect(determineMoveTarget(5, 3, 8)).toBe(-1);
    });

    it('returns -1 when item is at the start and already first', () => {
      // Task at line 2 is already the first item, next is line 5
      expect(determineMoveTarget(2, -1, 5)).toBe(-1);
    });

    it('returns -1 when item is at the end and already last', () => {
      // Task at line 10 is already the last item, prev is line 5
      expect(determineMoveTarget(10, 5, -1)).toBe(-1);
    });

    it('returns -1 when item is the only draggable', () => {
      expect(determineMoveTarget(5, -1, -1)).toBe(-1);
    });

    // Regression: items far down in the document should produce the correct
    // target, not jump to the top.
    it('handles items far down the document correctly (moved up)', () => {
      // Task at line 50 dragged up past task at line 30
      // New DOM: [..., line 25, line 50, line 30, ...]
      expect(determineMoveTarget(50, 25, 30)).toBe(30);
    });

    it('handles items far down the document correctly (moved down)', () => {
      // Task at line 5 dragged down past task at line 50
      // New DOM: [..., line 50, line 5, line 55, ...]
      expect(determineMoveTarget(5, 50, 55)).toBe(50);
    });

    // Non-contiguous tasks (with non-task lines between them)
    it('works with non-contiguous task lines', () => {
      // Tasks at lines 2, 7, 12, 18. Drag line 18 up to between 7 and 12.
      // New DOM: [line 2, line 7, line 18, line 12]
      expect(determineMoveTarget(18, 7, 12)).toBe(12);
    });
  });

  describe('getSortableIndices', () => {
    it('prefers draggable indices when available', () => {
      const result = getSortableIndices({
        oldIndex: 5,
        newIndex: 7,
        oldDraggableIndex: 1,
        newDraggableIndex: 2,
      });

      expect(result).toEqual({ oldIndex: 1, newIndex: 2 });
    });

    it('falls back to regular indices when draggable indices are missing', () => {
      const result = getSortableIndices({
        oldIndex: 3,
        newIndex: 4,
      });

      expect(result).toEqual({ oldIndex: 3, newIndex: 4 });
    });

    it('returns null when one index is missing', () => {
      const result = getSortableIndices({
        oldDraggableIndex: 2,
      });

      expect(result).toBeNull();
    });

    it('requires both draggable indices in draggable-only mode', () => {
      const result = getSortableIndices(
        {
          oldDraggableIndex: 1,
          newIndex: 2,
        },
        { draggableOnly: true }
      );

      expect(result).toBeNull();
    });
  });

  describe('stopCMObserver / startCMObserver', () => {
    it('calls observer.stop() on the EditorView', () => {
      const stop = vi.fn();
      const mockView = { observer: { stop, start: vi.fn() } };

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      stopCMObserver(mockView as any);
      expect(stop).toHaveBeenCalledOnce();
    });

    it('calls observer.start() on the EditorView', () => {
      const start = vi.fn();
      const mockView = { observer: { stop: vi.fn(), start } };

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      startCMObserver(mockView as any);
      expect(start).toHaveBeenCalledOnce();
    });

    it('handles missing observer gracefully', () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const mockView = {} as any;

      expect(() => stopCMObserver(mockView)).not.toThrow();
      expect(() => startCMObserver(mockView)).not.toThrow();
    });
  });
});
