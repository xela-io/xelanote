import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { taskCollapse } from './task-collapse';

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
  });

  afterEach(() => {
    localStorage.clear();
    document.body.innerHTML = '';
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
});
