import type { OfflineNoteContext } from '../offline/types';
import { request } from './client';
import { withQuery } from './query';
import type { Backlink, Job, Note, NotePayload, NoteUserState, RenameResult } from './types';

export async function listNotes(
  options: {
    limit?: number;
    cursor?: string;
    folder?: string;
    fields?: 'slim';
    updated_since?: string;
  } = {}
): Promise<{ notes: Note[]; next_cursor?: string; sync_token?: string }> {
  return request(
    withQuery('/notes', (params) => {
      if (options.limit) params.set('limit', options.limit.toString());
      if (options.cursor) params.set('cursor', options.cursor);
      if (options.folder) params.set('folder', options.folder);
      if (options.fields) params.set('fields', options.fields);
      if (options.updated_since) params.set('updated_since', options.updated_since);
    })
  );
}

export async function getNote(id: string): Promise<Note> {
  return request(`/notes/${id}`);
}

export async function createNote(
  data: NotePayload,
  offlineContext?: OfflineNoteContext
): Promise<Note> {
  return request('/notes', {
    method: 'POST',
    body: JSON.stringify(data),
    _offlineContext: offlineContext,
    _offlineAllowed: !!offlineContext,
  });
}

export async function updateNote(
  id: string,
  data: NotePayload,
  version: number,
  offlineContext?: OfflineNoteContext
): Promise<Note> {
  return request(`/notes/${id}`, {
    method: 'PUT',
    headers: {
      'If-Match': version.toString(),
    },
    body: JSON.stringify(data),
    _offlineContext: offlineContext,
    _offlineAllowed: !!offlineContext,
  });
}

export async function moveNote(id: string, folderPath: string, version: number): Promise<Note> {
  const note = await getNote(id);
  const payload: NotePayload = { title: note.title, folder_path: folderPath };

  if (note.content_encrypted && note.encrypted_content && note.wrapped_dek) {
    // Preserve all encryption fields for encrypted notes
    payload.encrypted_content = note.encrypted_content;
    payload.wrapped_dek = note.wrapped_dek;
    payload.encryption_metadata = note.encryption_metadata;
    payload.encrypted_title = note.encrypted_title;
    payload.title_encrypted = note.title_encrypted;
  } else {
    payload.content = note.content;
  }

  return updateNote(id, payload, version);
}

export async function deleteNote(id: string, offlineAllowed = false): Promise<void> {
  return request(`/notes/${id}`, {
    method: 'DELETE',
    _offlineAllowed: offlineAllowed,
  });
}

export async function renameNote(id: string, newTitle: string): Promise<RenameResult> {
  return request(`/notes/${id}/rename`, {
    method: 'POST',
    body: JSON.stringify({ newTitle }),
  });
}

export async function renameNoteAsync(
  id: string,
  newTitle: string
): Promise<{ job_id: string; status: string }> {
  return request(
    withQuery(`/notes/${id}/rename`, (params) => params.set('async', 'true')),
    {
      method: 'POST',
      body: JSON.stringify({ newTitle }),
    }
  );
}

export async function getJobStatus(jobId: string): Promise<Job> {
  return request(`/jobs/${jobId}`);
}

export async function getBacklinks(noteId: string): Promise<{ backlinks: Backlink[] }> {
  return request(`/notes/${noteId}/backlinks`);
}

export async function getNoteUserState(noteId: string): Promise<NoteUserState> {
  return request(`/notes/${noteId}/user-state`);
}

export async function updateNoteUserCollapseState(
  noteId: string,
  state: Record<string, boolean>
): Promise<{ status: string }> {
  return request(`/notes/${noteId}/user-state`, {
    method: 'PUT',
    body: JSON.stringify({ collapse_state: state }),
  });
}
