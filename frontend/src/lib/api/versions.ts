import { request } from './client';
import { withQuery } from './query';
import type { CompareResponse, Note, NoteVersion, VersionListResponse } from './types';

export async function listVersions(
  noteId: string,
  options: { limit?: number; cursor?: string } = {}
): Promise<VersionListResponse> {
  return request(
    withQuery(`/notes/${noteId}/versions`, (params) => {
      if (options.limit) params.set('limit', options.limit.toString());
      if (options.cursor) params.set('cursor', options.cursor);
    })
  );
}

export async function getVersion(noteId: string, version: number): Promise<NoteVersion> {
  return request(`/notes/${noteId}/versions/${version}`);
}

export async function compareVersions(
  noteId: string,
  v1: number,
  v2: number
): Promise<CompareResponse> {
  return request(
    withQuery(`/notes/${noteId}/versions/compare`, (params) => {
      params.set('v1', v1.toString());
      params.set('v2', v2.toString());
    })
  );
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
