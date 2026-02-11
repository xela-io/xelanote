/**
 * History Store - Manages Undo/Redo functionality with localStorage persistence
 *
 * This store implements the Command Pattern for reversible operations.
 * Commands are persisted to localStorage to survive page reloads.
 *
 * Note: Content edits are NOT handled here - CodeMirror has its own undo/redo.
 * This only handles property changes: delete, create, rename title, move folder.
 */

import { CreateCommand } from '$lib/commands/CreateCommand';
import { DeleteCommand } from '$lib/commands/DeleteCommand';
import { MoveFolderCommand } from '$lib/commands/MoveFolderCommand';
import { RenameTitleCommand } from '$lib/commands/RenameTitleCommand';
import type { Command, CommandData } from '$lib/commands/types';

const MAX_HISTORY_SIZE = 50;
const STORAGE_KEY = 'xelanote_command_history';
const MAX_STORAGE_SIZE = 500 * 1024; // 500KB

// State
let undoStack = $state<Command[]>([]);
let redoStack = $state<Command[]>([]);
let isExecuting = $state(false);

/**
 * Executes a command and adds it to the undo stack.
 * Clears the redo stack (you can't redo after a new action).
 */
export async function executeCommand(command: Command): Promise<boolean> {
  if (isExecuting) {
    console.warn('Command already executing, please wait');
    return false;
  }

  isExecuting = true;
  try {
    const success = await command.execute();
    if (success) {
      undoStack.push(command);
      redoStack = []; // Clear redo stack on new action

      // Enforce max size
      if (undoStack.length > MAX_HISTORY_SIZE) {
        undoStack.shift(); // Remove oldest command
      }

      saveHistory();
    }
    return success;
  } finally {
    isExecuting = false;
  }
}

/**
 * Undoes the last command.
 */
export async function undo(): Promise<boolean> {
  if (!canUndo() || isExecuting) {
    return false;
  }

  isExecuting = true;
  try {
    const command = undoStack.pop();
    if (!command) return false;

    const success = await command.undo();
    if (success) {
      redoStack.push(command);
      saveHistory();
    } else {
      // Undo failed, put command back
      undoStack.push(command);
    }
    return success;
  } finally {
    isExecuting = false;
  }
}

/**
 * Redoes the last undone command.
 */
export async function redo(): Promise<boolean> {
  if (!canRedo() || isExecuting) {
    return false;
  }

  isExecuting = true;
  try {
    const command = redoStack.pop();
    if (!command) return false;

    const success = await command.execute();
    if (success) {
      undoStack.push(command);
      saveHistory();
    } else {
      // Redo failed, put command back
      redoStack.push(command);
    }
    return success;
  } finally {
    isExecuting = false;
  }
}

/**
 * Checks if undo is available.
 */
export function canUndo(): boolean {
  return undoStack.length > 0 && !isExecuting;
}

/**
 * Checks if redo is available.
 */
export function canRedo(): boolean {
  return redoStack.length > 0 && !isExecuting;
}

/**
 * Returns the description of the next undo command.
 */
export function getUndoDescription(): string | null {
  if (!canUndo()) return null;
  return undoStack[undoStack.length - 1].getDescription();
}

/**
 * Returns the description of the next redo command.
 */
export function getRedoDescription(): string | null {
  if (!canRedo()) return null;
  return redoStack[redoStack.length - 1].getDescription();
}

/**
 * Clears all history (both undo and redo stacks).
 */
export function clearHistory(): void {
  undoStack = [];
  redoStack = [];
  saveHistory();
}

/**
 * Saves history to localStorage.
 */
function saveHistory(): void {
  if (typeof localStorage === 'undefined') return;

  try {
    const serializedUndo = undoStack.map((cmd) => cmd.serialize());
    const serializedRedo = redoStack.map((cmd) => cmd.serialize());

    const historyData = {
      undo: serializedUndo,
      redo: serializedRedo,
    };

    const json = JSON.stringify(historyData);

    // Check size limit
    if (json.length > MAX_STORAGE_SIZE) {
      console.warn('History exceeds storage limit, trimming...');
      // Remove oldest undo commands until under limit
      while (undoStack.length > 0 && JSON.stringify(historyData).length > MAX_STORAGE_SIZE) {
        undoStack.shift();
        historyData.undo = undoStack.map((cmd) => cmd.serialize());
      }
    }

    localStorage.setItem(STORAGE_KEY, JSON.stringify(historyData));
  } catch (error) {
    if (error instanceof Error && error.name === 'QuotaExceededError') {
      console.error('localStorage quota exceeded, clearing history');
      clearHistory();
    } else {
      console.error('Failed to save history to localStorage:', error);
    }
  }
}

/**
 * Loads history from localStorage.
 */
export function loadHistory(): void {
  if (typeof localStorage === 'undefined') return;

  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) return;

    const historyData = parseHistoryData(stored);
    if (!historyData) {
      clearHistory();
      return;
    }

    // Deserialize commands
    undoStack = historyData.undo.map((data: CommandData) => deserializeCommand(data));
    redoStack = historyData.redo.map((data: CommandData) => deserializeCommand(data));
  } catch (error) {
    console.error('Failed to load history from localStorage:', error);
    clearHistory();
  }
}

type StoredHistoryData = { undo: CommandData[]; redo: CommandData[] };

function parseHistoryData(raw: string): StoredHistoryData | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }

  if (!parsed || typeof parsed !== 'object') return null;
  const data = parsed as { undo?: unknown; redo?: unknown };
  if (!Array.isArray(data.undo) || !Array.isArray(data.redo)) return null;
  if (!isCommandDataArray(data.undo) || !isCommandDataArray(data.redo)) return null;

  return {
    undo: data.undo,
    redo: data.redo,
  };
}

function isCommandDataArray(value: unknown[]): value is CommandData[] {
  return value.every((entry) => isCommandData(entry));
}

function isCommandData(value: unknown): value is CommandData {
  if (!value || typeof value !== 'object') return false;

  const maybe = value as {
    type?: unknown;
    timestamp?: unknown;
    noteId?: unknown;
    data?: unknown;
  };

  if (
    typeof maybe.type !== 'string' ||
    typeof maybe.timestamp !== 'number' ||
    typeof maybe.noteId !== 'string' ||
    !maybe.data ||
    typeof maybe.data !== 'object'
  ) {
    return false;
  }

  switch (maybe.type) {
    case 'delete':
      return isDeleteData(maybe.data);
    case 'create':
      return isCreateData(maybe.data);
    case 'rename-title':
      return isRenameTitleData(maybe.data);
    case 'move-folder':
      return isMoveFolderData(maybe.data);
    default:
      return false;
  }
}

function isDeleteData(value: unknown): boolean {
  if (!value || typeof value !== 'object') return false;
  const data = value as {
    noteId?: unknown;
    snapshot?: {
      title?: unknown;
      content?: unknown;
      folder_path?: unknown;
      version?: unknown;
    };
  };
  return (
    typeof data.noteId === 'string' &&
    !!data.snapshot &&
    typeof data.snapshot.title === 'string' &&
    typeof data.snapshot.content === 'string' &&
    typeof data.snapshot.folder_path === 'string' &&
    typeof data.snapshot.version === 'number'
  );
}

function isCreateData(value: unknown): boolean {
  if (!value || typeof value !== 'object') return false;
  const data = value as {
    noteId?: unknown;
    title?: unknown;
    content?: unknown;
    folder_path?: unknown;
  };
  return (
    typeof data.noteId === 'string' &&
    typeof data.title === 'string' &&
    typeof data.content === 'string' &&
    typeof data.folder_path === 'string'
  );
}

function isRenameTitleData(value: unknown): boolean {
  if (!value || typeof value !== 'object') return false;
  const data = value as {
    noteId?: unknown;
    oldTitle?: unknown;
    newTitle?: unknown;
    version?: unknown;
  };
  return (
    typeof data.noteId === 'string' &&
    typeof data.oldTitle === 'string' &&
    typeof data.newTitle === 'string' &&
    typeof data.version === 'number'
  );
}

function isMoveFolderData(value: unknown): boolean {
  if (!value || typeof value !== 'object') return false;
  const data = value as {
    noteId?: unknown;
    oldFolder?: unknown;
    newFolder?: unknown;
    version?: unknown;
  };
  return (
    typeof data.noteId === 'string' &&
    typeof data.oldFolder === 'string' &&
    typeof data.newFolder === 'string' &&
    typeof data.version === 'number'
  );
}

/**
 * Deserializes command data back into Command instances.
 */
function deserializeCommand(data: CommandData): Command {
  switch (data.type) {
    case 'delete':
      return new DeleteCommand(data.data);
    case 'create':
      return new CreateCommand(data.data);
    case 'rename-title':
      return new RenameTitleCommand(data.data);
    case 'move-folder':
      return new MoveFolderCommand(data.data);
    default: {
      // Exhaustive check
      const _exhaustive: never = data;
      throw new Error(`Unknown command type: ${String(_exhaustive)}`);
    }
  }
}

// Derived state for reactive checks
export const history = {
  get canUndo() {
    return canUndo();
  },
  get canRedo() {
    return canRedo();
  },
  get undoDescription() {
    return getUndoDescription();
  },
  get redoDescription() {
    return getRedoDescription();
  },
  get isExecuting() {
    return isExecuting;
  },
};
