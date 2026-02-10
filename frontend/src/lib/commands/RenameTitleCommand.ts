import * as api from '$lib/api';

import type { Command, CommandData, RenameTitleCommandData } from './types';

/**
 * RenameTitleCommand - Renames a note's title with link refactoring.
 *
 * Execute: Renames the note to the new title
 * Undo: Renames the note back to the old title
 */
export class RenameTitleCommand implements Command {
  private data: RenameTitleCommandData;

  constructor(data: RenameTitleCommandData) {
    this.data = data;
  }

  async execute(): Promise<boolean> {
    try {
      await api.renameNote(this.data.noteId, this.data.newTitle);
      return true;
    } catch (error) {
      console.error('RenameTitleCommand execute failed:', error);
      return false;
    }
  }

  async undo(): Promise<boolean> {
    try {
      // Rename back to old title
      await api.renameNote(this.data.noteId, this.data.oldTitle);
      return true;
    } catch (error) {
      console.error('RenameTitleCommand undo failed:', error);
      return false;
    }
  }

  serialize(): CommandData {
    return {
      type: 'rename-title',
      timestamp: Date.now(),
      noteId: this.data.noteId,
      data: this.data,
    };
  }

  getDescription(): string {
    return `Rename note "${this.data.oldTitle}" to "${this.data.newTitle}"`;
  }
}
