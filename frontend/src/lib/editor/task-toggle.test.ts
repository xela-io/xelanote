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
});
