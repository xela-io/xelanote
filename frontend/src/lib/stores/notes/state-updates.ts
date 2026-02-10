import type { Note } from '$lib/api';

export interface NoteStateDeps {
  getCurrentNote: () => Note | null;
  setCurrentNote: (note: Note | null) => void;
  getNotes: () => Note[];
  setNotes: (notes: Note[]) => void;
  setDirty: (dirty: boolean) => void;
  incrementSaveCounter: () => void;
  getAutoSaveTimeout: () => ReturnType<typeof setTimeout> | null;
  setAutoSaveTimeout: (handle: ReturnType<typeof setTimeout> | null) => void;
  setAutoSaveStatus: (status: 'idle' | 'pending' | 'saving' | 'saved' | 'error') => void;
  setAutoSaveError: (value: string | null) => void;
  clearError: () => void;
  clearBacklinks: () => void;
  clearLastSaved: () => void;
}

export function updateCurrentNoteContent(content: string, deps: NoteStateDeps) {
  const currentNote = deps.getCurrentNote();
  if (!currentNote) return;
  if (currentNote.content === content) return;
  deps.setCurrentNote({ ...currentNote, content });
  deps.setDirty(true);
  deps.incrementSaveCounter();
}

export function updateCurrentNoteTitle(title: string, deps: NoteStateDeps) {
  const currentNote = deps.getCurrentNote();
  if (!currentNote) return;
  if (currentNote.title === title) return;
  deps.setCurrentNote({ ...currentNote, title });
  deps.setDirty(true);
  deps.incrementSaveCounter();
}

export function updateCurrentNoteAIEnabled(aiEnabled: boolean, deps: NoteStateDeps) {
  const currentNote = deps.getCurrentNote();
  if (!currentNote) return;
  const updated = { ...currentNote, ai_enabled: aiEnabled };
  deps.setCurrentNote(updated);
  const notes = deps.getNotes();
  const idx = notes.findIndex((n) => n.id === updated.id);
  if (idx !== -1) {
    const nextNotes = [...notes];
    nextNotes[idx] = { ...nextNotes[idx], ai_enabled: aiEnabled };
    deps.setNotes(nextNotes);
  }
}

export function clearCurrentNote(deps: NoteStateDeps) {
  const timeout = deps.getAutoSaveTimeout();
  if (timeout) {
    clearTimeout(timeout);
    deps.setAutoSaveTimeout(null);
  }
  deps.setAutoSaveStatus('idle');
  deps.setAutoSaveError(null);
  deps.clearLastSaved();
  deps.setCurrentNote(null);
  deps.clearBacklinks();
  deps.setDirty(false);
  deps.clearError();
}

export function replaceTempId(tempId: string, realNote: Note, deps: NoteStateDeps) {
  deps.setNotes(deps.getNotes().map((note) => (note.id === tempId ? realNote : note)));
  if (deps.getCurrentNote()?.id === tempId) {
    deps.setCurrentNote(realNote);
  }
}
