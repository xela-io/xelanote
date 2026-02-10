import type { Note } from '$lib/api';

export interface RenameResult {
  note: Note;
  updated_note_count: number;
}

export interface RenameNoteDeps {
  getCurrentNote: () => Note | null;
  getNotes: () => Note[];
  setCurrentNote: (note: Note | null) => void;
  setNotes: (notes: Note[]) => void;
  setDirty: (dirty: boolean) => void;
  setError: (value: string | null) => void;
  getBacklinks: (id: string) => Promise<{ backlinks: unknown[] }>;
  renameNote: (id: string, title: string) => Promise<RenameResult>;
  renameNoteAsync: (id: string, title: string) => Promise<{ job_id: string }>;
  getJobStatus: (id: string) => Promise<{ status: string; result?: unknown; error?: string }>;
  notifyInfo: (message: string) => void;
  notifySuccess: (message: string) => void;
  notifyError: (message: string) => void;
  loadNotes: () => Promise<void>;
}

function isRenameResult(value: unknown): value is RenameResult {
  if (!value || typeof value !== 'object') return false;
  const result = value as { note?: unknown; updated_note_count?: unknown };
  if (!result.note || typeof result.note !== 'object') return false;
  return typeof result.updated_note_count === 'number';
}

async function pollJobCompletion(
  getJobStatus: RenameNoteDeps['getJobStatus'],
  jobId: string,
  maxAttempts = 60
) {
  for (let i = 0; i < maxAttempts; i++) {
    const job = await getJobStatus(jobId);
    if (job.status === 'completed' || job.status === 'failed') {
      return job;
    }
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error('Job timeout - operation took too long');
}

export async function renameCurrentNote(newTitle: string, deps: RenameNoteDeps) {
  const currentNote = deps.getCurrentNote();
  if (!currentNote) return;

  const snapshot = {
    note: { ...currentNote },
    notesList: [...deps.getNotes()],
  };

  const optimistic = { ...currentNote, title: newTitle };
  deps.setCurrentNote(optimistic);
  deps.setNotes(deps.getNotes().map((n) => (n.id === optimistic.id ? optimistic : n)));
  deps.setDirty(false);
  deps.setError(null);

  try {
    const backlinksResult = await deps.getBacklinks(snapshot.note.id);
    const backlinksCount = backlinksResult.backlinks.length;
    const useAsync = backlinksCount > 100;

    if (useAsync) {
      deps.notifyInfo('Renaming note in background...');
      const { job_id } = await deps.renameNoteAsync(snapshot.note.id, newTitle);
      const job = await pollJobCompletion(deps.getJobStatus, job_id);

      if (job.status === 'completed') {
        if (!isRenameResult(job.result)) {
          throw new Error('Unexpected job result from rename');
        }
        deps.setCurrentNote(job.result.note);
        await deps.loadNotes();
        deps.notifySuccess(
          `Note renamed successfully (${job.result.updated_note_count} references updated)`
        );
        return job.result;
      }

      deps.setCurrentNote(snapshot.note);
      deps.setNotes(snapshot.notesList);
      const errorMsg = job.error || 'Failed to rename note';
      deps.notifyError(errorMsg);
      throw new Error(errorMsg);
    }

    const result = await deps.renameNote(snapshot.note.id, newTitle);
    deps.setCurrentNote(result.note);
    await deps.loadNotes();
    return result;
  } catch (err) {
    deps.setCurrentNote(snapshot.note);
    deps.setNotes(snapshot.notesList);
    deps.setError(err instanceof Error ? err.message : 'Failed to rename note');
    throw err;
  }
}
