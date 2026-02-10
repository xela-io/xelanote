import { request } from './client';
import type { Folder, FolderInfo } from './types';

// Folders API (legacy)

export async function getFoldersLegacy(): Promise<{ folders: FolderInfo[] }> {
  return request('/folders-legacy');
}

// Folders API (new folders table)

export async function getFolders(): Promise<{ folders: Folder[] }> {
  return request('/folders');
}

export async function createFolder(path: string): Promise<Folder> {
  return request('/folders', {
    method: 'POST',
    body: JSON.stringify({ path }),
  });
}

export async function moveFolder(id: number, newParentPath: string): Promise<void> {
  return request(`/folders/${id}/move`, {
    method: 'PUT',
    body: JSON.stringify({ new_parent_path: newParentPath }),
  });
}

export async function deleteFolder(id: number): Promise<void> {
  return request(`/folders/${id}`, {
    method: 'DELETE',
  });
}

export async function renameFolder(id: number, newName: string): Promise<void> {
  return request(`/folders/${id}/rename`, {
    method: 'PUT',
    body: JSON.stringify({ new_name: newName }),
  });
}

export async function reorderFolders(parentID: number | null, items: number[]): Promise<void> {
  return request(`/folders/reorder`, {
    method: 'POST',
    body: JSON.stringify({ parent_id: parentID, items }),
  });
}

export async function reorderNotes(folderPath: string, items: string[]): Promise<void> {
  return request(`/notes/reorder`, {
    method: 'POST',
    body: JSON.stringify({ folder_path: folderPath, items }),
  });
}

export async function updateFolderColor(id: number, color: string | null): Promise<void> {
  return request(`/folders/${id}/color`, {
    method: 'PUT',
    body: JSON.stringify({ color }),
  });
}

export async function updateNoteColor(id: string, color: string | null): Promise<void> {
  return request(`/notes/${id}/color`, {
    method: 'PUT',
    body: JSON.stringify({ color }),
  });
}
