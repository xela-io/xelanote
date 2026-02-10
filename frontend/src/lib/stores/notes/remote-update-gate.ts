import type { Note } from '$lib/api';

export interface RemoteUpdateGateDeps {
  hasPendingForNote: (noteId: string) => Promise<boolean>;
  onUpdate: (note: Note) => void;
}

export async function handleRemoteUpdateWithPendingCheck(
  remoteNote: Note,
  deps: RemoteUpdateGateDeps
) {
  try {
    const hasPending = await deps.hasPendingForNote(remoteNote.id);
    if (hasPending) {
      console.log(
        '[WebSocket] Skipping remote update for note with pending offline ops:',
        remoteNote.id
      );
      return;
    }
  } catch {
    // IndexedDB error - continue with normal flow
  }

  deps.onUpdate(remoteNote);
}
