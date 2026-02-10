import { request } from './client';
import type { CompareResponse, Note, NoteVersion, VersionListResponse } from './types';

export async function listVersions(
  noteId: string,
  options: { limit?: number; cursor?: string } = {}
): Promise<VersionListResponse> {
  const params = new URLSearchParams();
  if (options.limit) params.set('limit', options.limit.toString());
  if (options.cursor) params.set('cursor', options.cursor);

  const query = params.toString();
  return request(`/notes/${noteId}/versions${query ? '?' + query : ''}`);
}

export async function getVersion(noteId: string, version: number): Promise<NoteVersion> {
  return request(`/notes/${noteId}/versions/${version}`);
}

export async function compareVersions(
  noteId: string,
  v1: number,
  v2: number
): Promise<CompareResponse> {
  const params = new URLSearchParams({
    v1: v1.toString(),
    v2: v2.toString(),
  });
  return request(`/notes/${noteId}/versions/compare?${params}`);
}

export async function restoreVersion(
  noteId: string,
  version: number,
  currentVersion: number
): Promise<Note> {
  return request(`/notes/${noteId}/versions/${version}/restore`, {
    method: 'POST',
    headers: {
      'If-Match': currentVersion.toString(),
    },
  });
}
