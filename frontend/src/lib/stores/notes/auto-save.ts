import { SvelteDate } from 'svelte/reactivity';

export type AutoSaveStatus = 'idle' | 'pending' | 'saving' | 'saved' | 'error';

export interface TriggerAutoSaveDeps {
  getCurrentNoteId: () => string | null;
  isDirty: () => boolean;
  setStatus: (status: AutoSaveStatus) => void;
  setError: (error: string | null) => void;
  setLastAutoSave: (date: Date | null) => void;
  saveNote: () => Promise<void>;
  isConflictError: (err: unknown) => boolean;
  handleConflict: () => Promise<void>;
  conflictMessage: string;
  defaultError: string;
}

export async function triggerAutoSave(deps: TriggerAutoSaveDeps) {
  const noteIdAtStart = deps.getCurrentNoteId();
  if (!noteIdAtStart || !deps.isDirty()) return;

  deps.setStatus('saving');
  deps.setError(null);

  try {
    if (deps.getCurrentNoteId() !== noteIdAtStart) {
      console.log('Auto-save cancelled: note changed during debounce');
      deps.setStatus('idle');
      return;
    }

    await deps.saveNote();
    deps.setLastAutoSave(new SvelteDate());
  } catch (err) {
    if (deps.isConflictError(err)) {
      deps.setStatus('error');
      deps.setError(deps.conflictMessage);
      await deps.handleConflict();
    } else {
      deps.setStatus('error');
      deps.setError(err instanceof Error ? err.message : deps.defaultError);
    }
    console.error('Auto-save failed:', err);
  }
}

export interface ScheduleAutoSaveDeps {
  isEnabled: () => boolean;
  isDirty: () => boolean;
  isLoading: () => boolean;
  getCurrentNoteId: () => string | null;
  setStatus: (status: AutoSaveStatus) => void;
  getTimeout: () => ReturnType<typeof setTimeout> | null;
  setTimeoutHandle: (handle: ReturnType<typeof setTimeout> | null) => void;
  getDelay: () => number;
  trigger: () => void;
}

export function scheduleAutoSave(deps: ScheduleAutoSaveDeps) {
  if (!deps.isEnabled() || !deps.getCurrentNoteId() || !deps.isDirty() || deps.isLoading()) {
    return;
  }

  deps.setStatus('pending');

  const existing = deps.getTimeout();
  if (existing) clearTimeout(existing);

  deps.setTimeoutHandle(
    setTimeout(() => {
      deps.trigger();
    }, deps.getDelay())
  );
}

export interface CancelAutoSaveDeps {
  getTimeout: () => ReturnType<typeof setTimeout> | null;
  setTimeoutHandle: (handle: ReturnType<typeof setTimeout> | null) => void;
  getStatus: () => AutoSaveStatus;
  setStatus: (status: AutoSaveStatus) => void;
}

export function cancelAutoSave(deps: CancelAutoSaveDeps) {
  const existing = deps.getTimeout();
  if (existing) {
    clearTimeout(existing);
    deps.setTimeoutHandle(null);
  }
  if (deps.getStatus() === 'pending') {
    deps.setStatus('idle');
  }
}
