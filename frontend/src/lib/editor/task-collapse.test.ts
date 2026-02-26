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

  describe('server sync', () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    async function flushFrameFake(): Promise<void> {
      // With fake timers, advance by a frame (~16ms) to trigger RAF
      await vi.advanceTimersByTimeAsync(20);
    }

    it('calls updateNoteUserCollapseState after toggle', async () => {
      const { updateNoteUserCollapseState } = await import('$lib/api/notes');
      const mockUpdate = vi.mocked(updateNoteUserCollapseState);
      mockUpdate.mockClear();

      const options = {
        completedLabel: (count: number) => `${count} erledigt`,
        completedAriaLabel: (count: number) => `${count} erledigte Aufgaben`,
        noteId: 'note-sync-1',
        revision: 'r1',
      };

      const container = createTaskListContainer();
      const action = taskCollapse(container, options);
      await flushFrameFake();

      const details = container.querySelector(
        'details.completed-tasks-group'
      ) as HTMLDetailsElement;
      expect(details).not.toBeNull();

      // Toggle open
      details.open = true;
      details.dispatchEvent(new Event('toggle'));

      // Wait for the 500ms debounce timer to fire
      await vi.advanceTimersByTimeAsync(600);

      expect(mockUpdate).toHaveBeenCalledTimes(1);
      expect(mockUpdate).toHaveBeenCalledWith('note-sync-1', expect.any(Object));

      // The state object should contain the group hash → true
      const stateArg = mockUpdate.mock.calls[0][1];
      const keys = Object.keys(stateArg);
      expect(keys.length).toBeGreaterThan(0);
      expect(Object.values(stateArg).every((v) => typeof v === 'boolean')).toBe(true);

      action.destroy();
    });

    it('does NOT call server sync when noteId is empty', async () => {
      const { updateNoteUserCollapseState, getNoteUserState } = await import('$lib/api/notes');
      const mockUpdate = vi.mocked(updateNoteUserCollapseState);
      const mockGet = vi.mocked(getNoteUserState);
      mockUpdate.mockClear();
      mockGet.mockClear();

      const options = {
        completedLabel: (count: number) => `${count} erledigt`,
        completedAriaLabel: (count: number) => `${count} erledigte Aufgaben`,
        noteId: '', // empty noteId
        revision: 'r1',
      };

      const container = createTaskListContainer();
      const action = taskCollapse(container, options);
      await flushFrameFake();

      const details = container.querySelector(
        'details.completed-tasks-group'
      ) as HTMLDetailsElement;

      if (details) {
        details.open = true;
        details.dispatchEvent(new Event('toggle'));
        await vi.advanceTimersByTimeAsync(600);
      }

      // Should NOT have called server sync with empty noteId
      expect(mockUpdate).not.toHaveBeenCalled();
      // Should NOT have called loadFromServer with empty noteId
      expect(mockGet).not.toHaveBeenCalled();

      action.destroy();
    });

    it('calls server sync after action update sets correct noteId', async () => {
      const { updateNoteUserCollapseState } = await import('$lib/api/notes');
      const mockUpdate = vi.mocked(updateNoteUserCollapseState);
      mockUpdate.mockClear();

      const options = {
        completedLabel: (count: number) => `${count} erledigt`,
        completedAriaLabel: (count: number) => `${count} erledigte Aufgaben`,
        noteId: '', // starts empty
        revision: 'r1',
      };

      const container = createTaskListContainer();
      const action = taskCollapse(container, options);
      await flushFrameFake();

      // Simulate $effect setting noteId — triggers action update
      const updatedOptions = { ...options, noteId: 'note-updated-1' };
      action.update(updatedOptions);
      await flushFrameFake();

      const details = container.querySelector(
        'details.completed-tasks-group'
      ) as HTMLDetailsElement;
      expect(details).not.toBeNull();

      details!.open = true;
      details!.dispatchEvent(new Event('toggle'));
      await vi.advanceTimersByTimeAsync(600);

      expect(mockUpdate).toHaveBeenCalledTimes(1);
      expect(mockUpdate).toHaveBeenCalledWith('note-updated-1', expect.any(Object));

      action.destroy();
    });
  });

  describe('page navigation lifecycle', () => {
    it('preserves state across destroy/recreate with empty noteId → real noteId lifecycle', async () => {
      // Simulate real Svelte lifecycle:
      // 1. Component mounts: taskCollapse created with noteId ''
      // 2. $effect fires: action.update() with real noteId
      // 3. User toggles a group open
      // 4. Component destroyed (navigation away)
      // 5. Component recreated: taskCollapse created with noteId ''
      // 6. $effect fires: action.update() with real noteId
      // 7. State should be restored

      const baseOptions = {
        completedLabel: (count: number) => `${count} erledigt`,
        completedAriaLabel: (count: number) => `${count} erledigte Aufgaben`,
      };

      // --- First visit ---
      const container1 = createTaskListContainer();
      const action1 = taskCollapse(container1, { ...baseOptions, noteId: '', revision: '' });
      await flushFrame();

      // $effect sets noteId
      action1.update({ ...baseOptions, noteId: 'note-lifecycle-1', revision: 'r1' });
      await flushFrame();

      const details1 = container1.querySelector(
        'details.completed-tasks-group'
      ) as HTMLDetailsElement | null;
      expect(details1).not.toBeNull();
      expect(details1?.open).toBe(false);

      // User opens the completed group
      details1!.open = true;
      details1!.dispatchEvent(new Event('toggle'));

      // Verify state is saved in localStorage
      const storedBefore = localStorage.getItem('xelanote-task-collapse-v2');
      expect(storedBefore).not.toBeNull();
      const parsedBefore = JSON.parse(storedBefore!);
      // State must be under the real noteId, not under ''
      expect(parsedBefore['note-lifecycle-1']).toBeDefined();
      expect(Object.values(parsedBefore['note-lifecycle-1']).some((v) => v === true)).toBe(true);

      // Navigation away: destroy
      action1.destroy();

      // --- Second visit (navigate back) ---
      const container2 = createTaskListContainer();
      const action2 = taskCollapse(container2, { ...baseOptions, noteId: '', revision: '' });
      await flushFrame();

      // $effect sets noteId
      action2.update({ ...baseOptions, noteId: 'note-lifecycle-1', revision: 'r1' });
      await flushFrame();

      const details2 = container2.querySelector(
        'details.completed-tasks-group'
      ) as HTMLDetailsElement | null;
      expect(details2).not.toBeNull();
      // State must be preserved: user had opened the group
      expect(details2?.open).toBe(true);

      action2.destroy();
    });

    it('does NOT store toggle state under empty noteId', async () => {
      const baseOptions = {
        completedLabel: (count: number) => `${count} erledigt`,
        completedAriaLabel: (count: number) => `${count} erledigte Aufgaben`,
      };

      // Mount with empty noteId, content is already rendered
      const container = createTaskListContainer();
      const action = taskCollapse(container, { ...baseOptions, noteId: '', revision: 'r1' });
      await flushFrame();

      const details = container.querySelector(
        'details.completed-tasks-group'
      ) as HTMLDetailsElement | null;

      if (details) {
        // Toggle while noteId is still ''
        details.open = true;
        details.dispatchEvent(new Event('toggle'));

        // Check localStorage — state should NOT be under '' key
        const stored = localStorage.getItem('xelanote-task-collapse-v2');
        if (stored) {
          const parsed = JSON.parse(stored);
          expect(parsed['']).toBeUndefined();
        }
      }

      action.destroy();
    });

    it('preserves state when component is reused (SvelteKit same-route navigation)', async () => {
      // Simulates navigating /note/A → /note/B → /note/A
      // where the component is reused (not destroyed)
      const baseOptions = {
        completedLabel: (count: number) => `${count} erledigt`,
        completedAriaLabel: (count: number) => `${count} erledigte Aufgaben`,
      };

      const container = createTaskListContainer();
      const action = taskCollapse(container, { ...baseOptions, noteId: 'note-A', revision: 'r1' });
      await flushFrame();

      // Toggle on note A
      const detailsA = container.querySelector(
        'details.completed-tasks-group'
      ) as HTMLDetailsElement;
      expect(detailsA).not.toBeNull();
      detailsA.open = true;
      detailsA.dispatchEvent(new Event('toggle'));

      // Navigate to note B (action.update with new noteId, new content)
      // Replace container content to simulate different note
      container.innerHTML = `
        <ul class="contains-task-list">
          <li class="task-list-item">
            <input class="task-list-item-checkbox" type="checkbox">
            Task B open
          </li>
          <li class="task-list-item">
            <input class="task-list-item-checkbox" type="checkbox" checked>
            Task B done
          </li>
        </ul>
      `;
      action.update({ ...baseOptions, noteId: 'note-B', revision: 'r2' });
      await flushFrame();

      // Navigate back to note A (action.update with original noteId and content)
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
      action.update({ ...baseOptions, noteId: 'note-A', revision: 'r1' });
      await flushFrame();

      // Check that note A's collapse state is restored
      const detailsA2 = container.querySelector(
        'details.completed-tasks-group'
      ) as HTMLDetailsElement | null;
      expect(detailsA2).not.toBeNull();
      expect(detailsA2?.open).toBe(true);

      action.destroy();
    });
  });

  describe('Idiomorph integration', () => {
    const TASK_LIST_HTML = `
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

    const TASK_LIST_HTML_B = `
      <ul class="contains-task-list">
        <li class="task-list-item">
          <input class="task-list-item-checkbox" type="checkbox">
          Other open task
        </li>
        <li class="task-list-item">
          <input class="task-list-item-checkbox" type="checkbox" checked>
          Other done task
        </li>
      </ul>
    `;

    async function morphContainer(container: HTMLElement, html: string) {
      const { Idiomorph } = await import('idiomorph');
      Idiomorph.morph(container, html, {
        morphStyle: 'innerHTML',
        ignoreActiveValue: true,
        callbacks: {
          beforeNodeMorphed(oldNode: Node, newNode: Node) {
            if (oldNode instanceof HTMLDetailsElement && newNode instanceof HTMLDetailsElement) {
              (newNode as HTMLDetailsElement).open = (oldNode as HTMLDetailsElement).open;
            }
            return true;
          },
        },
      });
    }

    it('restores state after Idiomorph removes wrappers (same content re-render)', async () => {
      const baseOptions = {
        completedLabel: (count: number) => `${count} erledigt`,
        completedAriaLabel: (count: number) => `${count} erledigte Aufgaben`,
      };

      const container = document.createElement('div');
      container.innerHTML = TASK_LIST_HTML;
      document.body.appendChild(container);

      // Create action with real noteId
      const action = taskCollapse(container, {
        ...baseOptions,
        noteId: 'note-morph-1',
        revision: 'r1',
      });
      await flushFrame();

      // Wrappers should exist
      let details = container.querySelector('details.completed-tasks-group') as HTMLDetailsElement;
      expect(details).not.toBeNull();
      expect(details.open).toBe(false);

      // User opens the group
      details.open = true;
      details.dispatchEvent(new Event('toggle'));

      // Simulate previewRenderer morph (same content, destroys wrappers)
      await morphContainer(container, TASK_LIST_HTML);

      // After morph, wrappers should be gone
      expect(container.querySelector('details.completed-tasks-group')).toBeNull();

      // Simulate revision change → action.update()
      action.update({ ...baseOptions, noteId: 'note-morph-1', revision: 'r2' });
      await flushFrame();

      // Wrappers should be recreated with restored state
      details = container.querySelector('details.completed-tasks-group') as HTMLDetailsElement;
      expect(details).not.toBeNull();
      expect(details.open).toBe(true); // State should be preserved!

      action.destroy();
    });

    it('restores state after full note-to-note navigation cycle with morph', async () => {
      const baseOptions = {
        completedLabel: (count: number) => `${count} erledigt`,
        completedAriaLabel: (count: number) => `${count} erledigte Aufgaben`,
      };

      // --- Page load: note A ---
      const container = document.createElement('div');
      container.innerHTML = TASK_LIST_HTML;
      document.body.appendChild(container);

      // 1. Mount with empty noteId (simulates Svelte mount before $effect)
      const action = taskCollapse(container, { ...baseOptions, noteId: '', revision: '' });
      await flushFrame();

      // No wrappers yet (noteId is empty)
      expect(container.querySelector('details.completed-tasks-group')).toBeNull();

      // 2. $effect sets noteId and revision
      action.update({ ...baseOptions, noteId: 'note-A', revision: 'r1' });
      await flushFrame();

      let details = container.querySelector('details.completed-tasks-group') as HTMLDetailsElement;
      expect(details).not.toBeNull();

      // 3. User opens the completed group
      details.open = true;
      details.dispatchEvent(new Event('toggle'));

      // --- Navigate to note B: previewRenderer morphs, noteId changes ---

      // 4. previewRenderer morphs to note B's content
      await morphContainer(container, TASK_LIST_HTML_B);

      // 5. noteId and revision change
      action.update({ ...baseOptions, noteId: 'note-B', revision: 'r2' });
      await flushFrame();

      // Note B should have its own wrapper
      const detailsB = container.querySelector(
        'details.completed-tasks-group'
      ) as HTMLDetailsElement;
      expect(detailsB).not.toBeNull();

      // --- Navigate back to note A: previewRenderer morphs back ---

      // 6. previewRenderer morphs back to note A's content
      await morphContainer(container, TASK_LIST_HTML);

      // 7. noteId and revision change back
      action.update({ ...baseOptions, noteId: 'note-A', revision: 'r1' });
      await flushFrame();

      // 8. Check: note A's state should be restored!
      details = container.querySelector('details.completed-tasks-group') as HTMLDetailsElement;
      expect(details).not.toBeNull();
      expect(details.open).toBe(true); // User had opened it in step 3

      action.destroy();
    });
  });

  it('survives full page reload (module state lost, only localStorage remains)', async () => {
    // Simulate: toggle → save to localStorage → full reload → state restored
    const baseOptions = {
      completedLabel: (count: number) => `${count} erledigt`,
      completedAriaLabel: (count: number) => `${count} erledigte Aufgaben`,
    };

    // --- First session ---
    const container1 = createTaskListContainer();
    const action1 = taskCollapse(container1, {
      ...baseOptions,
      noteId: 'note-reload-1',
      revision: 'r1',
    });
    await flushFrame();

    const details1 = container1.querySelector(
      'details.completed-tasks-group'
    ) as HTMLDetailsElement;
    expect(details1).not.toBeNull();

    // User opens the group
    details1.open = true;
    details1.dispatchEvent(new Event('toggle'));
    action1.destroy();

    // Verify localStorage has the state
    const stored = localStorage.getItem('xelanote-task-collapse-v2');
    expect(stored).not.toBeNull();
    expect(JSON.parse(stored!)['note-reload-1']).toBeDefined();

    // --- Simulate full page reload: clear module state but keep localStorage ---
    _resetForTest();

    // --- Second session ---
    const container2 = createTaskListContainer();
    const action2 = taskCollapse(container2, {
      ...baseOptions,
      noteId: 'note-reload-1',
      revision: 'r1',
    });
    await flushFrame();

    const details2 = container2.querySelector(
      'details.completed-tasks-group'
    ) as HTMLDetailsElement;
    expect(details2).not.toBeNull();
    expect(details2.open).toBe(true); // State restored from localStorage!

    action2.destroy();
  });

  it('survives full page reload with empty-noteId lifecycle', async () => {
    // Simulate: toggle → save → full reload → mount with noteId '' → update → state restored
    const baseOptions = {
      completedLabel: (count: number) => `${count} erledigt`,
      completedAriaLabel: (count: number) => `${count} erledigte Aufgaben`,
    };

    // --- First session ---
    const container1 = createTaskListContainer();
    const action1 = taskCollapse(container1, { ...baseOptions, noteId: '', revision: '' });
    await flushFrame();
    action1.update({ ...baseOptions, noteId: 'note-reload-2', revision: 'r1' });
    await flushFrame();

    const details1 = container1.querySelector(
      'details.completed-tasks-group'
    ) as HTMLDetailsElement;
    expect(details1).not.toBeNull();

    details1.open = true;
    details1.dispatchEvent(new Event('toggle'));
    action1.destroy();

    // --- Simulate full page reload ---
    _resetForTest();

    // --- Second session: same lifecycle (mount with '' → update to real noteId) ---
    const container2 = createTaskListContainer();
    const action2 = taskCollapse(container2, { ...baseOptions, noteId: '', revision: '' });
    await flushFrame();
    action2.update({ ...baseOptions, noteId: 'note-reload-2', revision: 'r1' });
    await flushFrame();

    const details2 = container2.querySelector(
      'details.completed-tasks-group'
    ) as HTMLDetailsElement;
    expect(details2).not.toBeNull();
    expect(details2.open).toBe(true); // State restored from localStorage!

    action2.destroy();
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
