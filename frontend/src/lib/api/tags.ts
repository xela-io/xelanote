import { request } from './client';
import type { Tag } from './types';

export async function getTags(): Promise<Tag[]> {
  const result = await request<Tag[] | null>('/tags');
  return result || [];
}

export async function getNoteTags(noteId: string): Promise<Tag[]> {
  const result = await request<Tag[] | null>(`/notes/${noteId}/tags`);
  return result || [];
}

export async function setNoteTags(noteId: string, tags: string[]): Promise<Tag[]> {
  const result = await request<Tag[] | null>(`/notes/${noteId}/tags`, {
    method: 'PUT',
    body: JSON.stringify({ tags }),
  });
  return result || [];
}

export async function deleteTag(tagId: number): Promise<void> {
  return request(`/tags/${tagId}`, {
    method: 'DELETE',
  });
}
