import { request } from './client';
import type { FolderShare, NoteShare, SharedFolder, SharedNote, UserSearchResult } from './types';

export async function shareNote(
  noteId: string,
  identifier: string,
  role: string
): Promise<NoteShare> {
  return request<NoteShare>(`/notes/${noteId}/shares`, {
    method: 'POST',
    body: JSON.stringify({ identifier, role }),
  });
}

export async function getNoteShares(noteId: string): Promise<NoteShare[]> {
  const result = await request<{ shares: NoteShare[] }>(`/notes/${noteId}/shares`);
  return result.shares;
}

export async function updateShareRole(noteId: string, userId: number, role: string): Promise<void> {
  return request<void>(`/notes/${noteId}/shares/${userId}`, {
    method: 'PUT',
    body: JSON.stringify({ role }),
  });
}

export async function removeShare(noteId: string, userId: number): Promise<void> {
  return request<void>(`/notes/${noteId}/shares/${userId}`, {
    method: 'DELETE',
  });
}

export async function getSharedNotes(): Promise<SharedNote[]> {
  const result = await request<{ notes: SharedNote[] }>('/shared');
  return result.notes;
}

export async function getSharedNote(id: string): Promise<SharedNote> {
  return request<SharedNote>(`/shared/${id}`);
}

export async function updateSharedNote(
  id: string,
  title: string,
  content: string,
  expectedVersion: number
): Promise<SharedNote> {
  return request<SharedNote>(`/shared/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ title, content, expected_version: expectedVersion }),
  });
}

export async function searchUsers(query: string): Promise<UserSearchResult[]> {
  const result = await request<{ users: UserSearchResult[] }>(
    `/users/search?q=${encodeURIComponent(query)}`
  );
  return result.users;
}

export async function shareFolder(
  folderId: number,
  identifier: string,
  role: string
): Promise<FolderShare> {
  return request<FolderShare>(`/folders/${folderId}/shares`, {
    method: 'POST',
    body: JSON.stringify({ identifier, role }),
  });
}

export async function getFolderShares(folderId: number): Promise<FolderShare[]> {
  const result = await request<{ shares: FolderShare[] }>(`/folders/${folderId}/shares`);
  return result.shares;
}

export async function updateFolderShareRole(
  folderId: number,
  userId: number,
  role: string
): Promise<void> {
  return request<void>(`/folders/${folderId}/shares/${userId}`, {
    method: 'PUT',
    body: JSON.stringify({ role }),
  });
}

export async function removeFolderShare(folderId: number, userId: number): Promise<void> {
  return request<void>(`/folders/${folderId}/shares/${userId}`, {
    method: 'DELETE',
  });
}

export async function getSharedFolders(): Promise<SharedFolder[]> {
  const result = await request<{ folders: SharedFolder[] }>('/shared/folders');
  return result.folders;
}

export async function getSharedFolderNotes(folderId: number): Promise<SharedNote[]> {
  const result = await request<{ notes: SharedNote[] }>(`/shared/folders/${folderId}/notes`);
  return result.notes;
}

export async function placeSharedNote(noteId: string, folderId: number): Promise<void> {
  return request<void>(`/shared/${noteId}/placement`, {
    method: 'POST',
    body: JSON.stringify({ folder_id: folderId }),
  });
}

export async function removePlacement(noteId: string): Promise<void> {
  return request<void>(`/shared/${noteId}/placement`, {
    method: 'DELETE',
  });
}
