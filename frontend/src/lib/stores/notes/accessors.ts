import { SvelteDate } from 'svelte/reactivity';

import type { Backlink, Note } from '$lib/api';

export type AutoSaveStatus = 'idle' | 'pending' | 'saving' | 'saved' | 'error';

export interface NotesAccessorsDeps {
  getCurrentNote: () => Note | null;
  getBacklinks: () => Backlink[];
  getIsLoading: () => boolean;
  getIsSaving: () => boolean;
  getError: () => string | null;
  setError: (value: string | null) => void;
  getIsDirty: () => boolean;
  setDirty: (dirty: boolean) => void;
  getNotes: () => Note[];
  getNotesLoading: () => boolean;
  getAutoSaveStatus: () => AutoSaveStatus;
  getLastAutoSave: () => Date | null;
  getAutoSaveError: () => string | null;
}

export function createNotesAccessors(deps: NotesAccessorsDeps) {
  return {
    getCurrentNote: deps.getCurrentNote,
    getBacklinks: deps.getBacklinks,
    getIsLoading: deps.getIsLoading,
    getIsSaving: deps.getIsSaving,
    getError: deps.getError,
    clearError: () => deps.setError(null),
    getIsDirty: deps.getIsDirty,
    setDirty: deps.setDirty,
    getNotes: deps.getNotes,
    getRecentNotes: (limit = 5) =>
      [...deps.getNotes()]
        .sort(
          (a, b) => new SvelteDate(b.updated_at).getTime() - new SvelteDate(a.updated_at).getTime()
        )
        .slice(0, limit),
    getNotesLoading: deps.getNotesLoading,
    getAutoSaveStatus: deps.getAutoSaveStatus,
    getLastAutoSave: deps.getLastAutoSave,
    getAutoSaveError: deps.getAutoSaveError,
  };
}
