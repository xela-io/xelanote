import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { _resetForTest, taskCollapse } from './task-collapse';

// Mock the API modules
vi.mock('$lib/api/notes', () => ({
  getNoteUserState: vi.fn().mockResolvedValue({ collapse_state: null }),
  updateNoteUserCollapseState: vi.fn().mockResolvedValue({ status: 'ok' }),
}));

vi.mock('$lib/api/client', () => ({
  ApiError: class ApiError extends Error {
    status: number;
    constructor(message: string, status: number) {
      super(message);
      this.status = status;
      this.name = 'ApiError';
    }
  },
}));

function createTaskListContainer(): HTMLDivElement {
  const container = document.createElement('div');
  container.innerHTML = `
    <ul class="contains-task-list">
      <li class="task-list-item">
        <input class="task-list-item-checkbox" type="checkbox">
        Offen
      </li>
      <li class="task-list-item">
        <input class="task-list-item-checkbox" type="checkbox" checked>
        Erledigt A
      </li>
      <li class="task-list-item">
        <input class="task-list-item-checkbox" type="checkbox" checked>
        Erledigt B
      </li>
    </ul>
  `;
  document.body.appendChild(container);
  return container;
}

async function flushFrame(): Promise<void> {
  await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
}

describe('task-collapse', () => {
  beforeEach(() => {
    localStorage.clear();
    document.body.innerHTML = '';
    _resetForTest();
  });

  afterEach(() => {
    localStorage.clear();
    document.body.innerHTML = '';
    _resetForTest();
  });

  it('persists collapsed-group open state across remounts for the same note', async () => {
    const options = {
      completedLabel: (count: number) => `${count} erledigt`,
      completedAriaLabel: (count: number) => `${count} erledigte Aufgaben`,
      noteId: 'note-1',
      revision: 'r1',
    };

    const first = createTaskListContainer();
    const firstAction = taskCollapse(first, options);
    await flushFrame();

    const firstDetails = first.querySelector(
      'details.completed-tasks-group'
    ) as HTMLDetailsElement | null;
    expect(firstDetails).not.toBeNull();
    expect(firstDetails?.open).toBe(false);

    firstDetails!.open = true;
    firstDetails!.dispatchEvent(new Event('toggle'));
    firstAction.destroy();

    const second = createTaskListContainer();
    const secondAction = taskCollapse(second, options);
    await flushFrame();

    const secondDetails = second.querySelector(
      'details.completed-tasks-group'
    ) as HTMLDetailsElement | null;
    expect(secondDetails).not.toBeNull();
    expect(secondDetails?.open).toBe(true);

    secondAction.destroy();
  });

  it('sets data-group-hash attribute on details elements', async () => {
    const options = {
      completedLabel: (count: number) => `${count} erledigt`,
      completedAriaLabel: (count: number) => `${count} erledigte Aufgaben`,
      noteId: 'note-1',
    };

    const container = createTaskListContainer();
    const action = taskCollapse(container, options);
    await flushFrame();

    const details = container.querySelector(
      'details.completed-tasks-group'
    ) as HTMLDetailsElement | null;
    expect(details).not.toBeNull();
    expect(details?.getAttribute('data-group-hash')).toBeTruthy();
    // Hash should be a base36 string
    expect(details?.getAttribute('data-group-hash')).toMatch(/^[0-9a-z]+$/);

    action.destroy();
  });

  describe('localStorage v1 → v2 migration', () => {
    it('migrates v1 flat keys to v2 nested structure', async () => {
      // Set up v1 data: "noteId-hash" → boolean
      localStorage.setItem(
        'xelanote-task-collapse-v1',
        JSON.stringify({
          'abc123-def-456-xy1': true,
          'abc123-def-456-xy2': false,
          'other-note-id-abc': true,
        })
      );

      const options = {
        completedLabel: (count: number) => `${count} erledigt`,
        completedAriaLabel: (count: number) => `${count} erledigte Aufgaben`,
        noteId: 'note-1',
      };

      const container = createTaskListContainer();
      const action = taskCollapse(container, options);
      await flushFrame();
      action.destroy();

      // v1 key should be removed
      expect(localStorage.getItem('xelanote-task-collapse-v1')).toBeNull();

      // v2 key should exist with nested structure
      const v2Raw = localStorage.getItem('xelanote-task-collapse-v2');
      expect(v2Raw).not.toBeNull();
      const v2 = JSON.parse(v2Raw!);
      // UUID "abc123-def-456" with hash "xy1" and "xy2"
      expect(v2['abc123-def-456']).toEqual({ xy1: true, xy2: false });
      // "other-note-id" with hash "abc"
      expect(v2['other-note-id']).toEqual({ abc: true });
    });

    it('skips migration if v2 already exists', async () => {
      localStorage.setItem('xelanote-task-collapse-v1', JSON.stringify({ 'note-1-abc': true }));
      localStorage.setItem(
        'xelanote-task-collapse-v2',
        JSON.stringify({ 'note-1': { xyz: false } })
      );

      const options = {
        completedLabel: (count: number) => `${count} erledigt`,
        completedAriaLabel: (count: number) => `${count} erledigte Aufgaben`,
        noteId: 'note-1',
      };

      const container = createTaskListContainer();
      const action = taskCollapse(container, options);
      await flushFrame();
      action.destroy();

      // v1 should still be present (no migration ran)
      expect(localStorage.getItem('xelanote-task-collapse-v1')).not.toBeNull();

      // v2 should have original data, not migrated data
      const v2 = JSON.parse(localStorage.getItem('xelanote-task-collapse-v2')!);
      expect(v2['note-1']['xyz']).toBe(false);
    });

    it('handles empty v1 state gracefully', async () => {
      localStorage.setItem('xelanote-task-collapse-v1', JSON.stringify({}));

      const options = {
        completedLabel: (count: number) => `${count} erledigt`,
        completedAriaLabel: (count: number) => `${count} erledigte Aufgaben`,
        noteId: 'note-1',
      };

      const container = createTaskListContainer();
      const action = taskCollapse(container, options);
      await flushFrame();
      action.destroy();

      // v1 should be removed
      expect(localStorage.getItem('xelanote-task-collapse-v1')).toBeNull();
    });
  });
});
