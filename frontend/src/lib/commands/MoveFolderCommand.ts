import { moveNote } from '$lib/stores/notes.svelte';

import type { Command, CommandData, MoveFolderCommandData } from './types';

/**
 * MoveFolderCommand - Moves a note to a different folder.
 *
 * Execute: Moves the note to the new folder
 * Undo: Moves the note back to the old folder
 */
export class MoveFolderCommand implements Command {
  private data: MoveFolderCommandData;

  constructor(data: MoveFolderCommandData) {
    this.data = data;
  }

  async execute(): Promise<boolean> {
    try {
      await moveNote(this.data.noteId, this.data.newFolder);
      return true;
    } catch (error) {
      console.error('MoveFolderCommand execute failed:', error);
      return false;
    }
  }

  async undo(): Promise<boolean> {
    try {
      await moveNote(this.data.noteId, this.data.oldFolder);
      return true;
    } catch (error) {
      console.error('MoveFolderCommand undo failed:', error);
      return false;
    }
  }

  serialize(): CommandData {
    return {
      type: 'move-folder',
      timestamp: Date.now(),
      noteId: this.data.noteId,
      data: this.data,
    };
  }

  getDescription(): string {
    return `Move note from "${this.data.oldFolder}" to "${this.data.newFolder}"`;
  }
}
