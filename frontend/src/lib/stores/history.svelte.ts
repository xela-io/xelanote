/**
 * History Store - Manages Undo/Redo functionality with localStorage persistence
 *
 * This store implements the Command Pattern for reversible operations.
 * Commands are persisted to localStorage to survive page reloads.
 *
 * Note: Content edits are NOT handled here - CodeMirror has its own undo/redo.
 * This only handles property changes: delete, create, rename title, move folder.
 */

import type { Command, CommandData } from '$lib/commands/types';
import { DeleteCommand } from '$lib/commands/DeleteCommand';
import { CreateCommand } from '$lib/commands/CreateCommand';
import { RenameTitleCommand } from '$lib/commands/RenameTitleCommand';
import { MoveFolderCommand } from '$lib/commands/MoveFolderCommand';

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

    const historyData = JSON.parse(stored);

    // Deserialize commands
    undoStack = historyData.undo.map((data: CommandData) => deserializeCommand(data));
    redoStack = historyData.redo.map((data: CommandData) => deserializeCommand(data));
  } catch (error) {
    console.error('Failed to load history from localStorage:', error);
    clearHistory();
  }
}

/**
 * Deserializes command data back into Command instances.
 */
function deserializeCommand(data: CommandData): Command {
  switch (data.type) {
    case 'delete':
      return new DeleteCommand(data.data as any);
    case 'create':
      return new CreateCommand(data.data as any);
    case 'rename-title':
      return new RenameTitleCommand(data.data as any);
    case 'move-folder':
      return new MoveFolderCommand(data.data as any);
    default:
      throw new Error(`Unknown command type: ${data.type}`);
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
