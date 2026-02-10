import type { Note } from '$lib/api';
import type { DeleteCommand } from '$lib/commands/DeleteCommand';

interface CommonStrings {
  confirmTitle: string;
  cancelText: string;
}

interface WikilinkDeps {
  goto: (path: string) => void;
  confirm: (opts: {
    title: string;
    message: string;
    confirmText: string;
    cancelText: string;
  }) => Promise<boolean>;
  getCurrentNote: () => Note | null;
  getAllNotes: () => Note[];
  createNote: (title: string, content: string, folderPath: string) => Promise<Note>;
  loadFolders: () => Promise<void>;
  strings: CommonStrings & {
    createMissingMessage: string;
    createMissingConfirmText: string;
  };
}

interface DeleteDeps {
  goto: (path: string) => void;
  confirm: (opts: {
    title: string;
    message: string;
    confirmText: string;
    cancelText: string;
    variant?: 'danger';
  }) => Promise<boolean>;
  createDeleteCommand: (snapshot: {
    noteId: string;
    snapshot: {
      title: string;
      content: string;
      folder_path: string;
      version: number;
    };
  }) => DeleteCommand;
  executeCommand: (cmd: DeleteCommand) => Promise<boolean>;
  undoCommand: () => Promise<boolean>;
  getCurrentNote: () => Note | null;
  loadNotes: () => Promise<void>;
  loadTree: () => Promise<void>;
  clearCurrentNote: () => void;
  incrementTrash: () => void;
  decrementTrash: () => void;
  toast: {
    error: (message: string) => void;
    success: (message: string) => void;
    warning: (message: string, opts?: { label?: string; handler?: () => void }) => void;
    undoToast: (message: string, handler: () => Promise<void>) => void;
  };
  strings: CommonStrings & {
    deleteConfirmMessage: string;
    deleteConfirmText: string;
    deleteError: string;
    noteTrashed: string;
    noteRestored: string;
    restoreError: string;
  };
}

export function exportNoteMarkdown(note: Note | null) {
  if (!note) return;

  const title = note.title || 'Untitled';
  const content = note.content || '';
  const markdown = `---\ntitle: "${title.replace(/"/g, '\\"')}"\n---\n\n${content}`;

  const sanitizedTitle = title.replace(/[<>:"/\\|?*]/g, '_').trim() || 'note';
  const blob = new Blob([markdown], { type: 'text/markdown;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `${sanitizedTitle}.md`;
  a.click();
  URL.revokeObjectURL(url);
}

export async function handleWikilinkClick(title: string, deps: WikilinkDeps) {
  const allNotes = deps.getAllNotes();
  const targetNote = allNotes.find((n) => n.title.toLowerCase() === title.toLowerCase());

  if (targetNote) {
    deps.goto(`/note/${targetNote.id}`);
    return;
  }

  const confirmed = await deps.confirm({
    title: deps.strings.confirmTitle,
    message: deps.strings.createMissingMessage,
    confirmText: deps.strings.createMissingConfirmText,
    cancelText: deps.strings.cancelText,
  });

  if (!confirmed) return;

  const currentFolder = deps.getCurrentNote()?.folder_path || '/';
  const note = await deps.createNote(title, '', currentFolder);
  await deps.loadFolders();
  deps.goto(`/note/${note.id}`);
}

export async function handleDeleteNote(deps: DeleteDeps) {
  const currentNote = deps.getCurrentNote();
  if (!currentNote) return;

  const confirmed = await deps.confirm({
    title: deps.strings.confirmTitle,
    message: deps.strings.deleteConfirmMessage,
    confirmText: deps.strings.deleteConfirmText,
    cancelText: deps.strings.cancelText,
    variant: 'danger',
  });

  if (!confirmed) return;

  try {
    const snapshot = {
      noteId: currentNote.id,
      snapshot: {
        title: currentNote.title,
        content: currentNote.content,
        folder_path: currentNote.folder_path,
        version: currentNote.version,
      },
    };

    const deleteCmd = deps.createDeleteCommand(snapshot);
    const success = await deps.executeCommand(deleteCmd);

    if (!success) {
      deps.toast.error(deps.strings.deleteError);
      return;
    }

    deps.clearCurrentNote();
    await deps.loadNotes();
    await deps.loadTree();
    deps.incrementTrash();

    deps.toast.undoToast(deps.strings.noteTrashed, async () => {
      const undoSuccess = await deps.undoCommand();
      if (undoSuccess) {
        deps.toast.success(deps.strings.noteRestored);
        await deps.loadNotes();
        await deps.loadTree();
        deps.decrementTrash();
      } else {
        deps.toast.error(deps.strings.restoreError);
      }
    });

    deps.goto('/');
  } catch {
    deps.toast.error(deps.strings.deleteError);
  }
}
