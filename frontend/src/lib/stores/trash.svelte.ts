/**
 * Trash Store - Manages deleted notes (trash functionality)
 *
 * This store provides state management for the trash page and trash count badge.
 */

import * as api from '$lib/api';
import type { Note } from '$lib/api';

// State
let trashedNotes = $state<Note[]>([]);
let trashCount = $state<number>(0);
let isLoading = $state(false);
let error = $state<string | null>(null);
let nextCursor = $state<string | undefined>(undefined);

/**
 * Loads deleted notes from the trash.
 */
export async function loadTrash(cursor?: string): Promise<void> {
  isLoading = true;
  error = null;

  try {
    const result = await api.listTrash({ limit: 50, cursor });

    if (cursor) {
      // Append to existing notes (pagination)
      trashedNotes = [...trashedNotes, ...result.notes];
    } else {
      // Replace notes (initial load)
      trashedNotes = result.notes;
    }

    nextCursor = result.next_cursor;
  } catch (err) {
    error = err instanceof Error ? err.message : 'Failed to load trash';
    console.error('Failed to load trash:', err);
  } finally {
    isLoading = false;
  }
}

/**
 * Loads the count of deleted notes for the badge.
 */
export async function loadTrashCount(): Promise<void> {
  try {
    const result = await api.getTrashCount();
    trashCount = result.count;
  } catch (err) {
    console.error('Failed to load trash count:', err);
  }
}

/**
 * Restores a note from trash.
 */
export async function restoreNote(id: string): Promise<boolean> {
  try {
    await api.restoreNote(id);

    // Remove from local state
    trashedNotes = trashedNotes.filter((note) => note.id !== id);
    trashCount = Math.max(0, trashCount - 1);

    return true;
  } catch (err) {
    error = err instanceof Error ? err.message : 'Failed to restore note';
    console.error('Failed to restore note:', err);
    return false;
  }
}

/**
 * Permanently deletes a note (hard delete).
 */
export async function permanentlyDeleteNote(id: string): Promise<boolean> {
  try {
    await api.permanentlyDeleteNote(id);

    // Remove from local state
    trashedNotes = trashedNotes.filter((note) => note.id !== id);
    trashCount = Math.max(0, trashCount - 1);

    return true;
  } catch (err) {
    error = err instanceof Error ? err.message : 'Failed to delete note';
    console.error('Failed to permanently delete note:', err);
    return false;
  }
}

/**
 * Empties the trash (permanently deletes all deleted notes).
 */
export async function emptyTrash(): Promise<boolean> {
  try {
    await api.emptyTrash();

    // Clear local state
    trashedNotes = [];
    trashCount = 0;

    return true;
  } catch (err) {
    error = err instanceof Error ? err.message : 'Failed to empty trash';
    console.error('Failed to empty trash:', err);
    return false;
  }
}

/**
 * Increments the trash count (used when a note is deleted).
 */
export function incrementTrashCount(): void {
  trashCount = trashCount + 1;
}

/**
 * Decrements the trash count (used when a note is restored).
 */
export function decrementTrashCount(): void {
  trashCount = Math.max(0, trashCount - 1);
}

// Getters for reactive access
export function getTrashedNotes(): Note[] {
  return trashedNotes;
}

export function getTrashCount(): number {
  return trashCount;
}

export function getIsLoading(): boolean {
  return isLoading;
}

export function getError(): string | null {
  return error;
}

export function hasMore(): boolean {
  return nextCursor !== undefined;
}

export function getNextCursor(): string | undefined {
  return nextCursor;
}

// Derived reactive state
export const trash = {
  get notes() {
    return trashedNotes;
  },
  get count() {
    return trashCount;
  },
  get isLoading() {
    return isLoading;
  },
  get error() {
    return error;
  },
  get hasMore() {
    return nextCursor !== undefined;
  },
};
