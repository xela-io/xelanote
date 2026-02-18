import { request } from './client';
import { withQuery } from './query';
import type { Note } from './types';

export async function listTrash(
  options: {
    limit?: number;
    cursor?: string;
  } = {}
): Promise<{ notes: Note[]; next_cursor?: string }> {
  return request(
    withQuery('/trash', (params) => {
      if (options.limit) params.set('limit', options.limit.toString());
      if (options.cursor) params.set('cursor', options.cursor);
    })
  );
}

export async function getTrashCount(): Promise<{ count: number }> {
  return request('/trash/count');
}

export async function restoreNote(id: string): Promise<Note> {
  return request(`/notes/${id}/restore`, {
    method: 'POST',
  });
}

export async function permanentlyDeleteNote(id: string): Promise<void> {
  return request(`/notes/${id}/permanent`, {
    method: 'DELETE',
  });
}

export async function emptyTrash(): Promise<{ deleted_count: number }> {
  return request('/trash', {
    method: 'DELETE',
  });
}
