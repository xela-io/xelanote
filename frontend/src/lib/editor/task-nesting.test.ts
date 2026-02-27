import { describe, expect, it } from 'vitest';

import {
  buildTaskTree,
  computeDownwardPropagations,
  computeUpwardPropagations,
  findNodeByLine,
  getSubtreeLineRange,
} from './task-nesting';

describe('task-nesting', () => {
  describe('buildTaskTree', () => {
    it('builds flat list as all roots', () => {
      const roots = buildTaskTree([
        { lineNumber: 1, indent: '', checked: false },
        { lineNumber: 2, indent: '', checked: true },
        { lineNumber: 3, indent: '', checked: false },
      ]);
      expect(roots).toHaveLength(3);
      expect(roots[0].children).toHaveLength(0);
      expect(roots[1].children).toHaveLength(0);
    });

    it('builds one parent with children', () => {
      const roots = buildTaskTree([
        { lineNumber: 1, indent: '', checked: false },
        { lineNumber: 2, indent: '  ', checked: false },
        { lineNumber: 3, indent: '  ', checked: true },
      ]);
      expect(roots).toHaveLength(1);
      expect(roots[0].children).toHaveLength(2);
      expect(roots[0].children[0].lineNumber).toBe(2);
      expect(roots[0].children[1].lineNumber).toBe(3);
      expect(roots[0].children[0].parent).toBe(roots[0]);
    });

    it('builds multi-level nesting', () => {
      const roots = buildTaskTree([
        { lineNumber: 1, indent: '', checked: false },
        { lineNumber: 2, indent: '  ', checked: false },
        { lineNumber: 3, indent: '    ', checked: false },
      ]);
      expect(roots).toHaveLength(1);
      expect(roots[0].children).toHaveLength(1);
      expect(roots[0].children[0].children).toHaveLength(1);
      expect(roots[0].children[0].children[0].lineNumber).toBe(3);
    });

    it('handles mixed indents correctly', () => {
      const roots = buildTaskTree([
        { lineNumber: 1, indent: '', checked: false },
        { lineNumber: 2, indent: '  ', checked: false },
        { lineNumber: 3, indent: '', checked: true },
        { lineNumber: 4, indent: '  ', checked: false },
      ]);
      expect(roots).toHaveLength(2);
      expect(roots[0].children).toHaveLength(1);
      expect(roots[1].children).toHaveLength(1);
    });

    it('handles tab indentation', () => {
      const roots = buildTaskTree([
        { lineNumber: 1, indent: '', checked: false },
        { lineNumber: 2, indent: '\t', checked: false },
      ]);
      expect(roots).toHaveLength(1);
      expect(roots[0].children).toHaveLength(1);
      // Tab = 4 spaces, so nestLevel = 2
      expect(roots[0].children[0].nestLevel).toBe(2);
    });
  });

  describe('computeDownwardPropagations', () => {
    it('propagates checked state to all children', () => {
      const roots = buildTaskTree([
        { lineNumber: 1, indent: '', checked: false },
        { lineNumber: 2, indent: '  ', checked: false },
        { lineNumber: 3, indent: '  ', checked: false },
      ]);
      const changes = computeDownwardPropagations(roots[0], true);
      expect(changes).toHaveLength(2);
      expect(changes[0]).toEqual({ lineNumber: 2, newChecked: true });
      expect(changes[1]).toEqual({ lineNumber: 3, newChecked: true });
    });

    it('propagates unchecked state to all children', () => {
      const roots = buildTaskTree([
        { lineNumber: 1, indent: '', checked: true },
        { lineNumber: 2, indent: '  ', checked: true },
        { lineNumber: 3, indent: '  ', checked: true },
      ]);
      const changes = computeDownwardPropagations(roots[0], false);
      expect(changes).toHaveLength(2);
      expect(changes[0]).toEqual({ lineNumber: 2, newChecked: false });
      expect(changes[1]).toEqual({ lineNumber: 3, newChecked: false });
    });

    it('skips children already in target state', () => {
      const roots = buildTaskTree([
        { lineNumber: 1, indent: '', checked: false },
        { lineNumber: 2, indent: '  ', checked: true },
        { lineNumber: 3, indent: '  ', checked: false },
      ]);
      const changes = computeDownwardPropagations(roots[0], true);
      // Only line 3 needs change; line 2 is already checked
      expect(changes).toHaveLength(1);
      expect(changes[0]).toEqual({ lineNumber: 3, newChecked: true });
    });

    it('propagates through deep nesting', () => {
      const roots = buildTaskTree([
        { lineNumber: 1, indent: '', checked: false },
        { lineNumber: 2, indent: '  ', checked: false },
        { lineNumber: 3, indent: '    ', checked: false },
      ]);
      const changes = computeDownwardPropagations(roots[0], true);
      expect(changes).toHaveLength(2);
      expect(changes.map((c) => c.lineNumber)).toEqual([2, 3]);
    });
  });

  describe('computeUpwardPropagations', () => {
    it('checks parent when all siblings become checked', () => {
      const roots = buildTaskTree([
        { lineNumber: 1, indent: '', checked: false },
        { lineNumber: 2, indent: '  ', checked: true },
        { lineNumber: 3, indent: '  ', checked: false },
      ]);
      const child = findNodeByLine(roots, 3)!;
      const changes = computeUpwardPropagations(child, true);
      expect(changes).toHaveLength(1);
      expect(changes[0]).toEqual({ lineNumber: 1, newChecked: true });
    });

    it('does not check parent when not all siblings checked', () => {
      const roots = buildTaskTree([
        { lineNumber: 1, indent: '', checked: false },
        { lineNumber: 2, indent: '  ', checked: false },
        { lineNumber: 3, indent: '  ', checked: false },
      ]);
      const child = findNodeByLine(roots, 3)!;
      const changes = computeUpwardPropagations(child, true);
      // Line 2 is still unchecked, so parent stays unchecked
      expect(changes).toHaveLength(0);
    });

    it('unchecks parent when a child is unchecked', () => {
      const roots = buildTaskTree([
        { lineNumber: 1, indent: '', checked: true },
        { lineNumber: 2, indent: '  ', checked: true },
        { lineNumber: 3, indent: '  ', checked: true },
      ]);
      const child = findNodeByLine(roots, 2)!;
      const changes = computeUpwardPropagations(child, false);
      expect(changes).toHaveLength(1);
      expect(changes[0]).toEqual({ lineNumber: 1, newChecked: false });
    });

    it('propagates recursively to grandparent', () => {
      const roots = buildTaskTree([
        { lineNumber: 1, indent: '', checked: false },
        { lineNumber: 2, indent: '  ', checked: false },
        { lineNumber: 3, indent: '    ', checked: false },
      ]);
      const grandchild = findNodeByLine(roots, 3)!;
      const changes = computeUpwardPropagations(grandchild, true);
      // Line 3 is only child of line 2 → line 2 checked
      // Line 2 is only child of line 1 → line 1 checked
      expect(changes).toHaveLength(2);
      expect(changes[0]).toEqual({ lineNumber: 2, newChecked: true });
      expect(changes[1]).toEqual({ lineNumber: 1, newChecked: true });
    });

    it('does not uncheck parent that is already unchecked', () => {
      const roots = buildTaskTree([
        { lineNumber: 1, indent: '', checked: false },
        { lineNumber: 2, indent: '  ', checked: true },
      ]);
      const child = findNodeByLine(roots, 2)!;
      const changes = computeUpwardPropagations(child, false);
      // Parent already unchecked → no change
      expect(changes).toHaveLength(0);
    });
  });

  describe('getSubtreeLineRange', () => {
    it('returns only own line for task without children', () => {
      const lines = ['- [ ] Task A', '- [ ] Task B'];
      const range = getSubtreeLineRange(lines, 0);
      expect(range).toEqual({ start: 0, end: 0 });
    });

    it('returns full range for task with children', () => {
      const lines = ['- [ ] Parent', '  - [ ] Child 1', '  - [ ] Child 2', '- [ ] Next'];
      const range = getSubtreeLineRange(lines, 0);
      expect(range).toEqual({ start: 0, end: 2 });
    });

    it('returns full range for deeply nested', () => {
      const lines = [
        '- [ ] Parent',
        '  - [ ] Child',
        '    - [ ] Grandchild',
        '- [ ] Next',
      ];
      const range = getSubtreeLineRange(lines, 0);
      expect(range).toEqual({ start: 0, end: 2 });
    });

    it('stops at empty line', () => {
      const lines = ['- [ ] Parent', '  - [ ] Child', '', '- [ ] Other'];
      const range = getSubtreeLineRange(lines, 0);
      expect(range).toEqual({ start: 0, end: 1 });
    });

    it('returns child range correctly', () => {
      const lines = [
        '- [ ] Parent',
        '  - [ ] Child',
        '    - [ ] Grandchild',
        '  - [ ] Child 2',
      ];
      const range = getSubtreeLineRange(lines, 1);
      expect(range).toEqual({ start: 1, end: 2 });
    });

    it('handles last line in document', () => {
      const lines = ['- [ ] Parent', '  - [ ] Child'];
      const range = getSubtreeLineRange(lines, 0);
      expect(range).toEqual({ start: 0, end: 1 });
    });
  });
});
