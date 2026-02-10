import type { Command, CommandData, CreateCommandData } from './types';
import * as api from '$lib/api';

/**
 * CreateCommand - Creates a new note.
 *
 * Execute: Creates the note
 * Undo: Deletes the created note (soft-delete)
 */
export class CreateCommand implements Command {
  private data: CreateCommandData;

  constructor(data: CreateCommandData) {
    this.data = data;
  }

  async execute(): Promise<boolean> {
    try {
      const note = await api.createNote({
        title: this.data.title,
        content: this.data.content,
        folder_path: this.data.folder_path,
      });

      // Store the created note ID for undo
      this.data.noteId = note.id;

      return true;
    } catch (error) {
      console.error('CreateCommand execute failed:', error);
      return false;
    }
  }

  async undo(): Promise<boolean> {
    try {
      // Soft-delete the created note
      await api.deleteNote(this.data.noteId);
      return true;
    } catch (error) {
      console.error('CreateCommand undo failed:', error);
      return false;
    }
  }

  serialize(): CommandData {
    return {
      type: 'create',
      timestamp: Date.now(),
      noteId: this.data.noteId,
      data: this.data,
    };
  }

  getDescription(): string {
    return `Create note "${this.data.title}"`;
  }
}
