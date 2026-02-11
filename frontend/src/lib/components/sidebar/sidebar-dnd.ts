import type { DropPosition, TouchDragData } from '$lib/actions/touchdrag';
import type { FolderTreeNode, NoteTreeNode } from '$lib/stores/tree.svelte';
import { validateDrop } from '$lib/utils/tree-drop-validation';

interface SidebarDndDeps {
  moveFolder: (id: number, path: string) => Promise<void>;
  moveNote: (id: string, path: string) => Promise<unknown>;
  loadTree: () => Promise<void>;
  reorderFolders: (parentID: number | null, folderIds: number[]) => Promise<void>;
  reorderNotes: (folderPath: string, noteIds: string[]) => Promise<void>;
  findParentOfNodeById: (
    type: 'folder' | 'note',
    id: string | number
  ) => FolderTreeNode | null;
  getFolderChildren: (parent: FolderTreeNode) => FolderTreeNode[];
  getNoteChildren: (parent: FolderTreeNode) => NoteTreeNode[];
  alert: (opts: { title: string; message: string; variant: 'danger' | 'warning' }) => Promise<void>;
  strings: {
    errorTitle: string;
    errorMovingTopLevel: string;
    moveError: string;
  };
}

type SidebarDraggedItem =
  | { type: 'folder'; id: number; path: string }
  | { type: 'note'; id: string; folder_path: string };

function parseSidebarDraggedItem(raw: string): SidebarDraggedItem | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }

  if (!parsed || typeof parsed !== 'object') return null;
  const data = parsed as { type?: unknown; id?: unknown; path?: unknown; folder_path?: unknown };

  if (data.type === 'folder') {
    if (typeof data.id !== 'number' || typeof data.path !== 'string') return null;
    return { type: 'folder', id: data.id, path: data.path };
  }
  if (data.type === 'note') {
    if (typeof data.id !== 'string' || typeof data.folder_path !== 'string') return null;
    return { type: 'note', id: data.id, folder_path: data.folder_path };
  }
  return null;
}

export function handleDropZoneDragOver(e: DragEvent, setActive: (active: boolean) => void) {
  if (e.dataTransfer?.types.includes('application/x-xelanote-item')) {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    setActive(true);
  }
}

export function handleDropZoneDragLeave(setActive: (active: boolean) => void) {
  setActive(false);
}

export async function handleDropZoneDrop(
  e: DragEvent,
  deps: SidebarDndDeps,
  setActive: (active: boolean) => void
) {
  e.preventDefault();
  setActive(false);

  const data = e.dataTransfer?.getData('application/x-xelanote-item');
  if (!data) return;

  try {
    const dragData = parseSidebarDraggedItem(data);
    if (!dragData) throw new Error('invalid drag payload');

    if (dragData.type === 'folder') {
      await deps.moveFolder(dragData.id, '/');
    } else if (dragData.type === 'note') {
      await deps.moveNote(dragData.id, '/');
      await deps.loadTree();
    }
  } catch {
    await deps.alert({
      title: deps.strings.errorTitle,
      message: deps.strings.errorMovingTopLevel,
      variant: 'danger',
    });
  }
}

export async function handleTouchDrop(
  dragData: TouchDragData,
  targetData: TouchDragData,
  position: DropPosition,
  deps: SidebarDndDeps,
  translate: (key: string) => string
) {
  const validation = validateDrop(
    {
      type: dragData.type,
      id: dragData.id,
      path: dragData.path,
      folder_path: dragData.folder_path,
    },
    {
      type: targetData.type,
      id: targetData.id,
      path: targetData.path,
      folder_path: targetData.folder_path,
    },
    position
  );

  if (!validation.valid) {
    if (validation.errorKey) {
      await deps.alert({
        title: deps.strings.errorTitle,
        message: translate(validation.errorKey),
        variant: 'warning',
      });
    }
    return;
  }

  try {
    if (targetData.type === 'root-dropzone') {
      if (dragData.type === 'folder') {
        await deps.moveFolder(Number(dragData.id), '/');
      } else {
        await deps.moveNote(dragData.id, '/');
        await deps.loadTree();
      }
      return;
    }

    if (position === 'before' || position === 'after') {
      if (dragData.type === 'folder' && targetData.type === 'folder') {
        const reordered = await handleTouchFolderReorder(
          Number(dragData.id),
          Number(targetData.id),
          position,
          deps
        );
        if (reordered) return;
      }

      if (dragData.type === 'note' && targetData.type === 'note') {
        if (dragData.folder_path === targetData.folder_path) {
          await handleTouchNoteReorder(dragData.id, targetData.id, position, deps);
          return;
        }
        await deps.moveNote(dragData.id, targetData.folder_path!);
        await deps.loadTree();
        return;
      }
    }

    if (targetData.type === 'folder') {
      if (dragData.type === 'note') {
        await deps.moveNote(dragData.id, targetData.path!);
        await deps.loadTree();
      } else if (dragData.type === 'folder') {
        await deps.moveFolder(Number(dragData.id), targetData.path!);
      }
    }
  } catch {
    await deps.alert({
      title: deps.strings.errorTitle,
      message: deps.strings.moveError,
      variant: 'danger',
    });
  }
}

async function handleTouchFolderReorder(
  draggedId: number,
  targetId: number,
  position: 'before' | 'after',
  deps: SidebarDndDeps
): Promise<boolean> {
  const parent = deps.findParentOfNodeById('folder', targetId);
  if (!parent) return false;

  const draggedParent = deps.findParentOfNodeById('folder', draggedId);
  if (!draggedParent || draggedParent !== parent) return false;

  const siblings = deps.getFolderChildren(parent);
  const draggedIndex = siblings.findIndex((s) => s.id === draggedId);
  const targetIndex = siblings.findIndex((s) => s.id === targetId);
  if (draggedIndex === -1 || targetIndex === -1) return false;

  const newOrder = [...siblings];
  const [draggedItem] = newOrder.splice(draggedIndex, 1);
  const insertIndex = position === 'before' ? targetIndex : targetIndex + 1;
  newOrder.splice(draggedIndex < targetIndex ? insertIndex - 1 : insertIndex, 0, draggedItem);

  const folderIds = newOrder.map((s) => s.id).filter((id) => id !== 0);
  const parentID: number | null = parent.id === 0 ? 1 : parent.id || null;
  await deps.reorderFolders(parentID, folderIds);
  return true;
}

async function handleTouchNoteReorder(
  draggedId: string,
  targetId: string,
  position: 'before' | 'after',
  deps: SidebarDndDeps
) {
  const parent = deps.findParentOfNodeById('note', targetId);
  if (!parent) return;

  const siblings = deps.getNoteChildren(parent);
  const draggedIndex = siblings.findIndex((s) => s.id === draggedId);
  const targetIndex = siblings.findIndex((s) => s.id === targetId);
  if (draggedIndex === -1 || targetIndex === -1) return;

  const newOrder = [...siblings];
  const [draggedItem] = newOrder.splice(draggedIndex, 1);
  const insertIndex = position === 'before' ? targetIndex : targetIndex + 1;
  newOrder.splice(draggedIndex < targetIndex ? insertIndex - 1 : insertIndex, 0, draggedItem);

  const noteIds = newOrder.map((s) => s.id);
  const folderPath = parent.path || '/';
  await deps.reorderNotes(folderPath, noteIds);
}
