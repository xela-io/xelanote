import type { EditorView } from '@codemirror/view';

import { ApiError, getNote, updateNoteAIEnabled } from '$lib/api';

interface SaveNoteDeps {
  editorView?: EditorView;
  saveNote: () => Promise<void>;
  getCurrentNoteId: () => string | undefined;
  reloadNote: (id: string) => Promise<void>;
  toast: {
    warning: (message: string, opts?: { label?: string; handler?: () => void }) => void;
    error: (message: string) => void;
  };
  strings: {
    conflictWarning: (version: number) => string;
    conflictLoadRemote: string;
    errorRemote: string;
    errorSave: string;
  };
}

interface AutoSaveDeps {
  getAutoSaveEnabled: () => boolean;
  setAutoSaveEnabled: (value: boolean) => void;
  getIsDirty: () => boolean;
  scheduleAutoSave: () => void;
}

interface TitleDeps {
  updateTitle: (title: string) => void;
  scheduleAutoSave: () => void;
}

interface AIToggleDeps {
  getCurrentNote: () => { id: string; ai_enabled: boolean } | null;
  updateCurrentAI: (value: boolean) => void;
  reloadTree: () => Promise<void>;
  toast: {
    success: (message: string) => void;
    info: (message: string) => void;
    error: (message: string) => void;
  };
  strings: {
    enabled: string;
    disabled: string;
    error: string;
  };
}

interface EncryptionToggleDeps {
  getIsEncrypted: () => boolean;
  confirm: (opts: {
    title: string;
    message: string;
    confirmText: string;
    cancelText: string;
    variant?: 'danger';
  }) => Promise<boolean>;
  toggleEncryption: () => Promise<void>;
  toast: {
    success: (message: string) => void;
    error: (message: string) => void;
  };
  strings: {
    decryptTitle: string;
    decryptMessage: string;
    decryptConfirm: string;
    encryptTitle: string;
    encryptMessage: string;
    encryptConfirm: string;
    cancel: string;
    decrypted: string;
    encrypted: string;
    error: string;
  };
}

export async function handleSaveNote(deps: SaveNoteDeps) {
  try {
    const hadFocus = deps.editorView?.hasFocus;

    await deps.saveNote();

    if (hadFocus && deps.editorView) {
      deps.editorView.focus();
    }
  } catch (e) {
    if (e instanceof ApiError && e.status === 409) {
      const noteId = deps.getCurrentNoteId();
      if (noteId) {
        try {
          const latest = await getNote(noteId);
          deps.toast.warning(deps.strings.conflictWarning(latest.version), {
            label: deps.strings.conflictLoadRemote,
            handler: () => deps.reloadNote(noteId),
          });
        } catch {
          deps.toast.error(deps.strings.errorRemote);
        }
      }
    } else {
      deps.toast.error(deps.strings.errorSave);
    }
  }
}

export function handleTitleInput(e: Event, deps: TitleDeps) {
  const title = (e.currentTarget as HTMLInputElement).value;
  deps.updateTitle(title);
  deps.scheduleAutoSave();
}

export function handleAutoSaveToggle(deps: AutoSaveDeps) {
  const next = !deps.getAutoSaveEnabled();
  deps.setAutoSaveEnabled(next);
  if (next && deps.getIsDirty()) {
    deps.scheduleAutoSave();
  }
}

export async function handleAIToggle(deps: AIToggleDeps) {
  const currentNote = deps.getCurrentNote();
  if (!currentNote) return;

  const newValue = !currentNote.ai_enabled;

  try {
    await updateNoteAIEnabled(currentNote.id, newValue);
    deps.updateCurrentAI(newValue);
    await deps.reloadTree();
    if (newValue) {
      deps.toast.success(deps.strings.enabled);
    } else {
      deps.toast.info(deps.strings.disabled);
    }
  } catch {
    deps.toast.error(deps.strings.error);
  }
}

export async function handleEncryptionToggle(deps: EncryptionToggleDeps) {
  const isCurrentlyEncrypted = deps.getIsEncrypted();

  const confirmed = await deps.confirm({
    title: isCurrentlyEncrypted ? deps.strings.decryptTitle : deps.strings.encryptTitle,
    message: isCurrentlyEncrypted ? deps.strings.decryptMessage : deps.strings.encryptMessage,
    confirmText: isCurrentlyEncrypted ? deps.strings.decryptConfirm : deps.strings.encryptConfirm,
    cancelText: deps.strings.cancel,
    variant: isCurrentlyEncrypted ? undefined : 'danger',
  });
  if (!confirmed) return;

  try {
    await deps.toggleEncryption();
    if (isCurrentlyEncrypted) {
      deps.toast.success(deps.strings.decrypted);
    } else {
      deps.toast.success(deps.strings.encrypted);
    }
  } catch {
    deps.toast.error(deps.strings.error);
  }
}
