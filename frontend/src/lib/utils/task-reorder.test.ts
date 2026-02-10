/**
 * Tests for task-reorder utility functions
 */

import { Text } from '@codemirror/state';
import { describe, expect, it } from 'vitest';

import { calculateMoveChanges, getTasksInDocument } from './task-reorder';

describe('task-reorder', () => {
  describe('getTasksInDocument', () => {
    it('finds all task items in document', () => {
      const doc = Text.of([
        '# Header',
        '- [ ] Task 1',
        '- [x] Task 2',
        '- Normal item',
        '- [ ] Task 3',
      ]);

      const tasks = getTasksInDocument(doc);

      expect(tasks).toHaveLength(3);
      expect(tasks[0].index).toBe(0);
      expect(tasks[0].text).toBe('- [ ] Task 1');
      expect(tasks[1].index).toBe(1);
      expect(tasks[1].text).toBe('- [x] Task 2');
      expect(tasks[2].index).toBe(2);
      expect(tasks[2].text).toBe('- [ ] Task 3');
    });

    it('handles empty document', () => {
      const doc = Text.of(['']);
      const tasks = getTasksInDocument(doc);
      expect(tasks).toHaveLength(0);
    });

    it('handles document with no tasks', () => {
      const doc = Text.of(['# Header', '- Item 1', '- Item 2']);
      const tasks = getTasksInDocument(doc);
      expect(tasks).toHaveLength(0);
    });

    it('handles indented tasks', () => {
      const doc = Text.of(['- [ ] Task 1', '  - [ ] Nested task', '    - [x] Deep nested']);

      const tasks = getTasksInDocument(doc);

      expect(tasks).toHaveLength(3);
      expect(tasks[1].text).toBe('  - [ ] Nested task');
    });

    it('handles different list markers', () => {
      const doc = Text.of(['- [ ] Dash task', '* [ ] Star task', '+ [ ] Plus task']);

      const tasks = getTasksInDocument(doc);

      expect(tasks).toHaveLength(3);
    });
  });

  describe('calculateMoveChanges', () => {
    it('returns empty for same index', () => {
      const doc = Text.of(['- [ ] Task 1', '- [ ] Task 2']);

      const changes = calculateMoveChanges(doc, 0, 0);
      expect(changes).toHaveLength(0);
    });

    it('returns empty for invalid fromIndex', () => {
      const doc = Text.of(['- [ ] Task 1']);
      const changes = calculateMoveChanges(doc, 5, 0);
      expect(changes).toHaveLength(0);
    });

    it('returns empty for invalid toIndex', () => {
      const doc = Text.of(['- [ ] Task 1']);
      const changes = calculateMoveChanges(doc, 0, 5);
      expect(changes).toHaveLength(0);
    });

    it('calculates move down changes', () => {
      const doc = Text.of(['- [ ] Task 1', '- [ ] Task 2', '- [ ] Task 3']);

      const changes = calculateMoveChanges(doc, 0, 2);

      expect(changes).toHaveLength(2);
      // Should have delete and insert operations
      expect(changes.some((c) => 'insert' in c && c.insert === '')).toBe(true);
      expect(changes.some((c) => 'insert' in c && (c.insert as string).includes('Task 1'))).toBe(
        true
      );
    });

    it('calculates move up changes', () => {
      const doc = Text.of(['- [ ] Task 1', '- [ ] Task 2', '- [ ] Task 3']);

      const changes = calculateMoveChanges(doc, 2, 0);

      expect(changes).toHaveLength(2);
      // Should have insert and delete operations
      expect(changes.some((c) => 'insert' in c && (c.insert as string).includes('Task 3'))).toBe(
        true
      );
      expect(changes.some((c) => 'insert' in c && c.insert === '')).toBe(true);
    });
  });
});
