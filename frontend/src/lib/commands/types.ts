/**
 * Command Pattern for Undo/Redo System
 *
 * This implements the Command Pattern to support undo/redo functionality
 * for note operations (delete, create, rename title, move folder).
 *
 * Note: Content edits are handled separately by CodeMirror's built-in undo/redo.
 */

/**
 * Base Command interface that all commands must implement.
 */
export interface Command {
  /**
   * Executes the command.
   * @returns true if successful, false otherwise
   */
  execute(): Promise<boolean>;

  /**
   * Undoes the command, reverting to the previous state.
   * @returns true if successful, false otherwise
   */
  undo(): Promise<boolean>;

  /**
   * Serializes the command for localStorage persistence.
   * @returns Command data that can be JSON.stringify'd
   */
  serialize(): CommandData;

  /**
   * Returns a human-readable description of the command.
   * Used for debugging and UI display.
   */
  getDescription(): string;
}

/**
 * Serialized command data for localStorage persistence.
 */
export type CommandData =
  | {
      type: 'delete';
      timestamp: number;
      noteId: string;
      data: DeleteCommandData;
    }
  | {
      type: 'create';
      timestamp: number;
      noteId: string;
      data: CreateCommandData;
    }
  | {
      type: 'rename-title';
      timestamp: number;
      noteId: string;
      data: RenameTitleCommandData;
    }
  | {
      type: 'move-folder';
      timestamp: number;
      noteId: string;
      data: MoveFolderCommandData;
    };

/**
 * Data for DeleteCommand - stores full note snapshot for restoration.
 */
export interface DeleteCommandData {
  noteId: string;
  snapshot: {
    title: string;
    content: string;
    folder_path: string;
    version: number;
  };
}

/**
 * Data for CreateCommand - stores note details for undo (deletion).
 */
export interface CreateCommandData {
  noteId: string;
  title: string;
  content: string;
  folder_path: string;
}

/**
 * Data for RenameTitleCommand - stores old and new titles.
 */
export interface RenameTitleCommandData {
  noteId: string;
  oldTitle: string;
  newTitle: string;
  version: number;
}

/**
 * Data for MoveFolderCommand - stores old and new folder paths.
 */
export interface MoveFolderCommandData {
  noteId: string;
  oldFolder: string;
  newFolder: string;
  version: number;
}
