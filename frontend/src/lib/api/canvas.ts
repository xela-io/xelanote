import { request } from './client';
import type { Note } from './types';

/**
 * List all canvas notes for the current user.
 */
export async function listCanvasNotes(): Promise<Note[]> {
  const data = await request<{ notes: Note[] }>('/canvas');
  return data.notes || [];
}

/**
 * Export a canvas note as a .canvas file (JSON Canvas format).
 * Returns the raw JSON string for download.
 */
export async function exportCanvas(noteId: string): Promise<string> {
  return request<string>(`/canvas/${noteId}/export`);
}
