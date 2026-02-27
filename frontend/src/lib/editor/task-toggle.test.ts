import { describe, expect, it, vi } from 'vitest';

import { toggleTaskByIndex, toggleTaskByLine } from './task-toggle';

describe('toggleTaskByIndex', () => {
  it('queues the correct task text when a toggled task is reordered', () => {
    const initial = '- [ ] A\n- [ ] B\n- [x] C';
    let content = initial;
    const queueTaskEvent = vi.fn();
    const scheduleAutoSave = vi.fn();

    toggleTaskByIndex({
      checkboxIndex: 0,
      checked: true,
      getContent: () => content,
      setContent: (next) => {
        content = next;
      },
      scheduleAutoSave,
      queueTaskEvent,
      noteId: 'note-1',
    });

    expect(content).toBe('- [ ] B\n- [x] A\n- [x] C');
    expect(queueTaskEvent).toHaveBeenCalledTimes(1);
    expect(queueTaskEvent).toHaveBeenCalledWith('note-1', 'A', 0, 'completed');
    expect(scheduleAutoSave).toHaveBeenCalledTimes(1);
  });

  it('toggles task state without reordering when not needed', () => {
    let content = '- [ ] A\n- [x] B';
    const queueTaskEvent = vi.fn();

    toggleTaskByIndex({
      checkboxIndex: 1,
      checked: false,
      getContent: () => content,
      setContent: (next) => {
        content = next;
      },
      scheduleAutoSave: vi.fn(),
      queueTaskEvent,
      noteId: 'note-2',
    });

    expect(content).toBe('- [ ] A\n- [ ] B');
    expect(queueTaskEvent).toHaveBeenCalledWith('note-2', 'B', 1, 'reopened');
  });

  it('ignores empty task markers so preview indices stay aligned', () => {
    let content = '- [ ] \n- [ ] A\n- [ ] B';
    const queueTaskEvent = vi.fn();

    // Preview index 0 refers to "A" because "- [ ] " is not rendered as checkbox.
    toggleTaskByIndex({
      checkboxIndex: 0,
      checked: true,
      getContent: () => content,
      setContent: (next) => {
        content = next;
      },
      scheduleAutoSave: vi.fn(),
      queueTaskEvent,
      noteId: 'note-3',
    });

    expect(content).toBe('- [ ] \n- [ ] B\n- [x] A');
    expect(queueTaskEvent).toHaveBeenCalledWith('note-3', 'A', 0, 'completed');
  });

  it('can toggle by source line number for stable preview targeting', () => {
    let content = '- [ ] \n- [ ] A\n- [ ] B';
    const queueTaskEvent = vi.fn();

    toggleTaskByLine({
      lineNumber: 2,
      checked: true,
      getContent: () => content,
      setContent: (next) => {
        content = next;
      },
      scheduleAutoSave: vi.fn(),
      queueTaskEvent,
      noteId: 'note-4',
    });

    expect(content).toBe('- [ ] \n- [ ] B\n- [x] A');
    expect(queueTaskEvent).toHaveBeenCalledWith('note-4', 'A', 0, 'completed');
  });

  it('child click does not reorder — child is toggled, parent stays', () => {
    let content = '- [ ] Parent\n  - [ ] Child A\n  - [ ] Child B';
    toggleTaskByIndex({
      checkboxIndex: 1,
      checked: true,
      getContent: () => content,
      setContent: (next) => { content = next; },
      scheduleAutoSave: vi.fn(),
      queueTaskEvent: vi.fn(),
    });

    // Child A is checked, parent unchanged, no reorder
    expect(content).toBe('- [ ] Parent\n  - [x] Child A\n  - [ ] Child B');
  });

  it('checking last unchecked child auto-checks parent', () => {
    let content = '- [ ] Parent\n  - [x] Child A\n  - [ ] Child B';
    toggleTaskByIndex({
      checkboxIndex: 2, // Child B
      checked: true,
      getContent: () => content,
      setContent: (next) => { content = next; },
      scheduleAutoSave: vi.fn(),
      queueTaskEvent: vi.fn(),
    });

    // Both children checked → parent auto-checked
    expect(content).toBe('- [x] Parent\n  - [x] Child A\n  - [x] Child B');
  });

  it('parent check propagates to all children', () => {
    let content = '- [ ] Parent\n  - [ ] Child A\n  - [ ] Child B';
    toggleTaskByIndex({
      checkboxIndex: 0, // Parent
      checked: true,
      getContent: () => content,
      setContent: (next) => { content = next; },
      scheduleAutoSave: vi.fn(),
      queueTaskEvent: vi.fn(),
    });

    // Parent checked → all children checked, then moved
    const lines = content.split('\n');
    // All three should be checked
    expect(lines[lines.length - 3]).toContain('[x] Parent');
    expect(lines[lines.length - 2]).toContain('[x] Child A');
    expect(lines[lines.length - 1]).toContain('[x] Child B');
  });

  it('parent uncheck propagates to all children', () => {
    let content = '- [ ] Other\n- [x] Parent\n  - [x] Child A\n  - [x] Child B';
    toggleTaskByIndex({
      checkboxIndex: 1, // Parent (index 1 because "Other" is index 0)
      checked: false,
      getContent: () => content,
      setContent: (next) => { content = next; },
      scheduleAutoSave: vi.fn(),
      queueTaskEvent: vi.fn(),
    });

    // Parent unchecked → all children unchecked
    expect(content).toContain('- [ ] Parent');
    expect(content).toContain('  - [ ] Child A');
    expect(content).toContain('  - [ ] Child B');
  });

  it('unchecking a child moves entire parent subtree back to unchecked section', () => {
    // Scenario: parent was checked (with children), sitting in the checked area.
    // User unchecks one child → parent auto-unchecks → entire block moves back.
    let content = '- [ ] A\n- [x] B\n- [x] Obst\n  - [x] Äpfel\n  - [x] Bananen';
    toggleTaskByIndex({
      checkboxIndex: 3, // Äpfel (indices: A=0, B=1, Obst=2, Äpfel=3, Bananen=4)
      checked: false,
      getContent: () => content,
      setContent: (next) => { content = next; },
      scheduleAutoSave: vi.fn(),
      queueTaskEvent: vi.fn(),
    });

    const lines = content.split('\n');
    // Obst (now unchecked) should be among unchecked items, BEFORE checked B
    // Expected order: A (unchecked), Obst+children (unchecked subtree), B (checked)
    expect(lines[0]).toBe('- [ ] A');
    expect(lines[1]).toBe('- [ ] Obst');
    expect(lines[2]).toBe('  - [ ] Äpfel');
    expect(lines[3]).toBe('  - [x] Bananen');
    expect(lines[4]).toBe('- [x] B');
  });

  it('unchecking a child when parent is sole checked top-level: subtree stays in place', () => {
    // If parent is the only top-level task and it becomes unchecked, no move needed.
    let content = '- [x] Obst\n  - [x] Äpfel\n  - [x] Bananen';
    toggleTaskByIndex({
      checkboxIndex: 1, // Äpfel
      checked: false,
      getContent: () => content,
      setContent: (next) => { content = next; },
      scheduleAutoSave: vi.fn(),
      queueTaskEvent: vi.fn(),
    });

    // Parent unchecked, child unchecked, Bananen still checked. No reorder needed.
    expect(content).toBe('- [ ] Obst\n  - [ ] Äpfel\n  - [x] Bananen');
  });

  it('checking last child auto-checks parent and moves subtree to checked section', () => {
    let content = '- [ ] A\n- [ ] Obst\n  - [x] Äpfel\n  - [ ] Bananen\n- [x] C';
    toggleTaskByIndex({
      checkboxIndex: 3, // Bananen (A=0, Obst=1, Äpfel=2, Bananen=3, C=4)
      checked: true,
      getContent: () => content,
      setContent: (next) => { content = next; },
      scheduleAutoSave: vi.fn(),
      queueTaskEvent: vi.fn(),
    });

    const lines = content.split('\n');
    // Obst (now checked via propagation) should move to checked section after A
    // Expected: A (unchecked), then Obst+children (checked), C (checked)
    expect(lines[0]).toBe('- [ ] A');
    expect(lines[1]).toBe('- [x] Obst');
    expect(lines[2]).toBe('  - [x] Äpfel');
    expect(lines[3]).toBe('  - [x] Bananen');
    expect(lines[4]).toBe('- [x] C');
  });
});
