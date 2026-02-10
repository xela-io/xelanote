import * as api from '$lib/api';

import type { Command, CommandData, DeleteCommandData } from './types';

/**
 * DeleteCommand - Soft-deletes a note and stores a snapshot for undo.
 *
 * Execute: Soft-deletes the note (moves to trash)
 * Undo: Restores the note and updates it with the snapshot data
 */
export class DeleteCommand implements Command {
  private data: DeleteCommandData;

  constructor(data: DeleteCommandData) {
    this.data = data;
  }

  async execute(): Promise<boolean> {
    try {
      await api.deleteNote(this.data.noteId);
      return true;
    } catch (error) {
      console.error('DeleteCommand execute failed:', error, 'Note ID:', this.data.noteId);
      // Log more details about the error
      if (error instanceof Error) {
        console.error('Error message:', error.message);
        console.error('Error stack:', error.stack);
      }
      return false;
    }
  }

  async undo(): Promise<boolean> {
    try {
      // First, restore the note from trash
      await api.restoreNote(this.data.noteId);

      // Then update it with the snapshot data to restore the exact state
      // Fetch fresh version to avoid conflicts
      const freshNote = await api.getNote(this.data.noteId);

      await api.updateNote(
        this.data.noteId,
        {
          title: this.data.snapshot.title,
          content: this.data.snapshot.content,
          folder_path: this.data.snapshot.folder_path,
        },
        freshNote.version
      );

      return true;
    } catch (error) {
      console.error('DeleteCommand undo failed:', error);
      return false;
    }
  }

  serialize(): CommandData {
    return {
      type: 'delete',
      timestamp: Date.now(),
      noteId: this.data.noteId,
      data: this.data,
    };
  }

  getDescription(): string {
    return `Delete note "${this.data.snapshot.title}"`;
  }
}
