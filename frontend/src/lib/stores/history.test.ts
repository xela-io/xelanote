import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { Command, CommandData } from '$lib/commands/types';
import * as history from '$lib/stores/history.svelte';

function createCommand(options?: {
  id?: string;
  executeResult?: boolean;
  undoResult?: boolean;
  description?: string;
}): Command {
  const id = options?.id ?? 'note-1';
  const description = options?.description ?? 'create';
  const executeResult = options?.executeResult ?? true;
  const undoResult = options?.undoResult ?? true;

  return {
    execute: vi.fn().mockResolvedValue(executeResult),
    undo: vi.fn().mockResolvedValue(undoResult),
    serialize: (): CommandData => ({
      type: 'create',
      timestamp: Date.now(),
      noteId: id,
      data: {
        noteId: id,
        title: 'title',
        content: 'content',
        folder_path: '/',
      },
    }),
    getDescription: () => description,
  };
}

describe('history store', () => {
  beforeEach(() => {
    history.clearHistory();
    localStorage.clear();
  });

  it('should push command to undo stack when executeCommand succeeds', async () => {
    const command = createCommand({ description: 'cmd-1' });

    const result = await history.executeCommand(command);

    expect(result).toBe(true);
    expect(history.canUndo()).toBe(true);
    expect(history.getUndoDescription()).toBe('cmd-1');
  });

  it('should not push command when executeCommand fails', async () => {
    const command = createCommand({ executeResult: false });

    const result = await history.executeCommand(command);

    expect(result).toBe(false);
    expect(history.canUndo()).toBe(false);
  });

  it('should clear redo stack when executing after undo', async () => {
    const first = createCommand({ description: 'first' });
    const second = createCommand({ description: 'second' });
    const third = createCommand({ description: 'third' });

    await history.executeCommand(first);
    await history.executeCommand(second);
    await history.undo();

    expect(history.canRedo()).toBe(true);

    await history.executeCommand(third);

    expect(history.canRedo()).toBe(false);
  });
});
