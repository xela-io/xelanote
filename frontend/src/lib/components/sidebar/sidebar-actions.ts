export interface SidebarActionsDeps {
  getSelectedFolderPath: () => string | null;
  createNote: (title: string, content: string, folderPath: string) => Promise<{ id: string }>;
  createFolder: (path: string) => Promise<void>;
  loadTree: () => Promise<void>;
  closeSidebarOnMobile: () => void;
  goto: (path: string) => void;
  confirm: (opts: {
    title: string;
    message: string;
    confirmText: string;
    cancelText: string;
  }) => Promise<boolean>;
  alert: (opts: { title: string; message: string; variant: 'danger' }) => Promise<void>;
  stopAutoLock: () => void;
  logout: () => Promise<void>;
  strings: {
    confirmTitle: string;
    confirmLogout: string;
    logout: string;
    cancel: string;
    errorTitle: string;
    createFolderError: (error: string) => string;
  };
}

export async function handleCreateNoteConfirm(
  title: string,
  deps: SidebarActionsDeps
) {
  const selectedPath = deps.getSelectedFolderPath();
  const folderPath = selectedPath || '/';
  const note = await deps.createNote(title, '', folderPath);
  await deps.loadTree();
  deps.goto(`/note/${note.id}`);
  deps.closeSidebarOnMobile();
}

export async function handleCreateFolderConfirm(
  path: string,
  deps: SidebarActionsDeps
) {
  try {
    await deps.createFolder(path);
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    await deps.alert({
      title: deps.strings.errorTitle,
      message: deps.strings.createFolderError(message),
      variant: 'danger',
    });
  }
}

export async function handleLogout(deps: SidebarActionsDeps) {
  const confirmed = await deps.confirm({
    title: deps.strings.confirmTitle,
    message: deps.strings.confirmLogout,
    confirmText: deps.strings.logout,
    cancelText: deps.strings.cancel,
  });

  if (!confirmed) return;

  try {
    deps.stopAutoLock();
    await deps.logout();
    window.location.href = '/login';
  } catch {
    window.location.href = '/login';
  }
}
